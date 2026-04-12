package mailprovider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTMailorReceive(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var data struct {
			Action string `json:"action"`
		}
		_ = json.NewDecoder(r.Body).Decode(&data)
		switch {
		case data.Action == "newemail":
			_, _ = w.Write([]byte(`{
    "msg": "ok",
    "email": "ngzhkh@tiksofi.uk",
    "create": 1775961111,
    "sort": 1775961111,
    "permission_desc": "note_email_nosave",
    "accesstoken": "eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJlIjoicHprS1pUOGxNM3lpTEhIMXBTRVZxeFcySTN5aUZ5QXdvMU9KTXlNNkZKcVpGYXl6Rm1XR3JSMUlJM3lqWjFjMkRhTUtuUjBtTDJXaFphSUJwSUU1TUtObEJLY2hFd0hrb3prSk15TXVwS3lacndJMW8wY1dwMDFIQkpxWkZheWJJYXF2cWFTSFpLSWhGemdjcFVMMXEyOGxaVU1NSFNxNEdIZ0FMMGpsRlVNUHF5cG1HSGNKcXl5REkzeWlGeUF3bzFEMXJLUmtCSkFBSFNMMkp4cTRaSWNVcVFPUEhLTm1KSU9LWktObEZKa1laYXk0SWFxdm5hQVluMjFQRTNEMXBUYjlDRD09In0.AFfJpGaFtad2Y9qjMjiNBJfYht9pZg2kcZuvGPA3dzY",
    "client-block": 0
}`))
		case data.Action == "listinbox":
			_, _ = w.Write([]byte(`{
    "msg": "ok",
    "email": "ngzhkh@tiksofi.uk",
    "code": "82f1c3a6-9b16-4f0c-a037-b8e2ef57ce8b",
    "data": {
        "20299837-1eda-4e65-9961-c9e65fa1f63d": {
            "id": "20299837-1eda-4e65-9961-c9e65fa1f63d",
            "uuid": "20299837-1eda-4e65-9961-c9e65fa1f63d",
            "email_id": "b3pxYTZuVHRnYkRtVUVjdW4zQXhpTXp1eGhxZ0pnOHZad054bEJHZng0WmJtcGdzWkpJdnhMRmgwME14R0wxMllHeGE1QXd0UmdMbW15eXVBd0l4ekxHdVN6QWd3QXh2c1F1eHpCR2Z4NVpibVY1c0FRWnZrTEpoU3lBeHdFeDJaR0Fhdk1KdFN5TW13QXp1WkdMeG1aMnVSMVlnejVhdnJ6dXhlblJmTzBuYkpnbXNvMk12Y1lhaEllc3hRUjMyQW1IYTVBd3RSM1ptd1c4dVpOPXg9",
            "subject": "hi",
            "sender_name": "Anonymousemail",
            "sender_email": "noreply@anonymousemail.se",
            "avatar": "A",
            "read": 0,
            "push": 0,
            "receive_time": 1775961722,
            "rtime": 1775961722,
            "sort": 1775961722,
            "exp_count": 86400
        }
    }
}`))
		case data.Action == "read":
			_, _ = w.Write([]byte(`{
    "msg": "ok",
    "data": {
        "id": "20299837-1eda-4e65-9961-c9e65fa1f63d",
        "email_id": "b3pxYTZuVHRnYkRtVUVjdW4zQXhpTXp1eGhxZ0pnOHZad054bEJHZng0WmJtcGdzWkpJdnhMRmgwME14R0wxMllHeGE1QXd0UmdMbW15eXVBd0l4ekxHdVN6QWd3QXh2c1F1eHpCR2Z4NVpibVY1c0FRWnZrTEpoU3lBeHdFeDJaR0Fhdk1KdFN5TW13QXp1WkdMeG1aMnVSMVlnejVhdnJ6dXhlblJmTzBuYkpnbXNvMk12Y1lhaEllc3hRUjMyQW1IYTVBd3RSM1ptd1Y9",
        "subject": "hi",
        "sender_name": "Anonymousemail",
        "sender_email": "noreply@anonymousemail.se",
        "avatar": "",
        "read": 1,
        "push": 0,
        "receive_time": 1775961722,
        "rtime": 1775961722,
        "sort": 1775961722,
        "exp_count": 86310,
        "md5": "e876d0eb115288405e98c13560f42b2c",
        "body": "<p><span style=\"color:#c0392b\">Powered by <strong>Anonymousemail<\/strong><\/span><\/p><p>code 123456<\/p>",
        "url_body": null
    }
}`))
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer ts.Close()

	provider, _ := NewTMailor(map[string]string{
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

// func TestTMailorReceiveOnline(t *testing.T) {
// 	provider := NewTMailor(map[string]string{})

// 	ctx := context.Background()
// 	email, err := provider.GetEmail(ctx)
// 	if err != nil {
// 		t.Fatalf("GetEmail: %v", err)
// 	}
// 	t.Logf("Created email: %s", email.Email)

// 	code, err := provider.WaitForCode(ctx, email, "code", 60)
// 	if err != nil {
// 		t.Fatalf("WaitForCode: %v", err)
// 	}
// 	t.Logf("Received code: %s", code)
// 	if code != "123456" {
// 		t.Fatalf("Expected code 123456, got %s", code)
// 	}
// }
