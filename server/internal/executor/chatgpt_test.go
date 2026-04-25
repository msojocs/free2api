package executor

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/msojocs/ai-auto-register/server/internal/core"
	"github.com/msojocs/ai-auto-register/server/pkg/mailprovider"
)

func TestChatGPT(t *testing.T) {
	var tsURL string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		// Cloudflare IP check
		case r.Method == http.MethodGet && r.URL.Path == "/cdn-cgi/trace":
			w.Header().Set("Content-Type", "text/plain")
			fmt.Fprint(w, "ip=1.2.3.4\nloc=US\n")

		// Auth: authorize — must set oai-did cookie
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/oauth/authorize"):
			http.SetCookie(w, &http.Cookie{Name: "oai-did", Value: "mock-did", Path: "/"})
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, "<html>ok</html>")

		// Sentinel: proof
		case r.Method == http.MethodGet && r.URL.Path == "/proof":
			w.Header().Set("Content-Type", "text/plain")
			fmt.Fprint(w, "mockproof")

		// Sentinel: req (mocks sentinel.openai.com endpoint)
		case r.Method == http.MethodPost && r.URL.Path == "/sentinel/req":
			fmt.Fprint(w, `{"token":"mockSentinelToken","persona":"default","expire_after":0,"expire_at":0,"turnstile":{"required":false,"dx":""},"proofofwork":{"required":false,"seed":"","difficulty":""}}`)

		// Sentinel: turnstile
		case r.Method == http.MethodPost && r.URL.Path == "/turnstile":
			fmt.Fprint(w, `{"enforcementToken":"mockEnfToken","turnstileToken":"mockTurnstileToken"}`)

		// Auth: check email → needs sign-up
		case r.Method == http.MethodPost && r.URL.Path == "/api/accounts/authorize/continue":
			fmt.Fprint(w, `{"page":{"type":"create_account_password"}}`)

		// Auth: set password
		case r.Method == http.MethodPost && r.URL.Path == "/api/accounts/user/register":
			fmt.Fprint(w, `{"page":{"type":"email_otp_send"}}`)

		// Auth: trigger OTP send
		case r.Method == http.MethodGet && r.URL.Path == "/api/accounts/email-otp/send":
			fmt.Fprint(w, `{}`)

		// Mail: create inbox (TempMailLol)
		case r.Method == http.MethodPost && r.URL.Path == "/v2/inbox/create":
			fmt.Fprint(w, `{"address":"mock@test.com","token":"mailtoken"}`)

		// Mail: list messages — returns one message containing keyword + code
		case r.Method == http.MethodGet && r.URL.Path == "/v2/inbox":
			fmt.Fprint(w, `{"emails":[{"_id":"msg1","subject":"OpenAI Email Verification","body":"Your openai verification code is 123456","date":1234567}]}`)

		// Auth: verify OTP
		case r.Method == http.MethodPost && r.URL.Path == "/api/accounts/email-otp/validate":
			fmt.Fprint(w, `{"page":{"type":"about_you"}}`)

		// Auth: create account
		case r.Method == http.MethodPost && r.URL.Path == "/api/accounts/create_account":
			fmt.Fprint(w, `{"page":{"type":"about_you"}}`)

		// Auth: get workspace ID (raw body contains escaped JSON with id field)
		case r.Method == http.MethodGet && r.URL.Path == "/sign-in-with-chatgpt/codex/consent":
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `},\"id\",\"03fc13b9-59cb-4c67-a0d3-ecdfa3296744\",\"name\"`)

		// Auth: select workspace → returns continue_url pointing to redirect chain
		case r.Method == http.MethodPost && r.URL.Path == "/api/accounts/workspace/select":
			fmt.Fprintf(w, `{"continue_url":"%s/redirect1"}`, tsURL)

		// Redirect chain: 3 hops → final localhost callback URL
		case r.Method == http.MethodGet && r.URL.Path == "/redirect1":
			http.Redirect(w, r, tsURL+"/redirect2", http.StatusFound)
		case r.Method == http.MethodGet && r.URL.Path == "/redirect2":
			http.Redirect(w, r, tsURL+"/redirect3", http.StatusFound)
		case r.Method == http.MethodGet && r.URL.Path == "/redirect3":
			http.Redirect(w, r, "http://localhost:1455/auth/callback?code=mockcode&state=mockstate", http.StatusFound)

		// Auth: exchange code for token
		case r.Method == http.MethodPost && r.URL.Path == "/oauth/token":
			fmt.Fprint(w, `{"access_token":"mockAccessToken","expires_in":3600,"id_token":"mockIdToken","refresh_token":"mockRefreshToken","token_type":"bearer","scope":"openid"}`)

		default:
			http.Error(w, "unexpected: "+r.Method+" "+r.URL.Path, http.StatusNotFound)
		}
	}))
	defer ts.Close()
	tsURL = ts.URL

	gpt := NewChatGPTExecutor(ts.URL) // sentinel base = ts.URL
	gpt.authBaseURL = ts.URL
	gpt.cloudflareURL = ts.URL + "/cdn-cgi/trace"
	gpt.sentinelReqURL = ts.URL + "/sentinel/req"

	cfg := map[string]interface{}{
		"mail_provider": "tempmail",
		"mail_api_url":  ts.URL + "/v2",
	}
	result, err := gpt.Execute(context.Background(), 0, cfg, func(p core.ProgressUpdate) {
		t.Logf("progress [%d%%]: %s", p.Progress, p.Message)
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result == nil || result.Account == nil {
		t.Fatal("expected non-nil result with account")
	}
	t.Logf("Result: email=%s type=%s", result.Account.Email, result.Account.Type)
}

func TestHeaderEmpty(t *testing.T) {
	req, err := http.NewRequest("GET", "https://example.com", nil)
	if err != nil {
		t.Fatalf("newRequest returned error: %v", err)
	}
	if req == nil {
		t.Fatalf("newRequest returned nil")
	}
	if len(req.Header) != 0 {
		t.Fatalf("expected empty header, got: %v", req.Header)
	}
}

func TestWorkspaceIdParse(t *testing.T) {
	str := `},\"id\",\"03fc13b9-59cb-4c67-a0d3-ecdfa3296744\",\"name\"`

	// 格式："id\",\"90ae07ea-04d5-4dd5-93bc-ee71ef33f3cd\",\"profile_picture_alt_text
	reg := regexp.MustCompile(`"id\\",\\"([0-9a-fA-F-]+)\\"`)

	matches := reg.FindStringSubmatch(str)
	if len(matches) < 2 {
		t.Fatalf("Unexpected response format; 'id' field not found. %s", str)
	}
	workspaceId := matches[1]
	t.Logf("Workspace ID: %s", workspaceId)
}

func TestOtpSend(t *testing.T) {
	proxyURL := "http://127.0.0.1:7890"
	sentinelBaseURL := "http://127.0.0.1:3001"
	authBaseURL := "https://auth.openai.com"
	cloudflareURL := "https://cloudflare.com/cdn-cgi/trace"
	sentinelReqURL := "https://sentinel.openai.com/backend-api/sentinel/req"
	sess, err := newOpenAISession(proxyURL, sentinelBaseURL, authBaseURL, cloudflareURL, sentinelReqURL)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	ctx := context.Background()

	_, err = sess.prepareSession(ctx)
	if err != nil {
		t.Fatalf("prepareSession failed: %v", err)
	}

	email := "gmlkitt3n@mediaeast.uk"

	step := "prepare_session"
	var stepContext struct {
		prepareResult    *openAIPrepareResult
		password         string
		callbackUrl      string
		tokenInfo        *openAITokenResult
		resendCount      int
		maxRetryAttempts int // resend otp email
	}
	stepContext.maxRetryAttempts = 3
	mp, err := mailprovider.NewLinshiyouxiang(map[string]string{
		"proxy_url": "http://127.0.0.1:8866",
	})
	if err != nil {
		t.Fatalf("Failed to create mail provider: %v", err)
	}
	mailAccount := &mailprovider.MailAccount{
		Email:     email,
		AccountID: email,
		Token:     "e20dbd8e8ec85177f23b9bfc686708d5489811f5027b2d0c0ef49ddf9922ff6c",
	}

executor_loop:
	for {
		switch step {
		case "prepare_session":
			// Seed session / get state
			prepareResult, err := sess.prepareSession(ctx)
			if err != nil {
				t.Fatalf("prepareSession failed: %v", err)
			}
			if prepareResult.OaiDid == "" {
				// This typically means Cloudflare blocked the request.
				t.Fatalf("could not obtain auth state – Cloudflare protection may be active; try with a residential proxy")
			}
			stepContext.prepareResult = prepareResult
			step = "check_email"
		case "check_email":
			nextPage, err := sess.isEmailNeedSignUp(ctx, email, stepContext.prepareResult)
			if err != nil {
				t.Fatalf("isEmailNeedSignUp failed: %v", err)
			}

			switch nextPage {
			case "login_password":
				// otp登录，邮箱已经被使用了，不需要注册
				step = "passwordless_otp"
			case "create_account_password":
				// 填写密码，邮箱没被使用了，需要注册
				step = "dump_session"
			default:
				t.Fatalf("Unexpected page type: %s", nextPage)
			}
		case "dump_session":
			// Dump session for debugging
			dump, err := sess.authSessionDump()
			if err != nil {
				t.Fatalf("Session dump failed: %v", err)
			}
			t.Logf("Session dump: %s", dump)
			step = "set_password"
		case "passwordless_otp":
			t.Logf("Passwordless login...")
			err := sess.passwordlessLogin(ctx)
			if err != nil {
				t.Fatalf("Passwordless login failed: %v", err)
			}
			step = "wait_for_otp"
		case "set_password":
			// Set password
			t.Logf("Set password...")
			password := randPassword()
			t.Logf("Setting password: %s …", password)
			_, err := sess.setPassword(ctx, email, password, stepContext.prepareResult)
			if err != nil {
				t.Fatalf("Failed to set password: %v", err)
			}
			step = "send_otp"
		case "send_otp":
			// Trigger OTP email
			t.Logf("Triggering OTP email…")
			// The OTP email should be triggered by the previous step; if not, we can try to resend or just wait for it.
			err := sess.sendOtp(ctx, stepContext.prepareResult)
			if err != nil {
				t.Fatalf("Failed to trigger OTP email: %v", err)
			}
			// For simplicity, we proceed to wait for the OTP.
			step = "wait_for_otp"
		case "resend_otp":
			err := sess.resendOtp(ctx)
			if err != nil {
				t.Fatalf("Failed to resend OTP email: %v", err)
			}
			t.Logf("Resent OTP email (%d/%d)…", stepContext.resendCount, stepContext.maxRetryAttempts)
			step = "wait_for_otp"
		case "wait_for_otp":
			// Wait for OTP and verify
			t.Logf("Waiting for email verification code…")
			otp, err := mp.WaitForCode(ctx, mailAccount, "openai", 30)
			if err != nil {
				if stepContext.resendCount < stepContext.maxRetryAttempts {
					t.Logf("Failed to get OTP, resend. reason: %v", err)
					stepContext.resendCount++
					step = "resend_otp"
					continue
				}
				t.Fatalf("Failed to get OTP after %d attempts: %v", stepContext.resendCount, err)

			}
			t.Logf("Got OTP: %s – verifying…", otp)
			if err := sess.verifyEmailOTP(ctx, email, otp, stepContext.prepareResult); err != nil {
				if strings.Contains(err.Error(), "wrong_email_otp_code") && stepContext.resendCount < stepContext.maxRetryAttempts {
					step = "resend_otp"
					stepContext.resendCount++
					continue
				}
				t.Fatalf("Failed to verify OTP: %v", err)
			}
			step = "create_account"
		// case "create_account":
		// 	continueType, err := sess.createAccount(ctx, stepContext.prepareResult)
		// 	if err != nil {
		// 		t.Fatalf("Account creation failed: %v", err)
		// 	}

		// 	if continueType == "add_phone" {
		// 		t.Logf("[WARN] chatgpt createAccount: phone verification required; account may be limited or blocked")
		// 	}
		// 	t.Logf("Account created successfully!")
		// 	step = "get_callback_url"
		// case "get_callback_url":
		// 	// Obtain session/access token
		// 	t.Logf("Get callback url…")
		// 	callbackUrl, tokenErr := sess.getCallbackUrl(ctx)
		// 	// token errors are non-fatal – the account is still usable with email+password.
		// 	if tokenErr != nil {
		// 		t.Logf("Note: %v", tokenErr)
		// 	}

		// 	stepContext.callbackUrl = callbackUrl
		// 	step = "obtain_token"
		// case "obtain_token":
		// 	t.Logf("Obtaining session token…")
		// 	tokenInfo, err := sess.getTokenInfo(ctx, stepContext.callbackUrl, stepContext.prepareResult)
		// 	if err != nil {
		// 		t.Fatalf("Failed to obtain account info: %v", err)
		// 	}
		// 	stepContext.tokenInfo = tokenInfo
		// 	step = "done"
		default:
			t.Logf("Loop end.")
			break executor_loop
		}
	}
}
