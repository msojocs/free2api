package repository

import (
	"errors"

	"github.com/msojocs/ai-auto-register/server/internal/model"
	"gorm.io/gorm"
)

type accountRepository struct {
	db *gorm.DB
}

// NewAccountRepository returns a new AccountRepository backed by the given DB.
func NewAccountRepository(db *gorm.DB) AccountRepository {
	return &accountRepository{db: db}
}

func (r *accountRepository) Create(account *model.Account) error {
	return r.db.Create(account).Error
}

func (r *accountRepository) Update(account *model.Account) error {
	return r.db.Save(account).Error
}

func (r *accountRepository) FindByID(id uint) (*model.Account, error) {
	var account model.Account
	err := r.db.First(&account, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &account, err
}

func (r *accountRepository) List(offset, limit int, accountType string) ([]model.Account, int64, error) {
	var accounts []model.Account
	var total int64
	query := r.db.Model(&model.Account{})
	if accountType != "" {
		query = query.Where("type = ?", accountType)
	}
	query.Count(&total)
	err := query.Order("created_at desc").Offset(offset).Limit(limit).Find(&accounts).Error
	return accounts, total, err
}

func (r *accountRepository) ListAll(accountType string) ([]model.Account, error) {
	var accounts []model.Account
	query := r.db.Model(&model.Account{})
	if accountType != "" {
		query = query.Where("type = ?", accountType)
	}
	err := query.Find(&accounts).Error
	return accounts, err
}

func (r *accountRepository) FindByEmail(email string) (*model.Account, error) {
	var account model.Account
	err := r.db.Where("email = ?", email).First(&account).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &account, err
}

func (r *accountRepository) Delete(id uint) error {
	return r.db.Delete(&model.Account{}, id).Error
}

func (r *accountRepository) CountByStatus(status string) (int64, error) {
	var count int64
	err := r.db.Model(&model.Account{}).Where("status = ?", status).Count(&count).Error
	return count, err
}

func (r *accountRepository) Count() (int64, error) {
	var count int64
	err := r.db.Model(&model.Account{}).Count(&count).Error
	return count, err
}
