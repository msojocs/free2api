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

// LuckMailProvider implements Provider for the LuckMail service.
// LuckMail assigns a disposable email via an "order" placed against a
// registered project code. The verification code is then polled from the order.
//
// API base: https://mails.luckyous.com/
//
// Config keys:
//
//	api_url       – base URL (default: https://mails.luckyous.com)
//	api_key       – API key (required)
//	project_code  – project identifier assigned by LuckMail (required)
//	email_type    – optional filter for email type
type LuckMailProvider struct {
	apiURL      string
	apiKey      string
	projectCode string
	emailType   string
	client      *http.Client
}

const defaultLuckMailURL = "https://mails.luckyous.com"

// NewLuckMail returns a LuckMailProvider.
func NewLuckMail(config map[string]string) *LuckMailProvider {
	u := config["api_url"]
	if u == "" {
		u = defaultLuckMailURL
	}
	return &LuckMailProvider{
		apiURL:      strings.TrimRight(u, "/"),
		apiKey:      config["api_key"],
		projectCode: config["project_code"],
		emailType:   config["email_type"],
		client:      &http.Client{Timeout: 25 * time.Second, Transport: buildTransport(config["proxy_url"])},
	}
}

func (p *LuckMailProvider) authHeaders() map[string]string {
	return map[string]string{
		"Accept":        "application/json",
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + p.apiKey,
	}
}

func (p *LuckMailProvider) do(ctx context.Context, method, path string, payload interface{}) ([]byte, error) {
	var body io.Reader
	if payload != nil {
		b, _ := json.Marshal(payload)
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, p.apiURL+path, body)
	if err != nil {
		return nil, err
	}
	for k, v := range p.authHeaders() {
		req.Header.Set(k, v)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return data, fmt.Errorf("luckmail: HTTP %d %s: %s", resp.StatusCode, path, string(data))
	}
	return data, nil
}

// GetEmail creates a LuckMail order and returns the assigned email address.
// The AccountID is set to the order_no for polling.
func (p *LuckMailProvider) GetEmail(ctx context.Context) (*MailAccount, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("luckmail: api_key is required")
	}
	if p.projectCode == "" {
		return nil, fmt.Errorf("luckmail: project_code is required")
	}

	reqBody := map[string]interface{}{
		"project_code": p.projectCode,
	}
	if p.emailType != "" {
		reqBody["email_type"] = p.emailType
	}

	data, err := p.do(ctx, http.MethodPost, "/api/orders", reqBody)
	if err != nil {
		return nil, fmt.Errorf("luckmail GetEmail: %w", err)
	}

	var resp struct {
		OrderNo      string `json:"order_no"`
		EmailAddress string `json:"email_address"`
		Status       string `json:"status"`
		ExpiredAt    string `json:"expired_at"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("luckmail GetEmail parse: %w (body: %s)", err, string(data))
	}
	if resp.EmailAddress == "" {
		return nil, fmt.Errorf("luckmail GetEmail: no email_address in response: %s", string(data))
	}
	return &MailAccount{
		Email:     resp.EmailAddress,
		AccountID: resp.OrderNo,
		Token:     resp.OrderNo,
		Extra: map[string]string{
			"order_no":   resp.OrderNo,
			"expired_at": resp.ExpiredAt,
		},
	}, nil
}

// pollOrder queries a LuckMail order for its current status and verification code.
func (p *LuckMailProvider) pollOrder(ctx context.Context, orderNo string) (status, code string, err error) {
	data, err := p.do(ctx, http.MethodGet, "/api/orders/"+orderNo, nil)
	if err != nil {
		return "", "", err
	}
	var resp struct {
		Status           string `json:"status"`
		VerificationCode string `json:"verification_code"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", "", fmt.Errorf("luckmail poll parse: %w", err)
	}
	return resp.Status, resp.VerificationCode, nil
}

// WaitForCode polls the LuckMail order for a verification code.
func (p *LuckMailProvider) WaitForCode(ctx context.Context, account *MailAccount, _ string, timeoutSec int) (string, error) {
	orderNo := account.AccountID
	if orderNo == "" {
		if account.Extra != nil {
			orderNo = account.Extra["order_no"]
		}
	}
	if orderNo == "" {
		return "", fmt.Errorf("luckmail: no order_no in account")
	}

	deadline := time.Now().Add(time.Duration(timeoutSec) * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(3 * time.Second):
		}

		status, code, err := p.pollOrder(ctx, orderNo)
		if err != nil {
			continue
		}
		if code != "" {
			return code, nil
		}
		if status == "failed" || status == "expired" || status == "cancelled" {
			return "", fmt.Errorf("luckmail: order %s is %s", orderNo, status)
		}
	}
	return "", fmt.Errorf("luckmail: timeout waiting for OTP after %ds", timeoutSec)
}

// WaitForLink is not supported by LuckMail's order API.
func (p *LuckMailProvider) WaitForLink(_ context.Context, _ *MailAccount, _ string, _ int) (string, error) {
	return "", fmt.Errorf("luckmail: WaitForLink is not supported; use WaitForCode")
}
