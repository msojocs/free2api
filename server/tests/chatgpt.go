package main

import (
	"context"
	"log"
	"time"

	"github.com/msojocs/free2api/server/internal/core"
	"github.com/msojocs/free2api/server/internal/executor"
)

func main() {

	for i := 0; i < 1000; i++ {
		gpt := executor.NewChatGPTExecutor()
		ctx := context.Background()
		cfg := map[string]interface{}{
			"proxy": "http://127.0.0.1:7890",
		}
		result, err := gpt.Execute(ctx, 0, cfg, func(p core.ProgressUpdate) {
			log.Printf("%v", p)
		})
		if err != nil {
			log.Printf("Execute failed: %v", err)
		}
		log.Printf("Result: %v", result)
		time.Sleep(time.Second * 20)
	}
}
