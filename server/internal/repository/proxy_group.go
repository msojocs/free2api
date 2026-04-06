package repository

import (
	"errors"

	"github.com/msojocs/free2api/server/internal/model"
	"gorm.io/gorm"
)

type proxyGroupRepository struct {
	db *gorm.DB
}

func NewProxyGroupRepository(db *gorm.DB) ProxyGroupRepository {
	return &proxyGroupRepository{db: db}
}

func (r *proxyGroupRepository) Create(group *model.ProxyGroup) error {
	return r.db.Create(group).Error
}

func (r *proxyGroupRepository) FindByID(id uint) (*model.ProxyGroup, error) {
	var group model.ProxyGroup
	err := r.db.First(&group, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &group, err
}

func (r *proxyGroupRepository) FindByName(name string) (*model.ProxyGroup, error) {
	var group model.ProxyGroup
	err := r.db.Where("name = ?", name).First(&group).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &group, err
}

func (r *proxyGroupRepository) List() ([]model.ProxyGroup, error) {
	var groups []model.ProxyGroup
	err := r.db.Order("name asc").Find(&groups).Error
	return groups, err
}

func (r *proxyGroupRepository) Update(group *model.ProxyGroup) error {
	return r.db.Save(group).Error
}

func (r *proxyGroupRepository) Delete(id uint) error {
	return r.db.Delete(&model.ProxyGroup{}, id).Error
}
