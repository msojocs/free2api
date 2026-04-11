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

// CFWorkerProvider implements Provider for the cloudflare_temp_email project.
// See: https://github.com/dreamhunter2333/cloudflare_temp_email
//
// Config keys:
//
//	api_url     – Cloudflare Worker backend API URL (required)
//	admin_token – admin password / x-custom-auth value (required)
//	domain      – email domain for generated addresses (required)
type CFWorkerProvider struct {
	apiURL     string
	adminToken string
	domain     string
	client     *http.Client
}

// NewCFWorker returns a CFWorkerProvider.
func NewCFWorker(config map[string]string) *CFWorkerProvider {
	return &CFWorkerProvider{
		apiURL:     strings.TrimRight(config["api_url"], "/"),
		adminToken: config["admin_token"],
		domain:     config["domain"],
		client:     &http.Client{Timeout: 20 * time.Second, Transport: buildTransport(config["proxy_url"])},
	}
}

func (p *CFWorkerProvider) newRequest(ctx context.Context, method, path string, payload interface{}, bearerToken string) (*http.Response, error) {
	var body io.Reader
	if payload != nil {
		b, _ := json.Marshal(payload)
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, p.apiURL+path, body)
	if err != nil {
		return nil, err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	if bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+bearerToken)
	} else if p.adminToken != "" {
		req.Header.Set("x-custom-auth", p.adminToken)
	}
	return p.client.Do(req)
}

func (p *CFWorkerProvider) do(ctx context.Context, method, path string, payload interface{}, bearerToken string) ([]byte, error) {
	resp, err := p.newRequest(ctx, method, path, payload, bearerToken)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return data, fmt.Errorf("cfworker: HTTP %d: %s", resp.StatusCode, string(data))
	}
	return data, nil
}

// GetEmail creates a new temporary email address via the admin API.
func (p *CFWorkerProvider) GetEmail(ctx context.Context) (*MailAccount, error) {
	if p.apiURL == "" || p.domain == "" {
		return nil, fmt.Errorf("cfworker: api_url and domain are required")
	}

	name := randStr(12)
	address := fmt.Sprintf("%s@%s", name, p.domain)

	// Create address
	data, err := p.do(ctx, http.MethodPost, "/api/new_address", map[string]string{
		"cf_email": address,
	}, "")
	if err != nil {
		return nil, fmt.Errorf("cfworker: create address: %w", err)
	}

	var createResp struct {
		Email string `json:"email"`
		Msg   string `json:"msg"`
	}
	_ = json.Unmarshal(data, &createResp)
	email := createResp.Email
	if email == "" {
		email = address
	}

	// Authenticate to get a per-address JWT
	authData, err := p.do(ctx, http.MethodPost, "/api/auth", map[string]string{
		"cf_email": email,
	}, "")
	if err != nil {
		return nil, fmt.Errorf("cfworker: auth: %w", err)
	}

	var authResp struct {
		JWT   string `json:"jwt"`
		Token string `json:"token"`
	}
	if err := json.Unmarshal(authData, &authResp); err != nil {
		return nil, fmt.Errorf("cfworker: parse auth response: %s", string(authData))
	}
	token := authResp.JWT
	if token == "" {
		token = authResp.Token
	}

	return &MailAccount{
		Email:     email,
		AccountID: email,
		Token:     token,
	}, nil
}

type cfMail struct {
	ID      string `json:"id"`
	Subject string `json:"subject"`
	Source  string `json:"source"`
	Message string `json:"message"`
	HTML    string `json:"html"`
}

func (m *cfMail) fullText() string {
	return m.Subject + " " + m.Message + " " + m.HTML
}

func (p *CFWorkerProvider) listMails(ctx context.Context, token string) ([]cfMail, error) {
	data, err := p.do(ctx, http.MethodGet, "/api/mails?limit=20", nil, token)
	if err != nil {
		return nil, err
	}
	var mails []cfMail
	if err := json.Unmarshal(data, &mails); err != nil {
		return nil, fmt.Errorf("cfworker: parse mails: %w (body: %s)", err, string(data))
	}
	return mails, nil
}

func (p *CFWorkerProvider) snapshot(ctx context.Context, token string) map[string]bool {
	ids := make(map[string]bool)
	mails, _ := p.listMails(ctx, token)
	for _, m := range mails {
		ids[m.ID] = true
	}
	return ids
}

// WaitForCode polls for a new email and extracts an OTP code.
func (p *CFWorkerProvider) WaitForCode(ctx context.Context, account *MailAccount, keyword string, timeoutSec int) (string, error) {
	deadline := time.Now().Add(time.Duration(timeoutSec) * time.Second)
	seen := p.snapshot(ctx, account.Token)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(3 * time.Second):
		}

		mails, err := p.listMails(ctx, account.Token)
		if err != nil {
			continue
		}
		for _, m := range mails {
			if seen[m.ID] {
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
	return "", fmt.Errorf("cfworker: timeout waiting for OTP after %ds", timeoutSec)
}

// WaitForLink polls for a new email and extracts a verification URL.
func (p *CFWorkerProvider) WaitForLink(ctx context.Context, account *MailAccount, keyword string, timeoutSec int) (string, error) {
	deadline := time.Now().Add(time.Duration(timeoutSec) * time.Second)
	seen := p.snapshot(ctx, account.Token)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(3 * time.Second):
		}

		mails, err := p.listMails(ctx, account.Token)
		if err != nil {
			continue
		}
		for _, m := range mails {
			if seen[m.ID] {
				continue
			}
			seen[m.ID] = true

			content := m.fullText()
			if link := extractLink(content, keyword); link != "" {
				return link, nil
			}
		}
	}
	return "", fmt.Errorf("cfworker: timeout waiting for link after %ds", timeoutSec)
}
