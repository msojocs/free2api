package executor

import (
	"context"
	"net/http"
	"regexp"
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
