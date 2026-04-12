package service

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/msojocs/ai-auto-register/server/internal/model"
	"github.com/msojocs/ai-auto-register/server/internal/repository"
	"github.com/msojocs/ai-auto-register/server/internal/resource"
	"github.com/msojocs/ai-auto-register/server/pkg/httputil"
)

type ProxyService struct {
	repo      repository.ProxyRepository
	groupRepo repository.ProxyGroupRepository
	resource  *resource.ProxyResource
}

func NewProxyService(repo repository.ProxyRepository, groupRepo repository.ProxyGroupRepository, res *resource.ProxyResource) *ProxyService {
	return &ProxyService{repo: repo, groupRepo: groupRepo, resource: res}
}

func (s *ProxyService) List(page, limit int) ([]model.Proxy, int64, error) {
	offset := (page - 1) * limit
	return s.repo.List(offset, limit)
}

func (s *ProxyService) Create(host, port string, proxyGroupID *uint, username, password, protocol string) (*model.Proxy, error) {
	if host == "" || port == "" {
		return nil, errors.New("host and port are required")
	}
	if protocol == "" {
		protocol = "http"
	}
	if proxyGroupID != nil {
		group, err := s.groupRepo.FindByID(*proxyGroupID)
		if err != nil {
			return nil, err
		}
		if group == nil {
			return nil, errors.New("proxy group not found")
		}
	}
	proxy := &model.Proxy{
		Host:         host,
		Port:         port,
		ProxyGroupID: proxyGroupID,
		Username:     username,
		Password:     password,
		Protocol:     protocol,
		Status:       "active",
	}
	if err := s.repo.Create(proxy); err != nil {
		return nil, err
	}
	s.resource.Reload()
	return s.repo.FindByID(proxy.ID)
}

func (s *ProxyService) Update(id uint, host, port string, proxyGroupID *uint, username, password, protocol string) (*model.Proxy, error) {
	proxy, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if proxy == nil {
		return nil, errors.New("proxy not found")
	}
	if proxyGroupID != nil {
		group, err := s.groupRepo.FindByID(*proxyGroupID)
		if err != nil {
			return nil, err
		}
		if group == nil {
			return nil, errors.New("proxy group not found")
		}
	}
	proxy.Host = host
	proxy.Port = port
	proxy.ProxyGroupID = proxyGroupID
	proxy.Username = username
	proxy.Password = password
	proxy.Protocol = protocol
	if err := s.repo.Update(proxy); err != nil {
		return nil, err
	}
	s.resource.Reload()
	return s.repo.FindByID(proxy.ID)
}

func (s *ProxyService) Delete(id uint) error {
	err := s.repo.Delete(id)
	if err == nil {
		s.resource.Reload()
	}
	return err
}

func (s *ProxyService) Test(id uint) (bool, error) {
	proxy, err := s.repo.FindByID(id)
	if err != nil {
		return false, err
	}
	if proxy == nil {
		return false, errors.New("proxy not found")
	}
	client := httputil.NewProxyClient(proxy.Host, proxy.Port, proxy.Username, proxy.Password, proxy.Protocol)
	client.Timeout = 10 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.google.com", nil)
	if err != nil {
		return false, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	resp.Body.Close()
	return resp.StatusCode < 500, nil
}
