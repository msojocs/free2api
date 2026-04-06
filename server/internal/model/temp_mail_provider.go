package model

// TempMailProvider stores configuration for a temporary email provider used
// during automatic account registration. It is distinct from the Mail model
// which holds IMAP credentials for pre-provisioned mailboxes.
type TempMailProvider struct {
	BaseModel
	Name         string  `gorm:"not null" json:"name"`
	ProviderType string  `gorm:"not null" json:"provider_type"` // mailtm, cfworker, tempmail, moemail, freemail, laoudo, maliapi, luckmail
	Config       JSONMap `gorm:"type:text" json:"config"`       // provider-specific key/value pairs
	Enabled      bool    `gorm:"default:true" json:"enabled"`
	Description  string  `json:"description"`
	IsSystem     bool    `gorm:"default:false" json:"is_system"` // true if this is a built-in provider that should not be deleted
}
