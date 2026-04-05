package executor

import (
	"context"
	"net/http"
	"testing"

	"github.com/msojocs/free2api/server/internal/core"
)

func TestChatGPT(t *testing.T) {
	gpt := NewChatGPTExecutor()
	ctx := context.Background()
	cfg := map[string]interface{}{
		"proxy": "http://127.0.0.1:8866",
	}
	result, err := gpt.Execute(ctx, 0, cfg, func(p core.ProgressUpdate) {
		t.Logf("%v", p)
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	t.Logf("Result: %v", result)
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
