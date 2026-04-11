package mailprovider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// MaliAPIProvider implements Provider for YYDS Mail / MaliAPI.
// API documentation: https://maliapi.215.im/v1
//
// Config keys:
//
//	api_url  – base URL (default: https://maliapi.215.im/v1)
//	api_key  – API key (required)
//	domain   – preferred email domain (optional)
type MaliAPIProvider struct {
	apiURL string
	apiKey string
	domain string
	client *http.Client
}

const defaultMaliAPIURL = "https://maliapi.215.im/v1"

// NewMaliAPI returns a MaliAPIProvider.
func NewMaliAPI(config map[string]string) *MaliAPIProvider {
	u := config["api_url"]
	if u == "" {
		u = defaultMaliAPIURL
	}
	return &MaliAPIProvider{
		apiURL: strings.TrimRight(u, "/"),
		apiKey: config["api_key"],
		domain: config["domain"],
		client: &http.Client{Timeout: 20 * time.Second, Transport: buildTransport(config["proxy_url"])},
	}
}

func (p *MaliAPIProvider) headers(bearer string) map[string]string {
	h := map[string]string{
		"Accept":       "application/json",
		"Content-Type": "application/json",
	}
	if p.apiKey != "" {
		h["X-API-Key"] = p.apiKey
	}
	if bearer != "" {
		h["Authorization"] = "Bearer " + bearer
	}
	return h
}

func (p *MaliAPIProvider) do(ctx context.Context, method, path string, payload interface{}, bearer string) ([]byte, error) {
	var body io.Reader
	if payload != nil {
		b, _ := json.Marshal(payload)
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, p.apiURL+path, body)
	if err != nil {
		return nil, err
	}
	for k, v := range p.headers(bearer) {
		req.Header.Set(k, v)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return data, fmt.Errorf("maliapi: HTTP %d %s: %s", resp.StatusCode, path, string(data))
	}
	return data, nil
}

// unwrap extracts the "data" field if present, otherwise returns the raw payload.
func (p *MaliAPIProvider) unwrap(data []byte) (json.RawMessage, error) {
	var outer struct {
		Data    json.RawMessage `json:"data"`
		Success *bool           `json:"success"`
		Error   string          `json:"error"`
	}
	if err := json.Unmarshal(data, &outer); err != nil {
		return data, nil
	}
	if outer.Success != nil && !*outer.Success {
		return nil, fmt.Errorf("maliapi: %s", outer.Error)
	}
	if outer.Data != nil {
		return outer.Data, nil
	}
	return data, nil
}

// GetEmail creates a new temporary email address via the MaliAPI.
func (p *MaliAPIProvider) GetEmail(ctx context.Context) (*MailAccount, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("maliapi: api_key is required")
	}
	body := map[string]string{}
	if p.domain != "" {
		body["domain"] = p.domain
	}
	data, err := p.do(ctx, http.MethodPost, "/accounts", body, "")
	if err != nil {
		return nil, fmt.Errorf("maliapi GetEmail: %w", err)
	}
	raw, err := p.unwrap(data)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Address   string `json:"address"`
		Email     string `json:"email"`
		TempToken string `json:"tempToken"`
		Token     string `json:"token"`
		ID        string `json:"id"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("maliapi GetEmail parse: %w (body: %s)", err, string(raw))
	}
	email := resp.Address
	if email == "" {
		email = resp.Email
	}
	if email == "" {
		return nil, fmt.Errorf("maliapi GetEmail: empty address in response: %s", string(raw))
	}
	token := resp.TempToken
	if token == "" {
		token = resp.Token
	}
	if token == "" {
		token = resp.ID
	}
	return &MailAccount{
		Email:     email,
		AccountID: token,
		Token:     token,
	}, nil
}

// maliAPIMsg represents a message from the /messages list endpoint.
type maliAPIMsg struct {
	ID       interface{} `json:"id"`
	Subject  string      `json:"subject"`
	Snippet  string      `json:"snippet"`
	Text     string      `json:"text"`
	HTML     string      `json:"html"`
	Messages interface{} `json:"messages"` // may be an array of body parts
}

func (m *maliAPIMsg) id() string {
	return msgID(m.ID)
}

func (m *maliAPIMsg) fullText() string {
	return m.Subject + " " + m.Snippet + " " + m.Text + " " + m.HTML
}

// emailAddrRe strips email addresses to reduce false code matches.
var emailAddrRe = regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)

func (p *MaliAPIProvider) listMessages(ctx context.Context, email string) ([]maliAPIMsg, error) {
	data, err := p.do(ctx, http.MethodGet, "/messages?address="+email, nil, "")
	if err != nil {
		return nil, err
	}
	raw, err := p.unwrap(data)
	if err != nil {
		return nil, err
	}
	// Response may be {"messages": [...]} or a direct array
	var direct []maliAPIMsg
	if err := json.Unmarshal(raw, &direct); err == nil {
		return direct, nil
	}
	var wrapper struct {
		Messages []maliAPIMsg `json:"messages"`
	}
	if err := json.Unmarshal(raw, &wrapper); err == nil {
		return wrapper.Messages, nil
	}
	return nil, fmt.Errorf("maliapi: unexpected message list format: %s", string(raw))
}

func (p *MaliAPIProvider) getMessage(ctx context.Context, msgID string) (*maliAPIMsg, error) {
	data, err := p.do(ctx, http.MethodGet, "/messages/"+msgID, nil, "")
	if err != nil {
		return nil, err
	}
	raw, err := p.unwrap(data)
	if err != nil {
		return nil, err
	}
	var outer struct {
		Message *maliAPIMsg `json:"message"`
	}
	if err := json.Unmarshal(raw, &outer); err == nil && outer.Message != nil {
		return outer.Message, nil
	}
	var msg maliAPIMsg
	if err := json.Unmarshal(raw, &msg); err != nil {
		return nil, fmt.Errorf("maliapi getMessage parse: %w", err)
	}
	return &msg, nil
}

func (p *MaliAPIProvider) snapshot(ctx context.Context, email string) map[string]bool {
	ids := make(map[string]bool)
	msgs, _ := p.listMessages(ctx, email)
	for _, m := range msgs {
		if id := m.id(); id != "" {
			ids[id] = true
		}
	}
	return ids
}

// WaitForCode polls the MaliAPI inbox for an OTP code.
func (p *MaliAPIProvider) WaitForCode(ctx context.Context, account *MailAccount, keyword string, timeoutSec int) (string, error) {
	deadline := time.Now().Add(time.Duration(timeoutSec) * time.Second)
	seen := p.snapshot(ctx, account.Email)

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
			id := m.id()
			if id == "" || seen[id] {
				continue
			}
			seen[id] = true

			// Fetch full message for body text
			detail, err := p.getMessage(ctx, id)
			if err != nil {
				detail = &m
			}
			content := emailAddrRe.ReplaceAllString(detail.fullText(), "")
			if keyword != "" && !strings.Contains(strings.ToLower(content), strings.ToLower(keyword)) {
				continue
			}
			if code := extractCode(content); code != "" {
				return code, nil
			}
		}
	}
	return "", fmt.Errorf("maliapi: timeout waiting for OTP after %ds", timeoutSec)
}

// WaitForLink polls the MaliAPI inbox for a verification link.
func (p *MaliAPIProvider) WaitForLink(ctx context.Context, account *MailAccount, keyword string, timeoutSec int) (string, error) {
	deadline := time.Now().Add(time.Duration(timeoutSec) * time.Second)
	seen := p.snapshot(ctx, account.Email)

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
			id := m.id()
			if id == "" || seen[id] {
				continue
			}
			seen[id] = true

			detail, err := p.getMessage(ctx, id)
			if err != nil {
				detail = &m
			}
			if link := extractLink(detail.fullText(), keyword); link != "" {
				return link, nil
			}
		}
	}
	return "", fmt.Errorf("maliapi: timeout waiting for link after %ds", timeoutSec)
}
