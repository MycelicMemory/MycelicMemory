# Testing Guide

## Overview

Ultrathink uses Go's built-in testing framework with additional utilities for database and integration testing.

## Running Tests

### All Tests
```bash
make test
# or
go test ./...
```

### With Coverage
```bash
make test-coverage
# Opens coverage.html in browser
```

### Verbose Output
```bash
make test-verbose
# or
go test ./... -v
```

### Specific Package
```bash
go test ./pkg/config/...
go test ./internal/database/...
```

### With Race Detection
```bash
go test ./... -race
```

## Test Structure

### Unit Tests

Unit tests test individual functions and methods in isolation.

**Naming Convention:**
- File: `package_test.go`
- Function: `TestFunctionName`

**Example:**
```go
func TestConfigLoad(t *testing.T) {
    cfg, err := Load()
    if err != nil {
        t.Fatalf("Load failed: %v", err)
    }
    if cfg == nil {
        t.Error("Expected config, got nil")
    }
}
```

### Table-Driven Tests

Use table-driven tests for testing multiple scenarios:

```go
func TestValidate(t *testing.T) {
    tests := []struct {
        name      string
        input     *Config
        expectErr bool
    }{
        {
            name:      "valid config",
            input:     DefaultConfig(),
            expectErr: false,
        },
        {
            name: "invalid port",
            input: &Config{
                RestAPI: RestAPIConfig{Port: 99999},
            },
            expectErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.input.Validate()
            if (err != nil) != tt.expectErr {
                t.Errorf("Expected error=%v, got %v", tt.expectErr, err)
            }
        })
    }
}
```

### Database Tests

Use `testutil.NewTestDB` for database tests:

```go
func TestDatabaseInsert(t *testing.T) {
    db := testutil.NewTestDB(t)
    db.InitSchema()

    // Insert test data
    db.MustExec("INSERT INTO memories (id, content) VALUES (?, ?)",
        "test-id", "test content")

    // Verify
    db.AssertRowCount("memories", 1)
}
```

### Integration Tests

Integration tests are tagged with `//go:build integration`:

```go
//go:build integration

package api_test

import (
    "testing"
    "net/http/httptest"
)

func TestAPIEndpoint(t *testing.T) {
    // Test actual API integration
}
```

Run integration tests:
```bash
go test -tags=integration ./...
```

## Test Utilities

### testutil.NewTestDB(t)

Creates a temporary SQLite database for testing:

```go
db := testutil.NewTestDB(t)  // Auto-cleanup
db.InitSchema()              // Initialize schema
db.MustExec("INSERT ...")    // Execute SQL (fails test on error)
db.AssertRowCount("table", 5) // Assert row count
```

### testutil.TempDir(t)

Creates a temporary directory:

```go
dir := testutil.TempDir(t)  // Auto-cleanup
```

### testutil.TempFile(t, name, content)

Creates a temporary file:

```go
path := testutil.TempFile(t, "config.yaml", []byte("..."))
```

### testutil.AssertNoError(t, err)

Fails test if error is not nil:

```go
testutil.AssertNoError(t, err)
```

### testutil.AssertEqual(t, got, want)

Fails test if values don't match:

```go
testutil.AssertEqual(t, result, expected)
```

### testutil.AssertStringContains(t, str, substr)

Fails test if string doesn't contain substring:

```go
testutil.AssertStringContains(t, output, "expected text")
```

## Coverage Goals

- **Overall Coverage**: 80%+
- **Critical Paths**: 95%+ (database, API, CLI)
- **Utilities**: 70%+

Check coverage:
```bash
make test-coverage
open coverage.html
```

## Best Practices

### 1. Use t.Helper()

Mark helper functions with `t.Helper()`:

```go
func assertValid(t *testing.T, cfg *Config) {
    t.Helper()
    if err := cfg.Validate(); err != nil {
        t.Fatalf("Validation failed: %v", err)
    }
}
```

### 2. Use t.Cleanup()

Always clean up resources:

```go
func TestWithServer(t *testing.T) {
    server := startServer()
    t.Cleanup(func() {
        server.Stop()
    })
    // test code
}
```

### 3. Test Error Cases

Don't just test happy paths:

```go
func TestErrorHandling(t *testing.T) {
    _, err := OperationThatShouldFail()
    if err == nil {
        t.Fatal("Expected error, got nil")
    }
}
```

### 4. Use Subtests

Group related tests:

```go
func TestConfig(t *testing.T) {
    t.Run("Load", func(t *testing.T) { /* ... */ })
    t.Run("Validate", func(t *testing.T) { /* ... */ })
    t.Run("Save", func(t *testing.T) { /* ... */ })
}
```

### 5. Don't Use Global State

Each test should be independent:

```go
// Bad
var globalDB *sql.DB

// Good
func TestOperation(t *testing.T) {
    db := testutil.NewTestDB(t)
    // use local db
}
```

## Performance Testing

### Benchmarks

Use benchmarks for performance-critical code:

```go
func BenchmarkSearch(b *testing.B) {
    db := setupBenchmarkDB(b)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        db.Search("query")
    }
}
```

Run benchmarks:
```bash
go test -bench=. ./...
go test -bench=Search -benchmem ./internal/search/
```

### Performance Targets (Verified)

- FTS5 keyword search: <5ms
- Graph traversal: 4ms target
- UUID generation: <1μs
- Memory CRUD: <10ms

## Continuous Integration

Tests run automatically on:
- Every commit (via GitHub Actions)
- Pull requests
- Pre-release builds

CI configuration: `.github/workflows/test.yml`

## Debugging Tests

### Verbose Output
```bash
go test -v ./pkg/config/
```

### Run Specific Test
```bash
go test -run TestConfigLoad ./pkg/config/
```

### With Debug Logging
```bash
go test -v -args -debug ./...
```

### With Race Detector
```bash
go test -race ./...
```

## Test Organization

```
pkg/config/
├── config.go           # Implementation
├── config_test.go      # Unit tests
└── doc.go             # Package docs

internal/database/
├── database.go         # Implementation
├── database_test.go    # Unit tests
├── integration_test.go # Integration tests (//go:build integration)
└── README.md          # Package docs

internal/testutil/
├── testutil.go         # Test utilities
└── testutil_test.go    # Utility tests
```

## Common Patterns

### Testing Configuration
```go
func TestConfig(t *testing.T) {
    cfg := DefaultConfig()
    AssertNoError(t, cfg.Validate())
}
```

### Testing Database Operations
```go
func TestCRUD(t *testing.T) {
    db := NewTestDB(t)
    db.InitSchema()

    // Create
    id := createMemory(t, db, "test")

    // Read
    mem := getMemory(t, db, id)
    AssertEqual(t, mem.Content, "test")

    // Update
    updateMemory(t, db, id, "updated")

    // Delete
    deleteMemory(t, db, id)
    db.AssertRowCount("memories", 0)
}
```

### Testing HTTP APIs
```go
func TestAPIHandler(t *testing.T) {
    req := httptest.NewRequest("GET", "/api/v1/health", nil)
    w := httptest.NewRecorder()

    handler(w, req)

    AssertEqual(t, w.Code, 200)
    AssertStringContains(t, w.Body.String(), "healthy")
}
```

## Questions?

See `CONTRIBUTING.md` for more details on development workflow and code standards.
