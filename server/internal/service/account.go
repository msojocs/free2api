package service

import (
	"context"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/msojocs/free2api/server/internal/model"
	"github.com/msojocs/free2api/server/internal/repository"
	"github.com/msojocs/free2api/server/pkg/crypto"
	"github.com/msojocs/free2api/server/pkg/openai"
)

type AccountService struct {
	repo repository.AccountRepository
}

func NewAccountService(repo repository.AccountRepository) *AccountService {
	return &AccountService{repo: repo}
}

func (s *AccountService) List(page, limit int, accountType string) ([]model.Account, int64, error) {
	offset := (page - 1) * limit
	return s.repo.List(offset, limit, accountType)
}

func (s *AccountService) Delete(id uint) error {
	return s.repo.Delete(id)
}

func (s *AccountService) Export(accountType string) (string, error) {
	accounts, err := s.repo.ListAll(accountType)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	w := csv.NewWriter(&sb)
	_ = w.Write([]string{"id", "email", "password", "type", "status", "task_batch_id", "created_at"})
	for _, a := range accounts {
		password, err := crypto.Decrypt(a.Password)
		if err != nil {
			password = "[decryption error]"
		}
		_ = w.Write([]string{
			fmt.Sprintf("%d", a.ID),
			a.Email,
			password,
			a.Type,
			a.Status,
			fmt.Sprintf("%d", a.TaskBatchID),
			a.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	w.Flush()
	return sb.String(), w.Error()
}

type AccountCheckResult struct {
	Supported bool   `json:"supported"`
	Valid     bool   `json:"valid"`
	Message   string `json:"message"`
}

func (s *AccountService) Check(ctx context.Context, id uint) (*AccountCheckResult, error) {
	account, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
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

	client, err := openai.NewCodexClient(openai.CodexConfig{
		AccountId:   accountID,
		AccessToken: accessToken,
	})
	if err != nil {
		return nil, err
	}

	if _, err := client.CheckAccount(); err != nil {
		return &AccountCheckResult{
			Supported: true,
			Valid:     false,
			Message:   fmt.Sprintf("account check failed: %v", err),
		}, nil
	}

	return &AccountCheckResult{
		Supported: true,
		Valid:     true,
		Message:   "access token is valid",
	}, nil
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
