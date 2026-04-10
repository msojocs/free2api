package mailprovider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSeceMailReceive(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/":
			w.Header().Set("Set-Cookie", "XSRF-TOKEN=eyJpdiI6IjU3RHZyVys5RUFtRTBMaGE2b3RrRVE9PSIsInZhbHVlIjoiOHc0VU5icFdUSmhsMFYwQmE3S1lzbXphR0JHUGNSSEVSQXpMSmxGQzMxRWxLSDVHbVVoUHdqVHZ5dU1sQU4vSy9LdGV3ajQ4d0IremJhSUJ5TklUekErWG5GS3kyWjJTak5lS3E5S3FTVFErOVRrN0hwL21sb0NpbnlpeFhneDMiLCJtYWMiOiI5MWM2YjY5MzUwOWY5ZGIyNzk0MDc3MDFhYmNjZDkyNjY0ZGUzYzIwZmFjNTFkYTFiNDBhZWJlMTFiNDg2Mzk5IiwidGFnIjoiIn0=; expires=Tue, 09 Jun 2099 14:50:29 GMT; Max-Age=5184000; path=/; samesite=lax")
			_, _ = w.Write([]byte(`<meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta name="csrf-token" content="NAHdT2yagBXiVHeJyxZq5jsFghTdiBGXGL77mrAu">
    <`))
		case r.Method == http.MethodPost && r.URL.Path == "/get_messages":
			xsrfToken := r.Header.Get("x-xsrf-token")
			if xsrfToken != "eyJpdiI6IjU3RHZyVys5RUFtRTBMaGE2b3RrRVE9PSIsInZhbHVlIjoiOHc0VU5icFdUSmhsMFYwQmE3S1lzbXphR0JHUGNSSEVSQXpMSmxGQzMxRWxLSDVHbVVoUHdqVHZ5dU1sQU4vSy9LdGV3ajQ4d0IremJhSUJ5TklUekErWG5GS3kyWjJTak5lS3E5S3FTVFErOVRrN0hwL21sb0NpbnlpeFhneDMiLCJtYWMiOiI5MWM2YjY5MzUwOWY5ZGIyNzk0MDc3MDFhYmNjZDkyNjY0ZGUzYzIwZmFjNTFkYTFiNDBhZWJlMTFiNDg2Mzk5IiwidGFnIjoiIn0=" {
				http.Error(w, "invalid XSRF token", http.StatusUnauthorized)
				return
			}
			var body struct {
				Token string `json:"_token"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body.Token != "NAHdT2yagBXiVHeJyxZq5jsFghTdiBGXGL77mrAu" {
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}
			_, _ = w.Write([]byte(`{
    "status": true,
    "mailbox": "ngczflp19@emailgenerator.xyz",
    "email_token": "Mi9QRHEvb3JWRi9SUmp4eGpQQWZZUy9VcVJxU0pXbDU0Q0pEZVVQQUROTT0",
    "messages": [
        {
            "is_seen": false,
            "subject": "hi",
            "from": "Anonymousemail",
            "from_email": "noreply@anonymousemail.se",
            "to": "ngczflp19@emailgenerator.xyz",
            "receivedAt": "2026-04-10 20:44:37",
            "id": "aGkvDdo1VY58JqmglWPBxJ443zRbOAZNyn3470x6eKL9w1",
            "html": true,
            "content": "<p><span style=\"color:#c0392b\">Powered by <strong>Anonymousemail<\/strong><\/span><\/p><p>code:123456<\/p>",
            "attachments": []
        }
    ],
    "histories": [
        {
            "email": "ngczflp19@emailgenerator.xyz",
            "current": true,
            "time": "2026-04-10T13:57:03.000000Z"
        }
    ]
}`))
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer ts.Close()

	provider := NewSeceMail(map[string]string{
		"api_url": ts.URL,
	})

	ctx := context.Background()
	email, err := provider.GetEmail(ctx)
	if err != nil {
		t.Fatalf("GetEmail: %v", err)
	}
	t.Logf("Created email: %s", email.Email)

	code, err := provider.WaitForCode(ctx, email, "code", 10)
	if err != nil {
		t.Fatalf("WaitForCode: %v", err)
	}
	t.Logf("Received code: %s", code)
	if code != "123456" {
		t.Fatalf("Expected code 123456, got %s", code)
	}
}
