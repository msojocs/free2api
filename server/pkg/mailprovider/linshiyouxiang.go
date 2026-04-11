package mailprovider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/publicsuffix"
)

// LuckMailProvider implements Provider for the LuckMail service.
// LuckMail assigns a disposable email via an "order" placed against a
// registered project code. The verification code is then polled from the order.
//
// API base: https://mails.luckyous.com/
//
// Config keys:
//
//	api_url       – base URL (default: https://deepmails.org)
//	api_key       – API key (required)
//	project_code  – project identifier assigned by Linshiyouxiang (required)
//	email_type    – optional filter for email type
type LinshiyouxiangProvider struct {
	apiURL string
	client *http.Client
}

const defaultLinshiyouxiangURL = "https://deepmails.org"

// NewLinshiyouxiang returns a LinshiyouxiangProvider.
func NewLinshiyouxiang(config map[string]string) *LinshiyouxiangProvider {
	u := config["api_url"]
	if u == "" {
		u = defaultLinshiyouxiangURL
	}
	// cookies
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return nil
	}

	return &LinshiyouxiangProvider{
		apiURL: strings.TrimRight(u, "/"),
		client: &http.Client{
			Timeout:   25 * time.Second,
			Jar:       jar,
			Transport: buildTransport(config["proxy_url"]),
		},
	}
}

func (p *LinshiyouxiangProvider) authHeaders() map[string]string {
	return map[string]string{
		"Accept":             "application/json",
		"origin":             "https://deepmails.org",
		"referer":            "https://deepmails.org/",
		"sec-ch-ua":          `"Chromium";v="146", "Not-A.Brand";v="24", "Microsoft Edge";v="146"`,
		"sec-ch-ua-mobile":   "?0",
		"sec-ch-ua-platform": `"Windows"`,
		"sec-fetch-dest":     "empty",
		"sec-fetch-mode":     "cors",
		"sec-fetch-site":     "same-origin",
		"user-agent":         "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36 Edg/146.0.0.0",
	}
}

func (p *LinshiyouxiangProvider) do(ctx context.Context, method, path string, payload interface{}) ([]byte, error) {
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
		return data, fmt.Errorf("linshiyouxiang: HTTP %d %s: %s", resp.StatusCode, path, string(data))
	}
	return data, nil
}

// GetEmail creates a Linshiyouxiang order and returns the assigned email address.
// The AccountID is set to the order_no for polling.
func (p *LinshiyouxiangProvider) GetEmail(ctx context.Context) (*MailAccount, error) {

	reqBody := map[string]interface{}{}

	data, err := p.do(ctx, http.MethodGet, "/", reqBody)
	if err != nil {
		return nil, fmt.Errorf("linshiyouxiang GetEmail: %w", err)
	}

	mailAddressReg := regexp.MustCompile(`tempMailGlobal = '([^']+)';`)
	mailTokenReg := regexp.MustCompile(`mailCodeGlobal = '([^']+)';`)

	mailAddressMatch := mailAddressReg.FindSubmatch(data)
	mailTokenMatch := mailTokenReg.FindSubmatch(data)

	if len(mailAddressMatch) < 2 || len(mailTokenMatch) < 2 {
		return nil, fmt.Errorf("linshiyouxiang GetEmail: failed to parse email address or token from response")
	}

	email := string(mailAddressMatch[1])
	token := string(mailTokenMatch[1])
	return &MailAccount{
		Email:     email,
		AccountID: email,
		Token:     token,
		Extra:     map[string]string{},
	}, nil
}

type linshiyouxiangMsg struct {
	Code       string `json:"Code"`
	FromEmail  string `json:"FromEmail"`
	FromName   string `json:"FromName"`
	SendTime   int64  `json:"SendTime"`
	Status     int    `json:"Status"`
	Subject    string `json:"subject"`
	DetailHtml string
}

func (m *linshiyouxiangMsg) fullText() string {
	return m.Subject + " " + m.DetailHtml
}

func (p *LinshiyouxiangProvider) listMessages(ctx context.Context, email, token string) ([]linshiyouxiangMsg, error) {
	body := map[string]string{
		"code":  token,
		"email": email,
	}
	data, err := p.do(ctx, http.MethodPost, "/get-messages", body)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Emails []linshiyouxiangMsg `json:"emails"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("linshiyouxiang list parse: %w", err)
	}
	for i := range resp.Emails {
		detail, err := p.getMessageDetail(ctx, resp.Emails[i].Code)
		if err != nil {
			log.Printf("linshiyouxiang: failed to get message detail for code %s: %v", resp.Emails[i].Code, err)
			continue
		}
		resp.Emails[i].DetailHtml = detail
	}
	return resp.Emails, nil
}

// get message detail
func (p *LinshiyouxiangProvider) getMessageDetail(ctx context.Context, code string) (string, error) {
	url := fmt.Sprintf("/mail/view/%s", code)
	data, err := p.do(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	// <div class="table-responsive msglist  pb-4">(xxx)"site-description"
	detailReg := regexp.MustCompile(`emailContent = ([\s\S]+?)iframe.srcdoc = emailContent`)
	detailMatch := detailReg.FindSubmatch(data)
	if len(detailMatch) < 2 {
		return "", fmt.Errorf("linshiyouxiang getMessageDetail: failed to parse message detail from response")
	}
	return string(detailMatch[1]), nil
}

// WaitForCode polls for a new message and extracts an OTP code.
func (p *LinshiyouxiangProvider) WaitForCode(ctx context.Context, account *MailAccount, keyword string, timeoutSec int) (string, error) {
	deadline := time.Now().Add(time.Duration(timeoutSec) * time.Second)
	seen := map[string]bool{}

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(3 * time.Second):
		}

		msgs, err := p.listMessages(ctx, account.Email, account.Token)
		if err != nil {
			continue
		}
		log.Printf("收到邮件数量：%d", len(msgs))
		for _, m := range msgs {
			if m.Code == "" || seen[m.Code] {
				continue
			}
			seen[m.Code] = true
			content := m.fullText()
			if keyword != "" && !strings.Contains(strings.ToLower(content), strings.ToLower(keyword)) {
				continue
			}
			if code := extractCode(content); code != "" {
				return code, nil
			}
		}
	}
	return "", fmt.Errorf("linshiyouxiang: timeout waiting for OTP after %ds", timeoutSec)
}

// WaitForLink polls for a new message and extracts a verification URL.
func (p *LinshiyouxiangProvider) WaitForLink(ctx context.Context, account *MailAccount, keyword string, timeoutSec int) (string, error) {
	deadline := time.Now().Add(time.Duration(timeoutSec) * time.Second)
	seen := map[string]bool{}

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(3 * time.Second):
		}

		msgs, err := p.listMessages(ctx, account.Email, account.Token)
		if err != nil {
			continue
		}
		for _, m := range msgs {
			if m.Code == "" || seen[m.Code] {
				continue
			}
			seen[m.Code] = true
			content := m.fullText()
			if link := extractLink(content, keyword); link != "" {
				return link, nil
			}
		}
	}
	return "", fmt.Errorf("linshiyouxiang: timeout waiting for link after %ds", timeoutSec)
}
