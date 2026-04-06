package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config is the top-level application configuration.
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Auth     AuthConfig     `yaml:"auth"`
	Captcha  CaptchaConfig  `yaml:"captcha"`
	Executor ExecutorConfig `yaml:"executor"`
}

// ExecutorConfig holds settings for job executors.
type ExecutorConfig struct {
	// SentinelBaseURL is the base URL of the local sentinel-token proxy server
	// used by the ChatGPT executor to obtain proof-of-work tokens.
	// Env: SENTINEL_BASE_URL
	SentinelBaseURL string `yaml:"sentinel_base_url"`
}

// ServerConfig controls the HTTP listener.
type ServerConfig struct {
	// Port the server listens on. Env: PORT
	Port string `yaml:"port"`
}

// DatabaseConfig selects the backend and supplies connection parameters.
type DatabaseConfig struct {
	// Driver is either "sqlite" (default) or "postgres".
	// Env: DB_DRIVER
	Driver string `yaml:"driver"`

	// --- SQLite ---
	// Path to the SQLite database file.
	// Env: DB_PATH
	Path string `yaml:"path"`

	// --- PostgreSQL ---
	// Host is the PostgreSQL server hostname. Env: DB_HOST
	Host string `yaml:"host"`
	// DBPort of the PostgreSQL server (default: 5432). Env: DB_PORT
	DBPort string `yaml:"db_port"`
	// Name is the PostgreSQL database name. Env: DB_NAME
	Name string `yaml:"name"`
	// User is the PostgreSQL user. Env: DB_USER
	User string `yaml:"user"`
	// Password is the PostgreSQL password. Env: DB_PASSWORD
	Password string `yaml:"password"`
	// SSLMode controls SSL for the PostgreSQL connection (default: disable). Env: DB_SSL_MODE
	SSLMode string `yaml:"ssl_mode"`
	// TimeZone sets the session time zone for PostgreSQL (default: Local). Env: DB_TIMEZONE
	TimeZone string `yaml:"timezone"`
}

// DSN builds a DSN string from the DatabaseConfig.
// For SQLite it returns the file path.
// For PostgreSQL it constructs a libpq key=value DSN.
func (d DatabaseConfig) DSN() string {
	switch strings.ToLower(d.Driver) {
	case "postgres", "postgresql":
		host := d.Host
		if host == "" {
			host = "127.0.0.1"
		}
		port := d.DBPort
		if port == "" {
			port = "5432"
		}
		sslMode := d.SSLMode
		if sslMode == "" {
			sslMode = "disable"
		}
		tz := d.TimeZone
		if tz == "" {
			tz = "Local"
		}
		return fmt.Sprintf(
			"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=%s",
			host, d.User, d.Password, d.Name, port, sslMode, tz,
		)
	default: // sqlite
		if d.Path == "" {
			return "data/free2api.db"
		}
		return d.Path
	}
}

// AuthConfig holds authentication / JWT settings.
type AuthConfig struct {
	// JWTSecret is the signing key for JWT tokens.
	// Env: JWT_SECRET
	JWTSecret string `yaml:"jwt_secret"`

	// DefaultAdminUsername is used to create the first admin account on an empty database.
	// Env: DEFAULT_ADMIN_USERNAME
	DefaultAdminUsername string `yaml:"default_admin_username"`

	// DefaultAdminPassword is used to create the first admin account on an empty database.
	// Env: DEFAULT_ADMIN_PASSWORD
	DefaultAdminPassword string `yaml:"default_admin_password"`
}

// CaptchaConfig configures the CAPTCHA solver provider.
type CaptchaConfig struct {
	// Provider is the captcha backend name (e.g. "2captcha", "yescaptcha").
	// Env: CAPTCHA_PROVIDER
	Provider string `yaml:"provider"`
	// APIKey is the key used to call the captcha API.
	// Env: CAPTCHA_API_KEY
	APIKey string `yaml:"api_key"`
}

// Load reads the YAML file at path and then applies environment-variable
// overrides. If the file does not exist the function still succeeds and returns
// a config populated only from environment variables and defaults.
func Load(path string) (*Config, error) {
	cfg := defaults()

	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("config: read %s: %w", path, err)
	}
	if err == nil {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("config: parse %s: %w", path, err)
		}
	}

	applyEnv(cfg)
	return cfg, nil
}

// defaults returns a Config pre-filled with safe defaults.
func defaults() *Config {
	return &Config{
		Server: ServerConfig{Port: "8080"},
		Database: DatabaseConfig{
			Driver:   "sqlite",
			Path:     "data/free2api.db",
			DBPort:   "5432",
			SSLMode:  "disable",
			TimeZone: "Local",
		},
		Auth: AuthConfig{
			JWTSecret:            "free2api_jwt_secret_change_in_production",
			DefaultAdminUsername: "admin",
			DefaultAdminPassword: "admin123456",
		},
		Captcha: CaptchaConfig{
			Provider: "2captcha",
		},
		Executor: ExecutorConfig{
			SentinelBaseURL: "http://127.0.0.1:3000",
		},
	}
}

// applyEnv overrides config fields with environment variables when set.
func applyEnv(cfg *Config) {
	if v := os.Getenv("PORT"); v != "" {
		cfg.Server.Port = v
	}
	if v := os.Getenv("DB_DRIVER"); v != "" {
		cfg.Database.Driver = v
	}
	if v := os.Getenv("DB_PATH"); v != "" {
		cfg.Database.Path = v
	}
	if v := os.Getenv("DB_HOST"); v != "" {
		cfg.Database.Host = v
	}
	if v := os.Getenv("DB_PORT"); v != "" {
		cfg.Database.DBPort = v
	}
	if v := os.Getenv("DB_NAME"); v != "" {
		cfg.Database.Name = v
	}
	if v := os.Getenv("DB_USER"); v != "" {
		cfg.Database.User = v
	}
	if v := os.Getenv("DB_PASSWORD"); v != "" {
		cfg.Database.Password = v
	}
	if v := os.Getenv("DB_SSL_MODE"); v != "" {
		cfg.Database.SSLMode = v
	}
	if v := os.Getenv("DB_TIMEZONE"); v != "" {
		cfg.Database.TimeZone = v
	}
	if v := os.Getenv("JWT_SECRET"); v != "" {
		cfg.Auth.JWTSecret = v
	}
	if v := os.Getenv("DEFAULT_ADMIN_USERNAME"); v != "" {
		cfg.Auth.DefaultAdminUsername = v
	}
	if v := os.Getenv("DEFAULT_ADMIN_PASSWORD"); v != "" {
		cfg.Auth.DefaultAdminPassword = v
	}
	if v := os.Getenv("CAPTCHA_PROVIDER"); v != "" {
		cfg.Captcha.Provider = v
	}
	if v := os.Getenv("CAPTCHA_API_KEY"); v != "" {
		cfg.Captcha.APIKey = v
	}
	if v := os.Getenv("SENTINEL_BASE_URL"); v != "" {
		cfg.Executor.SentinelBaseURL = v
	}
}
