package model

// SystemSetting stores global runtime settings in the database.
// Only a single row (ID=1) is used.
type SystemSetting struct {
	ID                          uint   `gorm:"primaryKey;default:1" json:"id"`
	SentinelBaseURL             string `gorm:"column:sentinel_base_url;not null;default:'http://127.0.0.1:3000'" json:"sentinel_base_url"`
	AccountActionProxyGroupID   *uint  `gorm:"column:account_action_proxy_group_id" json:"account_action_proxy_group_id,omitempty"`
	AccountCheckEnabled         bool   `gorm:"column:account_check_enabled;default:false" json:"account_check_enabled"`
	AccountCheckIntervalMinutes int    `gorm:"column:account_check_interval_minutes;default:60" json:"account_check_interval_minutes"`
}
