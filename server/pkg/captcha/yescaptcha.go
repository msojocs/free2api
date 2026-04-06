package captcha

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const yesCaptchaBaseURL = "https://api.yescaptcha.com"

// YesCaptcha implements Solver using the YesCaptcha API.
// See: https://yescaptcha.com/i/0xlYxj (docs at /docs)
type YesCaptcha struct {
	clientKey string
}

// NewYesCaptcha returns a YesCaptcha solver with the given client key.
func NewYesCaptcha(clientKey string) *YesCaptcha {
	return &YesCaptcha{clientKey: clientKey}
}

func (y *YesCaptcha) postJSON(ctx context.Context, path string, payload interface{}) (map[string]interface{}, error) {
	b, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, yesCaptchaBaseURL+path, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("yescaptcha: parse response: %s", string(data))
	}
	return result, nil
}

// SolveTurnstile submits a TurnstileTaskProxyless task and polls for the token.
func (y *YesCaptcha) SolveTurnstile(ctx context.Context, pageURL, siteKey string) (string, error) {
	result, err := y.postJSON(ctx, "/createTask", map[string]interface{}{
		"clientKey": y.clientKey,
		"task": map[string]string{
			"type":       "TurnstileTaskProxyless",
			"websiteURL": pageURL,
			"websiteKey": siteKey,
		},
	})
	if err != nil {
		return "", fmt.Errorf("yescaptcha: createTask: %w", err)
	}

	taskID, _ := result["taskId"]
	if taskID == nil {
		return "", fmt.Errorf("yescaptcha: no taskId in response: %v", result)
	}

	for i := 0; i < 60; i++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(3 * time.Second):
		}

		res, err := y.postJSON(ctx, "/getTaskResult", map[string]interface{}{
			"clientKey": y.clientKey,
			"taskId":    taskID,
		})
		if err != nil {
			continue
		}

		status, _ := res["status"].(string)
		if status == "ready" {
			if sol, ok := res["solution"].(map[string]interface{}); ok {
				if token, ok := sol["token"].(string); ok && token != "" {
					return token, nil
				}
			}
			return "", fmt.Errorf("yescaptcha: ready but no token: %v", res)
		}
		if errID, _ := res["errorId"].(float64); errID != 0 {
			return "", fmt.Errorf("yescaptcha: error %v: %v", res["errorCode"], res["errorDescription"])
		}
	}
	return "", fmt.Errorf("yescaptcha: SolveTurnstile timed out")
}
