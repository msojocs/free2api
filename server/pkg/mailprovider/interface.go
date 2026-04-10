// Package mailprovider abstracts temporary email services used during account registration.
// Providers acquire a disposable email address and poll for incoming verification codes / links.
package mailprovider

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"regexp"
	"strings"
)

// MailAccount holds the credentials for a single temporary email address.
type MailAccount struct {
	Email     string
	AccountID string
	Token     string
	Extra     map[string]string // provider-specific metadata
}

// Provider is the common interface for temp-email backends.
type Provider interface {
	// GetEmail creates a new disposable address and returns its credentials.
	GetEmail(ctx context.Context) (*MailAccount, error)

	// WaitForCode polls until an email containing the keyword arrives and a 6-digit
	// OTP is found, or until timeoutSec seconds have elapsed.
	// Pass an empty keyword to accept any message.
	WaitForCode(ctx context.Context, account *MailAccount, keyword string, timeoutSec int) (string, error)

	// WaitForLink polls until an email arrives and a verification URL is found.
	// keyword is matched against the link URL; pass "" to return the first URL.
	WaitForLink(ctx context.Context, account *MailAccount, keyword string, timeoutSec int) (string, error)
}

// New returns a Provider for the given type.
//
// Supported types and their config keys:
//
//	"mailtm" / "duckmail"
//	  api_url  – base URL (default: https://api.duckmail.sbs)
//
//	"cfworker" / "cloudflare_worker"
//	  api_url      – Cloudflare Worker backend URL
//	  admin_token  – x-custom-auth token used to create addresses
//	  domain       – email domain for generated addresses
//
//	"tempmail" / "tempmail_lol"
//	  api_url  – override base URL (default: https://api.tempmail.lol/v2)
//
//	"moemail"
//	  api_url  – base URL (default: https://sall.cc)
//
//	"freemail"
//	  api_url      – Freemail backend base URL (required)
//	  admin_token  – admin bearer token (or use username+password)
//	  username     – account username
//	  password     – account password
//
//	"laoudo"
//	  auth_token  – Authorization header value (required)
//	  email       – pre-configured email address (required)
//	  account_id  – laoudo account ID (required)
//
//	"maliapi" / "yydsmail"
//	  api_url  – base URL (default: https://maliapi.215.im/v1)
//	  api_key  – API key (required)
//	  domain   – preferred email domain (optional)
//
//	"luckmail"
//	  api_url       – base URL (default: https://mails.luckyous.com)
//	  api_key       – API key (required)
//	  project_code  – project identifier (required)
//	  email_type    – optional email type filter
func New(providerType string, config map[string]string) (Provider, error) {
	switch strings.ToLower(strings.TrimSpace(providerType)) {
	case "mailtm", "mail.tm", "duckmail", "duck":
		return NewMailTm(config), nil
	case "cfworker", "cloudflare_worker", "cf_worker", "cloudflare":
		return NewCFWorker(config), nil
	case "tempmail", "tempmail_lol", "tempmail.lol":
		return NewTempMailLol(config), nil
	case "moemail", "moeMail":
		return NewMoeMail(config), nil
	case "freemail":
		return NewFreemail(config), nil
	case "laoudo":
		return NewLaoudo(config), nil
	case "maliapi", "yydsmail", "mali":
		return NewMaliAPI(config), nil
	case "luckmail":
		return NewLuckMail(config), nil
	case "linshiyouxiang", "lsyx":
		return NewLinshiyouxiang(config), nil
	case "tempmailorg", "tempmail_org":
		return NewTempMailOrg(config), nil
	case "secemail", "1secemail", "1secmail", "1sec":
		return NewSeceMail(config), nil
	default:
		return nil, fmt.Errorf("mailprovider: unknown provider type %q (supported: mailtm, duckmail, cfworker, tempmail, moemail, freemail, laoudo, maliapi, luckmail, linshiyouxiang, tempmailorg)", providerType)
	}
}

// ---------------------------------------------------------------------------
// Shared helpers
// ---------------------------------------------------------------------------

var (
	codeRe = regexp.MustCompile(`\b(\d{6,8})\b`)
	linkRe = regexp.MustCompile(`https?://[^\s<>"'\]]+`)
)

// extractCode returns the first 6-to-8 digit standalone number found in text.
func extractCode(text string) string {
	m := codeRe.FindStringSubmatch(text)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

// extractLink returns the first URL in text that contains keyword (case-insensitive).
// If keyword is empty it returns the first URL found.
func extractLink(text, keyword string) string {
	links := linkRe.FindAllString(text, -1)
	for _, l := range links {
		if keyword == "" || strings.Contains(strings.ToLower(l), strings.ToLower(keyword)) {
			return strings.TrimRight(l, ").,;")
		}
	}
	return ""
}

// randIntN returns a cryptographically random non-negative integer in [0, n).
func randIntN(n int) int {
	v, _ := rand.Int(rand.Reader, big.NewInt(int64(n)))
	return int(v.Int64())
}

func randAlphanumUpperStr(n int) string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		idx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		b[i] = chars[idx.Int64()]
	}
	return string(b)
}

// randStr generates a random lowercase ASCII string of length n.
func randStr(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz"
	b := make([]byte, n)
	for i := range b {
		idx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		b[i] = letters[idx.Int64()]
	}
	return string(b)
}
