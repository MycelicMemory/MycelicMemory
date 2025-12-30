// Package testutil provides testing utilities and helpers for Ultrathink
package testutil

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// TestDB represents a test database instance
type TestDB struct {
	*sql.DB
	Path string
	t    *testing.T
}

// NewTestDB creates a new temporary SQLite database for testing
// The database is automatically cleaned up after the test completes
func NewTestDB(t *testing.T) *TestDB {
	t.Helper()

	// Create temp directory
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Open database
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Enable foreign keys (required for CASCADE constraints)
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	testDB := &TestDB{
		DB:   db,
		Path: dbPath,
		t:    t,
	}

	// Cleanup on test completion
	t.Cleanup(func() {
		db.Close()
		os.Remove(dbPath)
	})

	return testDB
}

// InitSchema initializes the database with the verified schema
// This will be implemented in Phase 2 with actual schema
func (db *TestDB) InitSchema() error {
	// Placeholder - will be implemented in Phase 2
	// For now, create a simple test table
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS test_memories (
			id TEXT PRIMARY KEY,
			content TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	return err
}

// ExecScript executes a SQL script file
func (db *TestDB) ExecScript(path string) error {
	script, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read script: %w", err)
	}

	_, err = db.Exec(string(script))
	if err != nil {
		return fmt.Errorf("failed to execute script: %w", err)
	}

	return nil
}

// MustExec executes a SQL statement and fails the test on error
func (db *TestDB) MustExec(query string, args ...interface{}) sql.Result {
	db.t.Helper()

	result, err := db.Exec(query, args...)
	if err != nil {
		db.t.Fatalf("SQL exec failed: %v\nQuery: %s", err, query)
	}

	return result
}

// MustQuery executes a SQL query and fails the test on error
func (db *TestDB) MustQuery(query string, args ...interface{}) *sql.Rows {
	db.t.Helper()

	rows, err := db.Query(query, args...)
	if err != nil {
		db.t.Fatalf("SQL query failed: %v\nQuery: %s", err, query)
	}

	return rows
}

// Count returns the number of rows in a table
func (db *TestDB) Count(table string) int {
	db.t.Helper()

	var count int
	err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count)
	if err != nil {
		db.t.Fatalf("Failed to count rows in %s: %v", table, err)
	}

	return count
}

// AssertRowCount asserts that a table has exactly n rows
func (db *TestDB) AssertRowCount(table string, expected int) {
	db.t.Helper()

	actual := db.Count(table)
	if actual != expected {
		db.t.Errorf("Expected %d rows in %s, got %d", expected, table, actual)
	}
}

// TempDir creates a temporary directory for testing
// Automatically cleaned up after test completion
func TempDir(t *testing.T) string {
	t.Helper()
	return t.TempDir()
}

// TempFile creates a temporary file for testing
// Automatically cleaned up after test completion
func TempFile(t *testing.T, name string, content []byte) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, name)

	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	return path
}

// AssertNoError fails the test if err is not nil
func AssertNoError(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

// AssertError fails the test if err is nil
func AssertError(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Fatal("Expected error, got nil")
	}
}

// AssertEqual fails the test if got != want
func AssertEqual(t *testing.T, got, want interface{}) {
	t.Helper()

	if got != want {
		t.Errorf("Got %v, want %v", got, want)
	}
}

// AssertStringContains fails the test if str doesn't contain substr
func AssertStringContains(t *testing.T, str, substr string) {
	t.Helper()

	if !containsString(str, substr) {
		t.Errorf("String %q does not contain %q", str, substr)
	}
}

func containsString(str, substr string) bool {
	return len(str) >= len(substr) && (str == substr || findSubstring(str, substr))
}

func findSubstring(str, substr string) bool {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
