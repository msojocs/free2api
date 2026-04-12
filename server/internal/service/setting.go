package service

import (
	"errors"
	"strings"

	"github.com/msojocs/ai-auto-register/server/internal/model"
	"github.com/msojocs/ai-auto-register/server/internal/repository"
)

// SettingService manages global runtime settings.
type SettingService struct {
	repo               repository.SettingRepository
	proxyGroupRepo     repository.ProxyGroupRepository
	defaultSentinelURL string
}

func NewSettingService(
	repo repository.SettingRepository,
	proxyGroupRepo repository.ProxyGroupRepository,
	defaultSentinelURL string,
) *SettingService {
	return &SettingService{
		repo:               repo,
		proxyGroupRepo:     proxyGroupRepo,
		defaultSentinelURL: defaultSentinelURL,
	}
}

func (s *SettingService) Get() (*model.SystemSetting, error) {
	setting, err := s.repo.Get()
	if err != nil {
		return nil, err
	}
	// Use config default when the DB value is empty.
	if setting.SentinelBaseURL == "" {
		setting.SentinelBaseURL = s.defaultSentinelURL
	}
	return setting, nil
}

// GetSentinelBaseURL returns the configured URL, falling back to the config default.
func (s *SettingService) GetSentinelBaseURL() string {
	setting, err := s.repo.Get()
	if err != nil || strings.TrimSpace(setting.SentinelBaseURL) == "" {
		return s.defaultSentinelURL
	}
	return setting.SentinelBaseURL
}

func (s *SettingService) Save(
	sentinelBaseURL string,
	accountActionProxyGroupID *uint,
	accountCheckEnabled bool,
	accountCheckIntervalMinutes int,
) (*model.SystemSetting, error) {
	sentinelBaseURL = strings.TrimSpace(sentinelBaseURL)
	if sentinelBaseURL == "" {
		sentinelBaseURL = s.defaultSentinelURL
	}

	if accountActionProxyGroupID != nil {
		group, err := s.proxyGroupRepo.FindByID(*accountActionProxyGroupID)
		if err != nil {
			return nil, err
		}
		if group == nil {
			return nil, errors.New("proxy group not found")
		}
	}

	if accountCheckIntervalMinutes <= 0 {
		accountCheckIntervalMinutes = 60
	}

	setting := &model.SystemSetting{
		ID:                          1,
		SentinelBaseURL:             sentinelBaseURL,
		AccountActionProxyGroupID:   accountActionProxyGroupID,
		AccountCheckEnabled:         accountCheckEnabled,
		AccountCheckIntervalMinutes: accountCheckIntervalMinutes,
	}
	if err := s.repo.Save(setting); err != nil {
		return nil, err
	}
	return setting, nil
}
