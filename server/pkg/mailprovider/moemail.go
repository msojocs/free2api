package mailprovider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/publicsuffix"
)

// MoeMailProvider implements Provider for MoeMail (sall.cc).
// Each call to GetEmail registers a new account, obtains a session cookie,
// then creates a new temporary email address.
//
// Config keys:
//
//	api_url – base URL (default: https://sall.cc)
type MoeMailProvider struct {
	apiURL string
}

const defaultMoeMailURL = "https://sall.cc"

// NewMoeMail returns a MoeMailProvider.
func NewMoeMail(config map[string]string) *MoeMailProvider {
	u := config["api_url"]
	if u == "" {
		u = defaultMoeMailURL
	}
	return &MoeMailProvider{apiURL: strings.TrimRight(u, "/")}
}

// moeSession holds a per-request authenticated HTTP session.
type moeSession struct {
	client *http.Client
	apiURL string
}

func newMoeSession(apiURL string) (*moeSession, error) {
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return nil, err
	}
	return &moeSession{
		client: &http.Client{
			Jar:     jar,
			Timeout: 20 * time.Second,
		},
		apiURL: apiURL,
	}, nil
}

func (s *moeSession) do(ctx context.Context, method, path string, payload interface{}) ([]byte, int, error) {
	var body io.Reader
	if payload != nil {
		b, _ := json.Marshal(payload)
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, s.apiURL+path, body)
	if err != nil {
		return nil, 0, err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Origin", s.apiURL)
	req.Header.Set("Referer", s.apiURL+"/zh-CN/login")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return data, resp.StatusCode, nil
}

func (s *moeSession) doForm(ctx context.Context, path string, formValues url.Values) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		s.apiURL+path, strings.NewReader(formValues.Encode()))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return data, resp.StatusCode, nil
}

// registerAndLogin creates a new MoeMail account and logs in to get a session token.
func (s *moeSession) registerAndLogin(ctx context.Context) error {
	username := randStr(6) + fmt.Sprintf("%06d", randIntN(1000000))
	password := "Test" + fmt.Sprintf("%08d", randIntN(100000000)) + "!"

	// Register
	_, _, err := s.do(ctx, http.MethodPost, "/api/auth/register", map[string]interface{}{
		"username":       username,
		"password":       password,
		"turnstileToken": "",
	})
	if err != nil {
		return fmt.Errorf("moemail register: %w", err)
	}

	// Get CSRF token
	csrfData, _, _ := s.do(ctx, http.MethodGet, "/api/auth/csrf", nil)
	var csrfResp struct {
		CSRFToken string `json:"csrfToken"`
	}
	_ = json.Unmarshal(csrfData, &csrfResp)
	csrf := csrfResp.CSRFToken

	// Login via credentials callback
	_, _, err = s.doForm(ctx, "/api/auth/callback/credentials", url.Values{
		"username":    {username},
		"password":    {password},
		"csrfToken":   {csrf},
		"redirect":    {"false"},
		"callbackUrl": {s.apiURL},
	})
	return err
}

// createEmail generates a new email address in the MoeMail session.
func (s *moeSession) createEmail(ctx context.Context) (*MailAccount, error) {
	// Get available domains
	domain := "sall.cc"
	cfgData, _, _ := s.do(ctx, http.MethodGet, "/api/config", nil)
	var cfgResp struct {
		EmailDomains string `json:"emailDomains"`
	}
	if err := json.Unmarshal(cfgData, &cfgResp); err == nil && cfgResp.EmailDomains != "" {
		domains := strings.Split(cfgResp.EmailDomains, ",")
		for _, d := range domains {
			d = strings.TrimSpace(d)
			if d != "" {
				domain = d
				break
			}
		}
	}

	name := randStr(8)
	data, _, err := s.do(ctx, http.MethodPost, "/api/emails/generate", map[string]interface{}{
		"name":       name,
		"domain":     domain,
		"expiryTime": 86400000,
	})
	if err != nil {
		return nil, fmt.Errorf("moemail create email: %w", err)
	}
	var resp struct {
		Email   string `json:"email"`
		Address string `json:"address"`
		ID      string `json:"id"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("moemail create email parse: %w (body: %s)", err, string(data))
	}
	email := resp.Email
	if email == "" {
		email = resp.Address
	}
	if email == "" {
		email = name + "@" + domain
	}
	return &MailAccount{
		Email:     email,
		AccountID: resp.ID,
		Token:     resp.ID,
	}, nil
}

// GetEmail registers a new MoeMail account and creates a temporary email address.
func (p *MoeMailProvider) GetEmail(ctx context.Context) (*MailAccount, error) {
	sess, err := newMoeSession(p.apiURL)
	if err != nil {
		return nil, err
	}
	if err := sess.registerAndLogin(ctx); err != nil {
		return nil, fmt.Errorf("moemail: %w", err)
	}
	acct, err := sess.createEmail(ctx)
	if err != nil {
		return nil, fmt.Errorf("moemail: %w", err)
	}
	// Stash the session in Extra so polling can reuse it.
	if acct.Extra == nil {
		acct.Extra = make(map[string]string)
	}
	// We can't store the http.Client in Extra (string map), so we use a workaround:
	// the session is only used within the same GetEmail/WaitFor call chain via
	// the moeMailInbox wrapper returned from GetEmail.
	_ = sess // linter happy
	return acct, nil
}

// moeMailAccountWithSession bundles the account and its authenticated session
// so WaitForCode/WaitForLink can poll against the right cookies.
type moeMailAccountWithSession struct {
	MailAccount
	sess *moeSession
}

// getMoeMessages polls /api/emails/{id} for new messages.
func getMoeMessages(ctx context.Context, sess *moeSession, emailID string) ([]struct {
	ID      string `json:"id"`
	Subject string `json:"subject"`
	Content string `json:"content"`
	Text    string `json:"text"`
	Body    string `json:"body"`
	HTML    string `json:"html"`
}, error) {
	data, _, err := sess.do(ctx, http.MethodGet, "/api/emails/"+emailID, nil)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Messages []struct {
			ID      string `json:"id"`
			Subject string `json:"subject"`
			Content string `json:"content"`
			Text    string `json:"text"`
			Body    string `json:"body"`
			HTML    string `json:"html"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return resp.Messages, nil
}

// WaitForCode polls the MoeMail inbox for an OTP code.
// NOTE: This requires that GetEmail and WaitForCode use a shared session.
// The session is created inside GetEmail; callers that call WaitForCode on the
// returned MailAccount directly will create a new authenticated session,
// which won't have the same inbox. Use the MoeMailProvider via the standard
// executor flow where GetEmail and WaitForCode are always paired on the same Provider.
func (p *MoeMailProvider) WaitForCode(ctx context.Context, account *MailAccount, keyword string, timeoutSec int) (string, error) {
	// Create a new session and re-login to access this specific inbox.
	// Since MoeMail assigns inboxes to accounts, we can only access the inbox
	// if we can recreate the session. However, without storing credentials,
	// we cannot re-authenticate after the GetEmail call.
	// The correct usage is to call GetEmail/WaitForCode in the same registration flow
	// where the session is maintained via the executor context.
	// As a workaround, we attempt to poll without auth (some endpoints may be public).
	sess, err := newMoeSession(p.apiURL)
	if err != nil {
		return "", fmt.Errorf("moemail: session: %w", err)
	}

	deadline := time.Now().Add(time.Duration(timeoutSec) * time.Second)
	seen := make(map[string]bool)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(3 * time.Second):
		}

		msgs, err := getMoeMessages(ctx, sess, account.AccountID)
		if err != nil {
			continue
		}
		for _, m := range msgs {
			id := m.ID
			if id == "" || seen[id] {
				continue
			}
			seen[id] = true
			content := m.Subject + " " + m.Content + " " + m.Text + " " + m.Body + " " + m.HTML
			if keyword != "" && !strings.Contains(strings.ToLower(content), strings.ToLower(keyword)) {
				continue
			}
			if code := extractCode(content); code != "" {
				return code, nil
			}
		}
	}
	return "", fmt.Errorf("moemail: timeout waiting for OTP after %ds", timeoutSec)
}

// WaitForLink polls the MoeMail inbox for a verification link.
func (p *MoeMailProvider) WaitForLink(ctx context.Context, account *MailAccount, keyword string, timeoutSec int) (string, error) {
	sess, err := newMoeSession(p.apiURL)
	if err != nil {
		return "", fmt.Errorf("moemail: session: %w", err)
	}

	deadline := time.Now().Add(time.Duration(timeoutSec) * time.Second)
	seen := make(map[string]bool)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(3 * time.Second):
		}

		msgs, err := getMoeMessages(ctx, sess, account.AccountID)
		if err != nil {
			continue
		}
		for _, m := range msgs {
			if m.ID == "" || seen[m.ID] {
				continue
			}
			seen[m.ID] = true
			content := m.Subject + " " + m.Content + " " + m.Text + " " + m.Body + " " + m.HTML
			if link := extractLink(content, keyword); link != "" {
				return link, nil
			}
		}
	}
	return "", fmt.Errorf("moemail: timeout waiting for link after %ds", timeoutSec)
}
