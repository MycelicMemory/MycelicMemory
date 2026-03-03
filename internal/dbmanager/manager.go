package dbmanager

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/MycelicMemory/mycelicmemory/internal/database"
	"github.com/MycelicMemory/mycelicmemory/pkg/config"
)

// DatabaseInfo holds info about a database for display
type DatabaseInfo struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Description string `json:"description,omitempty"`
	CreatedAt   string `json:"created_at,omitempty"`
	SizeBytes   int64  `json:"size_bytes"`
	IsActive    bool   `json:"is_active"`
}

// Manager handles multi-database operations
type Manager struct {
	cfg *config.Config
}

// New creates a new database manager
func New(cfg *config.Config) *Manager {
	return &Manager{cfg: cfg}
}

// ListDatabases returns info about all configured databases
func (m *Manager) ListDatabases() ([]DatabaseInfo, error) {
	var result []DatabaseInfo

	// Always include the default database
	defaultPath := m.cfg.Database.Path
	activePath := m.cfg.GetActiveDBPath()

	// Check if default is in the profiles list
	hasDefault := false
	for _, p := range m.cfg.Databases {
		if p.Name == "default" {
			hasDefault = true
		}
		info := DatabaseInfo{
			Name:        p.Name,
			Path:        p.Path,
			Description: p.Description,
			CreatedAt:   p.CreatedAt,
			IsActive:    p.Path == activePath,
		}
		if stat, err := os.Stat(p.Path); err == nil {
			info.SizeBytes = stat.Size()
		}
		result = append(result, info)
	}

	if !hasDefault {
		info := DatabaseInfo{
			Name:     "default",
			Path:     defaultPath,
			IsActive: defaultPath == activePath,
		}
		if stat, err := os.Stat(defaultPath); err == nil {
			info.SizeBytes = stat.Size()
		}
		result = append([]DatabaseInfo{info}, result...)
	}

	return result, nil
}

// CreateDatabase creates a new named database with initialized schema
func (m *Manager) CreateDatabase(name, description string) (*DatabaseInfo, error) {
	// Check for duplicates
	for _, p := range m.cfg.Databases {
		if p.Name == name {
			return nil, fmt.Errorf("database %q already exists", name)
		}
	}
	if name == "default" {
		return nil, fmt.Errorf("cannot create database named 'default' — it is reserved")
	}

	dbDir := filepath.Join(config.ConfigPath(), "databases")
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create databases directory: %w", err)
	}

	dbPath := filepath.Join(dbDir, name+".db")

	// Open and initialize the new database
	db, err := database.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create database: %w", err)
	}
	if err := db.InitSchema(); err != nil {
		db.Close()
		os.Remove(dbPath)
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}
	if err := db.RunMigrations(); err != nil {
		db.Close()
		os.Remove(dbPath)
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}
	db.Close()

	now := time.Now().UTC().Format(time.RFC3339)
	profile := config.DatabaseProfile{
		Name:        name,
		Path:        dbPath,
		Description: description,
		CreatedAt:   now,
	}

	m.cfg.Databases = append(m.cfg.Databases, profile)
	if err := m.cfg.Save(); err != nil {
		return nil, fmt.Errorf("failed to save config: %w", err)
	}

	info := &DatabaseInfo{
		Name:        name,
		Path:        dbPath,
		Description: description,
		CreatedAt:   now,
		IsActive:    false,
	}
	if stat, err := os.Stat(dbPath); err == nil {
		info.SizeBytes = stat.Size()
	}

	return info, nil
}

// GetDatabase returns info about a specific database
func (m *Manager) GetDatabase(name string) (*DatabaseInfo, error) {
	activePath := m.cfg.GetActiveDBPath()

	if name == "default" {
		info := &DatabaseInfo{
			Name:     "default",
			Path:     m.cfg.Database.Path,
			IsActive: m.cfg.Database.Path == activePath,
		}
		if stat, err := os.Stat(m.cfg.Database.Path); err == nil {
			info.SizeBytes = stat.Size()
		}
		return info, nil
	}

	for _, p := range m.cfg.Databases {
		if p.Name == name {
			info := &DatabaseInfo{
				Name:        p.Name,
				Path:        p.Path,
				Description: p.Description,
				CreatedAt:   p.CreatedAt,
				IsActive:    p.Path == activePath,
			}
			if stat, err := os.Stat(p.Path); err == nil {
				info.SizeBytes = stat.Size()
			}
			return info, nil
		}
	}

	return nil, fmt.Errorf("database %q not found", name)
}

// SwitchDatabase changes the active database
func (m *Manager) SwitchDatabase(name string) error {
	// Verify the database exists
	if name == "default" {
		m.cfg.ActiveDatabase = ""
		return m.cfg.Save()
	}

	for _, p := range m.cfg.Databases {
		if p.Name == name {
			m.cfg.ActiveDatabase = name
			return m.cfg.Save()
		}
	}

	return fmt.Errorf("database %q not found", name)
}

// DeleteDatabase removes a database profile and optionally its file
func (m *Manager) DeleteDatabase(name string) error {
	if name == "default" {
		return fmt.Errorf("cannot delete the default database")
	}

	activePath := m.cfg.GetActiveDBPath()
	idx := -1
	for i, p := range m.cfg.Databases {
		if p.Name == name {
			idx = i
			if p.Path == activePath {
				return fmt.Errorf("cannot delete the active database — switch to another first")
			}
			break
		}
	}

	if idx < 0 {
		return fmt.Errorf("database %q not found", name)
	}

	dbPath := m.cfg.Databases[idx].Path

	// Remove from config
	m.cfg.Databases = append(m.cfg.Databases[:idx], m.cfg.Databases[idx+1:]...)
	if err := m.cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// Remove the file
	if err := os.Remove(dbPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("config updated but failed to delete file: %w", err)
	}

	return nil
}

// ArchiveDatabase creates a timestamped backup copy
func (m *Manager) ArchiveDatabase(name string) (string, error) {
	var dbPath string

	if name == "" || name == "default" {
		dbPath = m.cfg.Database.Path
	} else {
		found := false
		for _, p := range m.cfg.Databases {
			if p.Name == name {
				dbPath = p.Path
				found = true
				break
			}
		}
		if !found {
			return "", fmt.Errorf("database %q not found", name)
		}
	}

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return "", fmt.Errorf("database file not found: %s", dbPath)
	}

	backupDir := filepath.Join(config.ConfigPath(), "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create backups directory: %w", err)
	}

	if name == "" {
		name = "default"
	}
	timestamp := time.Now().UTC().Format("20060102-150405")
	backupPath := filepath.Join(backupDir, fmt.Sprintf("%s_%s.db", name, timestamp))

	if err := copyFile(dbPath, backupPath); err != nil {
		return "", fmt.Errorf("failed to archive: %w", err)
	}

	return backupPath, nil
}

// ImportDatabase imports a .db file as a new named database
func (m *Manager) ImportDatabase(srcPath, name string) (*DatabaseInfo, error) {
	// Validate it's a SQLite file
	f, err := os.Open(srcPath)
	if err != nil {
		return nil, fmt.Errorf("cannot open file: %w", err)
	}
	header := make([]byte, 16)
	_, err = f.Read(header)
	f.Close()
	if err != nil {
		return nil, fmt.Errorf("cannot read file: %w", err)
	}
	if string(header[:15]) != "SQLite format 3" {
		return nil, fmt.Errorf("not a valid SQLite database file")
	}

	// Check for name collision
	for _, p := range m.cfg.Databases {
		if p.Name == name {
			return nil, fmt.Errorf("database %q already exists", name)
		}
	}

	dbDir := filepath.Join(config.ConfigPath(), "databases")
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create databases directory: %w", err)
	}

	dstPath := filepath.Join(dbDir, name+".db")
	if err := copyFile(srcPath, dstPath); err != nil {
		return nil, fmt.Errorf("failed to copy database: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	profile := config.DatabaseProfile{
		Name:      name,
		Path:      dstPath,
		CreatedAt: now,
	}
	m.cfg.Databases = append(m.cfg.Databases, profile)
	if err := m.cfg.Save(); err != nil {
		return nil, fmt.Errorf("failed to save config: %w", err)
	}

	info := &DatabaseInfo{
		Name:      name,
		Path:      dstPath,
		CreatedAt: now,
		IsActive:  false,
	}
	if stat, err := os.Stat(dstPath); err == nil {
		info.SizeBytes = stat.Size()
	}

	return info, nil
}

// ExportDatabase copies a database to a specified path
func (m *Manager) ExportDatabase(name, dstPath string) error {
	var dbPath string

	if name == "" || name == "default" {
		dbPath = m.cfg.Database.Path
	} else {
		for _, p := range m.cfg.Databases {
			if p.Name == name {
				dbPath = p.Path
				break
			}
		}
	}

	if dbPath == "" {
		return fmt.Errorf("database %q not found", name)
	}

	return copyFile(dbPath, dstPath)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}
