package repository

import (
	"github.com/msojocs/ai-auto-register/server/internal/model"
	"gorm.io/gorm"
)

type pushTemplateRepository struct {
	db *gorm.DB
}

func NewPushTemplateRepository(db *gorm.DB) PushTemplateRepository {
	return &pushTemplateRepository{db: db}
}

func (r *pushTemplateRepository) Create(t *model.PushTemplate) error {
	return r.db.Create(t).Error
}

func (r *pushTemplateRepository) FindByID(id uint) (*model.PushTemplate, error) {
	var t model.PushTemplate
	if err := r.db.First(&t, id).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *pushTemplateRepository) List(offset, limit int) ([]model.PushTemplate, int64, error) {
	var templates []model.PushTemplate
	var total int64
	if err := r.db.Model(&model.PushTemplate{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := r.db.Offset(offset).Limit(limit).Find(&templates).Error; err != nil {
		return nil, 0, err
	}
	return templates, total, nil
}

func (r *pushTemplateRepository) ListEnabled() ([]model.PushTemplate, error) {
	var templates []model.PushTemplate
	if err := r.db.Where("enabled = ?", true).Find(&templates).Error; err != nil {
		return nil, err
	}
	return templates, nil
}

func (r *pushTemplateRepository) ListEnabledByType(accountType string) ([]model.PushTemplate, error) {
	var templates []model.PushTemplate
	// Return templates that are enabled AND (account_type matches OR account_type is empty / "all types")
	if err := r.db.Where("enabled = ? AND (account_type = ? OR account_type = '')", true, accountType).Find(&templates).Error; err != nil {
		return nil, err
	}
	return templates, nil
}

func (r *pushTemplateRepository) Update(t *model.PushTemplate) error {
	return r.db.Save(t).Error
}

func (r *pushTemplateRepository) Delete(id uint) error {
	return r.db.Delete(&model.PushTemplate{}, id).Error
}
