package mailprovider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// LaoudoProvider implements Provider for laoudo.com.
// Unlike other providers, Laoudo uses a pre-configured email address and
// an auth token; it does not auto-generate new addresses.
//
// Config keys:
//
//	auth_token  – Authorization header value (required)
//	email       – pre-configured email address (required)
//	account_id  – laoudo account ID (required)
type LaoudoProvider struct {
	authToken string
	email     string
	accountID string
	client    *http.Client
}

const laoudoAPIBase = "https://laoudo.com/api/email"
const laoudoUA = "Mozilla/5.0"

// NewLaoudo returns a LaoudoProvider.
func NewLaoudo(config map[string]string) *LaoudoProvider {
	return &LaoudoProvider{
		authToken: config["auth_token"],
		email:     config["email"],
		accountID: config["account_id"],
		client:    &http.Client{Timeout: 20 * time.Second},
	}
}

// GetEmail returns the pre-configured Laoudo email address.
func (p *LaoudoProvider) GetEmail(_ context.Context) (*MailAccount, error) {
	if p.email == "" {
		return nil, fmt.Errorf("laoudo: email is required (set config key 'email')")
	}
	if p.authToken == "" {
		return nil, fmt.Errorf("laoudo: auth_token is required")
	}
	return &MailAccount{
		Email:     p.email,
		AccountID: p.accountID,
		Token:     p.authToken,
	}, nil
}

func (p *LaoudoProvider) listMessages(ctx context.Context, accountID string) ([]struct {
	ID       interface{} `json:"id"`
	EmailID  interface{} `json:"emailId"`
	Subject  string      `json:"subject"`
	Content  string      `json:"content"`
	HTML     string      `json:"html"`
}, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, laoudoAPIBase+"/list", nil)
	if err != nil {
		return nil, err
	}
	q := req.URL.Query()
	q.Set("accountId", accountID)
	q.Set("allReceive", "0")
	q.Set("emailId", "0")
	q.Set("timeSort", "1")
	q.Set("size", "50")
	q.Set("type", "0")
	req.URL.RawQuery = q.Encode()
	req.Header.Set("Authorization", p.authToken)
	req.Header.Set("User-Agent", laoudoUA)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)

	var result struct {
		Data struct {
			List []struct {
				ID      interface{} `json:"id"`
				EmailID interface{} `json:"emailId"`
				Subject string      `json:"subject"`
				Content string      `json:"content"`
				HTML    string      `json:"html"`
			} `json:"list"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("laoudo list parse: %w", err)
	}
	return result.Data.List, nil
}

func msgID(id interface{}) string {
	if id == nil {
		return ""
	}
	switch v := id.(type) {
	case string:
		return v
	case float64:
		return fmt.Sprintf("%d", int64(v))
	default:
		return fmt.Sprintf("%v", v)
	}
}

// WaitForCode polls the Laoudo inbox for an OTP code.
func (p *LaoudoProvider) WaitForCode(ctx context.Context, account *MailAccount, keyword string, timeoutSec int) (string, error) {
	deadline := time.Now().Add(time.Duration(timeoutSec) * time.Second)
	seen := make(map[string]bool)

	// Snapshot existing message IDs
	if initial, err := p.listMessages(ctx, account.AccountID); err == nil {
		for _, m := range initial {
			if id := msgID(m.ID); id != "" {
				seen[id] = true
			}
			if id := msgID(m.EmailID); id != "" {
				seen[id] = true
			}
		}
	}

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(4 * time.Second):
		}

		msgs, err := p.listMessages(ctx, account.AccountID)
		if err != nil {
			continue
		}
		for _, m := range msgs {
			id := msgID(m.ID)
			if id == "" {
				id = msgID(m.EmailID)
			}
			if id == "" || seen[id] {
				continue
			}
			seen[id] = true

			content := m.Subject + " " + m.Content + " " + m.HTML
			if keyword != "" && !strings.Contains(strings.ToLower(content), strings.ToLower(keyword)) {
				continue
			}
			if code := extractCode(content); code != "" {
				return code, nil
			}
		}
	}
	return "", fmt.Errorf("laoudo: timeout waiting for OTP after %ds", timeoutSec)
}

// WaitForLink polls the Laoudo inbox for a verification link.
func (p *LaoudoProvider) WaitForLink(ctx context.Context, account *MailAccount, keyword string, timeoutSec int) (string, error) {
	deadline := time.Now().Add(time.Duration(timeoutSec) * time.Second)
	seen := make(map[string]bool)

	if initial, err := p.listMessages(ctx, account.AccountID); err == nil {
		for _, m := range initial {
			if id := msgID(m.ID); id != "" {
				seen[id] = true
			}
		}
	}

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(4 * time.Second):
		}

		msgs, err := p.listMessages(ctx, account.AccountID)
		if err != nil {
			continue
		}
		for _, m := range msgs {
			id := msgID(m.ID)
			if id == "" || seen[id] {
				continue
			}
			seen[id] = true
			content := m.Subject + " " + m.Content + " " + m.HTML
			if link := extractLink(content, keyword); link != "" {
				return link, nil
			}
		}
	}
	return "", fmt.Errorf("laoudo: timeout waiting for link after %ds", timeoutSec)
}
