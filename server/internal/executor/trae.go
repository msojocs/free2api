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

	"github.com/msojocs/ai-auto-register/server/internal/core"
	"github.com/msojocs/ai-auto-register/server/internal/model"
	"github.com/msojocs/ai-auto-register/server/pkg/crypto"
	"github.com/msojocs/ai-auto-register/server/pkg/mailprovider"
	"golang.org/x/net/publicsuffix"
)

// Trae.ai registration endpoints.
// Reference: https://github.com/lxf746/any-auto-register/blob/main/platforms/trae/core.py
const (
	traeBaseURL  = "https://ug-normal.trae.ai"
	traeAPIBase  = "https://api-sg-central.trae.ai"
	traeAID      = "677332"
	traeSDKVer   = "2.1.10-tiktok"
	traeVerifyFP = "verify_mmt7gooq_u1iacZ2Q_GkCW_4aPC_86Qf_nZN7GxQ7wzrX"
	traeUA       = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/145.0.0.0 Safari/537.36"
)

// TraeExecutor registers new Trae.ai accounts via the TikTok passport API.
type TraeExecutor struct{}

func NewTraeExecutor() *TraeExecutor {
	return &TraeExecutor{}
}

type traeSession struct {
	client *http.Client
}

func newTraeSession(proxyURL string) (*traeSession, error) {
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return nil, err
	}
	transport := &http.Transport{}
	if proxyURL != "" {
		u, err := url.Parse(proxyURL)
		if err != nil {
			return nil, fmt.Errorf("trae: invalid proxy URL: %w", err)
		}
		transport.Proxy = http.ProxyURL(u)
	}
	return &traeSession{client: &http.Client{
		Jar:       jar,
		Transport: transport,
		Timeout:   30 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}}, nil
}

func (s *traeSession) baseParams() url.Values {
	return url.Values{
		"aid":                {"677332"},
		"account_sdk_source": {"web"},
		"sdk_version":        {traeSDKVer},
		"language":           {"en"},
		"verifyFp":           {traeVerifyFP},
	}
}

func (s *traeSession) postForm(ctx context.Context, rawURL string, params, formData url.Values) ([]byte, error) {
	u, _ := url.Parse(rawURL)
	u.RawQuery = params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", traeUA)
	req.Header.Set("Accept", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func (s *traeSession) postJSON(ctx context.Context, rawURL string, params url.Values, payload interface{}) ([]byte, error) {
	b, _ := json.Marshal(payload)
	u, _ := url.Parse(rawURL)
	if params != nil {
		u.RawQuery = params.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", traeUA)
	req.Header.Set("Accept", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// step1Region initialises a regional session.
func (s *traeSession) step1Region(ctx context.Context) {
	_, _ = s.postForm(ctx, traeBaseURL+"/passport/web/region/",
		s.baseParams(), url.Values{"type": {"2"}})
}

// step2SendCode sends an OTP email to the given address.
func (s *traeSession) step2SendCode(ctx context.Context, email string) error {
	data, err := s.postForm(ctx, traeBaseURL+"/passport/web/email/send_code/",
		s.baseParams(),
		url.Values{
			"type":             {"1"},
			"email":            {email},
			"password":         {""},
			"email_logic_type": {"2"},
		})
	if err != nil {
		return fmt.Errorf("trae step2: %w", err)
	}
	var resp struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(data, &resp); err != nil || resp.Message != "success" {
		return fmt.Errorf("trae step2: send_code failed: %s", string(data))
	}
	return nil
}

// step3Register submits the OTP to complete registration and returns the user ID.
func (s *traeSession) step3Register(ctx context.Context, email, password, otp string) (string, error) {
	data, err := s.postForm(ctx, traeBaseURL+"/passport/web/email/register_verify_login/",
		s.baseParams(),
		url.Values{
			"type":             {"1"},
			"email":            {email},
			"password":         {password},
			"code":             {otp},
			"email_logic_type": {"2"},
		})
	if err != nil {
		return "", fmt.Errorf("trae step3: %w", err)
	}
	var resp struct {
		Message string `json:"message"`
		Data    struct {
			UserIDStr string `json:"user_id_str"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", fmt.Errorf("trae step3: parse: %w (%s)", err, string(data))
	}
	if resp.Message != "success" && resp.Data.UserIDStr == "" {
		return "", fmt.Errorf("trae step3: register failed: %s", string(data))
	}
	return resp.Data.UserIDStr, nil
}

// step4TraeLogin logs into the Trae IDE backend.
func (s *traeSession) step4TraeLogin(ctx context.Context) {
	_, _ = s.postJSON(ctx, traeBaseURL+"/cloudide/api/v3/trae/Login",
		url.Values{"type": {"email"}},
		map[string]string{
			"UtmSource": "", "UtmMedium": "", "UtmCampaign": "",
			"UtmTerm": "", "UtmContent": "", "BDVID": "",
			"LoginChannel": "ide_platform",
		})
}

// step5GetToken retrieves the user JWT token.
func (s *traeSession) step5GetToken(ctx context.Context) string {
	data, err := s.postJSON(ctx, traeAPIBase+"/cloudide/api/v3/common/GetUserToken", nil, map[string]string{})
	if err != nil {
		return ""
	}
	var resp struct {
		Result struct {
			Token string `json:"Token"`
		} `json:"Result"`
	}
	_ = json.Unmarshal(data, &resp)
	return resp.Result.Token
}

// step6CheckLogin checks login status and returns region info.
func (s *traeSession) step6CheckLogin(ctx context.Context) map[string]interface{} {
	data, err := s.postJSON(ctx, traeBaseURL+"/cloudide/api/v3/trae/CheckLogin",
		nil,
		map[string]bool{"GetAIPayHost": true, "GetNickNameEditStatus": true})
	if err != nil {
		return nil
	}
	var resp struct {
		Result map[string]interface{} `json:"Result"`
	}
	_ = json.Unmarshal(data, &resp)
	return resp.Result
}

// Execute runs the full Trae.ai registration flow.
//
// Config keys: proxy, mail_provider, mail_api_url, mail_admin_token, mail_domain
func (e *TraeExecutor) Execute(ctx context.Context, taskID uint, config map[string]interface{}, publish func(core.ProgressUpdate)) (*ExecutionResult, error) {
	sendProgress(publish, taskID, 0, "Starting Trae.ai account registration", "running")

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

	sendProgress(publish, taskID, 10, "Getting temporary email…", "running")
	mailAccount, err := mp.GetEmail(ctx)
	if err != nil {
		sendProgress(publish, taskID, 100, fmt.Sprintf("Get email failed: %v", err), "failed")
		return nil, err
	}
	email := mailAccount.Email
	sendProgress(publish, taskID, 18, fmt.Sprintf("Got email: %s", email), "running")

	sess, err := newTraeSession(proxyURL)
	if err != nil {
		sendProgress(publish, taskID, 100, fmt.Sprintf("Session init error: %v", err), "failed")
		return nil, err
	}

	// Step 1 – region
	sendProgress(publish, taskID, 22, "Step 1/6: Initialising region…", "running")
	sess.step1Region(ctx)

	// Step 2 – send OTP
	sendProgress(publish, taskID, 30, "Step 2/6: Sending OTP email…", "running")
	if err := sess.step2SendCode(ctx, email); err != nil {
		sendProgress(publish, taskID, 100, fmt.Sprintf("Step 2 failed: %v", err), "failed")
		return nil, err
	}

	// Wait for OTP
	sendProgress(publish, taskID, 40, "Waiting for OTP…", "running")
	otp, err := mp.WaitForCode(ctx, mailAccount, "", 120)
	if err != nil {
		sendProgress(publish, taskID, 100, fmt.Sprintf("OTP wait failed: %v", err), "failed")
		return nil, err
	}
	sendProgress(publish, taskID, 52, fmt.Sprintf("Got OTP: %s", otp), "running")

	// Step 3 – register
	password := randPassword()
	sendProgress(publish, taskID, 58, "Step 3/6: Submitting registration…", "running")
	userID, err := sess.step3Register(ctx, email, password, otp)
	if err != nil {
		sendProgress(publish, taskID, 100, fmt.Sprintf("Step 3 failed: %v", err), "failed")
		return nil, err
	}

	// Steps 4-6 – login + token
	sendProgress(publish, taskID, 68, "Step 4/6: Trae IDE login…", "running")
	sess.step4TraeLogin(ctx)

	sendProgress(publish, taskID, 75, "Step 5/6: Getting token…", "running")
	token := sess.step5GetToken(ctx)

	sendProgress(publish, taskID, 85, "Step 6/6: Checking login status…", "running")
	loginResult := sess.step6CheckLogin(ctx)

	region := ""
	if loginResult != nil {
		if r, ok := loginResult["Region"].(string); ok {
			region = r
		}
	}

	// Persist
	encPass, err := crypto.Encrypt(password)
	if err != nil {
		sendProgress(publish, taskID, 100, fmt.Sprintf("Encrypt error: %v", err), "failed")
		return nil, err
	}
	extraMap := map[string]string{"user_id": userID, "token": token, "region": region}
	extraJSON, _ := json.Marshal(extraMap)
	acct := &model.Account{
		Email:       email,
		Password:    encPass,
		Type:        "trae",
		Status:      "active",
		TaskBatchID: taskID,
		Extra:       string(extraJSON),
	}

	return &ExecutionResult{
		Account:        acct,
		SuccessMessage: fmt.Sprintf("✓ Trae.ai account registered: %s (user_id=%s)", email, userID),
	}, nil
}
