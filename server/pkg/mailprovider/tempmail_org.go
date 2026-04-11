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

// TempMailOrgProvider implements Provider for temp-mail.org.
// It requires no configuration — addresses are generated automatically.
//
// API: https://temp-mail.org/api
// Config keys: (none required)
//
//	api_url – override base URL (default: https://web2.temp-mail.org/)
type TempMailOrgProvider struct {
	apiURL string
	client *http.Client
}

const defaultTempMailOrgURL = "https://web2.temp-mail.org/"

// NewTempMailOrg returns a TempMailOrgProvider.
func NewTempMailOrg(config map[string]string) *TempMailOrgProvider {
	u := config["api_url"]
	if u == "" {
		u = defaultTempMailOrgURL
	}
	return &TempMailOrgProvider{
		apiURL: strings.TrimRight(u, "/"),
		client: &http.Client{Timeout: 20 * time.Second, Transport: buildTransport(config["proxy_url"])},
	}
}

func (p *TempMailOrgProvider) do(ctx context.Context, method, path string, body io.Reader, headers map[string]string) ([]byte, error) {
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
		"origin":             "https://temp-mail.org",
		"priority":           "u=1, i",
		"referer":            "https://temp-mail.org/",
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
		return data, fmt.Errorf("temp-mail.org: HTTP %d: %s", resp.StatusCode, string(data))
	}
	return data, nil
}

// GetEmail creates a new temporary inbox and returns its credentials.
func (p *TempMailOrgProvider) GetEmail(ctx context.Context) (*MailAccount, error) {
	data, err := p.do(ctx, http.MethodPost, "/mailbox", strings.NewReader("{}"), map[string]string{})
	if err != nil {
		return nil, fmt.Errorf("temp-mail.org GetEmail: %w", err)
	}
	var resp struct {
		Email string `json:"mailbox"`
		Token string `json:"token"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("temp-mail.org GetEmail parse: %w (body: %s)", err, string(data))
	}
	email := resp.Email
	if email == "" {
		return nil, fmt.Errorf("temp-mail.org GetEmail: empty address in response: %s", string(data))
	}
	return &MailAccount{
		Email:     email,
		AccountID: resp.Token,
		Token:     resp.Token,
	}, nil
}

type TempMailOrgMsg struct {
	ID          string `json:"_id"`
	Subject     string `json:"subject"`
	BodyPreview string `json:"bodyPreview"`
	ReceivedAt  int64  `json:"receivedAt"`
	DetailHtml  string
}

func (m *TempMailOrgMsg) fullText() string {
	return m.Subject + " " + m.BodyPreview + " " + m.DetailHtml
}

func (p *TempMailOrgProvider) getDetail(ctx context.Context, msgId string, token string) (string, error) {
	headers := map[string]string{
		"Authorization": "Bearer " + token,
	}
	data, err := p.do(ctx, http.MethodGet, "/messages/"+msgId, nil, headers)
	if err != nil {
		return "", err
	}
	var resp struct {
		BodyHtml string `json:"bodyHtml"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", fmt.Errorf("temp-mail.org list parse: %w", err)
	}
	return resp.BodyHtml, nil

}

func (p *TempMailOrgProvider) listMessages(ctx context.Context, token string) ([]TempMailOrgMsg, error) {
	headers := map[string]string{
		"Authorization": "Bearer " + token,
	}
	data, err := p.do(ctx, http.MethodGet, "/messages", nil, headers)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Emails []TempMailOrgMsg `json:"messages"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("temp-mail.org list parse: %w", err)
	}
	for i := range resp.Emails {
		detail, err := p.getDetail(ctx, resp.Emails[i].ID, token)
		if err != nil {
			log.Printf("temp-mail.org: failed to get message detail for ID %s: %v", resp.Emails[i].ID, err)
			continue
		}
		resp.Emails[i].DetailHtml = detail
	}
	return resp.Emails, nil
}

// WaitForCode polls for a new message and extracts an OTP code.
func (p *TempMailOrgProvider) WaitForCode(ctx context.Context, account *MailAccount, keyword string, timeoutSec int) (string, error) {
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
	return "", fmt.Errorf("temp-mail.org: timeout waiting for OTP after %ds", timeoutSec)
}

// WaitForLink polls for a new message and extracts a verification URL.
func (p *TempMailOrgProvider) WaitForLink(ctx context.Context, account *MailAccount, keyword string, timeoutSec int) (string, error) {
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
	return "", fmt.Errorf("temp-mail.org: timeout waiting for link after %ds", timeoutSec)
}
