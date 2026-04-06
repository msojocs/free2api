package mailprovider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// MailTmProvider implements Provider for mail.tm-compatible APIs,
// including DuckMail (https://api.duckmail.sbs) which shares the same API surface.
//
// Config keys:
//
//	api_url – base URL; defaults to https://api.duckmail.sbs
type MailTmProvider struct {
	apiURL string
}

const defaultMailTmURL = "https://api.duckmail.sbs"

// NewMailTm returns a MailTmProvider. If config["api_url"] is empty the
// default DuckMail endpoint is used.
func NewMailTm(config map[string]string) *MailTmProvider {
	u := config["api_url"]
	if u == "" {
		u = defaultMailTmURL
	}
	return &MailTmProvider{apiURL: strings.TrimRight(u, "/")}
}

func (p *MailTmProvider) doGet(ctx context.Context, path, token string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.apiURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func (p *MailTmProvider) doPost(ctx context.Context, path string, payload interface{}, token string) ([]byte, int, error) {
	b, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.apiURL+path, bytes.NewReader(b))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	return data, resp.StatusCode, err
}

func (p *MailTmProvider) getFirstDomain(ctx context.Context) (string, error) {
	data, err := p.doGet(ctx, "/domains?page=1", "")
	if err != nil {
		return "", err
	}
	var result struct {
		Members []struct {
			Domain string `json:"domain"`
		} `json:"hydra:member"`
	}
	if err := json.Unmarshal(data, &result); err != nil || len(result.Members) == 0 {
		return "", fmt.Errorf("mailtm: no domains available (response: %s)", string(data))
	}
	return result.Members[0].Domain, nil
}

// GetEmail creates a new temporary address and returns its MailAccount.
func (p *MailTmProvider) GetEmail(ctx context.Context) (*MailAccount, error) {
	domain, err := p.getFirstDomain(ctx)
	if err != nil {
		return nil, err
	}

	username := fmt.Sprintf("%s%04d", randStr(8), randIntN(10000))
	password := randStr(8) + randAlphanumUpperStr(4) + fmt.Sprintf("%04d", randIntN(10000)) + "!"
	address := fmt.Sprintf("%s@%s", username, domain)

	// Create account
	_, status, err := p.doPost(ctx, "/accounts", map[string]string{
		"address":  address,
		"password": password,
	}, "")
	if err != nil {
		return nil, fmt.Errorf("mailtm: create account: %w", err)
	}
	if status >= 400 {
		return nil, fmt.Errorf("mailtm: create account HTTP %d", status)
	}

	// Get authentication token
	data, status, err := p.doPost(ctx, "/token", map[string]string{
		"address":  address,
		"password": password,
	}, "")
	if err != nil {
		return nil, fmt.Errorf("mailtm: get token: %w", err)
	}
	if status >= 400 {
		return nil, fmt.Errorf("mailtm: get token HTTP %d: %s", status, string(data))
	}

	var tok struct {
		Token string `json:"token"`
		ID    string `json:"id"`
	}
	if err := json.Unmarshal(data, &tok); err != nil || tok.Token == "" {
		return nil, fmt.Errorf("mailtm: parse token response: %s", string(data))
	}

	return &MailAccount{
		Email:     address,
		AccountID: tok.ID,
		Token:     tok.Token,
	}, nil
}

// mailTmMsg represents a message from the /messages or /messages/{id} endpoints.
type mailTmMsg struct {
	ID      string          `json:"id"`
	Subject string          `json:"subject"`
	Text    string          `json:"text"`
	HTML    json.RawMessage `json:"html"` // can be []string or string
}

func (m *mailTmMsg) htmlText() string {
	if len(m.HTML) == 0 {
		return ""
	}
	var arr []string
	if err := json.Unmarshal(m.HTML, &arr); err == nil {
		return strings.Join(arr, "\n")
	}
	var s string
	if err := json.Unmarshal(m.HTML, &s); err == nil {
		return s
	}
	return ""
}

func (m *mailTmMsg) fullText() string {
	return m.Subject + " " + m.Text + " " + m.htmlText()
}

type mailTmList struct {
	Members []mailTmMsg `json:"hydra:member"`
}

func (p *MailTmProvider) listMessages(ctx context.Context, token string) ([]mailTmMsg, error) {
	data, err := p.doGet(ctx, "/messages?page=1", token)
	if err != nil {
		return nil, err
	}
	var list mailTmList
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, fmt.Errorf("mailtm: parse message list: %w", err)
	}
	return list.Members, nil
}

func (p *MailTmProvider) getMessage(ctx context.Context, token, id string) (*mailTmMsg, error) {
	data, err := p.doGet(ctx, "/messages/"+id, token)
	if err != nil {
		return nil, err
	}
	var msg mailTmMsg
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("mailtm: parse message: %w", err)
	}
	return &msg, nil
}

func (p *MailTmProvider) snapshot(ctx context.Context, token string) map[string]bool {
	ids := make(map[string]bool)
	msgs, _ := p.listMessages(ctx, token)
	for _, m := range msgs {
		ids[m.ID] = true
	}
	return ids
}

// WaitForCode polls for a new message containing an OTP code.
func (p *MailTmProvider) WaitForCode(ctx context.Context, account *MailAccount, keyword string, timeoutSec int) (string, error) {
	deadline := time.Now().Add(time.Duration(timeoutSec) * time.Second)
	seen := p.snapshot(ctx, account.Token)

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
			if seen[m.ID] {
				continue
			}
			full, err := p.getMessage(ctx, account.Token, m.ID)
			if err != nil {
				seen[m.ID] = true
				continue
			}
			seen[full.ID] = true

			content := full.fullText()
			if keyword != "" && !strings.Contains(strings.ToLower(content), strings.ToLower(keyword)) {
				continue
			}
			if code := extractCode(content); code != "" {
				return code, nil
			}
		}
	}
	return "", fmt.Errorf("mailtm: timeout waiting for OTP code after %ds", timeoutSec)
}

// WaitForLink polls for a new message containing a verification URL.
func (p *MailTmProvider) WaitForLink(ctx context.Context, account *MailAccount, keyword string, timeoutSec int) (string, error) {
	deadline := time.Now().Add(time.Duration(timeoutSec) * time.Second)
	seen := p.snapshot(ctx, account.Token)

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
			if seen[m.ID] {
				continue
			}
			full, err := p.getMessage(ctx, account.Token, m.ID)
			if err != nil {
				seen[m.ID] = true
				continue
			}
			seen[full.ID] = true

			content := full.fullText()
			if link := extractLink(content, keyword); link != "" {
				return link, nil
			}
		}
	}
	return "", fmt.Errorf("mailtm: timeout waiting for link after %ds", timeoutSec)
}
