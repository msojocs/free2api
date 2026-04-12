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
