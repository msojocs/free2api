package executor

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

	"github.com/msojocs/free2api/server/internal/core"
	"github.com/msojocs/free2api/server/internal/model"
	"github.com/msojocs/free2api/server/pkg/captcha"
	"github.com/msojocs/free2api/server/pkg/crypto"
	"github.com/msojocs/free2api/server/pkg/mailprovider"
	"golang.org/x/net/publicsuffix"
)

// OpenAI / ChatGPT registration endpoints.
// The signup flow uses auth0.openai.com with PKCE and email OTP verification.
// NOTE: OpenAI deploys Cloudflare WAF + Arkose FunCaptcha on the signup endpoint.
// This implementation performs the HTTP protocol flow; additional bot-bypass
// tooling (TLS fingerprinting, CAPTCHA solver) may be required in production.
const (
	openAIAuthBase   = "https://auth.openai.com"
	openAISignupPath = "/u/signup"

	chatGPTUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/145.0.0.0 Safari/537.36"
)

// ChatGPTExecutor registers new OpenAI / ChatGPT accounts using the HTTP protocol flow.
type ChatGPTExecutor struct{}

func NewChatGPTExecutor() *ChatGPTExecutor {
	return &ChatGPTExecutor{}
}

// openAISession holds the HTTP clients and helpers for the auth0 sign-up flow.
type openAISession struct {
	noRedirect   *http.Client
	withRedirect *http.Client
}

func newOpenAISession(proxyURL string) (*openAISession, error) {
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return nil, err
	}
	transport := &http.Transport{}
	if proxyURL != "" {
		u, err := url.Parse(proxyURL)
		if err != nil {
			return nil, fmt.Errorf("chatgpt: invalid proxy URL: %w", err)
		}
		transport.Proxy = http.ProxyURL(u)
	}
	noRedir := &http.Client{
		Jar:       jar,
		Transport: transport,
		Timeout:   30 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	withRedir := &http.Client{
		Jar:       jar,
		Transport: transport,
		Timeout:   60 * time.Second,
	}
	return &openAISession{noRedirect: noRedir, withRedirect: withRedir}, nil
}

func (s *openAISession) jsonPost(ctx context.Context, useRedir bool, targetURL string, payload interface{}, headers map[string]string) ([]byte, int, error) {
	b, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(b))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", chatGPTUA)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	client := s.noRedirect
	if useRedir {
		client = s.withRedirect
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	return data, resp.StatusCode, err
}

// step1SeedSession visits the signup page to obtain the CSRF token and session cookies.
func (s *openAISession) step1SeedSession(ctx context.Context) (string, error) {
	reqURL := openAIAuthBase + openAISignupPath
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", chatGPTUA)
	req.Header.Set("Accept", "text/html")
	resp, err := s.withRedirect.Do(req)
	if err != nil {
		return "", fmt.Errorf("chatgpt step1: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	// Extract state from the page or Location header (depending on auth0 version).
	// The state parameter is embedded in the redirect URL or the HTML form action.
	state := extractStateParam(string(body), resp.Request.URL.String())
	return state, nil
}

// extractStateParam tries to find the auth0 state value from the page URL or body.
func extractStateParam(body, finalURL string) string {
	// Try URL parameter first (after redirect)
	if u, err := url.Parse(finalURL); err == nil {
		if s := u.Query().Get("state"); s != "" {
			return s
		}
	}
	// Fallback: look for name="state" value="..." in the HTML form
	const needle = `name="state" value="`
	if idx := strings.Index(body, needle); idx != -1 {
		rest := body[idx+len(needle):]
		if end := strings.Index(rest, `"`); end > 0 {
			return rest[:end]
		}
	}
	return ""
}

// step2SignUpWithEmail submits the email to the auth0 signup endpoint.
func (s *openAISession) step2SignUpWithEmail(ctx context.Context, email, state string) (string, error) {
	reqURL := fmt.Sprintf("%s/u/signup/identifier?state=%s", openAIAuthBase, url.QueryEscape(state))

	formData := url.Values{
		"state":                       {state},
		"email":                       {email},
		"js-available":                {"true"},
		"webauthn-available":          {"true"},
		"is-brave":                    {"false"},
		"webauthn-platform-available": {"false"},
		"action":                      {"default"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", chatGPTUA)
	req.Header.Set("Referer", openAIAuthBase+openAISignupPath)

	resp, err := s.noRedirect.Do(req)
	if err != nil {
		return "", fmt.Errorf("chatgpt step2: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	// The redirect or response body should contain a new state for the next step.
	newState := state
	if loc := resp.Header.Get("Location"); loc != "" {
		if u, err := url.Parse(loc); err == nil {
			if s := u.Query().Get("state"); s != "" {
				newState = s
			}
		}
	}
	if s := extractStateParam(string(body), ""); s != "" {
		newState = s
	}
	return newState, nil
}

// step3SetPassword submits the password to the auth0 signup continuation endpoint.
func (s *openAISession) step3SetPassword(ctx context.Context, email, password, state string) (string, error) {
	reqURL := fmt.Sprintf("%s/u/signup/password?state=%s", openAIAuthBase, url.QueryEscape(state))

	formData := url.Values{
		"state":    {state},
		"email":    {email},
		"password": {password},
		"action":   {"default"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", chatGPTUA)

	resp, err := s.noRedirect.Do(req)
	if err != nil {
		return "", fmt.Errorf("chatgpt step3: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	loc := resp.Header.Get("Location")
	newState := state
	if u, err := url.Parse(loc); err == nil {
		if s := u.Query().Get("state"); s != "" {
			newState = s
		}
	}
	if s := extractStateParam(string(body), ""); s != "" {
		newState = s
	}
	return newState, nil
}

// step4VerifyEmail submits the email verification OTP.
func (s *openAISession) step4VerifyEmail(ctx context.Context, email, otp, state string) error {
	reqURL := fmt.Sprintf("%s/u/signup/email-verification?state=%s", openAIAuthBase, url.QueryEscape(state))

	formData := url.Values{
		"state":  {state},
		"email":  {email},
		"code":   {otp},
		"action": {"default"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", chatGPTUA)

	resp, err := s.noRedirect.Do(req)
	if err != nil {
		return fmt.Errorf("chatgpt step4: %w", err)
	}
	resp.Body.Close()
	return nil
}

// step5GetSessionToken completes the OAuth callback and extracts the session token.
func (s *openAISession) step5GetSessionToken(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://chat.openai.com/api/auth/session", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", chatGPTUA)

	resp, err := s.withRedirect.Do(req)
	if err != nil {
		return "", fmt.Errorf("chatgpt step5: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var sessionResp struct {
		AccessToken string `json:"accessToken"`
		User        struct {
			Email string `json:"email"`
		} `json:"user"`
	}
	if err := json.Unmarshal(body, &sessionResp); err != nil || sessionResp.AccessToken == "" {
		// Return partial info; the account is still created with the email/password.
		return "", fmt.Errorf("chatgpt: could not extract session token (Cloudflare protection may be active): %s", string(body))
	}
	return sessionResp.AccessToken, nil
}

// Execute runs the OpenAI / ChatGPT registration flow.
//
// NOTE: OpenAI uses Cloudflare WAF protection on the signup page.  The
// standard net/http client may receive a 403 / CAPTCHA challenge.  For
// production use, supplement with a TLS-fingerprinting library (e.g. utls)
// or a headless-browser executor mode.
//
// Relevant config keys (same as CursorExecutor):
//
//	proxy            – optional proxy URL
//	mail_provider    – "mailtm" (default) | "cfworker"
//	mail_api_url, mail_admin_token, mail_domain
//	captcha_provider – "yescaptcha" | "2captcha" (optional)
//	captcha_key      – captcha API key
func (e *ChatGPTExecutor) Execute(ctx context.Context, taskID uint, config map[string]interface{}, publish func(core.ProgressUpdate)) (*ExecutionResult, error) {
	sendProgress(publish, taskID, 0, "Starting ChatGPT account registration", "running")

	proxyURL := cfgStr(config, "proxy", "")

	// ── Temp email ────────────────────────────────────────────────────────────
	mailProviderType := cfgStr(config, "mail_provider", "tempmail")
	mailCfg := map[string]string{
		"api_url":     cfgStr(config, "mail_api_url", ""),
		"admin_token": cfgStr(config, "mail_admin_token", ""),
		"domain":      cfgStr(config, "mail_domain", ""),
	}
	sendProgress(publish, taskID, 8, fmt.Sprintf("Initialising mail provider: %s", mailProviderType), "running")
	mp, err := mailprovider.New(mailProviderType, mailCfg)
	if err != nil {
		sendProgress(publish, taskID, 100, fmt.Sprintf("Mail provider error: %v", err), "failed")
		return nil, err
	}

	sendProgress(publish, taskID, 12, "Getting temporary email address…", "running")
	mailAccount, err := mp.GetEmail(ctx)
	if err != nil {
		sendProgress(publish, taskID, 100, fmt.Sprintf("Get email failed: %v", err), "failed")
		return nil, err
	}
	email := mailAccount.Email
	sendProgress(publish, taskID, 18, fmt.Sprintf("Got email: %s", email), "running")

	// ── Captcha solver (optional) ─────────────────────────────────────────────
	captchaProvider := cfgStr(config, "captcha_provider", "")
	captchaKey := cfgStr(config, "captcha_key", "")
	_ = captchaProvider
	_ = captchaKey
	if captchaProvider != "" && captchaKey != "" {
		if _, err := captcha.New(captchaProvider, captchaKey); err != nil {
			sendProgress(publish, taskID, 100, fmt.Sprintf("Captcha provider error: %v", err), "failed")
			return nil, err
		}
	}

	// ── OpenAI HTTP session ───────────────────────────────────────────────────
	sess, err := newOpenAISession(proxyURL)
	if err != nil {
		sendProgress(publish, taskID, 100, fmt.Sprintf("Failed to init HTTP session: %v", err), "failed")
		return nil, err
	}

	// Step 1 – seed session / get state
	sendProgress(publish, taskID, 25, "Step 1/5: Seeding registration session…", "running")
	state, err := sess.step1SeedSession(ctx)
	if err != nil {
		sendProgress(publish, taskID, 100, fmt.Sprintf("Step 1 failed: %v", err), "failed")
		return nil, err
	}
	if state == "" {
		// This typically means Cloudflare blocked the request.
		err = fmt.Errorf("could not obtain auth state – Cloudflare protection may be active; try with a residential proxy")
		sendProgress(publish, taskID, 100, err.Error(), "failed")
		return nil, err
	}

	// Step 2 – submit email
	sendProgress(publish, taskID, 38, "Step 2/5: Submitting email…", "running")
	state, err = sess.step2SignUpWithEmail(ctx, email, state)
	if err != nil {
		sendProgress(publish, taskID, 100, fmt.Sprintf("Step 2 failed: %v", err), "failed")
		return nil, err
	}

	// Step 3 – set password
	password := randPassword()
	sendProgress(publish, taskID, 50, "Step 3/5: Setting password…", "running")
	state, err = sess.step3SetPassword(ctx, email, password, state)
	if err != nil {
		sendProgress(publish, taskID, 100, fmt.Sprintf("Step 3 failed: %v", err), "failed")
		return nil, err
	}

	// Step 4 – wait for OTP and verify
	sendProgress(publish, taskID, 58, "Waiting for email verification code…", "running")
	otp, err := mp.WaitForCode(ctx, mailAccount, "openai", 120)
	if err != nil {
		// Also try without keyword in case the email doesn't mention "openai"
		sendProgress(publish, taskID, 60, "Retrying OTP wait without keyword filter…", "running")
		otp, err = mp.WaitForCode(ctx, mailAccount, "", 60)
		if err != nil {
			sendProgress(publish, taskID, 100, fmt.Sprintf("OTP wait failed: %v", err), "failed")
			return nil, err
		}
	}
	sendProgress(publish, taskID, 70, fmt.Sprintf("Got OTP: %s – verifying…", otp), "running")
	if err := sess.step4VerifyEmail(ctx, email, otp, state); err != nil {
		sendProgress(publish, taskID, 100, fmt.Sprintf("Step 4 failed: %v", err), "failed")
		return nil, err
	}

	// Step 5 – obtain session/access token
	sendProgress(publish, taskID, 85, "Step 5/5: Obtaining session token…", "running")
	token, tokenErr := sess.step5GetSessionToken(ctx)
	// token errors are non-fatal – the account is still usable with email+password.
	if tokenErr != nil {
		sendProgress(publish, taskID, 90, fmt.Sprintf("Note: %v", tokenErr), "running")
	}

	// Persist account
	encPass, err := crypto.Encrypt(password)
	if err != nil {
		sendProgress(publish, taskID, 100, fmt.Sprintf("Encrypt error: %v", err), "failed")
		return nil, err
	}
	extra := ""
	if token != "" {
		extra = fmt.Sprintf(`{"access_token":"%s"}`, strings.ReplaceAll(token, `"`, `\"`))
	}
	acct := &model.Account{
		Email:       email,
		Password:    encPass,
		Type:        "chatgpt",
		Status:      "active",
		TaskBatchID: taskID,
		Extra:       extra,
	}

	return &ExecutionResult{
		Account:        acct,
		SuccessMessage: fmt.Sprintf("✓ ChatGPT account registered: %s", email),
	}, nil
}
