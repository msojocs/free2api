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
	"regexp"
	"strings"
	"time"

	"github.com/msojocs/free2api/server/internal/core"
	"github.com/msojocs/free2api/server/internal/model"
	"github.com/msojocs/free2api/server/pkg/captcha"
	"github.com/msojocs/free2api/server/pkg/crypto"
	"github.com/msojocs/free2api/server/pkg/mailprovider"
	"golang.org/x/net/publicsuffix"
)

// Grok (x.ai) registration.
// Reference: https://github.com/lxf746/any-auto-register/blob/main/platforms/grok/core.py
const (
	grokAccountsURL      = "https://accounts.x.ai"
	grokTurnstileSiteKey = "0x4AAAAAAAhr9JGVDZbrZOo0"
	grokNextAction       = "7f69646bb11542f4cad728680077c67a09624b94e0"
	grokUA               = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
)

// GrokExecutor registers new Grok (x.ai) accounts.
// It uses gRPC-web for email OTP and standard JSON for signup with Turnstile.
type GrokExecutor struct{}

func NewGrokExecutor() *GrokExecutor {
	return &GrokExecutor{}
}

// ---------------------------------------------------------------------------
// Minimal gRPC-web / protobuf helpers (no external dependency)
// ---------------------------------------------------------------------------

// gRPCVarint encodes n as a protobuf varint.
func gRPCVarint(n uint64) []byte {
	var buf []byte
	for {
		b := byte(n & 0x7F)
		n >>= 7
		if n != 0 {
			b |= 0x80
		}
		buf = append(buf, b)
		if n == 0 {
			break
		}
	}
	return buf
}

// gRPCStringField encodes fieldNumber + wire-type 2 (length-delimited) and the string value.
func gRPCStringField(fieldNumber int, value string) []byte {
	tag := uint64(fieldNumber<<3) | 2
	encoded := []byte(value)
	return append(gRPCVarint(tag), append(gRPCVarint(uint64(len(encoded))), encoded...)...)
}

// gRPCFrame wraps a protobuf body in a gRPC-web data frame (1 byte flag + 4 byte big-endian length).
func gRPCFrame(body []byte) []byte {
	length := len(body)
	frame := make([]byte, 5)
	frame[0] = 0 // no-compression flag
	frame[1] = byte(length >> 24)
	frame[2] = byte(length >> 16)
	frame[3] = byte(length >> 8)
	frame[4] = byte(length)
	return append(frame, body...)
}

// ---------------------------------------------------------------------------
// grok HTTP session
// ---------------------------------------------------------------------------

type grokSession struct {
	noRedirect   *http.Client
	withRedirect *http.Client
}

func newGrokSession(proxyURL string) (*grokSession, error) {
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return nil, err
	}
	transport := &http.Transport{}
	if proxyURL != "" {
		u, err := url.Parse(proxyURL)
		if err != nil {
			return nil, fmt.Errorf("grok: invalid proxy URL: %w", err)
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
	return &grokSession{noRedirect: noRedir, withRedirect: withRedir}, nil
}

func (s *grokSession) grpcPost(ctx context.Context, path string, body []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		grokAccountsURL+path, bytes.NewReader(gRPCFrame(body)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/grpc-web+proto")
	req.Header.Set("X-Grpc-Web", "1")
	req.Header.Set("Origin", "https://accounts.x.ai")
	req.Header.Set("Referer", "https://accounts.x.ai/sign-up")
	req.Header.Set("User-Agent", grokUA)
	resp, err := s.withRedirect.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// step1SendOTP sends a verification code to the email via gRPC-web.
func (s *grokSession) step1SendOTP(ctx context.Context, email string) error {
	body := gRPCStringField(1, email)
	_, err := s.grpcPost(ctx, "/auth_mgmt.AuthManagement/CreateEmailValidationCode", body)
	if err != nil {
		return fmt.Errorf("grok step1: %w", err)
	}
	return nil
}

// step2VerifyOTP verifies the OTP code. Returns true if successful.
func (s *grokSession) step2VerifyOTP(ctx context.Context, email, code string) bool {
	body := append(gRPCStringField(1, email), gRPCStringField(2, code)...)
	resp, err := s.grpcPost(ctx, "/auth_mgmt.AuthManagement/VerifyEmailValidationCode", body)
	if err != nil {
		return false
	}
	return bytes.Contains(resp, []byte("grpc-status:0"))
}

// step3Signup submits the full signup payload and returns the response body.
func (s *grokSession) step3Signup(ctx context.Context, email, password, code, givenName, familyName, captchaToken string) (string, error) {
	payload := []map[string]interface{}{
		{
			"emailValidationCode": code,
			"createUserAndSessionRequest": map[string]interface{}{
				"email":              email,
				"givenName":          givenName,
				"familyName":         familyName,
				"clearTextPassword":  password,
				"tosAcceptedVersion": 1,
			},
			"turnstileToken": captchaToken,
		},
	}
	b, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		grokAccountsURL+"/sign-up", bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Next-Action", grokNextAction)
	req.Header.Set("Origin", "https://accounts.x.ai")
	req.Header.Set("Referer", "https://accounts.x.ai/sign-up")
	req.Header.Set("User-Agent", grokUA)

	resp, err := s.withRedirect.Do(req)
	if err != nil {
		return "", fmt.Errorf("grok step3: %w", err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return string(data), nil
}

var setCookieURLRe = regexp.MustCompile(`https://auth\.[^\s"\\]+/set-cookie[^\s"\\]*`)

// step4SetCookies visits set-cookie URLs found in the signup response.
func (s *grokSession) step4SetCookies(ctx context.Context, signupBody string) {
	raw := strings.ReplaceAll(signupBody, `\u0026`, "&")
	raw = strings.ReplaceAll(raw, `\u003d`, "=")
	for _, u := range setCookieURLRe.FindAllString(raw, -1) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			continue
		}
		req.Header.Set("User-Agent", grokUA)
		req.Header.Set("Accept", "text/html")
		req.Header.Set("Referer", "https://accounts.x.ai/")
		resp, err := s.withRedirect.Do(req)
		if err == nil {
			resp.Body.Close()
		}
	}
}

// getCookies returns the current session cookie map.
func (s *grokSession) getCookies() map[string]string {
	base, _ := url.Parse("https://accounts.x.ai")
	result := make(map[string]string)
	for _, c := range s.withRedirect.Jar.Cookies(base) {
		result[c.Name] = c.Value
	}
	return result
}

// randName returns a random capitalised name.
func randName(n int) string {
	const lower = "abcdefghijklmnopqrstuvwxyz"
	b := make([]byte, n)
	for i := range b {
		b[i] = lower[randIntBig(len(lower))]
	}
	b[0] = b[0] - 32 // uppercase first letter
	return string(b)
}

// randIntBig is a small helper wrapping crypto/rand.Int used inside grok.go.
// (The full version lives in common.go but is declared in the same package, so we use it.)
func randIntBig(n int) int {
	// delegate to the package-level randAlphanumStr helpers which use crypto/rand
	// but here we only need the index — use the shared one from common.go via a local shim.
	return int(safeRandInt(n))
}

// Execute runs the full Grok registration flow.
//
// Config keys:
//
//	proxy, mail_provider, mail_api_url, mail_admin_token, mail_domain
//	captcha_provider, captcha_key  (Turnstile required for signup)
func (e *GrokExecutor) Execute(ctx context.Context, taskID uint, config map[string]interface{}, publish func(core.ProgressUpdate)) (*ExecutionResult, error) {
	sendProgress(publish, taskID, 0, "Starting Grok (x.ai) account registration", "running")

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
	mailAccount, err := mp.GetEmail(ctx)
	if err != nil {
		sendProgress(publish, taskID, 100, fmt.Sprintf("Get email failed: %v", err), "failed")
		return nil, err
	}
	email := mailAccount.Email
	sendProgress(publish, taskID, 18, fmt.Sprintf("Got email: %s", email), "running")

	// ── Captcha solver (required for Grok signup) ─────────────────────────────
	captchaToken := ""
	captchaProvider := cfgStr(config, "captcha_provider", "")
	captchaKey := cfgStr(config, "captcha_key", "")
	if captchaProvider != "" && captchaKey != "" {
		solver, err := captcha.New(captchaProvider, captchaKey)
		if err != nil {
			sendProgress(publish, taskID, 100, fmt.Sprintf("Captcha provider error: %v", err), "failed")
			return nil, err
		}
		sendProgress(publish, taskID, 20, "Solving Turnstile CAPTCHA…", "running")
		captchaToken, err = solver.SolveTurnstile(ctx, "https://accounts.x.ai/sign-up", grokTurnstileSiteKey)
		if err != nil {
			sendProgress(publish, taskID, 100, fmt.Sprintf("CAPTCHA failed: %v", err), "failed")
			return nil, err
		}
		sendProgress(publish, taskID, 28, "CAPTCHA solved", "running")
	}

	sess, err := newGrokSession(proxyURL)
	if err != nil {
		sendProgress(publish, taskID, 100, fmt.Sprintf("Session init error: %v", err), "failed")
		return nil, err
	}

	// Step 1 – send OTP
	sendProgress(publish, taskID, 32, "Step 1/4: Sending OTP email…", "running")
	if err := sess.step1SendOTP(ctx, email); err != nil {
		sendProgress(publish, taskID, 100, fmt.Sprintf("Step 1 failed: %v", err), "failed")
		return nil, err
	}

	// Wait for OTP
	sendProgress(publish, taskID, 42, "Waiting for OTP…", "running")
	otp, err := mp.WaitForCode(ctx, mailAccount, "", 120)
	if err != nil {
		sendProgress(publish, taskID, 100, fmt.Sprintf("OTP wait failed: %v", err), "failed")
		return nil, err
	}
	sendProgress(publish, taskID, 52, fmt.Sprintf("Got OTP: %s", otp), "running")

	// Step 2 – verify OTP
	sendProgress(publish, taskID, 55, "Step 2/4: Verifying OTP…", "running")
	sess.step2VerifyOTP(ctx, email, otp)

	// Step 3 – signup
	password := randPassword()
	givenName := randName(6)
	familyName := randName(6)
	sendProgress(publish, taskID, 65, "Step 3/4: Submitting signup…", "running")
	signupBody, err := sess.step3Signup(ctx, email, password, otp, givenName, familyName, captchaToken)
	if err != nil {
		sendProgress(publish, taskID, 100, fmt.Sprintf("Step 3 failed: %v", err), "failed")
		return nil, err
	}

	// Step 4 – set cookies
	sendProgress(publish, taskID, 78, "Step 4/4: Setting session cookies…", "running")
	sess.step4SetCookies(ctx, signupBody)

	cookies := sess.getCookies()
	sso := cookies["sso"]

	// Persist
	encPass, err := crypto.Encrypt(password)
	if err != nil {
		sendProgress(publish, taskID, 100, fmt.Sprintf("Encrypt error: %v", err), "failed")
		return nil, err
	}
	extraMap := map[string]string{
		"sso":         sso,
		"sso_rw":      cookies["sso-rw"],
		"given_name":  givenName,
		"family_name": familyName,
	}
	extraJSON, _ := json.Marshal(extraMap)
	acct := &model.Account{
		Email:       email,
		Password:    encPass,
		Type:        "grok",
		Status:      "active",
		TaskBatchID: taskID,
		Extra:       string(extraJSON),
	}

	msg := fmt.Sprintf("✓ Grok account registered: %s", email)
	if sso != "" {
		msg += fmt.Sprintf(" (sso=%s…)", sso[:min(20, len(sso))])
	} else {
		msg += " (no sso cookie — Cloudflare may have blocked the request)"
	}

	return &ExecutionResult{
		Account:        acct,
		SuccessMessage: msg,
	}, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
