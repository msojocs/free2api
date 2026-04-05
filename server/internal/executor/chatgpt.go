package executor

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/msojocs/free2api/server/internal/core"
	"github.com/msojocs/free2api/server/internal/model"
	"github.com/msojocs/free2api/server/pkg/mailprovider"
	"github.com/msojocs/free2api/server/pkg/openai"
	"golang.org/x/net/publicsuffix"
)

// OpenAI / ChatGPT registration endpoints.
// The signup flow uses auth0.openai.com with PKCE and email OTP verification.
// NOTE: OpenAI deploys Cloudflare WAF + Arkose FunCaptcha on the signup endpoint.
// This implementation performs the HTTP protocol flow; additional bot-bypass
// tooling (TLS fingerprinting, CAPTCHA solver) may be required in production.
const (
	openAIAuthBase = "https://auth.openai.com"

	chatGPTUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36"
)

// ChatGPTExecutor registers new OpenAI / ChatGPT accounts using the HTTP protocol flow.
type ChatGPTExecutor struct {
	step string
}

func NewChatGPTExecutor() *ChatGPTExecutor {
	return &ChatGPTExecutor{}
}

// openAISession holds the HTTP clients and helpers for the auth0 sign-up flow.
type openAISession struct {
	noRedirect    *http.Client
	withRedirect  *http.Client
	sentinelToken *openai.SentinelToken
}
type openAIPrepareResult struct {
	OaiDid       string
	CodeVerifier string
}

type openAITokenResult struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	IdToken      string `json:"id_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
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

func (s *openAISession) checkIpLocation(ctx context.Context) error {
	resp, err := s.withRedirect.Get("https://cloudflare.com/cdn-cgi/trace")
	if err != nil {
		return fmt.Errorf("IP location check failed: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	lines := strings.Split(string(body), "\n")
	loc := ""
	for _, line := range lines {
		if strings.HasPrefix(line, "loc=") {
			loc = strings.TrimPrefix(line, "loc=")
			break
		}
	}
	// OpenAI may block certain regions; adjust as needed.
	blockedRegions := map[string]bool{
		"CN": true,
		"HK": true,
		"MO": true,
		"TW": true,
	}
	if blockedRegions[loc] {
		return fmt.Errorf("IP geolocation check: access from region %s may be blocked by OpenAI", loc)
	}
	return nil
}

func (s *openAISession) generateCodeVerifier() (string, error) {
	// 生成 64 字节的随机数据
	b := make([]byte, 64)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	// 使用 URLEncoding 并移除填充字符 '='
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func (s *openAISession) generateCodeChallenge(codeVerifier string) string {
	h := sha256.New()
	h.Write([]byte(codeVerifier))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

// prepareSession visits the signup page to obtain the oid-did cookies and sentinel token.
func (s *openAISession) prepareSession(ctx context.Context) (*openAIPrepareResult, error) {

	result := &openAIPrepareResult{}
	// 0. 准备Code Verifier和Code Challenge
	codeVerifier, err := s.generateCodeVerifier()
	if err != nil {
		return nil, fmt.Errorf("failed to generate code verifier: %w", err)
	}
	result.CodeVerifier = codeVerifier
	codeChallenge := s.generateCodeChallenge(codeVerifier)

	// 1. 浏览器跳转authorize
	{
		reqURL := fmt.Sprintf("%s/oauth/authorize?response_type=code&client_id=app_EMoamEEZ73f0CkXaXp7hrann&redirect_uri=http%%3A%%2F%%2Flocalhost%%3A1455%%2Fauth%%2Fcallback&scope=openid%%20profile%%20email%%20offline_access%%20api.connectors.read%%20api.connectors.invoke&code_challenge=%s&code_challenge_method=S256&id_token_add_organizations=true&codex_cli_simplified_flow=true&state=KWXvVMO9vEH6BDLTcgQAjmSVeczW5h4FzI0FdEpUHEs&originator=Codex%%20Desktop", openAIAuthBase, codeChallenge)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", chatGPTUA)
		req.Header.Set("Accept", "text/html")
		resp, err := s.withRedirect.Do(req)
		if err != nil {
			return nil, fmt.Errorf("chatgpt step1: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("chatgpt step1: unexpected status code: %d, last url: %s", resp.StatusCode, resp.Request.URL)
		}
		cookies := s.withRedirect.Jar.Cookies(req.URL)
		var oaiDid string
		for _, c := range cookies {
			log.Printf("cookie name:%s", c.Name)
			if c.Name == "oai-did" {
				oaiDid = c.Value
				break
			}
		}
		if oaiDid == "" {
			return nil, fmt.Errorf("chatgpt step1: oai_did cookie not found; Cloudflare protection may be active")
		}
		result.OaiDid = oaiDid
	}

	// 2. sentinel token获取
	{
		// TODO: 地址配置化
		s.sentinelToken = openai.NewSentinelToken("http://127.0.0.1:3000", "authorize_continue", result.OaiDid)
		s.sentinelToken.Req(s.noRedirect)
	}
	return result, nil
}

// isEmailNeedSignUp submits the email to the auth0 signup endpoint.
func (s *openAISession) isEmailNeedSignUp(ctx context.Context, email string, prepareResult *openAIPrepareResult) (bool, error) {
	reqURL := fmt.Sprintf("%s/api/accounts/authorize/continue", openAIAuthBase)

	josnBody := map[string]interface{}{
		"username": map[string]string{
			"kind":  "email",
			"value": email,
		},
		"screen_hint": "signup",
	}
	str, err := json.Marshal(josnBody)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, strings.NewReader(string(str)))
	if err != nil {
		return false, err
	}

	{
		time.Sleep(time.Second * 3)
		headers, err := s.sentinelToken.GetSentinelHeader()
		if err != nil {
			return false, fmt.Errorf("Check email: failed to get sentinel header: %w", err)
		}
		sentinelTokenMap := map[string]string{
			"p":    headers["p"],
			"t":    headers["t"],
			"c":    headers["c"],
			"id":   prepareResult.OaiDid,
			"flow": "authorize_continue",
		}
		sentinelToken, err := json.Marshal(sentinelTokenMap)
		if err != nil {
			return false, fmt.Errorf("Check email: failed to marshal sentinel token: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", chatGPTUA)
		req.Header.Set("Referer", "https://auth.openai.com/create-account")
		req.Header.Set("openai-sentinel-token", string(sentinelToken))
	}

	resp, err := s.noRedirect.Do(req)
	if err != nil {
		return false, fmt.Errorf("Check email: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var continueResult struct {
		Page struct {
			Type string `json:"type"`
		} `json:"page"`
	}
	if err := json.Unmarshal(body, &continueResult); err != nil {
		return false, fmt.Errorf("Check email: failed to parse response: %w", err)
	}

	switch continueResult.Page.Type {
	case "email_otp_verification":
		// otp验证，邮箱已经被使用了，不需要注册
		return false, nil
	case "create_account_password":
		// 填写密码，邮箱没被使用了，需要注册
		return true, nil
	default:
		// 非预期的响应类型
		return false, fmt.Errorf("Check email: unexpected page type: %s; response: %s", continueResult.Page.Type, string(body))
	}

}

// setPassword submits the password to the auth0 signup continuation endpoint.
func (s *openAISession) setPassword(ctx context.Context, email, password string, prepareResult *openAIPrepareResult) (string, error) {
	reqURL := fmt.Sprintf("%s/api/accounts/user/register", openAIAuthBase)

	s.sentinelToken = openai.NewSentinelToken("http://127.0.0.1:3000", "username_password_create", prepareResult.OaiDid)
	s.sentinelToken.Req(s.noRedirect)
	jsonData := map[string]string{
		"username": email,
		"password": password,
	}
	str, err := json.Marshal(jsonData)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, strings.NewReader(string(str)))
	if err != nil {
		return "", err
	}

	{
		time.Sleep(time.Second * 3)
		headers, err := s.sentinelToken.GetSentinelHeader()
		if err != nil {
			return "", fmt.Errorf("chatgpt step3: failed to get sentinel header: %w", err)
		}
		sentinelTokenMap := map[string]string{
			"p":    headers["p"],
			"t":    headers["t"],
			"c":    headers["c"],
			"id":   prepareResult.OaiDid,
			"flow": "authorize_continue",
		}
		sentinelToken, err := json.Marshal(sentinelTokenMap)
		if err != nil {
			return "", fmt.Errorf("chatgpt step3: failed to marshal sentinel token: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", chatGPTUA)
		req.Header.Set("Referer", "https://auth.openai.com/create-account/password")
		req.Header.Set("openai-sentinel-token", string(sentinelToken))
	}

	resp, err := s.noRedirect.Do(req)
	if err != nil {
		return "", fmt.Errorf("chatgpt step3: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("chatgpt step3: unexpected status code: %d; response: %s", resp.StatusCode, string(body))
	}

	var passResult struct {
		Page struct {
			Type string `json:"type"`
		} `json:"page"`
	}
	if err := json.Unmarshal(body, &passResult); err != nil {
		return "", fmt.Errorf("chatgpt step3: failed to parse response: %w; response: %s", err, string(body))
	}
	if passResult.Page.Type != "email_otp_send" {
		return "", fmt.Errorf("chatgpt step3: unexpected page type: %s; response: %s", passResult.Page.Type, string(body))
	}
	return "", nil
}

func (s *openAISession) sendOtp(ctx context.Context, prepareResult *openAIPrepareResult) error {
	reqUrl := fmt.Sprintf("%s/api/accounts/email-otp/send", openAIAuthBase)
	// GET
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqUrl, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", chatGPTUA)
	req.Header.Set("Referer", "https://auth.openai.com/create-account/password")
	resp, err := s.withRedirect.Do(req)
	if err != nil {
		return fmt.Errorf("chatgpt sendOtp: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("chatgpt sendOtp: unexpected status code: %d; response: %s", resp.StatusCode, string(body))
	}
	return nil
}

func (s *openAISession) resendOtp(ctx context.Context) error {
	reqUrl := fmt.Sprintf("%s/api/accounts/email-otp/resend", openAIAuthBase)
	// GET
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqUrl, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", chatGPTUA)
	req.Header.Set("Referer", "https://auth.openai.com/create-account/password")
	resp, err := s.withRedirect.Do(req)
	if err != nil {
		return fmt.Errorf("chatgpt resendOtp: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("chatgpt resendOtp: unexpected status code: %d; response: %s", resp.StatusCode, string(body))
	}
	var result struct {
		Success bool `json:"success"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("chatgpt resendOtp: failed to parse response: %w; response: %s", err, string(body))
	}
	if !result.Success {
		return fmt.Errorf("chatgpt resendOtp: resend failed; response: %s", string(body))
	}
	return nil
}

// verifyEmailOTP submits the email verification OTP.
func (s *openAISession) verifyEmailOTP(ctx context.Context, email, otp string, prepareResult *openAIPrepareResult) error {
	reqURL := fmt.Sprintf("%s/api/accounts/email-otp/validate", openAIAuthBase)

	jsonData := map[string]string{
		"code": otp,
	}
	str, err := json.Marshal(jsonData)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, strings.NewReader(string(str)))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", chatGPTUA)
	req.Header.Set("Referer", "https://auth.openai.com/email-verification")

	time.Sleep(time.Second * 3)
	resp, err := s.noRedirect.Do(req)
	if err != nil {
		return fmt.Errorf("chatgpt step4: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("chatgpt step4: unexpected status code: %d; response: %s", resp.StatusCode, string(body))
	}
	var otpResult struct {
		Page struct {
			Type string `json:"type"`
		} `json:"page"`
	}
	if err := json.Unmarshal(body, &otpResult); err != nil {
		return fmt.Errorf("chatgpt step4: failed to parse response: %w; response: %s", err, string(body))
	}
	if otpResult.Page.Type != "about_you" {
		return fmt.Errorf("chatgpt step4: unexpected page type: %s; response: %s", otpResult.Page.Type, string(body))
	}
	return nil
}

func (s *openAISession) createAccount(ctx context.Context, prepareResult *openAIPrepareResult) error {
	reqUrl := fmt.Sprintf("%s/api/accounts/create_account", openAIAuthBase)
	jsonData := map[string]string{
		"name":      "micro jans",
		"birthdate": "1995-04-04",
	}
	str, err := json.Marshal(jsonData)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqUrl, strings.NewReader(string(str)))
	if err != nil {
		return err
	}

	s.sentinelToken = openai.NewSentinelToken("http://127.0.0.1:3000", "oauth_create_account", prepareResult.OaiDid)
	s.sentinelToken.Req(s.noRedirect)
	{
		time.Sleep(time.Second * 3)
		headers, err := s.sentinelToken.GetSentinelHeader()
		if err != nil {
			return fmt.Errorf("chatgpt createAccount: failed to get sentinel header: %w", err)
		}
		sentinelTokenMap := map[string]string{
			"p":    headers["p"],
			"t":    headers["t"],
			"c":    headers["c"],
			"id":   prepareResult.OaiDid,
			"flow": "oauth_create_account",
		}
		sentinelToken, err := json.Marshal(sentinelTokenMap)
		if err != nil {
			return fmt.Errorf("chatgpt createAccount: failed to marshal sentinel token: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", chatGPTUA)
		req.Header.Set("Referer", "https://auth.openai.com/about-you")
		req.Header.Set("openai-sentinel-token", string(sentinelToken))
	}
	resp, err := s.noRedirect.Do(req)
	if err != nil {
		return fmt.Errorf("chatgpt createAccount: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("chatgpt createAccount: unexpected status code: %d; response: %s", resp.StatusCode, string(body))
	}
	var createResult struct {
		Page struct {
			Type string `json:"type"`
		} `json:"page"`
	}
	if err := json.Unmarshal(body, &createResult); err != nil {
		return fmt.Errorf("chatgpt createAccount: failed to parse response: %w; response: %s", err, string(body))
	}
	if createResult.Page.Type == "add_phone" {
		log.Printf("[WARN] chatgpt createAccount: phone verification required; account may be limited or blocked")
		return nil
	}
	return nil
}

// getCallbackUrl completes the OAuth callback and extracts the session token.
func (s *openAISession) getCallbackUrl(ctx context.Context) (string, error) {
	reqUrl := fmt.Sprintf("%s/sign-in-with-chatgpt/codex/consent", openAIAuthBase)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqUrl, nil)
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

	// 格式："id\",\"90ae07ea-04d5-4dd5-93bc-ee71ef33f3cd\",\"profile_picture_alt_text
	bodyStr := string(body)
	hasWorkspaceId := strings.Contains(bodyStr, "\"id")
	if !hasWorkspaceId {
		return "", fmt.Errorf("chatgpt step5: workspace id not found in response. %s", bodyStr[0:200])
	}
	parts := strings.Split(bodyStr, "\"id\",\"")
	if len(parts) < 2 {
		return "", fmt.Errorf("chatgpt step5: unexpected response format; 'id' field not found. %s", bodyStr[0:200])
	}
	idPart := parts[1]
	idParts := strings.Split(idPart, "\",\"profile_picture_alt_text")
	if len(idParts) < 1 {
		return "", fmt.Errorf("chatgpt step5: unexpected response format; 'id' field not properly terminated. %s", bodyStr[0:200])
	}
	workspaceId := idParts[0]

	var continueUrl string
	{
		// https://auth.openai.com/api/accounts/workspace/select
		selectURL := fmt.Sprintf("%s/api/accounts/workspace/select", openAIAuthBase)
		jsonData := map[string]string{
			"workspace_id": workspaceId,
		}
		str, err := json.Marshal(jsonData)
		if err != nil {
			return "", err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, selectURL, strings.NewReader(string(str)))
		if err != nil {
			return "", err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", chatGPTUA)
		resp, err := s.withRedirect.Do(req)
		if err != nil {
			return "", fmt.Errorf("chatgpt step5 workspace select: %w", err)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)

		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("chatgpt step5 workspace select: unexpected status code: %d; response: %s", resp.StatusCode, string(body))
		}
		var selectResult struct {
			ContinueURL string `json:"continue_url"`
			Page        struct {
				Type string `json:"type"`
			} `json:"page"`
		}
		if err := json.Unmarshal(body, &selectResult); err != nil {
			return "", fmt.Errorf("chatgpt step5 workspace select: failed to parse response: %w; response: %s", err, string(body))
		}
		continueUrl = selectResult.ContinueURL
	}
	{
		// 访问 continue_url 获取最终的 session token cookie
		finalUrl := continueUrl
		for i := 0; i < 3; i++ {
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, finalUrl, nil)
			if err != nil {
				return "", err
			}
			req.Header.Set("User-Agent", chatGPTUA)
			resp, err := s.noRedirect.Do(req)
			if err != nil {
				return "", fmt.Errorf("chatgpt step5 final redirect: %w", err)
			}
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)

			if resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusSeeOther {
				return "", fmt.Errorf("chatgpt step5 final redirect: unexpected status code: %d; response: %s", resp.StatusCode, string(body))
			}
			finalUrl = resp.Header.Get("Location")
			if finalUrl == "" {
				return "", fmt.Errorf("chatgpt step5 final redirect: Location header not found; response: %s", string(body))
			}
		}
		if !strings.Contains(finalUrl, "http://localhost") || !strings.Contains(finalUrl, "code") || !strings.Contains(finalUrl, "state") {
			return "", fmt.Errorf("chatgpt step5 final redirect: unexpected final URL: %s", finalUrl)
		}

		return finalUrl, nil
	}

}

func (s *openAISession) getTokenInfo(ctx context.Context, callbackUrl string, prepareResult *openAIPrepareResult) (*openAITokenResult, error) {
	// https://auth.openai.com/oauth/token
	uri, err := url.Parse(callbackUrl)
	if err != nil {
		return nil, fmt.Errorf("chatgpt getAccountInfo: failed to parse callback URL: %w", err)
	}
	// code state
	query := uri.Query()
	code := query.Get("code")
	state := query.Get("state")
	if code == "" || state == "" {
		return nil, fmt.Errorf("chatgpt getAccountInfo: code or state not found in callback URL: %s", callbackUrl)
	}

	reqURL := fmt.Sprintf("%s/oauth/token", openAIAuthBase)
	formData := url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {"app_EMoamEEZ73f0CkXaXp7hrann"},
		"code":          {code},
		"redirect_uri":  {"http://localhost:1455/auth/callback"},
		"code_verifier": {prepareResult.CodeVerifier},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("chatgpt getAccountInfo: failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", chatGPTUA)

	resp, err := s.noRedirect.Do(req)
	if err != nil {
		return nil, fmt.Errorf("chatgpt getAccountInfo: request failed: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("chatgpt getAccountInfo: unexpected status code: %d; response: %s", resp.StatusCode, string(body))
	}
	var tokenResult openAITokenResult
	if err := json.Unmarshal(body, &tokenResult); err != nil {
		return nil, fmt.Errorf("chatgpt getAccountInfo: failed to parse response: %w; response: %s", err, string(body))
	}
	return &tokenResult, nil

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

	// ── OpenAI HTTP session ───────────────────────────────────────────────────
	sess, err := newOpenAISession(proxyURL)
	if err != nil {
		sendProgress(publish, taskID, 100, fmt.Sprintf("Failed to init HTTP session: %v", err), "failed")
		return nil, err
	}

	// Check location of the IP address, as OpenAI may block certain regions.
	sendProgress(publish, taskID, 8, "Checking IP geolocation…", "running")
	err = sess.checkIpLocation(ctx)
	if err != nil {
		sendProgress(publish, taskID, 100, fmt.Sprintf("IP location check failed: %v", err), "failed")
		return nil, err
	}

	// ── Temp email ────────────────────────────────────────────────────────────
	mailProviderType := cfgStr(config, "mail_provider", "tempmail")
	mailCfg := map[string]string{
		"api_url":     cfgStr(config, "mail_api_url", ""),
		"admin_token": cfgStr(config, "mail_admin_token", ""),
		"domain":      cfgStr(config, "mail_domain", ""),
	}
	sendProgress(publish, taskID, 12, fmt.Sprintf("Initialising mail provider: %s", mailProviderType), "running")
	mp, err := mailprovider.New(mailProviderType, mailCfg)
	if err != nil {
		sendProgress(publish, taskID, 100, fmt.Sprintf("Mail provider error: %v", err), "failed")
		return nil, err
	}

	sendProgress(publish, taskID, 22, "Getting temporary email address…", "running")
	mailAccount, err := mp.GetEmail(ctx)
	if err != nil {
		sendProgress(publish, taskID, 100, fmt.Sprintf("Get email failed: %v", err), "failed")
		return nil, err
	}
	email := mailAccount.Email
	sendProgress(publish, taskID, 24, fmt.Sprintf("Got email: %s", email), "running")

	e.step = "prepare_session"
	var stepContext struct {
		prepareResult *openAIPrepareResult
		password      string
		passResult    string
		callbackUrl   string
		tokenInfo     *openAITokenResult
		resendCount   int
	}
executor_loop:
	for {
		switch e.step {
		case "prepare_session":
			// Step 1 – seed session / get state
			sendProgress(publish, taskID, 30, "Step 1/5: Seeding registration session…", "running")
			prepareResult, err := sess.prepareSession(ctx)
			if err != nil {
				sendProgress(publish, taskID, 100, fmt.Sprintf("Step 1 failed: %v", err), "failed")
				return nil, err
			}
			if prepareResult.OaiDid == "" {
				// This typically means Cloudflare blocked the request.
				err = fmt.Errorf("could not obtain auth state – Cloudflare protection may be active; try with a residential proxy")
				sendProgress(publish, taskID, 100, err.Error(), "failed")
				return nil, err
			}
			stepContext.prepareResult = prepareResult
			e.step = "check_email"
		case "check_email":
			needSignUp, err := sess.isEmailNeedSignUp(ctx, email, stepContext.prepareResult)
			if err != nil {
				sendProgress(publish, taskID, 100, fmt.Sprintf("Check email failed: %v", err), "failed")
				return nil, err
			}
			if needSignUp {
				e.step = "set_password"
			} else {
				e.step = "wait_for_otp"
			}
		case "set_password":
			// Step 3 – set password
			password := randPassword()
			sendProgress(publish, taskID, 50, "Step 3/5: Setting password…", "running")
			passResult, err := sess.setPassword(ctx, email, password, stepContext.prepareResult)
			if err != nil {
				sendProgress(publish, taskID, 100, fmt.Sprintf("Step 3 failed: %v", err), "failed")
				return nil, err
			}
			stepContext.passResult = passResult
			e.step = "send_otp"
		case "send_otp":
			// Step 4a – trigger OTP email
			sendProgress(publish, taskID, 55, "Step 4/5: Triggering OTP email…", "running")
			// The OTP email should be triggered by the previous step; if not, we can try to resend or just wait for it.
			err := sess.sendOtp(ctx, stepContext.prepareResult)
			if err != nil {
				sendProgress(publish, taskID, 100, fmt.Sprintf("Failed to trigger OTP email: %v", err), "running")
				return nil, err
			}
			// For simplicity, we proceed to wait for the OTP.
			e.step = "wait_for_otp"
		case "resend_otp":
			err := sess.resendOtp(ctx)
			if err != nil {
				sendProgress(publish, taskID, 100, fmt.Sprintf("Failed to resend OTP email: %v", err), "running")
				return nil, err
			}
			stepContext.resendCount++
			sendProgress(publish, taskID, 55+stepContext.resendCount*5, fmt.Sprintf("Resent OTP email (%d)…", stepContext.resendCount), "running")
			e.step = "wait_for_otp"
		case "wait_for_otp":
			// Step 4 – wait for OTP and verify
			sendProgress(publish, taskID, 58, "Waiting for email verification code…", "running")
			otp, err := mp.WaitForCode(ctx, mailAccount, "openai", 30)
			if err != nil {
				if stepContext.resendCount >= 3 {
					return nil, fmt.Errorf("failed to get OTP after %d attempts: %w", stepContext.resendCount, err)
				}
				sendProgress(publish, taskID, 55+stepContext.resendCount*5, fmt.Sprintf("Failed to get OTP, resend. reason: %v", err), "failed")
				e.step = "resend_otp"
				continue
			}
			sendProgress(publish, taskID, 70, fmt.Sprintf("Got OTP: %s – verifying…", otp), "running")
			if err := sess.verifyEmailOTP(ctx, email, otp, stepContext.prepareResult); err != nil {
				sendProgress(publish, taskID, 100, fmt.Sprintf("Step 4 failed: %v", err), "failed")
				return nil, err
			}
			e.step = "create_account"
		case "create_account":
			err := sess.createAccount(ctx, stepContext.prepareResult)
			if err != nil {
				sendProgress(publish, taskID, 100, fmt.Sprintf("Account creation failed: %v", err), "failed")
				return nil, err
			}
			sendProgress(publish, taskID, 80, "Account created successfully!", "running")
			e.step = "get_callback_url"
		case "get_callback_url":
			// Step 5 – obtain session/access token
			sendProgress(publish, taskID, 85, "Step 5/5: Obtaining session token…", "running")
			callbackUrl, tokenErr := sess.getCallbackUrl(ctx)
			// token errors are non-fatal – the account is still usable with email+password.
			if tokenErr != nil {
				sendProgress(publish, taskID, 100, fmt.Sprintf("Note: %v", tokenErr), "running")
				return nil, tokenErr
			}

			stepContext.callbackUrl = callbackUrl
			e.step = "obtain_token"
		case "obtain_token":
			tokenInfo, err := sess.getTokenInfo(ctx, stepContext.callbackUrl, stepContext.prepareResult)
			if err != nil {
				sendProgress(publish, taskID, 100, fmt.Sprintf("Failed to obtain account info: %v", err), "failed")
				return nil, err
			}
			stepContext.tokenInfo = tokenInfo
			e.step = "done"
		default:
			break executor_loop
		}
	}

	var extra string
	if stepContext.tokenInfo != nil {
		if b, err := json.Marshal(stepContext.tokenInfo); err == nil {
			extra = string(b)
		} else {
			return nil, fmt.Errorf("failed to marshal token info: %w", err)
		}
	}
	acct := &model.Account{
		Email:       email,
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
