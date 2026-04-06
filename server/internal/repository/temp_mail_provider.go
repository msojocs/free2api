package repository

import (
	"github.com/msojocs/free2api/server/internal/model"
	"gorm.io/gorm"
)

type tempMailProviderRepository struct {
	db *gorm.DB
}

func NewTempMailProviderRepository(db *gorm.DB) TempMailProviderRepository {
	return &tempMailProviderRepository{db: db}
}

func (r *tempMailProviderRepository) Create(p *model.TempMailProvider) error {
	return r.db.Create(p).Error
}

func (r *tempMailProviderRepository) FindByID(id uint) (*model.TempMailProvider, error) {
	var p model.TempMailProvider
	if err := r.db.First(&p, id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *tempMailProviderRepository) List() ([]model.TempMailProvider, error) {
	var providers []model.TempMailProvider
	if err := r.db.Order("id asc").Find(&providers).Error; err != nil {
		return nil, err
	}
	return providers, nil
}

func (r *tempMailProviderRepository) ListEnabled() ([]model.TempMailProvider, error) {
	var providers []model.TempMailProvider
	if err := r.db.Where("enabled = ?", true).Order("id asc").Find(&providers).Error; err != nil {
		return nil, err
	}
	return providers, nil
}

func (r *tempMailProviderRepository) Update(p *model.TempMailProvider) error {
	return r.db.Save(p).Error
}

func (r *tempMailProviderRepository) Delete(id uint) error {
	return r.db.Delete(&model.TempMailProvider{}, id).Error
}
