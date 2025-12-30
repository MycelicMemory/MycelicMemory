package testutil

import (
	"os"
	"testing"
)

func TestNewTestDB(t *testing.T) {
	db := NewTestDB(t)

	// Verify database is open
	if err := db.Ping(); err != nil {
		t.Fatalf("Database ping failed: %v", err)
	}

	// Verify foreign keys are enabled
	var fkEnabled int
	err := db.QueryRow("PRAGMA foreign_keys").Scan(&fkEnabled)
	if err != nil {
		t.Fatalf("Failed to check foreign keys: %v", err)
	}
	if fkEnabled != 1 {
		t.Error("Foreign keys not enabled")
	}
}

func TestTestDB_InitSchema(t *testing.T) {
	db := NewTestDB(t)

	if err := db.InitSchema(); err != nil {
		t.Fatalf("InitSchema failed: %v", err)
	}

	// Verify test table was created
	var tableName string
	err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='test_memories'").Scan(&tableName)
	if err != nil {
		t.Fatalf("Test table not created: %v", err)
	}
	if tableName != "test_memories" {
		t.Errorf("Expected table name test_memories, got %s", tableName)
	}
}

func TestTestDB_MustExec(t *testing.T) {
	db := NewTestDB(t)
	db.InitSchema()

	// Should not panic on successful exec
	db.MustExec("INSERT INTO test_memories (id, content) VALUES (?, ?)", "test-id", "test content")

	// Verify insert worked
	var count int
	db.QueryRow("SELECT COUNT(*) FROM test_memories").Scan(&count)
	if count != 1 {
		t.Errorf("Expected 1 row, got %d", count)
	}
}

func TestTestDB_Count(t *testing.T) {
	db := NewTestDB(t)
	db.InitSchema()

	// Initially should be 0
	if count := db.Count("test_memories"); count != 0 {
		t.Errorf("Expected 0 rows, got %d", count)
	}

	// Insert some rows
	db.MustExec("INSERT INTO test_memories (id, content) VALUES (?, ?)", "id1", "content1")
	db.MustExec("INSERT INTO test_memories (id, content) VALUES (?, ?)", "id2", "content2")

	// Should be 2
	if count := db.Count("test_memories"); count != 2 {
		t.Errorf("Expected 2 rows, got %d", count)
	}
}

func TestTestDB_AssertRowCount(t *testing.T) {
	db := NewTestDB(t)
	db.InitSchema()

	// Initially 0
	db.AssertRowCount("test_memories", 0)

	// Insert one row
	db.MustExec("INSERT INTO test_memories (id, content) VALUES (?, ?)", "id1", "content1")
	db.AssertRowCount("test_memories", 1)
}

func TestTempDir(t *testing.T) {
	dir := TempDir(t)

	// Verify directory exists
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("Temp directory doesn't exist: %v", err)
	}
	if !info.IsDir() {
		t.Error("Path is not a directory")
	}
}

func TestTempFile(t *testing.T) {
	content := []byte("test content")
	path := TempFile(t, "test.txt", content)

	// Verify file exists
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read temp file: %v", err)
	}

	if string(data) != string(content) {
		t.Errorf("Expected content %q, got %q", string(content), string(data))
	}
}

func TestAssertNoError(t *testing.T) {
	// Should not fail with nil error
	AssertNoError(t, nil)

	// Test with actual error would fail the test, so we can't test that case here
}

func TestAssertEqual(t *testing.T) {
	AssertEqual(t, 1, 1)
	AssertEqual(t, "test", "test")
	AssertEqual(t, true, true)
}

func TestAssertStringContains(t *testing.T) {
	AssertStringContains(t, "hello world", "world")
	AssertStringContains(t, "hello world", "hello")
	AssertStringContains(t, "hello world", "o w")
}
