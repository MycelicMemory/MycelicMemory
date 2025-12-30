package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Database represents a connection to the SQLite database
type Database struct {
	db   *sql.DB
	path string
	mu   sync.RWMutex
}

// Open opens a database connection and initializes the schema if needed
func Open(path string) (*Database, error) {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open SQLite database with foreign key support
	// The _foreign_keys=on parameter enables FK constraints
	dsn := fmt.Sprintf("%s?_foreign_keys=on&_journal_mode=WAL", path)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(1) // SQLite only supports one writer
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Hour)

	// Verify connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	database := &Database{
		db:   db,
		path: path,
	}

	return database, nil
}

// InitSchema initializes the database schema
// This creates all tables, indexes, triggers, and FTS5 configuration
func (d *Database) InitSchema() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Begin transaction for schema initialization
	tx, err := d.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Execute core schema (tables, indexes, constraints)
	if _, err := tx.Exec(CoreSchema); err != nil {
		return fmt.Errorf("failed to create core schema: %w", err)
	}

	// Execute FTS5 schema (virtual table, triggers)
	if _, err := tx.Exec(FTS5Schema); err != nil {
		return fmt.Errorf("failed to create FTS5 schema: %w", err)
	}

	// Record schema version
	_, err = tx.Exec(`
		INSERT OR REPLACE INTO schema_version (version, applied_at)
		VALUES (?, CURRENT_TIMESTAMP)
	`, SchemaVersion)
	if err != nil {
		return fmt.Errorf("failed to record schema version: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit schema: %w", err)
	}

	return nil
}

// Close closes the database connection
func (d *Database) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db != nil {
		return d.db.Close()
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
