package mailprovider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// TMailorProvider implements Provider for tmailor.com.
// It requires no configuration — addresses are generated automatically.
//
// API: https://tmailor.com/api
// Config keys: (none required)
//
//	api_url – override base URL (default: https://web2.tmailor.com/)
type TMailorProvider struct {
	apiURL string
	client *http.Client
}

type TMailorResp[T any] struct {
	Msg  string `json:"msg"`
	Data T      `json:"data"`
}

type TMailorCreate struct {
	Msg   string `json:"msg"`
	Email string `json:"email"`
	Token string `json:"accesstoken"`
}

type TMailorListItem struct {
	ID          string `json:"id"`
	Subject     string `json:"subject"`
	SenderName  string `json:"sender_name"`
	SenderEmail string `json:"sender_email"`
	EmailToken  string `json:"email_id"`
	BodyHtml    string
}

type TMailorListResp TMailorResp[map[string]TMailorListItem]

type TMailorRead struct {
	Id   string `json:"id"`
	Body string `json:"body"`
}

func (m *TMailorListItem) fullText() string {
	return m.Subject + " " + m.BodyHtml
}

const defaultTMailorURL = "https://tmailor.com/"

// NewTMailor returns a TMailorProvider.
func NewTMailor(config map[string]string) (*TMailorProvider, error) {
	u := config["api_url"]
	if u == "" {
		u = defaultTMailorURL
	}
	return &TMailorProvider{
		apiURL: strings.TrimRight(u, "/"),
		client: &http.Client{Timeout: 20 * time.Second, Transport: buildTransport(config["proxy_url"])},
	}, nil
}

func (p *TMailorProvider) do(ctx context.Context, method, path string, body io.Reader, headers map[string]string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, p.apiURL+path, body)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36 Edg/146.0.0.0")

	defaultHeaders := map[string]string{
		"origin":             "https://tmailor.com",
		"priority":           "u=1, i",
		"referer":            "https://tmailor.com/",
		"sec-ch-ua":          `"Chromium";v="146", "Not-A.Brand";v="24", "Microsoft Edge";v="146"`,
		"sec-ch-ua-mobile":   "?0",
		"sec-ch-ua-platform": `"Windows"`,
		"sec-fetch-dest":     "empty",
		"sec-fetch-mode":     "cors",
		"sec-fetch-site":     "same-site",
	}
	for k, v := range defaultHeaders {
		req.Header.Set(k, v)
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return data, fmt.Errorf("tmailor.com: HTTP %d: %s", resp.StatusCode, string(data))
	}
	return data, nil
}

// GetEmail creates a new temporary inbox and returns its credentials.
func (p *TMailorProvider) GetEmail(ctx context.Context) (*MailAccount, error) {
	body := map[string]string{
		"action": "newemail",
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	data, err := p.do(ctx, http.MethodPost, "/api", strings.NewReader(string(bodyBytes)), map[string]string{})
	if err != nil {
		return nil, fmt.Errorf("tmailor.com GetEmail: %w", err)
	}
	var resp TMailorCreate
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("tmailor.com GetEmail parse: %w (body: %s)", err, string(data))
	}
	email := resp.Email
	if email == "" {
		return nil, fmt.Errorf("tmailor.com GetEmail: empty address in response: %s", string(data))
	}
	return &MailAccount{
		Email:     email,
		AccountID: resp.Token,
		Token:     resp.Token,
	}, nil
}

func (p *TMailorProvider) getDetail(ctx context.Context, emailCode, emailToken string, token string) (string, error) {
	body := map[string]string{
		"accesstoken":  token,
		"action":       "read",
		"currentToken": token,
		"email_code":   emailCode,
		"email_token":  emailToken,
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	data, err := p.do(ctx, http.MethodGet, "/api", strings.NewReader(string(bodyBytes)), map[string]string{})
	if err != nil {
		return "", err
	}
	var resp TMailorResp[TMailorRead]
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", fmt.Errorf("tmailor.com list parse: %w", err)
	}
	return resp.Data.Body, nil

}

func (p *TMailorProvider) listMessages(ctx context.Context, token string) ([]TMailorListItem, error) {
	body := map[string]string{
		"accesstoken":  token,
		"action":       "listinbox",
		"currentToken": token,
		"fbToken":      "",
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	data, err := p.do(ctx, http.MethodPost, "/api", strings.NewReader(string(bodyBytes)), map[string]string{})
	if err != nil {
		return nil, err
	}
	var resp TMailorListResp
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("tmailor.com list parse: %w", err)
	}
	result := make([]TMailorListItem, 0, len(resp.Data))
	for _, item := range resp.Data {
		detail, err := p.getDetail(ctx, item.ID, item.EmailToken, token)
		if err != nil {
			log.Printf("tmailor.com: failed to get message detail for ID %s: %v", item.ID, err)
			continue
		}
		item.BodyHtml = detail
		result = append(result, item)
	}
	return result, nil
}

// WaitForCode polls for a new message and extracts an OTP code.
func (p *TMailorProvider) WaitForCode(ctx context.Context, account *MailAccount, keyword string, timeoutSec int) (string, error) {
	deadline := time.Now().Add(time.Duration(timeoutSec) * time.Second)
	seen := map[string]bool{}

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(3 * time.Second):
		}

		msgs, err := p.listMessages(ctx, account.Token)
		if err != nil {
			continue
		}
		log.Printf("收到邮件数量：%d", len(msgs))
		for _, m := range msgs {
			if m.ID == "" || seen[m.ID] {
				continue
			}
			seen[m.ID] = true
			content := m.fullText()
			if keyword != "" && !strings.Contains(strings.ToLower(content), strings.ToLower(keyword)) {
				continue
			}
			if code := extractCode(content); code != "" {
				return code, nil
			}
		}
	}
	return "", fmt.Errorf("tmailor.com: timeout waiting for OTP after %ds", timeoutSec)
}

// WaitForLink polls for a new message and extracts a verification URL.
func (p *TMailorProvider) WaitForLink(ctx context.Context, account *MailAccount, keyword string, timeoutSec int) (string, error) {
	deadline := time.Now().Add(time.Duration(timeoutSec) * time.Second)
	seen := map[string]bool{}

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(3 * time.Second):
		}

		msgs, err := p.listMessages(ctx, account.Token)
		if err != nil {
			continue
		}
		for _, m := range msgs {
			if m.ID == "" || seen[m.ID] {
				continue
			}
			seen[m.ID] = true
			content := m.fullText()
			if link := extractLink(content, keyword); link != "" {
				return link, nil
			}
		}
	}
	return "", fmt.Errorf("tmailor.com: timeout waiting for link after %ds", timeoutSec)
}
