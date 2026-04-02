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
		"captcha_provider": "2captcha",
	}
	gpt.Execute(ctx, 0, cfg, func(core.ProgressUpdate) {})
}
