package service

import (
	"errors"
	"strings"

	"github.com/msojocs/free2api/server/internal/model"
	"github.com/msojocs/free2api/server/internal/repository"
)

type ProxyGroupService struct {
	repo      repository.ProxyGroupRepository
	proxyRepo repository.ProxyRepository
}

func NewProxyGroupService(repo repository.ProxyGroupRepository, proxyRepo repository.ProxyRepository) *ProxyGroupService {
	return &ProxyGroupService{repo: repo, proxyRepo: proxyRepo}
}

func (s *ProxyGroupService) List() ([]model.ProxyGroup, error) {
	return s.repo.List()
}

func (s *ProxyGroupService) Create(name string) (*model.ProxyGroup, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("name is required")
	}
	existing, err := s.repo.FindByName(name)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, errors.New("proxy group already exists")
	}
	group := &model.ProxyGroup{Name: name}
	if err := s.repo.Create(group); err != nil {
		return nil, err
	}
	return group, nil
}

func (s *ProxyGroupService) Update(id uint, name string) (*model.ProxyGroup, error) {
	group, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if group == nil {
		return nil, errors.New("proxy group not found")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("name is required")
	}
	existing, err := s.repo.FindByName(name)
	if err != nil {
		return nil, err
	}
	if existing != nil && existing.ID != group.ID {
		return nil, errors.New("proxy group already exists")
	}
	group.Name = name
	if err := s.repo.Update(group); err != nil {
		return nil, err
	}
	return group, nil
}

func (s *ProxyGroupService) Delete(id uint) error {
	group, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	if group == nil {
		return errors.New("proxy group not found")
	}
	count, err := s.proxyRepo.CountByGroupID(id)
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.New("proxy group is not empty")
	}
	return s.repo.Delete(id)
}
