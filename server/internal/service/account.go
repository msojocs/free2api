package service

import (
	"encoding/csv"
	"fmt"
	"strings"

	"github.com/msojocs/free2api/server/internal/model"
	"github.com/msojocs/free2api/server/internal/repository"
	"github.com/msojocs/free2api/server/pkg/crypto"
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

