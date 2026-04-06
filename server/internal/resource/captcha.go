package resource

import (
	"context"
	"fmt"

	"github.com/msojocs/free2api/server/internal/model"
	captchapkg "github.com/msojocs/free2api/server/pkg/captcha"
	"gorm.io/gorm"
)

type CaptchaResource struct {
	db       *gorm.DB
	provider string
	apiKey   string
}

func NewCaptchaResource(db *gorm.DB, provider, apiKey string) *CaptchaResource {
	return &CaptchaResource{db: db, provider: provider, apiKey: apiKey}
}

type CaptchaSolution struct {
	Token    string
	Provider string
}

// Solve resolves a CAPTCHA challenge using the configured provider (yescaptcha or 2captcha).
// pageURL and siteKey are the target page and site-key for Turnstile challenges.
func (r *CaptchaResource) Solve(taskBatchID uint, pageURL, siteKey, captchaType string) (*CaptchaSolution, error) {
	if r.apiKey == "" {
		return nil, fmt.Errorf("captcha API key not configured (set CAPTCHA_API_KEY)")
	}

	solver, err := captchapkg.New(r.provider, r.apiKey)
	if err != nil {
		return nil, fmt.Errorf("captcha: unsupported provider %q: %w", r.provider, err)
	}

	var token string
	switch captchaType {
	case "turnstile", "":
		token, err = solver.SolveTurnstile(context.Background(), pageURL, siteKey)
	default:
		return nil, fmt.Errorf("captcha: unsupported type %q", captchaType)
	}
	if err != nil {
		_ = r.logResult(taskBatchID, pageURL, captchaType, "failed", 0)
		return nil, err
	}

	const approxCost = 0.001
	_ = r.logResult(taskBatchID, pageURL, captchaType, "success", approxCost)

	return &CaptchaSolution{
		Token:    token,
		Provider: r.provider,
	}, nil
}

func (r *CaptchaResource) logResult(taskBatchID uint, email, captchaType, status string, cost float64) error {
	log := &model.CaptchaLog{
		TaskBatchID: taskBatchID,
		Email:       email,
		Type:        captchaType,
		Provider:    r.provider,
		Status:      status,
		Cost:        cost,
	}
	return r.db.Create(log).Error
}

func (r *CaptchaResource) GetStats() map[string]interface{} {
	var total int64
	var success int64
	var totalCost float64

	r.db.Model(&model.CaptchaLog{}).Count(&total)
	r.db.Model(&model.CaptchaLog{}).Where("status = ?", "success").Count(&success)

	var logs []model.CaptchaLog
	r.db.Model(&model.CaptchaLog{}).Find(&logs)
	for _, l := range logs {
		totalCost += l.Cost
	}

	return map[string]interface{}{
		"total":      total,
		"success":    success,
		"failed":     total - success,
		"total_cost": totalCost,
	}
}

