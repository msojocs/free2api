package mailprovider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"

	"golang.org/x/net/publicsuffix"
)

// FreemailProvider implements Provider for the self-hosted Freemail service.
// Project: https://github.com/idinging/freemail
//
// Supports two authentication modes:
//   - Admin token (Authorization: Bearer <admin_token>)
//   - Username/password (POST /api/login, then session cookie)
//
// Config keys:
//
//	api_url      – Freemail backend base URL (required)
//	admin_token  – admin bearer token (mutually exclusive with username/password)
//	username     – account username (used when admin_token is absent)
//	password     – account password (used when admin_token is absent)
type FreemailProvider struct {
	apiURL     string
	adminToken string
	username   string
	password   string
	proxyURL   string
}

// NewFreemail returns a FreemailProvider.
func NewFreemail(config map[string]string) *FreemailProvider {
	return &FreemailProvider{
		apiURL:     strings.TrimRight(config["api_url"], "/"),
		adminToken: config["admin_token"],
		username:   config["username"],
		password:   config["password"],
		proxyURL:   config["proxy_url"],
	}
}

// freemailSession holds an authenticated HTTP session.
type freemailSession struct {
	client     *http.Client
	apiURL     string
	adminToken string
}

func newFreemailSession(apiURL, adminToken, username, password, proxyURL string) (*freemailSession, error) {
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return nil, err
	}
	s := &freemailSession{
		client:     &http.Client{Jar: jar, Timeout: 20 * time.Second, Transport: buildTransport(proxyURL)},
		apiURL:     apiURL,
		adminToken: adminToken,
	}
	if adminToken == "" && username != "" && password != "" {
		if err := s.login(context.Background(), username, password); err != nil {
			return nil, fmt.Errorf("freemail: login: %w", err)
		}
	}
	return s, nil
}

func (s *freemailSession) do(ctx context.Context, method, path string, payload interface{}) ([]byte, error) {
	var body io.Reader
	if payload != nil {
		b, _ := json.Marshal(payload)
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, s.apiURL+path, body)
	if err != nil {
		return nil, err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	if s.adminToken != "" {
		req.Header.Set("Authorization", "Bearer "+s.adminToken)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return data, fmt.Errorf("freemail: HTTP %d %s: %s", resp.StatusCode, path, string(data))
	}
	return data, nil
}

func (s *freemailSession) login(ctx context.Context, username, password string) error {
	_, err := s.do(ctx, http.MethodPost, "/api/login", map[string]string{
		"username": username,
		"password": password,
	})
	return err
}

// GetEmail generates a new temporary email address.
func (p *FreemailProvider) GetEmail(ctx context.Context) (*MailAccount, error) {
	if p.apiURL == "" {
		return nil, fmt.Errorf("freemail: api_url is required")
	}
	sess, err := newFreemailSession(p.apiURL, p.adminToken, p.username, p.password, p.proxyURL)
	if err != nil {
		return nil, err
	}
	data, err := sess.do(ctx, http.MethodGet, "/api/generate", nil)
	if err != nil {
		return nil, fmt.Errorf("freemail GetEmail: %w", err)
	}
	var resp struct {
		Email string `json:"email"`
	}
	if err := json.Unmarshal(data, &resp); err != nil || resp.Email == "" {
		return nil, fmt.Errorf("freemail GetEmail: empty email in response: %s", string(data))
	}
	return &MailAccount{
		Email:     resp.Email,
		AccountID: resp.Email,
	}, nil
}

type freemailMsg struct {
	ID               string `json:"id"`
	Subject          string `json:"subject"`
	Preview          string `json:"preview"`
	VerificationCode string `json:"verification_code"`
}

func (m *freemailMsg) fullText() string {
	return m.Subject + " " + m.Preview + " " + m.VerificationCode
}

func (p *FreemailProvider) listMessages(ctx context.Context, email string) ([]freemailMsg, error) {
	sess, err := newFreemailSession(p.apiURL, p.adminToken, p.username, p.password, p.proxyURL)
	if err != nil {
		return nil, err
	}
	data, err := sess.do(ctx, http.MethodGet, "/api/emails?mailbox="+email+"&limit=20", nil)
	if err != nil {
		return nil, err
	}
	var msgs []freemailMsg
	if err := json.Unmarshal(data, &msgs); err != nil {
		return nil, fmt.Errorf("freemail list parse: %w (body: %s)", err, string(data))
	}
	return msgs, nil
}

// WaitForCode polls the Freemail inbox for an OTP code.
func (p *FreemailProvider) WaitForCode(ctx context.Context, account *MailAccount, keyword string, timeoutSec int) (string, error) {
	deadline := time.Now().Add(time.Duration(timeoutSec) * time.Second)
	seen := make(map[string]bool)

	// Snapshot existing messages
	if initial, err := p.listMessages(ctx, account.Email); err == nil {
		for _, m := range initial {
			seen[m.ID] = true
		}
	}

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(3 * time.Second):
		}

		msgs, err := p.listMessages(ctx, account.Email)
		if err != nil {
			continue
		}
		for _, m := range msgs {
			if m.ID == "" || seen[m.ID] {
				continue
			}
			seen[m.ID] = true

			// Prefer pre-extracted verification_code field
			if m.VerificationCode != "" && m.VerificationCode != "None" {
				return m.VerificationCode, nil
			}

			content := m.fullText()
			if keyword != "" && !strings.Contains(strings.ToLower(content), strings.ToLower(keyword)) {
				continue
			}
			if code := extractCode(content); code != "" {
				return code, nil
			}
		}
	}
	return "", fmt.Errorf("freemail: timeout waiting for OTP after %ds", timeoutSec)
}

// WaitForLink polls the Freemail inbox for a verification link.
func (p *FreemailProvider) WaitForLink(ctx context.Context, account *MailAccount, keyword string, timeoutSec int) (string, error) {
	deadline := time.Now().Add(time.Duration(timeoutSec) * time.Second)
	seen := make(map[string]bool)

	if initial, err := p.listMessages(ctx, account.Email); err == nil {
		for _, m := range initial {
			seen[m.ID] = true
		}
	}

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(3 * time.Second):
		}

		msgs, err := p.listMessages(ctx, account.Email)
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
	return "", fmt.Errorf("freemail: timeout waiting for link after %ds", timeoutSec)
}
