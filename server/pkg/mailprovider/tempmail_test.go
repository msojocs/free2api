package mailprovider

import (
	"context"
	"testing"
)

func TestTempMailReceive(t *testing.T) {
	provider := NewTempMailLol(nil)
	ctx := context.Background()
	email, err := provider.GetEmail(ctx)
	if err != nil {
		t.Fatalf("GetEmail: %v", err)
	}
	t.Logf("Created email: %s", email)
	t.Logf("去 https://anonymousemail.me/ 发邮件给：%s", email.Email)
	t.Logf("内容：code: 123456")

	code, err := provider.WaitForCode(ctx, email, "code", 600)
	if err != nil {
		t.Fatalf("WaitForCode: %v", err)
	}
	t.Logf("Received messages: %v", code)
	if code != "123456" {
		t.Fatalf("Expected code 123456, got %s", code)
	}
}
