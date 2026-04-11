package mailprovider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/publicsuffix"
)

// SeceMailProvider implements Provider for www.1secemail.com.
// It requires no configuration — addresses are generated automatically.
//
// API: https://www.1secemail.com/api
// Config keys: (none required)
//
//	api_url – override base URL (default: https://web2.www.1secemail.com/)
type SeceMailProvider struct {
	apiURL string
	client *http.Client
}

const defaultSeceMailURL = "https://www.1secemail.com"

// NewSeceMail returns a SeceMailProvider.
func NewSeceMail(config map[string]string) *SeceMailProvider {
	u := config["api_url"]
	if u == "" {
		u = defaultSeceMailURL
	}
	jar, err := cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})
	if err != nil {
		log.Fatalf("failed to create cookie jar: %v", err)
	}
	return &SeceMailProvider{
		apiURL: strings.TrimRight(u, "/"),
		client: &http.Client{
			Timeout:   20 * time.Second,
			Jar:       jar,
			Transport: buildTransport(config["proxy_url"]),
		},
	}
}

func (p *SeceMailProvider) do(ctx context.Context, method, path string, body io.Reader, headers map[string]string) ([]byte, error) {
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
		"origin":             "https://www.1secemail.com",
		"priority":           "u=1, i",
		"referer":            "https://www.1secemail.com/",
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
		return data, fmt.Errorf("www.1secemail.com: HTTP %d: %s", resp.StatusCode, string(data))
	}
	return data, nil
}

// GetEmail creates a new temporary inbox and returns its credentials.
func (p *SeceMailProvider) GetEmail(ctx context.Context) (*MailAccount, error) {
	data, err := p.do(ctx, http.MethodGet, "/", strings.NewReader("{}"), map[string]string{})
	if err != nil {
		return nil, fmt.Errorf("www.1secemail.com GetEmail: %w", err)
	}
	// html
	// 1. 从html取csrf-token <meta name="csrf-token" content="NAHdT2yagBXiVHeJyxZq5jsFghTdiBGXGL77mrAu">
	// 2. 从cookie取 XSRF-TOKEN
	csrfTokenReg := regexp.MustCompile(`<meta name="csrf-token" content="([^"]+)">`)
	matches := csrfTokenReg.FindSubmatch(data)
	if len(matches) < 2 {
		return nil, fmt.Errorf("www.1secemail.com GetEmail: failed to extract CSRF token from response: %s", string(data))
	}
	csrfToken := string(matches[1])
	log.Printf("提取到CSRF token: %s", csrfToken)

	uri, err := url.Parse(p.apiURL + "/") // to get cookies for the domain
	if err != nil {
		return nil, fmt.Errorf("www.1secemail.com GetEmail: failed to parse API URL: %w", err)
	}
	cookies := p.client.Jar.Cookies(uri)
	var xsrfToken string
	for _, c := range cookies {
		if c.Name == "XSRF-TOKEN" {
			xsrfToken = c.Value
			log.Printf("提取到XSRF token: %s", xsrfToken)
			break
		}
	}
	if xsrfToken == "" {
		return nil, fmt.Errorf("www.1secemail.com GetEmail: XSRF-TOKEN cookie not found")
	}

	headers := map[string]string{
		"x-xsrf-token": xsrfToken,
	}
	bytes, err := json.Marshal(map[string]string{
		"_token": csrfToken,
	})
	data, err = p.do(ctx, http.MethodPost, "/get_messages", strings.NewReader(string(bytes)), headers)
	var resp struct {
		Mailbox string        `json:"mailbox"`
		Emails  []SeceMailMsg `json:"messages"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("www.1secemail.com GetEmail parse: %w", err)
	}
	email := resp.Mailbox
	if email == "" {
		return nil, fmt.Errorf("www.1secemail.com GetEmail: empty address in response: %s", string(data))
	}
	return &MailAccount{
		Email:     email,
		AccountID: csrfToken,
		Token:     xsrfToken,
	}, nil
}

type SeceMailMsg struct {
	ID         string `json:"id"`
	Subject    string `json:"subject"`
	ReceivedAt string `json:"receivedAt"`
	Content    string `json:"content"`
}

func (m *SeceMailMsg) fullText() string {
	return m.Subject + " " + m.Content
}

func (p *SeceMailProvider) listMessages(ctx context.Context, account *MailAccount) ([]SeceMailMsg, error) {
	headers := map[string]string{
		"x-xsrf-token": account.Token,
	}
	auth := map[string]string{
		"_token": account.AccountID,
	}
	bodyBytes, err := json.Marshal(auth)
	if err != nil {
		return nil, fmt.Errorf("www.1secemail.com list messages: %w", err)
	}
	data, err := p.do(ctx, http.MethodPost, "/get_messages", strings.NewReader(string(bodyBytes)), headers)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Emails []SeceMailMsg `json:"messages"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("www.1secemail.com list parse: %w", err)
	}
	return resp.Emails, nil
}

// WaitForCode polls for a new message and extracts an OTP code.
func (p *SeceMailProvider) WaitForCode(ctx context.Context, account *MailAccount, keyword string, timeoutSec int) (string, error) {
	deadline := time.Now().Add(time.Duration(timeoutSec) * time.Second)
	seen := map[string]bool{}

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(3 * time.Second):
		}

		msgs, err := p.listMessages(ctx, account)
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
	return "", fmt.Errorf("www.1secemail.com: timeout waiting for OTP after %ds", timeoutSec)
}

// WaitForLink polls for a new message and extracts a verification URL.
func (p *SeceMailProvider) WaitForLink(ctx context.Context, account *MailAccount, keyword string, timeoutSec int) (string, error) {
	deadline := time.Now().Add(time.Duration(timeoutSec) * time.Second)
	seen := map[string]bool{}

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(3 * time.Second):
		}

		msgs, err := p.listMessages(ctx, account)
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
	return "", fmt.Errorf("www.1secemail.com: timeout waiting for link after %ds", timeoutSec)
}
