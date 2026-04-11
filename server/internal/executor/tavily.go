package executor

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
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

// Tavily registration (Auth0 PKCE flow).
// Reference: https://github.com/lxf746/any-auto-register/blob/main/platforms/tavily/core.py
const (
	tavilyAuth0ClientID = "RRIAvvXNFxpfTWIozX1mXqLnyUmYSTrQ"
	tavilyAuth0Base     = "https://auth.tavily.com"
	tavilyAppBase       = "https://app.tavily.com"
	tavilyRedirectURI   = "https://app.tavily.com/api/auth/callback"
	tavilyTurnstileKey  = "0x4AAAAAAAQFNSW6xordsuIq"
	tavilyUA            = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/145.0.0.0 Safari/537.36"
)

// TavilyExecutor registers new Tavily accounts.
type TavilyExecutor struct{}

func NewTavilyExecutor() *TavilyExecutor {
	return &TavilyExecutor{}
}

type tavilySession struct {
	noRedirect   *http.Client
	withRedirect *http.Client
}

func newTavilySession(proxyURL string) (*tavilySession, error) {
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return nil, err
	}
	transport := &http.Transport{}
	if proxyURL != "" {
		u, err := url.Parse(proxyURL)
		if err != nil {
			return nil, fmt.Errorf("tavily: invalid proxy URL: %w", err)
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
	return &tavilySession{noRedirect: noRedir, withRedirect: withRedir}, nil
}

func (s *tavilySession) get(ctx context.Context, client *http.Client, rawURL string, params url.Values) (*http.Response, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	if params != nil {
		u.RawQuery = params.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", tavilyUA)
	req.Header.Set("Accept", "text/html,application/json")
	return client.Do(req)
}

func (s *tavilySession) postForm(ctx context.Context, rawURL string, params, form url.Values) (*http.Response, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	if params != nil {
		u.RawQuery = params.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", tavilyUA)
	req.Header.Set("Accept", "text/html,application/json")
	return s.noRedirect.Do(req)
}

var stateParamRe = regexp.MustCompile(`[?&]state=([^&\s"]+)`)

func extractStateFromResponse(body []byte, loc string) string {
	for _, src := range []string{loc, string(body)} {
		if m := stateParamRe.FindStringSubmatch(src); len(m) >= 2 {
			v, _ := url.QueryUnescape(m[1])
			return v
		}
	}
	return ""
}

// generatePKCE creates a PKCE code_verifier and its S256 challenge.
func generatePKCE() (verifier, challenge string) {
	raw := make([]byte, 43)
	for i := range raw {
		raw[i] = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._~"[safeRandInt(66)]
	}
	verifier = string(raw)
	h := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(h[:])
	return
}

// step1Authorize initiates the Auth0 PKCE flow and returns the state parameter.
func (s *tavilySession) step1Authorize(ctx context.Context) (string, error) {
	nonce := randAlphanumStr(32)
	_, codeChallenge := generatePKCE()
	stateVal := base64.RawURLEncoding.EncodeToString([]byte(`{"returnTo":"` + tavilyAppBase + `/home"}`))

	params := url.Values{
		"client_id":             {tavilyAuth0ClientID},
		"scope":                 {"openid profile email"},
		"response_type":         {"code"},
		"redirect_uri":          {tavilyRedirectURI},
		"nonce":                 {nonce},
		"state":                 {stateVal},
		"screen_hint":           {"signup"},
		"code_challenge":        {codeChallenge},
		"code_challenge_method": {"S256"},
	}
	resp, err := s.get(ctx, s.noRedirect, tavilyAuth0Base+"/authorize", params)
	if err != nil {
		return "", fmt.Errorf("tavily step1: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	loc := resp.Header.Get("Location")
	if state := extractStateFromResponse(body, loc); state != "" {
		return state, nil
	}
	return stateVal, nil
}

// step2SolveCaptcha solves the Turnstile challenge.
func (s *tavilySession) step2SolveCaptcha(ctx context.Context, solver captcha.Solver) (string, error) {
	return solver.SolveTurnstile(ctx, tavilyAuth0Base, tavilyTurnstileKey)
}

// step3SubmitEmail posts the email to the Auth0 identifier endpoint and returns the next state.
func (s *tavilySession) step3SubmitEmail(ctx context.Context, email, state, captchaToken string) (string, error) {
	resp, err := s.postForm(ctx,
		tavilyAuth0Base+"/u/signup/identifier",
		url.Values{"state": {state}},
		url.Values{"state": {state}, "email": {email}, "captcha": {captchaToken}},
	)
	if err != nil {
		return state, fmt.Errorf("tavily step3: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	loc := resp.Header.Get("Location")
	if nextState := extractStateFromResponse(body, loc); nextState != "" {
		return nextState, nil
	}
	return state, nil
}

// step4SubmitOTP posts the email OTP and returns the next state.
func (s *tavilySession) step4SubmitOTP(ctx context.Context, otp, challengeState string) (string, error) {
	resp, err := s.postForm(ctx,
		tavilyAuth0Base+"/u/email-identifier/challenge",
		url.Values{"state": {challengeState}},
		url.Values{"state": {challengeState}, "code": {otp}},
	)
	if err != nil {
		return challengeState, fmt.Errorf("tavily step4: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	loc := resp.Header.Get("Location")
	if nextState := extractStateFromResponse(body, loc); nextState != "" {
		return nextState, nil
	}
	return challengeState, nil
}

// step5SubmitPassword sets the account password and returns the resume state.
func (s *tavilySession) step5SubmitPassword(ctx context.Context, email, password, pwState string) (string, error) {
	resp, err := s.postForm(ctx,
		tavilyAuth0Base+"/u/signup/password",
		url.Values{"state": {pwState}},
		url.Values{
			"state":                       {pwState},
			"email":                       {email},
			"password":                    {password},
			"passwordPolicy.isFlexible":   {"false"},
			"strengthPolicy":              {"good"},
			"complexityOptions.minLength": {"8"},
		},
	)
	if err != nil {
		return pwState, fmt.Errorf("tavily step5: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	loc := resp.Header.Get("Location")
	if nextState := extractStateFromResponse(body, loc); nextState != "" {
		return nextState, nil
	}
	return pwState, nil
}

// step6ResumeAndGetKey completes the OAuth redirect and retrieves the API key.
func (s *tavilySession) step6ResumeAndGetKey(ctx context.Context, resumeState string) (string, error) {
	resp, err := s.get(ctx, s.withRedirect, tavilyAuth0Base+"/authorize/resume",
		url.Values{"state": {resumeState}})
	if err != nil {
		return "", fmt.Errorf("tavily step6 resume: %w", err)
	}
	resp.Body.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, tavilyAppBase+"/api/keys", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", tavilyUA)
	keysResp, err := s.withRedirect.Do(req)
	if err != nil {
		return "", fmt.Errorf("tavily step6 get keys: %w", err)
	}
	defer keysResp.Body.Close()
	data, _ := io.ReadAll(keysResp.Body)

	var keys []map[string]interface{}
	if err := json.Unmarshal(data, &keys); err == nil && len(keys) > 0 {
		if k, ok := keys[0]["key"].(string); ok {
			return k, nil
		}
	}
	return "", nil
}

// Execute runs the full Tavily registration flow.
//
// Config keys:
//
//	proxy, mail_provider, mail_api_url, mail_admin_token, mail_domain
//	captcha_provider, captcha_key  (Turnstile required)
func (e *TavilyExecutor) Execute(ctx context.Context, taskID uint, config map[string]interface{}, publish func(core.ProgressUpdate)) (*ExecutionResult, error) {
	sendProgress(publish, taskID, 0, "Starting Tavily account registration", "running")

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
	sendProgress(publish, taskID, 15, fmt.Sprintf("Got email: %s", email), "running")

	// ── Captcha solver (required for Tavily step3) ────────────────────────────
	captchaProvider := cfgStr(config, "captcha_provider", "")
	captchaKey := cfgStr(config, "captcha_key", "")
	var solver captcha.Solver
	if captchaProvider != "" && captchaKey != "" {
		solver, err = captcha.New(captchaProvider, captchaKey)
		if err != nil {
			sendProgress(publish, taskID, 100, fmt.Sprintf("Captcha provider error: %v", err), "failed")
			return nil, err
		}
	}

	sess, err := newTavilySession(proxyURL)
	if err != nil {
		sendProgress(publish, taskID, 100, fmt.Sprintf("Session init error: %v", err), "failed")
		return nil, err
	}

	// Step 1 – authorize
	sendProgress(publish, taskID, 20, "Step 1/6: Auth0 authorize…", "running")
	state, err := sess.step1Authorize(ctx)
	if err != nil {
		sendProgress(publish, taskID, 100, fmt.Sprintf("Step 1 failed: %v", err), "failed")
		return nil, err
	}

	// Step 2 – Turnstile
	captchaToken := ""
	if solver != nil {
		sendProgress(publish, taskID, 28, "Step 2/6: Solving Turnstile…", "running")
		captchaToken, err = sess.step2SolveCaptcha(ctx, solver)
		if err != nil {
			sendProgress(publish, taskID, 100, fmt.Sprintf("Step 2 CAPTCHA failed: %v", err), "failed")
			return nil, err
		}
		sendProgress(publish, taskID, 35, "CAPTCHA solved", "running")
	}

	// Step 3 – submit email
	sendProgress(publish, taskID, 38, "Step 3/6: Submitting email…", "running")
	challengeState, err := sess.step3SubmitEmail(ctx, email, state, captchaToken)
	if err != nil {
		sendProgress(publish, taskID, 100, fmt.Sprintf("Step 3 failed: %v", err), "failed")
		return nil, err
	}

	// Wait for OTP
	sendProgress(publish, taskID, 45, "Waiting for OTP…", "running")
	otp, err := mp.WaitForCode(ctx, mailAccount, "tavily", 120)
	if err != nil {
		sendProgress(publish, taskID, 100, fmt.Sprintf("OTP wait failed: %v", err), "failed")
		return nil, err
	}
	sendProgress(publish, taskID, 55, fmt.Sprintf("Got OTP: %s", otp), "running")

	// Step 4 – submit OTP
	sendProgress(publish, taskID, 58, "Step 4/6: Submitting OTP…", "running")
	pwState, err := sess.step4SubmitOTP(ctx, otp, challengeState)
	if err != nil {
		sendProgress(publish, taskID, 100, fmt.Sprintf("Step 4 failed: %v", err), "failed")
		return nil, err
	}

	// Step 5 – set password
	password := randPassword()
	sendProgress(publish, taskID, 68, "Step 5/6: Setting password…", "running")
	resumeState, err := sess.step5SubmitPassword(ctx, email, password, pwState)
	if err != nil {
		sendProgress(publish, taskID, 100, fmt.Sprintf("Step 5 failed: %v", err), "failed")
		return nil, err
	}

	// Step 6 – resume and get API key
	sendProgress(publish, taskID, 80, "Step 6/6: Fetching API key…", "running")
	apiKey, err := sess.step6ResumeAndGetKey(ctx, resumeState)
	if err != nil {
		// non-fatal: account may exist without key
		sendProgress(publish, taskID, 85, fmt.Sprintf("Note: API key retrieval: %v", err), "running")
	}

	// Persist
	encPass, err := crypto.Encrypt(password)
	if err != nil {
		sendProgress(publish, taskID, 100, fmt.Sprintf("Encrypt error: %v", err), "failed")
		return nil, err
	}
	extraMap := map[string]string{"api_key": apiKey}
	extraJSON, _ := json.Marshal(extraMap)
	acct := &model.Account{
		Email:       email,
		Password:    encPass,
		Type:        "tavily",
		Status:      "active",
		TaskBatchID: taskID,
		Extra:       string(extraJSON),
	}

	msg := fmt.Sprintf("✓ Tavily account registered: %s", email)
	if apiKey != "" {
		msg += fmt.Sprintf(" (api_key=%s…)", apiKey[:min(20, len(apiKey))])
	}

	return &ExecutionResult{
		Account:        acct,
		SuccessMessage: msg,
	}, nil
}
