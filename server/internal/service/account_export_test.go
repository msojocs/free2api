package service

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/msojocs/ai-auto-register/server/internal/model"
	"github.com/msojocs/ai-auto-register/server/pkg/crypto"
)

type exportAccountRepoStub struct {
	listAllFunc func(accountType string, ids []uint) ([]model.Account, error)
}

func (s *exportAccountRepoStub) Create(account *model.Account) error {
	return nil
}

func (s *exportAccountRepoStub) Update(account *model.Account) error {
	return nil
}

func (s *exportAccountRepoStub) FindByID(id uint) (*model.Account, error) {
	return nil, nil
}

func (s *exportAccountRepoStub) FindByEmail(email string) (*model.Account, error) {
	return nil, nil
}

func (s *exportAccountRepoStub) List(offset, limit int, accountType string) ([]model.Account, int64, error) {
	return nil, 0, nil
}

func (s *exportAccountRepoStub) ListAll(accountType string, ids []uint) ([]model.Account, error) {
	if s.listAllFunc != nil {
		return s.listAllFunc(accountType, ids)
	}
	return nil, nil
}

func (s *exportAccountRepoStub) Delete(id uint) error {
	return nil
}

func (s *exportAccountRepoStub) CountByStatus(status string) (int64, error) {
	return 0, nil
}

func (s *exportAccountRepoStub) Count() (int64, error) {
	return 0, nil
}

func TestAccountServiceExportDecryptsPasswordsAndKeepsLegacyPlaintext(t *testing.T) {
	encryptedPassword, err := crypto.Encrypt("secret-123")
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	svc := &AccountService{
		repo: &exportAccountRepoStub{
			listAllFunc: func(accountType string, ids []uint) ([]model.Account, error) {
				if accountType != "chatgpt" {
					t.Fatalf("expected accountType chatgpt, got %q", accountType)
				}
				if ids != nil {
					t.Fatalf("expected nil ids for full export, got %v", ids)
				}
				return []model.Account{
					{
						Email:    "encrypted@example.com",
						Password: encryptedPassword,
						Type:     "chatgpt",
						Status:   "active",
					},
					{
						Email:    "legacy@example.com",
						Password: "legacy-plain-password",
						Type:     "chatgpt",
						Status:   "active",
					},
				}, nil
			},
		},
	}

	data, err := svc.Export("chatgpt", nil)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	var records []exportRecord
	if err := json.Unmarshal(data, &records); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	if records[0].Password != "secret-123" {
		t.Fatalf("expected decrypted password, got %q", records[0].Password)
	}
	if records[1].Password != "legacy-plain-password" {
		t.Fatalf("expected legacy plaintext password, got %q", records[1].Password)
	}
}

func TestAccountServiceExportPassesSelectedIDsToRepository(t *testing.T) {
	var gotType string
	var gotIDs []uint

	svc := &AccountService{
		repo: &exportAccountRepoStub{
			listAllFunc: func(accountType string, ids []uint) ([]model.Account, error) {
				gotType = accountType
				gotIDs = append([]uint(nil), ids...)
				return []model.Account{
					{
						Email:    "selected@example.com",
						Password: "selected-password",
						Type:     "cursor",
						Status:   "active",
					},
				}, nil
			},
		},
	}

	data, err := svc.Export("cursor", []uint{3, 8, 8})
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	if gotType != "cursor" {
		t.Fatalf("expected accountType cursor, got %q", gotType)
	}
	if !reflect.DeepEqual(gotIDs, []uint{3, 8, 8}) {
		t.Fatalf("expected ids [3 8 8], got %v", gotIDs)
	}

	var records []exportRecord
	if err := json.Unmarshal(data, &records); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if len(records) != 1 || records[0].Email != "selected@example.com" {
		t.Fatalf("unexpected export records: %+v", records)
	}
}
