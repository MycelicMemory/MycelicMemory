package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	// Test database defaults
	if cfg.Database.MaxBackups != 7 {
		t.Errorf("Expected MaxBackups=7, got %d", cfg.Database.MaxBackups)
	}
	if cfg.Database.BackupInterval != 24*time.Hour {
		t.Errorf("Expected BackupInterval=24h, got %v", cfg.Database.BackupInterval)
	}
	if !cfg.Database.AutoMigrate {
		t.Error("Expected AutoMigrate=true")
	}

	// Test REST API defaults
	if !cfg.RestAPI.Enabled {
		t.Error("Expected RestAPI.Enabled=true")
	}
	if cfg.RestAPI.Port != 3002 {
		t.Errorf("Expected Port=3002, got %d", cfg.RestAPI.Port)
	}
	if cfg.RestAPI.Host != "localhost" {
		t.Errorf("Expected Host=localhost, got %s", cfg.RestAPI.Host)
	}
	if !cfg.RestAPI.CORS {
		t.Error("Expected CORS=true")
	}

	// Test session defaults
	if !cfg.Session.AutoGenerate {
		t.Error("Expected Session.AutoGenerate=true")
	}
	if cfg.Session.Strategy != "git-directory" {
		t.Errorf("Expected Strategy=git-directory, got %s", cfg.Session.Strategy)
	}

	// Test Ollama defaults (verified from config.yaml)
	if cfg.Ollama.EmbeddingModel != "nomic-embed-text" {
		t.Errorf("Expected EmbeddingModel=nomic-embed-text, got %s", cfg.Ollama.EmbeddingModel)
	}
	if cfg.Ollama.ChatModel != "qwen2.5:3b" {
		t.Errorf("Expected ChatModel=qwen2.5:3b, got %s", cfg.Ollama.ChatModel)
	}
	if cfg.Ollama.BaseURL != "http://localhost:11434" {
		t.Errorf("Expected Ollama BaseURL=http://localhost:11434, got %s", cfg.Ollama.BaseURL)
	}

	// Test Qdrant defaults
	if cfg.Qdrant.URL != "http://localhost:6333" {
		t.Errorf("Expected Qdrant URL=http://localhost:6333, got %s", cfg.Qdrant.URL)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name      string
		modify    func(*Config)
		expectErr bool
	}{
		{
			name:      "valid config",
			modify:    func(c *Config) {},
			expectErr: false,
		},
		{
			name: "empty database path",
			modify: func(c *Config) {
				c.Database.Path = ""
			},
			expectErr: true,
		},
		{
			name: "negative max backups",
			modify: func(c *Config) {
				c.Database.MaxBackups = -1
			},
			expectErr: true,
		},
		{
			name: "invalid port",
			modify: func(c *Config) {
				c.RestAPI.Port = 99999
			},
			expectErr: true,
		},
		{
			name: "invalid session strategy",
			modify: func(c *Config) {
				c.Session.Strategy = "invalid"
			},
			expectErr: true,
		},
		{
			name: "invalid logging level",
			modify: func(c *Config) {
				c.Logging.Level = "invalid"
			},
			expectErr: true,
		},
		{
			name: "empty ollama base url when enabled",
			modify: func(c *Config) {
				c.Ollama.Enabled = true
				c.Ollama.BaseURL = ""
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			tt.modify(cfg)

			err := cfg.Validate()
			if tt.expectErr && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

func TestLoadConfig_NoFile(t *testing.T) {
	// Change to temp directory where no config exists
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd) //nolint:errcheck
	_ = os.Chdir(tmpDir)

	// Temporarily override HOME to prevent finding user's config
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Should return default config without error
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Expected no error with missing config, got: %v", err)
	}

	if cfg == nil {
		t.Fatal("Expected config, got nil")
	}

	// Verify it's using defaults
	if cfg.RestAPI.Port != 3002 {
		t.Errorf("Expected default port 3002, got %d", cfg.RestAPI.Port)
	}
}

func TestLoadConfig_WithFile(t *testing.T) {
	// Create temp directory and config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
profile: test
database:
  path: /tmp/test.db
  backup_interval: 12h
  max_backups: 3
  auto_migrate: false
rest_api:
  enabled: true
  port: 4000
  host: 127.0.0.1
  cors: false
session:
  auto_generate: false
  strategy: manual
logging:
  level: debug
  format: json
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Change to temp directory
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd) //nolint:errcheck
	_ = os.Chdir(tmpDir)

	// Load config
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify loaded values
	if cfg.Profile != "test" {
		t.Errorf("Expected profile=test, got %s", cfg.Profile)
	}
	if cfg.Database.Path != "/tmp/test.db" {
		t.Errorf("Expected database path=/tmp/test.db, got %s", cfg.Database.Path)
	}
	if cfg.Database.MaxBackups != 3 {
		t.Errorf("Expected max_backups=3, got %d", cfg.Database.MaxBackups)
	}
	if cfg.RestAPI.Port != 4000 {
		t.Errorf("Expected port=4000, got %d", cfg.RestAPI.Port)
	}
	if cfg.RestAPI.CORS {
		t.Error("Expected CORS=false, got true")
	}
	if cfg.Session.Strategy != "manual" {
		t.Errorf("Expected strategy=manual, got %s", cfg.Session.Strategy)
	}
	if cfg.Logging.Level != "debug" {
		t.Errorf("Expected level=debug, got %s", cfg.Logging.Level)
	}
}

func TestEnsureConfigDir(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &Config{
		Database: DatabaseConfig{
			Path: filepath.Join(tmpDir, "subdir", "test.db"),
		},
	}

	if err := cfg.EnsureConfigDir(); err != nil {
		t.Fatalf("EnsureConfigDir failed: %v", err)
	}

	// Verify directory was created
	if _, err := os.Stat(filepath.Join(tmpDir, "subdir")); os.IsNotExist(err) {
		t.Error("Config directory was not created")
	}
}

func TestConfigPath(t *testing.T) {
	path := ConfigPath()
	if path == "" {
		t.Error("ConfigPath returned empty string")
	}

	homeDir, _ := os.UserHomeDir()
	expected := filepath.Join(homeDir, ".mycelicmemory")
	if path != expected {
		t.Errorf("Expected %s, got %s", expected, path)
	}
}

func TestDatabasePath(t *testing.T) {
	path := DatabasePath()
	if path == "" {
		t.Error("DatabasePath returned empty string")
	}

	if filepath.Base(path) != "memories.db" {
		t.Errorf("Expected database file named memories.db, got %s", filepath.Base(path))
	}
}
