package repository

import (
	"github.com/msojocs/ai-auto-register/server/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SettingRepository manages the single-row SystemSetting record.
type SettingRepository interface {
	Get() (*model.SystemSetting, error)
	Save(s *model.SystemSetting) error
}

type settingRepository struct {
	db *gorm.DB
}

func NewSettingRepository(db *gorm.DB) SettingRepository {
	return &settingRepository{db: db}
}

// Get returns the settings row, creating it with defaults if absent.
func (r *settingRepository) Get() (*model.SystemSetting, error) {
	var s model.SystemSetting
	err := r.db.FirstOrCreate(&s, model.SystemSetting{ID: 1}).Error
	return &s, err
}

// Save upserts the settings row (always ID=1).
func (r *settingRepository) Save(s *model.SystemSetting) error {
	s.ID = 1
	return r.db.Clauses(clause.OnConflict{UpdateAll: true}).Create(s).Error
}
