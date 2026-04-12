package executor

import (
	"context"
	"crypto/rand"
	"math/big"

	"github.com/msojocs/ai-auto-register/server/internal/core"
	"github.com/msojocs/ai-auto-register/server/internal/model"
)

type Executor interface {
	Execute(ctx context.Context, taskID uint, config map[string]interface{}, publish func(core.ProgressUpdate)) (*ExecutionResult, error)
}

type ExecutionResult struct {
	Account        *model.Account
	SuccessMessage string
}

func sendProgress(publish func(core.ProgressUpdate), taskID uint, progress int, message, status string) {
	if publish == nil {
		return
	}
	publish(core.ProgressUpdate{
		TaskID:   taskID,
		Progress: progress,
		Message:  message,
		Status:   status,
	})
}

// randPassword generates a cryptographically random 16-character password that
// satisfies basic complexity: at least one lowercase, uppercase, digit and symbol.
func randPassword() string {
	const lower = "abcdefghijklmnopqrstuvwxyz"
	const upper = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	const digits = "0123456789"
	const special = "."
	const all = lower + upper + digits + special

	pick := func(charset string) byte {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		return charset[n.Int64()]
	}
	buf := make([]byte, 16)
	buf[0] = pick(lower)
	buf[1] = pick(upper)
	buf[2] = pick(digits)
	buf[3] = pick(special)
	for i := 4; i < 16; i++ {
		buf[i] = pick(all)
	}
	// Shuffle to avoid predictable prefix ordering.
	for i := len(buf) - 1; i > 0; i-- {
		j, _ := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		buf[i], buf[j.Int64()] = buf[j.Int64()], buf[i]
	}
	return string(buf)
}

// randAlphanumStr generates a cryptographically random alphanumeric string of length n.
func randAlphanumStr(n int) string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		idx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		b[i] = chars[idx.Int64()]
	}
	return string(b)
}

// cfgStr extracts a string value from a config map with a fallback default.
func cfgStr(config map[string]interface{}, key, def string) string {
	if v, ok := config[key]; ok {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	return def
}

// cfgBool extracts a bool from a config map. Returns def when the key is absent
// or holds a non-bool value. JSON-decoded configs carry booleans as bool.
func cfgBool(config map[string]interface{}, key string, def bool) bool {
	if v, ok := config[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return def
}

// safeRandInt returns a cryptographically random integer in [0, n).
func safeRandInt(n int) int64 {
	v, _ := rand.Int(rand.Reader, big.NewInt(int64(n)))
	return v.Int64()
}
