// Package captcha abstracts CAPTCHA solving services used during account registration.
package captcha

import (
	"context"
	"fmt"
	"strings"
)

// Solver abstracts CAPTCHA solving services.
type Solver interface {
	// SolveTurnstile submits a Cloudflare Turnstile challenge and returns the solution token.
	SolveTurnstile(ctx context.Context, pageURL, siteKey string) (string, error)
}

// New returns a Solver for the given provider name and API key.
//
// Supported providers:
//
//	"yescaptcha"         – https://yescaptcha.com
//	"2captcha" / "twocaptcha" – https://2captcha.com
func New(provider, apiKey string) (Solver, error) {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "yescaptcha":
		return NewYesCaptcha(apiKey), nil
	case "2captcha", "twocaptcha":
		return NewTwoCaptcha(apiKey), nil
	default:
		return nil, fmt.Errorf("captcha: unknown provider %q", provider)
	}
}
