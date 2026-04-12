package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/msojocs/ai-auto-register/server/internal/model"
	"github.com/msojocs/ai-auto-register/server/internal/repository"
	"github.com/msojocs/ai-auto-register/server/pkg/mailprovider"
)

// TempMailProviderService manages temporary email provider configurations used
// during auto-registration.
type TempMailProviderService struct {
	repo repository.TempMailProviderRepository
}

func NewTempMailProviderService(repo repository.TempMailProviderRepository) *TempMailProviderService {
	return &TempMailProviderService{repo: repo}
}

func (s *TempMailProviderService) List() ([]model.TempMailProvider, error) {
	return s.repo.List()
}

func (s *TempMailProviderService) GetByID(id uint) (*model.TempMailProvider, error) {
	p, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, errors.New("temp mail provider not found")
	}
	return p, nil
}

func (s *TempMailProviderService) Create(name, providerType, description string, config map[string]interface{}, enabled bool) (*model.TempMailProvider, error) {
	if name == "" {
		return nil, errors.New("name is required")
	}
	if providerType == "" {
		return nil, errors.New("provider_type is required")
	}
	// Validate provider type by attempting construction with the provided config.
	strCfg := toStringMap(config)
	if _, err := mailprovider.New(providerType, strCfg); err != nil {
		return nil, fmt.Errorf("invalid provider_type: %w", err)
	}
	p := &model.TempMailProvider{
		Name:         name,
		ProviderType: providerType,
		Config:       model.JSONMap(config),
		Enabled:      enabled,
		Description:  description,
	}
	if err := s.repo.Create(p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *TempMailProviderService) Update(id uint, name, providerType, description string, config map[string]interface{}, enabled bool) (*model.TempMailProvider, error) {
	p, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, errors.New("temp mail provider not found")
	}
	if name != "" {
		p.Name = name
	}
	if providerType != "" {
		strCfg := toStringMap(config)
		if _, err := mailprovider.New(providerType, strCfg); err != nil {
			return nil, fmt.Errorf("invalid provider_type: %w", err)
		}
		p.ProviderType = providerType
	}
	if config != nil {
		p.Config = model.JSONMap(config)
	}
	p.Enabled = enabled
	p.Description = description
	if err := s.repo.Update(p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *TempMailProviderService) Delete(id uint) error {
	p, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	if p == nil {
		return errors.New("temp mail provider not found")
	}
	return s.repo.Delete(id)
}

// BuildProvider instantiates the mailprovider.Provider for the given stored configuration ID.
func (s *TempMailProviderService) BuildProvider(id uint) (mailprovider.Provider, error) {
	p, err := s.repo.FindByID(id)
	if err != nil {
		return nil, fmt.Errorf("temp mail provider %d not found: %w", id, err)
	}
	strCfg := toStringMap(map[string]interface{}(p.Config))
	return mailprovider.New(p.ProviderType, strCfg)
}

// TestProvider creates a provider instance and calls GetEmail to verify the
// configuration is valid and the remote service is reachable.
func (s *TempMailProviderService) TestProvider(ctx context.Context, id uint) (string, error) {
	mp, err := s.BuildProvider(id)
	if err != nil {
		return "", err
	}
	acct, err := mp.GetEmail(ctx)
	if err != nil {
		return "", fmt.Errorf("get email failed: %w", err)
	}
	return acct.Email, nil
}

// toStringMap converts map[string]interface{} to map[string]string, skipping non-string values.
func toStringMap(m map[string]interface{}) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		if s, ok := v.(string); ok {
			out[k] = s
		}
	}
	return out
}
