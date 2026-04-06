package repository

import (
	"errors"

	"github.com/msojocs/free2api/server/internal/model"
	"gorm.io/gorm"
)

type proxyRepository struct {
	db *gorm.DB
}

// NewProxyRepository returns a new ProxyRepository backed by the given DB.
func NewProxyRepository(db *gorm.DB) ProxyRepository {
	return &proxyRepository{db: db}
}

func (r *proxyRepository) Create(proxy *model.Proxy) error {
	return r.db.Create(proxy).Error
}

func (r *proxyRepository) Update(proxy *model.Proxy) error {
	return r.db.Save(proxy).Error
}

func (r *proxyRepository) FindByID(id uint) (*model.Proxy, error) {
	var proxy model.Proxy
	err := r.db.Preload("ProxyGroup").First(&proxy, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &proxy, err
}

func (r *proxyRepository) List(offset, limit int) ([]model.Proxy, int64, error) {
	var proxies []model.Proxy
	var total int64
	r.db.Model(&model.Proxy{}).Count(&total)
	err := r.db.Preload("ProxyGroup").Order("created_at desc").Offset(offset).Limit(limit).Find(&proxies).Error
	return proxies, total, err
}

func (r *proxyRepository) ListActive() ([]model.Proxy, error) {
	var proxies []model.Proxy
	err := r.db.Preload("ProxyGroup").Where("status = ?", "active").Find(&proxies).Error
	return proxies, err
}

func (r *proxyRepository) Delete(id uint) error {
	return r.db.Delete(&model.Proxy{}, id).Error
}

func (r *proxyRepository) CountByStatus(status string) (int64, error) {
	var count int64
	err := r.db.Model(&model.Proxy{}).Where("status = ?", status).Count(&count).Error
	return count, err
}

func (r *proxyRepository) CountByGroupID(id uint) (int64, error) {
	var count int64
	err := r.db.Model(&model.Proxy{}).Where("proxy_group_id = ?", id).Count(&count).Error
	return count, err
}
