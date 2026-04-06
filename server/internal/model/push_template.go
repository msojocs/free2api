package model

type PushTemplate struct {
	BaseModel
	Name         string `gorm:"not null" json:"name"`
	Enabled      bool   `gorm:"default:true" json:"enabled"`
	URL          string `gorm:"not null" json:"url"`
	Method       string `gorm:"default:'POST'" json:"method"`
	Headers      string `gorm:"type:text" json:"headers"`       // JSON object string, e.g. {"Content-Type":"application/json"}
	BodyTemplate string `gorm:"type:text" json:"body_template"` // Go text/template
	Description  string `json:"description"`
	IsSystem     bool   `gorm:"default:false" json:"is_system"` // system templates cannot be deleted
	// AccountType filters which account type triggers this template (empty = all types).
	AccountType string `gorm:"default:''" json:"account_type"`
}
