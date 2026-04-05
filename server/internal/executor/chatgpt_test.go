package executor

import (
	"context"
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
