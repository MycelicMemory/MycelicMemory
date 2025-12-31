package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/MycelicMemory/ultrathink/internal/logging"
	_ "github.com/mattn/go-sqlite3"
)

var log = logging.GetLogger("database")

// Database represents a connection to the SQLite database
type Database struct {
	db   *sql.DB
	path string
	mu   sync.RWMutex
}

// Open opens a database connection and initializes the schema if needed
func Open(path string) (*Database, error) {
	log.Info("opening database", "path", path)

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Error("failed to create database directory", "error", err, "dir", dir)
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open SQLite database with foreign key support
	// The _foreign_keys=on parameter enables FK constraints
	dsn := fmt.Sprintf("%s?_foreign_keys=on&_journal_mode=WAL", path)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		log.Error("failed to open database", "error", err)
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(1) // SQLite only supports one writer
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Hour)

	// Verify connection
	if err := db.Ping(); err != nil {
		db.Close()
		log.Error("failed to ping database", "error", err)
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	database := &Database{
		db:   db,
		path: path,
	}

	log.Info("database connection established", "path", path)
	return database, nil
}

// InitSchema initializes the database schema
// This creates all tables, indexes, triggers, and FTS5 configuration
func (d *Database) InitSchema() error {
	log.Info("initializing database schema", "version", SchemaVersion)

	d.mu.Lock()
	defer d.mu.Unlock()

	// Check if schema already exists by checking for a key table
	var tableName string
	err := d.db.QueryRow(`
		SELECT name FROM sqlite_master
		WHERE type='table' AND name='memories'
		LIMIT 1
	`).Scan(&tableName)
	if err == nil && tableName != "" {
		log.Info("schema already initialized")
		return nil
	}
	log.Debug("schema not yet initialized", "check_err", err, "table_name", tableName)

	// Begin transaction for schema initialization
	tx, err := d.db.Begin()
	if err != nil {
		log.Error("failed to begin transaction", "error", err)
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Execute core schema (tables, indexes, constraints)
	log.Debug("creating core schema")
	if _, err := tx.Exec(CoreSchema); err != nil {
		log.Error("failed to create core schema", "error", err)
		return fmt.Errorf("failed to create core schema: %w", err)
	}

	// Execute FTS5 schema (virtual table, triggers)
	// FTS5 is optional, so skip if it fails
	log.Debug("creating FTS5 schema")
	if _, err := tx.Exec(FTS5Schema); err != nil {
		log.Warn("failed to create FTS5 schema (skipping)", "error", err)
		// Don't return error - FTS5 is optional
	}

	// Record schema version
	_, err = tx.Exec(`
		INSERT OR REPLACE INTO schema_version (version, applied_at)
		VALUES (?, CURRENT_TIMESTAMP)
	`, SchemaVersion)
	if err != nil {
		log.Error("failed to record schema version", "error", err)
		return fmt.Errorf("failed to record schema version: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		log.Error("failed to commit schema", "error", err)
		return fmt.Errorf("failed to commit schema: %w", err)
	}

	log.Info("database schema initialized successfully", "version", SchemaVersion)
	return nil
}

// Close closes the database connection
func (d *Database) Close() error {
	log.Info("closing database connection")
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db != nil {
		if err := d.db.Close(); err != nil {
			log.Error("failed to close database", "error", err)
			return err
		}
		log.Debug("database connection closed")
	}
	return nil
}

// DB returns the underlying sql.DB for advanced operations
func (d *Database) DB() *sql.DB {
	return d.db
}

// Path returns the database file path
func (d *Database) Path() string {
	return d.path
}

// Exec executes a SQL statement
func (d *Database) Exec(query string, args ...interface{}) (sql.Result, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.db.Exec(query, args...)
}

// Query executes a SQL query and returns rows
func (d *Database) Query(query string, args ...interface{}) (*sql.Rows, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.db.Query(query, args...)
}

// QueryRow executes a SQL query and returns a single row
func (d *Database) QueryRow(query string, args ...interface{}) *sql.Row {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.db.QueryRow(query, args...)
}

// Begin starts a new transaction
func (d *Database) Begin() (*sql.Tx, error) {
	return d.db.Begin()
}

// GetSchemaVersion returns the current schema version
func (d *Database) GetSchemaVersion() (int, error) {
	var version int
	err := d.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_version").Scan(&version)
	if err != nil {
		return 0, fmt.Errorf("failed to get schema version: %w", err)
	}
	return version, nil
}

// TableExists checks if a table exists in the database
func (d *Database) TableExists(name string) (bool, error) {
	var count int
	err := d.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master
		WHERE type='table' AND name=?
	`, name).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// CountRows returns the number of rows in a table
func (d *Database) CountRows(table string) (int, error) {
	var count int
	// Using parameterized table name is not possible in SQLite
	// Table name is validated before calling this function
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", table)
	if err := d.QueryRow(query).Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count rows in %s: %w", table, err)
	}
	return count, nil
}

// Vacuum runs VACUUM to optimize the database file
func (d *Database) Vacuum() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	_, err := d.db.Exec("VACUUM")
	return err
}

// Checkpoint forces a WAL checkpoint
func (d *Database) Checkpoint() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	_, err := d.db.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
	return err
}

// Stats returns database statistics
type Stats struct {
	Path          string
	SchemaVersion int
	TableCount    int
	MemoryCount   int
	RelationCount int
	CategoryCount int
	DomainCount   int
	SessionCount  int
	FileSizeBytes int64
}

// GetStats returns database statistics
func (d *Database) GetStats() (*Stats, error) {
	stats := &Stats{
		Path: d.path,
	}

	// Get schema version
	version, err := d.GetSchemaVersion()
	if err == nil {
		stats.SchemaVersion = version
	}

	// Count tables
	var tableCount int
	d.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table'").Scan(&tableCount)
	stats.TableCount = tableCount

	// Count records in main tables
	d.QueryRow("SELECT COUNT(*) FROM memories").Scan(&stats.MemoryCount)
	d.QueryRow("SELECT COUNT(*) FROM memory_relationships").Scan(&stats.RelationCount)
	d.QueryRow("SELECT COUNT(*) FROM categories").Scan(&stats.CategoryCount)
	d.QueryRow("SELECT COUNT(*) FROM domains").Scan(&stats.DomainCount)
	d.QueryRow("SELECT COUNT(*) FROM agent_sessions").Scan(&stats.SessionCount)

	// Get file size
	if info, err := os.Stat(d.path); err == nil {
		stats.FileSizeBytes = info.Size()
	}

	return stats, nil
}
