package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

// Config represents the complete application configuration
// Structure verified from Local Memory v1.2.0 config.yaml
type Config struct {
	Profile  string           `mapstructure:"profile"`
	Database DatabaseConfig   `mapstructure:"database"`
	Setup    SetupConfig      `mapstructure:"setup"`
	License  LicenseConfig    `mapstructure:"license"`
	RestAPI  RestAPIConfig    `mapstructure:"rest_api"`
	Session  SessionConfig    `mapstructure:"session"`
	Logging  LoggingConfig    `mapstructure:"logging"`
	Ollama   OllamaConfig     `mapstructure:"ollama"`
	Qdrant   QdrantConfig     `mapstructure:"qdrant"`
}

// DatabaseConfig holds database configuration
// Verified from config.yaml
type DatabaseConfig struct {
	Path           string        `mapstructure:"path"`
	BackupInterval time.Duration `mapstructure:"backup_interval"`
	MaxBackups     int           `mapstructure:"max_backups"`
	AutoMigrate    bool          `mapstructure:"auto_migrate"`
}

// SetupConfig holds setup wizard configuration
type SetupConfig struct {
	FirstRun     bool `mapstructure:"first_run"`
	WizardShown  bool `mapstructure:"wizard_shown"`
}

// LicenseConfig holds license and terms configuration
type LicenseConfig struct {
	Required       bool        `mapstructure:"required"`
	CheckOnStartup bool        `mapstructure:"check_on_startup"`
	Terms          TermsConfig `mapstructure:"terms"`
}

// TermsConfig holds terms of service configuration
type TermsConfig struct {
	Required bool   `mapstructure:"required"`
	Source   string `mapstructure:"source"`
}

// RestAPIConfig holds REST API server configuration
// Verified behavior: auto_port enables automatic port selection
type RestAPIConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	AutoPort bool   `mapstructure:"auto_port"`
	Port     int    `mapstructure:"port"`
	Host     string `mapstructure:"host"`
	CORS     bool   `mapstructure:"cors"`
}

// SessionConfig holds session management configuration
// Verified strategies: "git-directory", "manual", or "hash"
type SessionConfig struct {
	AutoGenerate bool   `mapstructure:"auto_generate"`
	Strategy     string `mapstructure:"strategy"`  // "git-directory", "manual", or "hash"
	ManualID     string `mapstructure:"manual_id"` // Used when strategy is "manual"
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level  string `mapstructure:"level"`  // debug, info, warn, error
	Format string `mapstructure:"format"` // console, json
}

// OllamaConfig holds Ollama AI service configuration
// Verified models: nomic-embed-text (768-dim), qwen2.5:3b
type OllamaConfig struct {
	Enabled        bool   `mapstructure:"enabled"`
	AutoDetect     bool   `mapstructure:"auto_detect"`
	BaseURL        string `mapstructure:"base_url"`
	EmbeddingModel string `mapstructure:"embedding_model"` // nomic-embed-text
	ChatModel      string `mapstructure:"chat_model"`      // qwen2.5:3b
}

// QdrantConfig holds Qdrant vector database configuration
// Verified: localhost:6333, HNSW (m=16, ef_construct=100)
type QdrantConfig struct {
	Enabled    bool   `mapstructure:"enabled"`
	AutoDetect bool   `mapstructure:"auto_detect"`
	URL        string `mapstructure:"url"`
}

// DefaultConfig returns configuration with verified default values
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, ".ultrathink")

	return &Config{
		Profile: "default",
		Database: DatabaseConfig{
			Path:           filepath.Join(configDir, "memories.db"),
			BackupInterval: 24 * time.Hour,
			MaxBackups:     7,
			AutoMigrate:    true,
		},
		Setup: SetupConfig{
			FirstRun:    true,
			WizardShown: false,
		},
		License: LicenseConfig{
			Required:       false, // Open source version
			CheckOnStartup: false,
			Terms: TermsConfig{
				Required: false,
				Source:   "embedded",
			},
		},
		RestAPI: RestAPIConfig{
			Enabled:  true,
			AutoPort: true,
			Port:     3002,
			Host:     "localhost",
			CORS:     true,
		},
		Session: SessionConfig{
			AutoGenerate: true,
			Strategy:     "git-directory",
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "console",
		},
		Ollama: OllamaConfig{
			Enabled:        true,
			AutoDetect:     true,
			BaseURL:        "http://localhost:11434",
			EmbeddingModel: "nomic-embed-text",
			ChatModel:      "qwen2.5:3b",
		},
		Qdrant: QdrantConfig{
			Enabled:    true,
			AutoDetect: true,
			URL:        "http://localhost:6333",
		},
	}
}

// Load loads configuration from YAML file with fallback to defaults
// Searches in multiple locations:
// 1. ./config.yaml (current directory)
// 2. ~/.ultrathink/config.yaml (user home)
// 3. /etc/ultrathink/config.yaml (system-wide)
func Load() (*Config, error) {
	v := viper.New()

	// Set config name and type
	v.SetConfigName("config")
	v.SetConfigType("yaml")

	// Add search paths
	v.AddConfigPath(".")                                          // Current directory
	homeDir, _ := os.UserHomeDir()
	v.AddConfigPath(filepath.Join(homeDir, ".ultrathink"))       // User config
	v.AddConfigPath("/etc/ultrathink")                           // System config

	// Set default values
	setDefaults(v)

	// Attempt to read config file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; use defaults
			return DefaultConfig(), nil
		}
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	// Unmarshal into Config struct
	config := &Config{}
	if err := v.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// setDefaults sets default values in Viper
func setDefaults(v *viper.Viper) {
	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, ".ultrathink")

	v.SetDefault("profile", "default")
	v.SetDefault("database.path", filepath.Join(configDir, "memories.db"))
	v.SetDefault("database.backup_interval", "24h")
	v.SetDefault("database.max_backups", 7)
	v.SetDefault("database.auto_migrate", true)

	v.SetDefault("rest_api.enabled", true)
	v.SetDefault("rest_api.auto_port", true)
	v.SetDefault("rest_api.port", 3002)
	v.SetDefault("rest_api.host", "localhost")
	v.SetDefault("rest_api.cors", true)

	v.SetDefault("session.auto_generate", true)
	v.SetDefault("session.strategy", "git-directory")

	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "console")

	v.SetDefault("ollama.enabled", true)
	v.SetDefault("ollama.auto_detect", true)
	v.SetDefault("ollama.base_url", "http://localhost:11434")
	v.SetDefault("ollama.embedding_model", "nomic-embed-text")
	v.SetDefault("ollama.chat_model", "qwen2.5:3b")

	v.SetDefault("qdrant.enabled", true)
	v.SetDefault("qdrant.auto_detect", true)
	v.SetDefault("qdrant.url", "http://localhost:6333")
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate database configuration
	if c.Database.Path == "" {
		return fmt.Errorf("database.path is required")
	}
	if c.Database.MaxBackups < 0 {
		return fmt.Errorf("database.max_backups must be >= 0")
	}

	// Validate REST API configuration
	if c.RestAPI.Enabled {
		if c.RestAPI.Port < 1 || c.RestAPI.Port > 65535 {
			return fmt.Errorf("rest_api.port must be between 1 and 65535")
		}
		if c.RestAPI.Host == "" {
			return fmt.Errorf("rest_api.host is required when REST API is enabled")
		}
	}

	// Validate session strategy
	if c.Session.Strategy != "git-directory" && c.Session.Strategy != "manual" {
		return fmt.Errorf("session.strategy must be 'git-directory' or 'manual'")
	}

	// Validate logging level
	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[c.Logging.Level] {
		return fmt.Errorf("logging.level must be one of: debug, info, warn, error")
	}

	// Validate logging format
	validFormats := map[string]bool{"console": true, "json": true}
	if !validFormats[c.Logging.Format] {
		return fmt.Errorf("logging.format must be one of: console, json")
	}

	// Validate Ollama configuration
	if c.Ollama.Enabled && c.Ollama.BaseURL == "" {
		return fmt.Errorf("ollama.base_url is required when Ollama is enabled")
	}

	// Validate Qdrant configuration
	if c.Qdrant.Enabled && c.Qdrant.URL == "" {
		return fmt.Errorf("qdrant.url is required when Qdrant is enabled")
	}

	return nil
}

// EnsureConfigDir creates the configuration directory if it doesn't exist
func (c *Config) EnsureConfigDir() error {
	configDir := filepath.Dir(c.Database.Path)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	return nil
}

// ConfigPath returns the path to the configuration directory
func ConfigPath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".ultrathink")
}

// DatabasePath returns the default database path
func DatabasePath() string {
	return filepath.Join(ConfigPath(), "memories.db")
}
