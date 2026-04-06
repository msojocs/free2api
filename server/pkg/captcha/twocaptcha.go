package captcha

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const twoCaptchaBaseURL = "https://2captcha.com"

// TwoCaptcha implements Solver using the 2Captcha API.
// See: https://2captcha.com/api-docs
type TwoCaptcha struct {
	apiKey string
}

// NewTwoCaptcha returns a TwoCaptcha solver with the given API key.
func NewTwoCaptcha(apiKey string) *TwoCaptcha {
	return &TwoCaptcha{apiKey: apiKey}
}

func (t *TwoCaptcha) postForm(ctx context.Context, path string, data url.Values) (map[string]interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, twoCaptchaBaseURL+path, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("2captcha: parse response: %s", string(body))
	}
	return result, nil
}

func (t *TwoCaptcha) getJSON(ctx context.Context, path string) (map[string]interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, twoCaptchaBaseURL+path, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("2captcha: parse response: %s", string(body))
	}
	return result, nil
}

// SolveTurnstile submits a Turnstile task to 2Captcha and polls for the solution.
func (t *TwoCaptcha) SolveTurnstile(ctx context.Context, pageURL, siteKey string) (string, error) {
	form := url.Values{
		"key":     {t.apiKey},
		"method":  {"turnstile"},
		"sitekey": {siteKey},
		"pageurl": {pageURL},
		"json":    {"1"},
	}
	result, err := t.postForm(ctx, "/in.php", form)
	if err != nil {
		return "", fmt.Errorf("2captcha: submit task: %w", err)
	}

	if status, _ := result["status"].(float64); status != 1 {
		return "", fmt.Errorf("2captcha: submit failed: %v", result)
	}
	taskID, _ := result["request"].(string)
	if taskID == "" {
		return "", fmt.Errorf("2captcha: no taskId in response: %v", result)
	}

	for i := 0; i < 60; i++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(3 * time.Second):
		}

		path := fmt.Sprintf("/res.php?key=%s&action=get&id=%s&json=1", t.apiKey, taskID)
		res, err := t.getJSON(ctx, path)
		if err != nil {
			continue
		}

		if status, _ := res["status"].(float64); status == 1 {
			if token, ok := res["request"].(string); ok && token != "" {
				return token, nil
			}
		}

		// Check for terminal error
		if req, _ := res["request"].(string); req != "CAPCHA_NOT_READY" && req != "CAPTCHA_NOT_READY" {
			return "", fmt.Errorf("2captcha: error: %v", res)
		}
	}
	return "", fmt.Errorf("2captcha: SolveTurnstile timed out")
}
