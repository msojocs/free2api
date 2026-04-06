package model

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// InitDB opens the database selected by driver and dsn and runs AutoMigrate.
//
// driver must be either "sqlite" (default) or "postgres".
// For SQLite, dsn is a file path.
// For PostgreSQL, dsn is a libpq key=value string (host=... user=... ...).
func InitDB(driver, dsn string) error {
	var dialector gorm.Dialector
	switch strings.ToLower(driver) {
	case "postgres", "postgresql":
		dialector = postgres.Open(dsn)
	default: // sqlite
		dir := filepath.Dir(dsn)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create db directory: %w", err)
		}
		dialector = sqlite.Open(dsn)
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		if strings.Contains(err.Error(), "CGO_ENABLED=0") {
			return fmt.Errorf("failed to open database (%s): sqlite database file is created automatically, but github.com/mattn/go-sqlite3 requires CGO_ENABLED=1 and a working C toolchain", driver)
		}
		return fmt.Errorf("failed to open database (%s): %w", driver, err)
	}
	DB = db
	return db.AutoMigrate(&User{}, &TaskBatch{}, &Account{}, &ProxyGroup{}, &Proxy{}, &CaptchaLog{}, &PushTemplate{}, &TempMailProvider{}, &SystemSetting{})
}

// SeedPushTemplate seeds the built-in CPA push template if no system template exists.
func SeedPushTemplate(db *gorm.DB) {
	var count int64
	db.Model(&PushTemplate{}).Where("is_system = ?", true).Count(&count)
	if count > 0 {
		return
	}
	cpa := &PushTemplate{
		Name:         "CLIProxyAPI (CPA)",
		Enabled:      false,
		URL:          "http://localhost:3000/api/accounts",
		Method:       "POST",
		Headers:      `{"Content-Type": "application/json"}`,
		BodyTemplate: `{"type": "{{.type}}", "email": "{{.email}}", "password": "{{.password}}", "token": "{{.extra}}"}`,
		Description:  "Built-in CPA (CLIProxyAPI) push template. Set the URL to your CLIProxyAPI instance and enable it.",
		IsSystem:     true,
		AccountType:  "",
	}
	db.Create(cpa)
}

func SeedTempMailProviders(db *gorm.DB) {
	providers := []TempMailProvider{
		{
			Name:         "TempMail",
			ProviderType: "tempmail",
			Config: JSONMap{
				"version": "1",
			},
			Description: "Temp mail provider with API at https://api.tempmail.lol. No auth required.",
			IsSystem:    true,
			Enabled:     true,
		},
	}
	for _, p := range providers {
		var existing TempMailProvider
		if err := db.Where("name = ?", p.Name).First(&existing).Error; err == gorm.ErrRecordNotFound {
			db.Create(&p)
		}
	}
}
