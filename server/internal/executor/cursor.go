package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/msojocs/ai-auto-register/server/internal/core"
	"github.com/msojocs/ai-auto-register/server/internal/model"
	"github.com/msojocs/ai-auto-register/server/pkg/captcha"
	"github.com/msojocs/ai-auto-register/server/pkg/crypto"
	"github.com/msojocs/ai-auto-register/server/pkg/mailprovider"
	"golang.org/x/net/publicsuffix"
)

// Cursor registration endpoints and Next.js Server-Action hashes.
// Source: https://github.com/lxf746/any-auto-register/blob/main/platforms/cursor/core.py
const (
	cursorAuthBase = "https://authenticator.cursor.sh"
	cursorBase     = "https://cursor.com"

	cursorActionSubmitEmail    = "d0b05a2a36fbe69091c2f49016138171d5c1e4cd"
	cursorActionSubmitPassword = "fef846a39073c935bea71b63308b177b113269b7"
	cursorActionMagicCode      = "f9e8ae3d58a7cd11cccbcdbf210e6f2a6a2550dd"

	cursorTurnstileSiteKey = "0x4AAAAAAAMNIvC45A4Wjjln"
	cursorUA               = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/145.0.0.0 Safari/537.36"

	// nextRouterState is the URL-encoded Next.js router state tree sent in every action request.
	nextRouterState = "%5B%22%22%2C%7B%22children%22%3A%5B%22(main)%22%2C%7B%22children%22%3A%5B%22(root)%22%2C%7B%22children%22%3A%5B%22(sign-in)%22%2C%7B%22children%22%3A%5B%22__PAGE__%22%2C%7B%7D%5D%7D%5D%7D%5D%7D%5D%7D%5D"
)

// CursorExecutor registers new Cursor.sh accounts using the protocol (HTTP) flow.
type CursorExecutor struct{}

func NewCursorExecutor() *CursorExecutor {
	return &CursorExecutor{}
}

// cursorSession wraps two http.Clients that share a cookie jar:
//   - noRedirect – for steps where we need to inspect redirect Location headers
//   - withRedirect – for the final callback step
type cursorSession struct {
	noRedirect   *http.Client
	withRedirect *http.Client
}

func newCursorSession(proxyURL string) (*cursorSession, error) {
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return nil, err
	}
	transport := &http.Transport{}
	if proxyURL != "" {
		u, err := url.Parse(proxyURL)
		if err != nil {
			return nil, fmt.Errorf("cursor: invalid proxy URL: %w", err)
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
	return &cursorSession{noRedirect: noRedir, withRedirect: withRedir}, nil
}

// actionHeaders returns common headers for Next.js Server-Action requests.
func actionHeaders(nextAction, referer, contentType string) map[string]string {
	return map[string]string{
		"User-Agent":             cursorUA,
		"Accept":                 "text/x-component",
		"Content-Type":           contentType,
		"Origin":                 cursorAuthBase,
		"Referer":                referer,
		"Next-Action":            nextAction,
		"Next-Router-State-Tree": nextRouterState,
	}
}

// buildMultipart serialises fields as multipart/form-data using a WebKit-style boundary.
func buildMultipart(fields map[string]string) ([]byte, string, error) {
	boundary := "----WebKitFormBoundary" + randAlphanumStr(16)
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	if err := w.SetBoundary(boundary); err != nil {
		return nil, "", err
	}
	for k, v := range fields {
		if err := w.WriteField(k, v); err != nil {
			return nil, "", err
		}
	}
	w.Close()
	return buf.Bytes(), w.FormDataContentType(), nil
}

func applyHeaders(req *http.Request, headers map[string]string) {
	for k, v := range headers {
		req.Header.Set(k, v)
	}
}

// step1GetSession visits the auth page to seed the session cookie and returns
// the URL-encoded state parameter.
func (s *cursorSession) step1GetSession(ctx context.Context) (string, error) {
	stateEncoded, err := buildCursorState()
	if err != nil {
		return "", err
	}

	reqURL := fmt.Sprintf("%s/?state=%s", cursorAuthBase, stateEncoded)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", cursorUA)
	req.Header.Set("Accept", "text/html")
	resp, err := s.withRedirect.Do(req)
	if err != nil {
		return "", fmt.Errorf("cursor step1: %w", err)
	}
	resp.Body.Close()
	return stateEncoded, nil
}

// step2SubmitEmail posts the email address to the sign-up Next.js action.
func (s *cursorSession) step2SubmitEmail(ctx context.Context, email, stateEncoded string) error {
	body, ct, err := buildMultipart(map[string]string{
		"1_state": stateEncoded,
		"email":   email,
	})
	if err != nil {
		return err
	}
	referer := fmt.Sprintf("%s/sign-up?state=%s", cursorAuthBase, stateEncoded)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cursorAuthBase+"/sign-up", bytes.NewReader(body))
	if err != nil {
		return err
	}
	applyHeaders(req, actionHeaders(cursorActionSubmitEmail, referer, ct))
	resp, err := s.noRedirect.Do(req)
	if err != nil {
		return fmt.Errorf("cursor step2: %w", err)
	}
	resp.Body.Close()
	return nil
}

// step3SubmitPassword posts the password (and optional Turnstile token) to the action.
func (s *cursorSession) step3SubmitPassword(ctx context.Context, email, password, stateEncoded, captchaToken string) error {
	body, ct, err := buildMultipart(map[string]string{
		"1_state":      stateEncoded,
		"email":        email,
		"password":     password,
		"captchaToken": captchaToken,
	})
	if err != nil {
		return err
	}
	referer := fmt.Sprintf("%s/sign-up?state=%s", cursorAuthBase, stateEncoded)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cursorAuthBase+"/sign-up", bytes.NewReader(body))
	if err != nil {
		return err
	}
	applyHeaders(req, actionHeaders(cursorActionSubmitPassword, referer, ct))
	resp, err := s.noRedirect.Do(req)
	if err != nil {
		return fmt.Errorf("cursor step3: %w", err)
	}
	resp.Body.Close()
	return nil
}

var authCodeRe = regexp.MustCompile(`code=([\w-]+)`)

// step4SubmitOTP posts the OTP and extracts the OAuth auth code from the redirect Location.
func (s *cursorSession) step4SubmitOTP(ctx context.Context, email, otp, stateEncoded string) (string, error) {
	body, ct, err := buildMultipart(map[string]string{
		"1_state": stateEncoded,
		"email":   email,
		"otp":     otp,
	})
	if err != nil {
		return "", err
	}
	referer := fmt.Sprintf("%s/sign-up?state=%s", cursorAuthBase, stateEncoded)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cursorAuthBase+"/sign-up", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	applyHeaders(req, actionHeaders(cursorActionMagicCode, referer, ct))
	resp, err := s.noRedirect.Do(req)
	if err != nil {
		return "", fmt.Errorf("cursor step4: %w", err)
	}
	defer resp.Body.Close()

	loc := resp.Header.Get("Location")
	if m := authCodeRe.FindStringSubmatch(loc); len(m) >= 2 {
		return m[1], nil
	}
	// Also search the response body (some versions return it there)
	rawBody, _ := io.ReadAll(resp.Body)
	if m := authCodeRe.FindStringSubmatch(string(rawBody)); len(m) >= 2 {
		return m[1], nil
	}
	return "", fmt.Errorf("cursor step4: auth code not found (location: %q)", loc)
}

// step5GetToken exchanges the auth code for a WorkosCursorSessionToken cookie.
func (s *cursorSession) step5GetToken(ctx context.Context, authCode, stateEncoded string) (string, error) {
	callbackURL := fmt.Sprintf("%s/api/auth/callback?code=%s&state=%s", cursorBase, authCode, stateEncoded)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, callbackURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", cursorUA)
	req.Header.Set("Accept", "text/html")

	// Follow redirects to reach the final page where the session cookie is set.
	resp, err := s.withRedirect.Do(req)
	if err != nil {
		return "", fmt.Errorf("cursor step5: %w", err)
	}
	resp.Body.Close()

	// Check the jar for the session token.
	base, _ := url.Parse(cursorBase)
	for _, c := range s.withRedirect.Jar.Cookies(base) {
		if c.Name == "WorkosCursorSessionToken" {
			val, _ := url.QueryUnescape(c.Value)
			return val, nil
		}
	}
	return "", fmt.Errorf("cursor step5: WorkosCursorSessionToken cookie not found")
}

// randAlphanumStr returns a random alphanumeric string of length n. (crypto/rand, defined in common.go)
// buildCursorState builds a double-URL-encoded JSON state for the Cursor auth flow.
func buildCursorState() (string, error) {
	nonce := randAlphanumStr(32)
	state := map[string]interface{}{
		"returnTo": "https://cursor.com/dashboard",
		"nonce":    nonce,
	}
	stateJSON, err := json.Marshal(state)
	if err != nil {
		return "", err
	}
	return url.QueryEscape(url.QueryEscape(string(stateJSON))), nil
}

// randPassword generates a random 16-char password (crypto/rand, defined in common.go)
// cfgStr extracts a string from a config map (defined in common.go)
// Execute runs the full Cursor registration flow.
//
// Relevant config keys:
//
//	proxy            – optional proxy URL (http/socks5)
//	mail_provider    – "mailtm" (default) | "cfworker"
//	mail_api_url     – base URL for the mail provider
//	mail_admin_token – admin token (cfworker only)
//	mail_domain      – email domain (cfworker only)
//	captcha_provider – "yescaptcha" | "2captcha" (optional but recommended)
//	captcha_key      – API key for the captcha provider
func (e *CursorExecutor) Execute(ctx context.Context, taskID uint, config map[string]interface{}, publish func(core.ProgressUpdate)) (*ExecutionResult, error) {
	sendProgress(publish, taskID, 0, "Starting Cursor account registration", "running")

	proxyURL := cfgStr(config, "proxy", "")

	// ── Temp email ────────────────────────────────────────────────────────────
	mailProviderType := cfgStr(config, "mail_provider", "mailtm")
	mailCfg := map[string]string{
		"api_url":     cfgStr(config, "mail_api_url", ""),
		"admin_token": cfgStr(config, "mail_admin_token", ""),
		"domain":      cfgStr(config, "mail_domain", ""),
	}
	if cfgBool(config, "mail_use_proxy", true) {
		mailCfg["proxy_url"] = proxyURL
	}
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
	var captchaSolver captcha.Solver
	captchaProvider := cfgStr(config, "captcha_provider", "")
	captchaKey := cfgStr(config, "captcha_key", "")
	if captchaProvider != "" && captchaKey != "" {
		captchaSolver, err = captcha.New(captchaProvider, captchaKey)
		if err != nil {
			sendProgress(publish, taskID, 100, fmt.Sprintf("Captcha provider error: %v", err), "failed")
			return nil, err
		}
		sendProgress(publish, taskID, 20, fmt.Sprintf("Captcha provider: %s", captchaProvider), "running")
	}

	// ── Cursor HTTP session ───────────────────────────────────────────────────
	sess, err := newCursorSession(proxyURL)
	if err != nil {
		sendProgress(publish, taskID, 100, fmt.Sprintf("Failed to init HTTP session: %v", err), "failed")
		return nil, err
	}

	// Step 1 – initialise session
	sendProgress(publish, taskID, 25, "Step 1/5: Initialising registration session…", "running")
	stateEncoded, err := sess.step1GetSession(ctx)
	if err != nil {
		sendProgress(publish, taskID, 100, fmt.Sprintf("Step 1 failed: %v", err), "failed")
		return nil, err
	}

	// Step 2 – submit email
	sendProgress(publish, taskID, 35, "Step 2/5: Submitting email…", "running")
	if err := sess.step2SubmitEmail(ctx, email, stateEncoded); err != nil {
		sendProgress(publish, taskID, 100, fmt.Sprintf("Step 2 failed: %v", err), "failed")
		return nil, err
	}

	// Step 3 – submit password (+ Turnstile if solver configured)
	password := randPassword()
	captchaToken := ""
	if captchaSolver != nil {
		sendProgress(publish, taskID, 42, "Solving Turnstile CAPTCHA…", "running")
		captchaToken, err = captchaSolver.SolveTurnstile(ctx, cursorAuthBase, cursorTurnstileSiteKey)
		if err != nil {
			sendProgress(publish, taskID, 100, fmt.Sprintf("CAPTCHA failed: %v", err), "failed")
			return nil, err
		}
		sendProgress(publish, taskID, 48, "CAPTCHA solved", "running")
	}
	sendProgress(publish, taskID, 50, "Step 3/5: Submitting password…", "running")
	if err := sess.step3SubmitPassword(ctx, email, password, stateEncoded, captchaToken); err != nil {
		sendProgress(publish, taskID, 100, fmt.Sprintf("Step 3 failed: %v", err), "failed")
		return nil, err
	}

	// Step 4 – wait for OTP then submit it
	sendProgress(publish, taskID, 55, "Waiting for OTP email…", "running")
	otp, err := mp.WaitForCode(ctx, mailAccount, "", 120)
	if err != nil {
		sendProgress(publish, taskID, 100, fmt.Sprintf("OTP wait failed: %v", err), "failed")
		return nil, err
	}
	sendProgress(publish, taskID, 68, fmt.Sprintf("Got OTP: %s – submitting…", otp), "running")
	authCode, err := sess.step4SubmitOTP(ctx, email, otp, stateEncoded)
	if err != nil {
		sendProgress(publish, taskID, 100, fmt.Sprintf("Step 4 failed: %v", err), "failed")
		return nil, err
	}

	// Step 5 – exchange auth code for session token
	sendProgress(publish, taskID, 80, "Step 5/5: Exchanging auth code for session token…", "running")
	token, err := sess.step5GetToken(ctx, authCode, stateEncoded)
	if err != nil {
		sendProgress(publish, taskID, 100, fmt.Sprintf("Step 5 failed: %v", err), "failed")
		return nil, err
	}

	// Persist account
	encPass, err := crypto.Encrypt(password)
	if err != nil {
		sendProgress(publish, taskID, 100, fmt.Sprintf("Encrypt error: %v", err), "failed")
		return nil, err
	}
	extra := fmt.Sprintf(`{"token":"%s"}`, strings.ReplaceAll(token, `"`, `\"`))
	acct := &model.Account{
		Email:       email,
		Password:    encPass,
		Type:        "cursor",
		Status:      "active",
		TaskBatchID: taskID,
		Extra:       extra,
	}

	return &ExecutionResult{
		Account:        acct,
		SuccessMessage: fmt.Sprintf("✓ Cursor account registered: %s", email),
	}, nil
}
