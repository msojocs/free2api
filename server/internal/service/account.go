package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/msojocs/ai-auto-register/server/internal/model"
	"github.com/msojocs/ai-auto-register/server/internal/repository"
	"github.com/msojocs/ai-auto-register/server/internal/resource"
	"github.com/msojocs/ai-auto-register/server/pkg/crypto"
	"github.com/msojocs/ai-auto-register/server/pkg/openai"
)

type AccountService struct {
	repo        repository.AccountRepository
	settingRepo repository.SettingRepository
	proxyRes    *resource.ProxyResource
}

func NewAccountService(
	repo repository.AccountRepository,
	settingRepo repository.SettingRepository,
	proxyRes *resource.ProxyResource,
) *AccountService {
	return &AccountService{
		repo:        repo,
		settingRepo: settingRepo,
		proxyRes:    proxyRes,
	}
}

func (s *AccountService) List(page, limit int, accountType string) ([]model.Account, int64, error) {
	offset := (page - 1) * limit
	return s.repo.List(offset, limit, accountType)
}

func (s *AccountService) Delete(id uint) error {
	return s.repo.Delete(id)
}

type exportRecord struct {
	Email    string          `json:"email"`
	Password string          `json:"password"`
	Type     string          `json:"type"`
	Status   string          `json:"status"`
	Extra    json.RawMessage `json:"extra,omitempty"`
}

func (s *AccountService) Export(accountType string) ([]byte, error) {
	accounts, err := s.repo.ListAll(accountType)
	if err != nil {
		return nil, err
	}

	records := make([]exportRecord, 0, len(accounts))
	for _, a := range accounts {
		password, decErr := crypto.Decrypt(a.Password)
		if decErr != nil {
			password = ""
		}
		var extra json.RawMessage
		if a.Extra != "" {
			extra = json.RawMessage(a.Extra)
		}
		records = append(records, exportRecord{
			Email:    a.Email,
			Password: password,
			Type:     a.Type,
			Status:   a.Status,
			Extra:    extra,
		})
	}
	return json.Marshal(records)
}

// ImportAccountRecord is the shape of each item in an import JSON file.
type ImportAccountRecord struct {
	Email    string          `json:"email"`
	Password string          `json:"password"`
	Type     string          `json:"type"`
	Status   string          `json:"status"`
	Extra    json.RawMessage `json:"extra"`
}

// ImportResult holds the outcome counts of an import operation.
type ImportResult struct {
	Imported int `json:"imported"`
	Skipped  int `json:"skipped"`
	Failed   int `json:"failed"`
}

// Import creates accounts from a slice of records.  Passwords are encrypted
// before storage.  Records missing email or type are silently skipped.
// Existing emails (same email) are skipped to avoid duplicates.
func (s *AccountService) Import(records []ImportAccountRecord) (*ImportResult, error) {
	result := &ImportResult{}
	for _, rec := range records {
		if rec.Email == "" || rec.Type == "" {
			result.Skipped++
			continue
		}
		existing, err := s.repo.FindByEmail(rec.Email)
		if err != nil {
			result.Failed++
			continue
		}
		if existing != nil {
			result.Skipped++
			continue
		}
		password, err := crypto.Encrypt(rec.Password)
		if err != nil {
			result.Failed++
			continue
		}
		extra := ""
		if len(rec.Extra) > 0 {
			extra = string(rec.Extra)
		}
		status := rec.Status
		if status == "" {
			status = "active"
		}
		account := &model.Account{
			Email:    rec.Email,
			Password: password,
			Type:     rec.Type,
			Status:   status,
			Extra:    extra,
		}
		if err := s.repo.Create(account); err != nil {
			result.Failed++
			continue
		}
		result.Imported++
	}
	return result, nil
}

type AccountCheckResult struct {
	Supported bool   `json:"supported"`
	Valid     bool   `json:"valid"`
	Message   string `json:"message"`
	Status    string `json:"status,omitempty"`
	Usage     any    `json:"usage,omitempty"`
}

type ChatGPTRefreshTokenResult struct {
	AccountID            string `json:"account_id"`
	AccessToken          string `json:"access_token"`
	AccessTokenExpiresAt string `json:"access_token_expires_at,omitempty"`
	RefreshToken         string `json:"refresh_token"`
}

type ChatGPTAccountDetailResult struct {
	AccountID        string                     `json:"account_id"`
	DefaultAccountID string                     `json:"default_account_id,omitempty"`
	Email            string                     `json:"email,omitempty"`
	PlanType         string                     `json:"plan_type,omitempty"`
	Accounts         []openai.CodexAccount      `json:"accounts,omitempty"`
	Usage            *openai.CodexUsageResponse `json:"usage,omitempty"`
	Extra            map[string]interface{}     `json:"extra,omitempty"`
}

func (s *AccountService) Check(ctx context.Context, id uint) (*AccountCheckResult, error) {
	account, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, errors.New("account not found")
	}

	switch account.Type {
	case "chatgpt":
		return s.checkChatGPTAccount(ctx, account)
	default:
		return &AccountCheckResult{
			Supported: false,
			Valid:     false,
			Message:   "account check is not implemented for this platform yet",
		}, nil
	}
}

func (s *AccountService) checkChatGPTAccount(_ context.Context, account *model.Account) (*AccountCheckResult, error) {
	extra, err := parseAccountExtra(account.Extra)
	if err != nil {
		return nil, err
	}

	accessToken, _ := extra["access_token"].(string)
	if strings.TrimSpace(accessToken) == "" {
		return nil, errors.New("missing access_token in account extra field")
	}

	accountID, _ := extra["account_id"].(string)
	if strings.TrimSpace(accountID) == "" {
		accountID = extractChatGPTAccountIDFromAccessToken(accessToken)
	}
	if strings.TrimSpace(accountID) == "" {
		return nil, errors.New("missing account_id in account extra field")
	}

	proxyURL, err := s.resolveAccountActionProxy()
	if err != nil {
		return nil, err
	}

	client, err := openai.NewCodexClient(openai.CodexConfig{
		AccountId:   accountID,
		AccessToken: accessToken,
		ProxyURL:    proxyURL,
	})
	if err != nil {
		return nil, err
	}

	if _, err := client.CheckAccount(); err != nil {
		account.Status = deriveFailedStatus(err)
		if updateErr := s.repo.Update(account); updateErr != nil {
			return nil, updateErr
		}
		return &AccountCheckResult{
			Supported: true,
			Valid:     false,
			Message:   fmt.Sprintf("account check failed: %v", err),
			Status:    account.Status,
		}, nil
	}

	usageResp, err := client.QueryUsage()
	if err != nil {
		account.Status = deriveFailedStatus(err)
		if updateErr := s.repo.Update(account); updateErr != nil {
			return nil, updateErr
		}
		return &AccountCheckResult{
			Supported: true,
			Valid:     false,
			Message:   fmt.Sprintf("account check passed but usage query failed: %v", err),
			Status:    account.Status,
		}, nil
	}

	usageMap, err := usageResponseToMap(usageResp)
	if err != nil {
		return nil, err
	}
	account.Status = "active"
	account.Usage = usageMap
	if err := s.repo.Update(account); err != nil {
		return nil, err
	}

	return &AccountCheckResult{
		Supported: true,
		Valid:     true,
		Message:   "access token is valid",
		Status:    account.Status,
		Usage:     usageMap,
	}, nil
}

func (s *AccountService) RefreshChatGPTToken(_ context.Context, id uint) (*ChatGPTRefreshTokenResult, error) {
	account, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, errors.New("account not found")
	}
	if account.Type != "chatgpt" {
		return nil, errors.New("account type is not chatgpt")
	}

	extra, err := parseAccountExtra(account.Extra)
	if err != nil {
		return nil, err
	}
	refreshToken, _ := extra["refresh_token"].(string)
	if strings.TrimSpace(refreshToken) == "" {
		return nil, errors.New("missing refresh_token in account extra field")
	}

	accessToken, _ := extra["access_token"].(string)
	accountID, _ := extra["account_id"].(string)
	if strings.TrimSpace(accountID) == "" {
		accountID = extractChatGPTAccountIDFromAccessToken(accessToken)
	}

	proxyURL, err := s.resolveAccountActionProxy()
	if err != nil {
		return nil, err
	}

	client, err := openai.NewCodexClient(openai.CodexConfig{
		AccountId:    accountID,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ProxyURL:     proxyURL,
	})
	if err != nil {
		return nil, err
	}

	refreshResp, err := client.RefreshToken()
	if err != nil {
		return nil, err
	}

	extra["access_token"] = refreshResp.AccessToken
	extra["refresh_token"] = refreshResp.RefreshToken

	newAccountID := extractChatGPTAccountIDFromAccessToken(refreshResp.AccessToken)
	if strings.TrimSpace(newAccountID) == "" {
		newAccountID = accountID
	}
	if strings.TrimSpace(newAccountID) != "" {
		extra["account_id"] = newAccountID
	}

	expiresAt := ""
	if refreshResp.ExpiresIn > 0 {
		expiresAt = time.Now().Add(time.Duration(refreshResp.ExpiresIn) * time.Second).UTC().Format(time.RFC3339)
		extra["access_token_expires_at"] = expiresAt
	}

	extraRaw, err := json.Marshal(extra)
	if err != nil {
		return nil, err
	}
	account.Extra = string(extraRaw)
	if err := s.repo.Update(account); err != nil {
		return nil, err
	}

	return &ChatGPTRefreshTokenResult{
		AccountID:            newAccountID,
		AccessToken:          refreshResp.AccessToken,
		AccessTokenExpiresAt: expiresAt,
		RefreshToken:         refreshResp.RefreshToken,
	}, nil
}

func (s *AccountService) GetChatGPTDetail(_ context.Context, id uint) (*ChatGPTAccountDetailResult, error) {
	account, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, errors.New("account not found")
	}
	if account.Type != "chatgpt" {
		return nil, errors.New("account type is not chatgpt")
	}

	extra, err := parseAccountExtra(account.Extra)
	if err != nil {
		return nil, err
	}
	accessToken, _ := extra["access_token"].(string)
	if strings.TrimSpace(accessToken) == "" {
		return nil, errors.New("missing access_token in account extra field")
	}

	accountID, _ := extra["account_id"].(string)
	if strings.TrimSpace(accountID) == "" {
		accountID = extractChatGPTAccountIDFromAccessToken(accessToken)
	}
	if strings.TrimSpace(accountID) == "" {
		return nil, errors.New("missing account_id in account extra field")
	}

	proxyURL, err := s.resolveAccountActionProxy()
	if err != nil {
		return nil, err
	}

	client, err := openai.NewCodexClient(openai.CodexConfig{
		AccountId:   accountID,
		AccessToken: accessToken,
		ProxyURL:    proxyURL,
	})
	if err != nil {
		return nil, err
	}

	checkResp, err := client.CheckAccount()
	if err != nil {
		return nil, err
	}
	usageResp, err := client.QueryUsage()
	if err != nil {
		return nil, err
	}

	return &ChatGPTAccountDetailResult{
		AccountID:        accountID,
		DefaultAccountID: checkResp.DefaultAccountId,
		Email:            usageResp.Email,
		PlanType:         usageResp.PlanType,
		Accounts:         checkResp.Accounts,
		Usage:            usageResp,
		Extra:            extra,
	}, nil
}

// CheckAndRefreshAll iterates all accounts, skips banned ones, refreshes
// near-expiry ChatGPT tokens, then checks availability and updates usage.
func (s *AccountService) CheckAndRefreshAll(ctx context.Context) {
	accounts, err := s.repo.ListAll("")
	if err != nil {
		log.Printf("[account-check] failed to list accounts: %v", err)
		return
	}

	for _, a := range accounts {
		if a.Status == "banned" {
			continue
		}

		if a.Type == "chatgpt" {
			if shouldRefreshToken(a.Extra) {
				log.Printf("[account-check] refreshing near-expiry token for account %d (%s)", a.ID, a.Email)
				if _, err := s.RefreshChatGPTToken(ctx, a.ID); err != nil {
					log.Printf("[account-check] token refresh failed for account %d: %v", a.ID, err)
				}
			}
		}

		log.Printf("[account-check] checking account %d (%s)", a.ID, a.Email)
		if _, err := s.Check(ctx, a.ID); err != nil {
			log.Printf("[account-check] check failed for account %d: %v", a.ID, err)
		}
	}
	log.Printf("[account-check] done, processed %d accounts", len(accounts))
}

// shouldRefreshToken returns true when the access_token_expires_at field is
// within 24 hours of now (i.e. it is about to expire).
func shouldRefreshToken(extraRaw string) bool {
	if strings.TrimSpace(extraRaw) == "" {
		return false
	}
	var extra map[string]interface{}
	if err := json.Unmarshal([]byte(extraRaw), &extra); err != nil {
		return false
	}
	expiresAtStr, _ := extra["access_token_expires_at"].(string)
	if strings.TrimSpace(expiresAtStr) == "" {
		return false
	}
	expiresAt, err := time.Parse(time.RFC3339, expiresAtStr)
	if err != nil {
		return false
	}
	return time.Until(expiresAt) <= 24*time.Hour
}

func parseAccountExtra(raw string) (map[string]interface{}, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, errors.New("account extra field is empty")
	}
	var extra map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &extra); err != nil {
		return nil, fmt.Errorf("failed to parse account extra field as JSON: %w", err)
	}
	return extra, nil
}

func extractChatGPTAccountIDFromAccessToken(accessToken string) string {
	parts := strings.Split(accessToken, ".")
	if len(parts) < 2 {
		return ""
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}
	var payloadData map[string]interface{}
	if err := json.Unmarshal(payload, &payloadData); err != nil {
		return ""
	}
	authData, ok := payloadData["https://api.openai.com/auth"].(map[string]interface{})
	if !ok {
		return ""
	}
	accountID, _ := authData["chatgpt_account_id"].(string)
	return strings.TrimSpace(accountID)
}

func usageResponseToMap(usageResp *openai.CodexUsageResponse) (model.JSONMap, error) {
	if usageResp == nil {
		return model.JSONMap{}, nil
	}
	metric := "rate_limit"
	usedPercent := usageResp.RateLimit.PrimaryWindow.UsedPercent
	limitReached := usageResp.RateLimit.LimitReached
	allowed := usageResp.RateLimit.Allowed
	limitWindowSeconds := usageResp.RateLimit.PrimaryWindow.LimitWindowSeconds
	resetAfterSeconds := usageResp.RateLimit.PrimaryWindow.ResetAfterSeconds
	resetAt := usageResp.RateLimit.PrimaryWindow.ResetAt

	return model.JSONMap{
		"metric":               metric,
		"used_percent":         usedPercent,
		"limit_reached":        limitReached,
		"allowed":              allowed,
		"limit_window_seconds": limitWindowSeconds,
		"reset_after_seconds":  resetAfterSeconds,
		"reset_at":             resetAt,
	}, nil
}

func deriveFailedStatus(err error) string {
	if err == nil {
		return "pending"
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "403") ||
		strings.Contains(msg, "forbidden") ||
		strings.Contains(msg, "banned") ||
		strings.Contains(msg, "suspended") ||
		strings.Contains(msg, "deactivated") {
		return "banned"
	} else if strings.Contains(msg, "invalidated") {
		return "expired"
	}
	return "pending"
}

func (s *AccountService) resolveAccountActionProxy() (string, error) {
	if s.settingRepo == nil || s.proxyRes == nil {
		return "", nil
	}
	setting, err := s.settingRepo.Get()
	if err != nil || setting == nil {
		return "", err
	}
	if setting.AccountActionProxyGroupID == nil || *setting.AccountActionProxyGroupID == 0 {
		return "", nil
	}
	proxy := s.proxyRes.NextByGroupID(*setting.AccountActionProxyGroupID)
	if proxy == nil {
		return "", nil
	}
	return buildAccountProxyURL(proxy), nil
}

func buildAccountProxyURL(proxy *model.Proxy) string {
	protocol := strings.TrimSpace(proxy.Protocol)
	if protocol == "" {
		protocol = "http"
	}
	u := &url.URL{
		Scheme: protocol,
		Host:   net.JoinHostPort(proxy.Host, proxy.Port),
	}
	if proxy.Username != "" || proxy.Password != "" {
		u.User = url.UserPassword(proxy.Username, proxy.Password)
	}
	return u.String()
}
