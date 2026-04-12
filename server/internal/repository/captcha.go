package repository

import (
	"github.com/msojocs/ai-auto-register/server/internal/model"
	"gorm.io/gorm"
)

type captchaRepository struct {
	db *gorm.DB
}

// NewCaptchaRepository returns a new CaptchaRepository backed by the given DB.
func NewCaptchaRepository(db *gorm.DB) CaptchaRepository {
	return &captchaRepository{db: db}
}

func (r *captchaRepository) Create(log *model.CaptchaLog) error {
	return r.db.Create(log).Error
}

func (r *captchaRepository) ListByTask(taskBatchID uint) ([]model.CaptchaLog, error) {
	var logs []model.CaptchaLog
	err := r.db.Where("task_batch_id = ?", taskBatchID).Find(&logs).Error
	return logs, err
}

func (r *captchaRepository) SumCostByProvider() (map[string]float64, error) {
	type result struct {
		Provider string
		Total    float64
	}
	var rows []result
	err := r.db.Model(&model.CaptchaLog{}).
		Select("provider, SUM(cost) as total").
		Group("provider").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	m := make(map[string]float64, len(rows))
	for _, row := range rows {
		m[row.Provider] = row.Total
	}
	return m, nil
}

func (r *captchaRepository) Count() (int64, error) {
	var count int64
	err := r.db.Model(&model.CaptchaLog{}).Count(&count).Error
	return count, err
}
