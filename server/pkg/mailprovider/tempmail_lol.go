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

// TempMailLolProvider implements Provider for tempmail.lol.
// It requires no configuration — addresses are generated automatically.
//
// API: https://tempmail.lol/api
// Config keys: (none required)
//
//	api_url – override base URL (default: https://api.tempmail.lol/v2)
type TempMailLolProvider struct {
	apiURL string
	client *http.Client
}

const defaultTempMailLolURL = "https://api.tempmail.lol/v2"

// NewTempMailLol returns a TempMailLolProvider.
func NewTempMailLol(config map[string]string) *TempMailLolProvider {
	u := config["api_url"]
	if u == "" {
		u = defaultTempMailLolURL
	}
	return &TempMailLolProvider{
		apiURL: strings.TrimRight(u, "/"),
		client: &http.Client{Timeout: 20 * time.Second, Transport: buildTransport(config["proxy_url"])},
	}
}

func (p *TempMailLolProvider) do(ctx context.Context, method, path string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, p.apiURL+path, body)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
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
		return data, fmt.Errorf("tempmail.lol: HTTP %d: %s", resp.StatusCode, string(data))
	}
	return data, nil
}

// GetEmail creates a new temporary inbox and returns its credentials.
func (p *TempMailLolProvider) GetEmail(ctx context.Context) (*MailAccount, error) {
	data, err := p.do(ctx, http.MethodPost, "/inbox/create", strings.NewReader("{}"))
	if err != nil {
		return nil, fmt.Errorf("tempmail.lol GetEmail: %w", err)
	}
	var resp struct {
		Address string `json:"address"`
		Email   string `json:"email"`
		Token   string `json:"token"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("tempmail.lol GetEmail parse: %w (body: %s)", err, string(data))
	}
	email := resp.Address
	if email == "" {
		email = resp.Email
	}
	if email == "" {
		return nil, fmt.Errorf("tempmail.lol GetEmail: empty address in response: %s", string(data))
	}
	return &MailAccount{
		Email:     email,
		AccountID: resp.Token,
		Token:     resp.Token,
	}, nil
}

type tempMailLolMsg struct {
	ID      string `json:"_id"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
	HTML    string `json:"html"`
	Date    int64  `json:"date"`
}

func (m *tempMailLolMsg) fullText() string {
	return m.Subject + " " + m.Body + " " + m.HTML
}

func (p *TempMailLolProvider) listMessages(ctx context.Context, token string) ([]tempMailLolMsg, error) {
	data, err := p.do(ctx, http.MethodGet, "/inbox?token="+token, nil)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Emails []tempMailLolMsg `json:"emails"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("tempmail.lol list parse: %w", err)
	}
	return resp.Emails, nil
}

// WaitForCode polls for a new message and extracts an OTP code.
func (p *TempMailLolProvider) WaitForCode(ctx context.Context, account *MailAccount, keyword string, timeoutSec int) (string, error) {
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
	return "", fmt.Errorf("tempmail.lol: timeout waiting for OTP after %ds", timeoutSec)
}

// WaitForLink polls for a new message and extracts a verification URL.
func (p *TempMailLolProvider) WaitForLink(ctx context.Context, account *MailAccount, keyword string, timeoutSec int) (string, error) {
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
	return "", fmt.Errorf("tempmail.lol: timeout waiting for link after %ds", timeoutSec)
}
