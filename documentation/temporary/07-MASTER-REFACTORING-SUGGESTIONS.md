# Master Refactoring Document

**MycelicMemory v1.2.0 | Comprehensive Refactoring Plan**
**Updated: 2026-01-28 | Audited against commit a6b01db (main)**

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Priority Matrix](#2-priority-matrix)
3. [Critical Refactoring: Database Layer](#3-critical-refactoring-database-layer)
4. [Critical Refactoring: MCP Server](#4-critical-refactoring-mcp-server)
5. [High Priority: AI Integration](#5-high-priority-ai-integration)
6. [High Priority: Search & Memory](#6-high-priority-search--memory)
7. [Medium Priority: CLI & API](#7-medium-priority-cli--api)
8. [Architecture Improvements](#8-architecture-improvements)
9. [Performance Optimization](#9-performance-optimization)
10. [Testing Improvements](#10-testing-improvements)
11. [Security Hardening](#11-security-hardening)
12. [Code Quality & Maintainability](#12-code-quality--maintainability)
13. [Implementation Roadmap](#13-implementation-roadmap)
14. [Conclusion](#14-conclusion)

---

## 1. Executive Summary

This document consolidates findings from 5 comprehensive code reviews covering all major modules of MycelicMemory. It has been audited against the current codebase to reflect resolved issues and updated priorities.

### Current State

| Metric | Original (v1.2.2) | Current (v1.2.0) | Status |
|--------|-------------------|-------------------|--------|
| Total Issues Identified | 54 | 54 | -- |
| Resolved | 0 | 18 | 33% done |
| Remaining | 54 | 36 | -- |
| Critical (P0) remaining | 8 | 3 | -- |
| High (P1) remaining | 16 | 8 | -- |
| Medium (P2) remaining | 20 | 16 | -- |
| Low (P3) remaining | 10 | 9 | -- |
| Largest file (LOC) | 1,687 | 1,240 | Improved |
| golangci-lint errors | ~60 | 0 | Resolved |

### Key Files (Current LOC)

| File | LOC | Status |
|------|-----|--------|
| `internal/database/operations.go` | 1,240 | Still needs splitting |
| `internal/mcp/handlers.go` | 1,116 | Still needs splitting |
| `internal/mcp/server.go` | 787 | Acceptable |
| `internal/mcp/formatter.go` | 743 | Acceptable |
| `internal/ai/ollama.go` | 574 | Acceptable |
| `internal/ai/manager.go` | 508 | Borderline |
| `internal/search/engine.go` | 449 | OK |
| `internal/memory/service.go` | 407 | OK |

### What's Been Fixed Since Original Document

1. **N+1 graph query** -- `GetGraphOptimized()` with recursive CTE (operations.go:674)
2. **Rate limiting** -- Full `internal/ratelimit/` package (bucket, config, limiter, metrics)
3. **Composite indexes** -- 9+ indexes on memories, 4 compound indexes on relationships (schema.go)
4. **Health endpoint** -- `GET /api/v1/health` (api/server.go:104)
5. **Pagination** -- `PaginationMetadata` with HasNextPage/HasPreviousPage (api/handlers_search.go)
6. **FTS5 escaping** -- `escapeFTS5Query()` function (operations.go:377)
7. **Graceful shutdown** -- `Stop(ctx)` with httpServer.Shutdown (api/server.go:183)
8. **CORS configuration** -- Fully configured middleware (api/server.go:49)
9. **Search merge/dedup** -- `mergeResults()` for hybrid search (search/engine.go:403)
10. **golangci-lint** -- All lint errors resolved (PR #66), CI enforced

### Top 5 Remaining Actions

1. **Split operations.go** (1,240 LOC) into domain-specific files
2. **Split handlers.go** (1,116 LOC) into tool-specific handlers
3. **Add embedding cache** for AI operations
4. **Consolidate Memory types** (4+ duplicates across packages)
5. **Add MCP request timeouts** (MCP server has no timeout protection)

---

## 2. Priority Matrix

### Critical (P0) -- 3 Remaining (5 Resolved)

| # | Issue | Location | Impact | Effort | Status |
|---|-------|----------|--------|--------|--------|
| ~~1~~ | ~~N+1 queries in graph~~ | ~~operations.go~~ | ~~Performance~~ | ~~Medium~~ | **RESOLVED** -- `GetGraphOptimized()` at operations.go:674 |
| 2 | handlers.go too large | mcp/handlers.go (1,116 LOC) | Unmaintainable | Medium | **OPEN** |
| 3 | operations.go too large | database/operations.go (1,240 LOC) | Unmaintainable | Medium | **OPEN** -- reduced from 1,687 but still needs split |
| 4 | 4+ duplicate Memory types | mcp/handlers.go:48,131,286 + api/handlers_memory.go | Confusion, bugs | Low | **OPEN** |
| ~~5~~ | ~~Tags as JSON string~~ | ~~database/models.go~~ | ~~Query inefficiency~~ | ~~High~~ | **RESOLVED** -- Current JSON approach adequate with FTS5 |
| ~~6~~ | ~~No rate limiting~~ | ~~api/server.go~~ | ~~Security risk~~ | ~~Low~~ | **RESOLVED** -- Full `internal/ratelimit/` package |
| ~~7~~ | ~~Fire-and-forget indexing~~ | ~~memory/service.go~~ | ~~Silent failures~~ | ~~Low~~ | **RESOLVED** -- Error handling fixed in PR #66 |
| ~~8~~ | ~~Binding error ignored~~ | ~~api/handlers.go~~ | ~~Bug source~~ | ~~Low~~ | **RESOLVED** -- errcheck lint pass enforced |

### High Priority (P1) -- 8 Remaining (8 Resolved)

| # | Issue | Location | Impact | Effort | Status |
|---|-------|----------|--------|--------|--------|
| 9 | No embedding cache | ai/manager.go | Performance | Medium | **OPEN** |
| 10 | Sequential hybrid search | search/engine.go:325 | Performance | Medium | **OPEN** -- keyword+semantic run sequentially |
| 11 | No batch operations | database/operations.go | Performance | Medium | **OPEN** |
| 12 | No MCP request timeout | mcp/server.go | Reliability | Low | **OPEN** -- API has timeouts, MCP does not |
| 13 | Character-based chunking | memory/chunker.go (231 LOC) | Accuracy | Medium | **OPEN** |
| 14 | Tool definitions inline | mcp/server.go | Maintainability | Medium | **OPEN** |
| 15 | Missing AI interfaces | ai/manager.go | Testability | Medium | **OPEN** -- tests use real DB, no mocks |
| 16 | No retry logic | ai/ollama.go | Reliability | Low | **OPEN** |
| ~~17~~ | ~~No status caching~~ | ~~ai/manager.go~~ | ~~Performance~~ | ~~Low~~ | **RESOLVED** -- `GetStatus()` at manager.go:46 |
| ~~18~~ | ~~N+1 in semanticSearch~~ | ~~search/engine.go~~ | ~~Performance~~ | ~~Medium~~ | **RESOLVED** -- Optimized with graph CTE |
| ~~19~~ | ~~Error message leakage~~ | ~~api/handlers.go~~ | ~~Security~~ | ~~Low~~ | **RESOLVED** -- Error sanitization in handlers |
| ~~20~~ | ~~No pagination~~ | ~~api/handlers.go~~ | ~~Scalability~~ | ~~Medium~~ | **RESOLVED** -- PaginationMetadata in handlers_search.go |
| ~~21~~ | ~~No merge sorting~~ | ~~search/engine.go~~ | ~~UX~~ | ~~Low~~ | **RESOLVED** -- `mergeResults()` at engine.go:403 |
| ~~22~~ | ~~Session detection cache~~ | ~~memory/session.go~~ | ~~Performance~~ | ~~Low~~ | **RESOLVED** -- Caching via `cacheDir` field |
| ~~23~~ | ~~No content length limit~~ | ~~memory/service.go~~ | ~~Security~~ | ~~Low~~ | **RESOLVED** -- Validation in MCP handlers |
| ~~24~~ | ~~Composite index missing~~ | ~~database/schema.go~~ | ~~Performance~~ | ~~Low~~ | **RESOLVED** -- 13+ indexes including compound |

### Medium Priority (P2) -- 16 Remaining (4 Resolved)

| # | Issue | Location | Impact | Effort | Status |
|---|-------|----------|--------|--------|--------|
| 25 | Query caching missing | database/operations.go | Performance | Medium | **OPEN** |
| ~~26~~ | ~~FTS5 escaping issues~~ | ~~database/operations.go~~ | ~~Search accuracy~~ | ~~Medium~~ | **RESOLVED** -- `escapeFTS5Query()` at operations.go:377 |
| ~~27~~ | ~~No health endpoint~~ | ~~api/server.go~~ | ~~Operations~~ | ~~Low~~ | **RESOLVED** -- `GET /api/v1/health` |
| 28 | CORS too permissive | api/server.go:49 | Security | Low | **OPEN** -- `AllowOrigins: ["*"]` |
| 29 | No structured output | cmd/*.go | Automation | Medium | **OPEN** |
| 30 | Two constructors | search/engine.go | Consistency | Low | **OPEN** |
| 31 | Manual field mapping | mcp/handlers.go | Error-prone | Medium | **OPEN** |
| 32 | Schema as strings | database/schema.go (280 LOC) | Maintainability | Medium | **OPEN** |
| 33 | No connection retry | database/database.go (275 LOC) | Reliability | Low | **OPEN** |
| 34 | Generic error messages | database/operations.go | Debugging | Low | **OPEN** |
| 35 | No down migrations | database/migrations.go (162 LOC) | Operations | Medium | **OPEN** -- V1ToV2 exists, no rollback |
| 36 | Global variables | cmd/root.go (115 LOC) | Testability | Medium | **OPEN** |
| 37 | Version hardcoded | cmd/root.go:19 | Maintenance | Low | **OPEN** -- `Version = "1.2.0"` (standard Go pattern) |
| 38 | No AI availability check | cmd/cmd_analyze.go | UX | Low | **OPEN** |
| 39 | O(n^2) relationship | ai/manager.go | Scalability | High | **OPEN** |
| 40 | No streaming | ai/ollama.go (574 LOC) | UX | Medium | **OPEN** |
| 41 | Verbose filter building | ai/manager.go | Readability | Low | **OPEN** |
| 42 | Embedding not persisted | ai/manager.go | Data loss | Low | **OPEN** |
| 43 | No change detection | memory/service.go | Efficiency | Low | **OPEN** |
| 44 | Post-query filtering | search/engine.go | Performance | Low | **OPEN** |
| ~~53~~ | ~~No graceful shutdown~~ | ~~cmd/root.go~~ | ~~Reliability~~ | ~~Low~~ | **RESOLVED** -- `Stop(ctx)` in api/server.go:183 |

### Low Priority (P3) -- 9 Remaining (1 Resolved)

| # | Issue | Location | Impact | Effort | Status |
|---|-------|----------|--------|--------|--------|
| 45 | No localization | mcp/formatter.go (743 LOC) | International | High | **OPEN** |
| 46 | ID type validation | mcp/types.go | Spec compliance | Low | **OPEN** |
| 47 | 8-byte hash | memory/session.go | Collision risk | Low | **OPEN** |
| 48 | No token counting | ai/ollama.go | Cost tracking | Medium | **OPEN** |
| 49 | Hardcoded split logic | memory/chunker.go (231 LOC) | Flexibility | Medium | **OPEN** |
| 50 | Manual field mapping | api/response.go | Maintenance | Low | **OPEN** |
| 51 | No progress indicator | cmd/*.go | UX | Low | **OPEN** |
| 52 | No PID file | cmd/cmd_service.go | Operations | Low | **OPEN** |
| 54 | No duplicate detection | memory/service.go (407 LOC) | Data quality | Medium | **OPEN** |

### New Issues (Found During Audit)

| # | Issue | Location | Impact | Effort | Priority |
|---|-------|----------|--------|--------|----------|
| N1 | No `.golangci.yml` config | project root | Code quality | Low | P2 |
| N2 | Old `GetGraph()` not removed | operations.go:573 | Dead code | Low | P3 |
| N3 | `ollama.go` exceeds 500 LOC | ai/ollama.go (574 LOC) | Maintainability | Medium | P2 |

---

## 3. Critical Refactoring: Database Layer

### 3.1 Split operations.go (P0 #3)

**Current State**: 1,240 LOC in single file (reduced from 1,687)

**Target Structure**:

```
internal/database/
├── database.go          # Connection management (275 LOC, existing)
├── models.go            # Domain structures (223 LOC, existing)
├── schema.go            # Table definitions (280 LOC, existing)
├── migrations.go        # Schema upgrades (162 LOC, existing)
├── memory_ops.go        # NEW: Memory CRUD (~250 LOC)
├── relationship_ops.go  # NEW: Relationship CRUD (~200 LOC)
├── category_ops.go      # NEW: Category operations (~100 LOC)
├── domain_ops.go        # NEW: Domain operations (~80 LOC)
├── session_ops.go       # NEW: Session operations (~80 LOC)
├── search_ops.go        # NEW: Search + FTS5 operations (~200 LOC)
├── graph_ops.go         # NEW: Graph traversal (both legacy + optimized, ~200 LOC)
├── benchmark_ops.go     # NEW: Benchmark operations (~80 LOC)
└── stats_ops.go         # NEW: Statistics (~50 LOC)
```

### 3.2 Add Batch Operations (P1 #11)

```go
// batch_ops.go
package database

func (d *Database) BatchCreateMemories(memories []*Memory) error {
    d.mu.Lock()
    defer d.mu.Unlock()

    tx, err := d.db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback() //nolint:errcheck

    stmt, err := tx.Prepare(`
        INSERT INTO memories (id, content, importance, tags, domain, session_id, created_at, updated_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?)
    `)
    if err != nil {
        return err
    }
    defer stmt.Close()

    now := time.Now()
    for _, m := range memories {
        if m.ID == "" {
            m.ID = uuid.New().String()
        }
        m.CreatedAt = now
        m.UpdatedAt = now
        tagsJSON, _ := json.Marshal(m.Tags)
        if _, err := stmt.Exec(m.ID, m.Content, m.Importance, string(tagsJSON),
            m.Domain, m.SessionID, m.CreatedAt, m.UpdatedAt); err != nil {
            return fmt.Errorf("failed to create memory %s: %w", m.ID, err)
        }
    }

    return tx.Commit()
}

func (d *Database) GetMemoriesByIDs(ids []string) (map[string]*Memory, error) {
    if len(ids) == 0 {
        return make(map[string]*Memory), nil
    }
    placeholders := make([]string, len(ids))
    args := make([]interface{}, len(ids))
    for i, id := range ids {
        placeholders[i] = "?"
        args[i] = id
    }
    query := fmt.Sprintf(`SELECT id, content, source, importance, tags, session_id, domain,
        embedding, created_at, updated_at FROM memories WHERE id IN (%s)`,
        strings.Join(placeholders, ","))
    rows, err := d.db.Query(query, args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    result := make(map[string]*Memory)
    for rows.Next() {
        m := &Memory{}
        var tagsJSON string
        if err := rows.Scan(&m.ID, &m.Content, &m.Source, &m.Importance,
            &tagsJSON, &m.SessionID, &m.Domain, &m.Embedding,
            &m.CreatedAt, &m.UpdatedAt); err != nil {
            continue
        }
        _ = json.Unmarshal([]byte(tagsJSON), &m.Tags)
        result[m.ID] = m
    }
    return result, nil
}
```

### 3.3 Add Down Migrations (P2 #35)

`internal/database/migrations.go` has `MigrationV1ToV2` but no rollback path. Add:

```go
func (d *Database) MigrationV2ToV1() error {
    tx, err := d.db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback() //nolint:errcheck

    // Remove V2-specific columns and tables
    // ... reverse each V1ToV2 step ...

    return tx.Commit()
}
```

---

## 4. Critical Refactoring: MCP Server

### 4.1 Split handlers.go (P0 #2)

**Current State**: 1,116 LOC in single file (reduced from 1,534)

**Target Structure**:

```
internal/mcp/
├── server.go              # Core server (787 LOC, existing)
├── types.go               # Protocol types (existing)
├── formatter.go           # Response formatting (743 LOC, existing)
├── tools/
│   ├── registry.go        # Tool registration + Handler interface
│   ├── definitions.go     # Tool JSON schemas
│   ├── memory.go          # store_memory, recall_memory, delete_memory
│   ├── search.go          # search_memory
│   ├── relationships.go   # create_relationship, find_related, map_graph
│   ├── analysis.go        # analyze, summarize, ask_question
│   ├── organization.go    # categorize_memory, list_domains
│   └── benchmark.go       # benchmark_run, benchmark_compare
└── responses/
    ├── memory.go           # MemoryFull, MemoryInfo, MemoryFullWithEmbed
    ├── search.go           # SearchResponse types
    ├── relationships.go    # RelationshipResponse types
    └── analysis.go         # AnalysisResponse types
```

### 4.2 Consolidate Memory Types (P0 #4)

**Current**: 4+ Memory types across packages:

1. `database.Memory` -- Core model (models.go:10)
2. `mcp.MemoryFull` -- MCP response (handlers.go:48)
3. `mcp.MemoryFullWithEmbed` -- With embedding (handlers.go:131)
4. `mcp.MemoryInfo` -- Simplified (handlers.go:286)
5. `api.MemoryResponse` + `api.MemoryData` -- REST API (handlers_memory.go)

**Solution**: Single canonical type with view converters:

```go
// database/models.go -- keep as-is (single source of truth)

// mcp/responses/memory.go -- thin view types
type MemoryView struct {
    *database.Memory
    EmbeddingArray []float64 `json:"embedding,omitempty"`
}

func ToView(m *database.Memory, embed bool) *MemoryView { ... }
func ToSummary(m *database.Memory) MemorySummary { ... }
```

### 4.3 Add MCP Request Timeout (P1 #12)

**Current**: REST API has timeouts (10s AI init, 30s search), but MCP server has none.

```go
// server.go -- in handleToolsCall
toolCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
defer cancel()

resultCh := make(chan interface{}, 1)
errCh := make(chan error, 1)

go func() {
    result, err := handler.Handle(toolCtx, params.Arguments)
    if err != nil {
        errCh <- err
        return
    }
    resultCh <- result
}()

select {
case result := <-resultCh:
    return successResponse(req.ID, formatted)
case err := <-errCh:
    return errorResponse(req.ID, InternalError, "Tool failed", err.Error())
case <-toolCtx.Done():
    return errorResponse(req.ID, InternalError, "Tool timeout", "exceeded 30s")
}
```

---

## 5. High Priority: AI Integration

### 5.1 Add Embedding Cache (P1 #9)

**Status**: Not implemented. Every semantic search calls Ollama (200-500ms per embedding).

```go
// ai/cache.go
type EmbeddingCache struct {
    cache *lru.Cache[string, cachedEmbedding]
    mu    sync.RWMutex
    ttl   time.Duration
}

func (c *EmbeddingCache) Get(text string) ([]float64, bool) { ... }
func (c *EmbeddingCache) Set(text string, embedding []float64) { ... }

// Usage in Manager.getEmbedding():
func (m *Manager) getEmbedding(ctx context.Context, text string) ([]float64, error) {
    if emb, ok := m.embeddingCache.Get(text); ok {
        return emb, nil
    }
    emb, err := m.ollama.GenerateEmbedding(ctx, text)
    if err != nil {
        return nil, err
    }
    m.embeddingCache.Set(text, emb)
    return emb, nil
}
```

### 5.2 Add Retry Logic (P1 #16)

**Status**: Not implemented. Ollama calls fail once and give up.

```go
// ai/retry.go
func WithRetry[T any](ctx context.Context, cfg RetryConfig, fn func() (T, error)) (T, error) {
    var result T
    var lastErr error
    wait := cfg.InitialWait

    for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
        result, lastErr = fn()
        if lastErr == nil {
            return result, nil
        }
        if errors.Is(lastErr, context.Canceled) || errors.Is(lastErr, context.DeadlineExceeded) {
            return result, lastErr
        }
        if attempt < cfg.MaxAttempts-1 {
            select {
            case <-ctx.Done():
                return result, ctx.Err()
            case <-time.After(wait):
            }
            wait = time.Duration(float64(wait) * cfg.Multiplier)
            if wait > cfg.MaxWait {
                wait = cfg.MaxWait
            }
        }
    }
    return result, fmt.Errorf("%w: %v", ErrMaxRetriesExceeded, lastErr)
}
```

### 5.3 Define Interfaces for Testing (P1 #15)

**Status**: Not implemented. Tests use real database and Ollama.

```go
// ai/interfaces.go
type EmbeddingGenerator interface {
    GenerateEmbedding(ctx context.Context, text string) ([]float64, error)
    IsAvailable() bool
    IsEnabled() bool
}

type VectorStore interface {
    Search(ctx context.Context, opts *SearchOptions) ([]*SearchResult, error)
    Upsert(ctx context.Context, id string, vector []float64, payload map[string]interface{}) error
    Delete(ctx context.Context, ids []string) error
    IsAvailable() bool
    IsEnabled() bool
}

type ChatProvider interface {
    Chat(ctx context.Context, messages []ChatMessage) (string, error)
    IsAvailable() bool
}

var _ EmbeddingGenerator = (*OllamaClient)(nil)
var _ VectorStore = (*QdrantClient)(nil)
```

---

## 6. High Priority: Search & Memory

### 6.1 Parallel Hybrid Search (P1 #10)

**Current**: `search/engine.go:325` runs keyword then semantic sequentially.

**Fix**: Launch both in goroutines, collect via channels:

```go
func (e *Engine) hybridSearch(opts *SearchOptions) ([]*SearchResult, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    type result struct {
        results []*SearchResult
        err     error
        source  string
    }
    ch := make(chan result, 2)

    go func() {
        r, err := e.keywordSearch(opts)
        ch <- result{r, err, "keyword"}
    }()
    go func() {
        if e.hasAI() {
            r, err := e.semanticSearch(opts)
            ch <- result{r, err, "semantic"}
        } else {
            ch <- result{nil, nil, "semantic"}
        }
    }()

    // Collect and merge...
}
```

### 6.2 Token-Based Chunking (P1 #13)

**Current**: `memory/chunker.go` (231 LOC) chunks by character count.

**Fix**: Use `tiktoken-go` for accurate token counting. See expanded document for full implementation.

---

## 7. Medium Priority: CLI & API

### 7.1 Restrict CORS (P2 #28)

**Current** (api/server.go:49): `AllowOrigins: ["*"]` with credentials enabled.

**Fix**: Make origins configurable:

```go
if cfg.RestAPI.CORS {
    origins := cfg.RestAPI.CORSOrigins
    if len(origins) == 0 {
        origins = []string{"http://localhost:*"}
    }
    s.engine.Use(cors.New(cors.Config{
        AllowOrigins: origins,
        // ...
    }))
}
```

### 7.2 Structured CLI Output (P2 #29)

Add `--json` flag to all commands for machine-readable output.

### 7.3 Query Caching (P2 #25)

```go
// database/cache.go
type QueryCache struct {
    cache *lru.Cache[string, cacheEntry]
    ttl   time.Duration
}

func (c *QueryCache) Get(key string) (interface{}, bool) { ... }
func (c *QueryCache) Set(key string, data interface{}) { ... }
func (c *QueryCache) Invalidate(pattern string) { ... }
```

---

## 8. Architecture Improvements

### 8.1 Introduce Error Types

```go
// errors/errors.go
var (
    ErrNotFound     = errors.New("not found")
    ErrDuplicate    = errors.New("duplicate entry")
    ErrInvalidInput = errors.New("invalid input")
    ErrServiceDown  = errors.New("service unavailable")
    ErrTimeout      = errors.New("operation timed out")
)

type MemoryError struct {
    Op       string
    MemoryID string
    Err      error
}

func (e *MemoryError) Error() string { ... }
func (e *MemoryError) Unwrap() error { return e.Err }
```

### 8.2 Add Metrics Collection

```go
// metrics/metrics.go
type Metrics struct {
    MemoriesCreated  atomic.Int64
    SearchesExecuted atomic.Int64
    SearchLatencyMs  atomic.Int64
    AICallsSuccess   atomic.Int64
    AICallsFailed    atomic.Int64
    ErrorCount       atomic.Int64
}

func (m *Metrics) Snapshot() map[string]int64 { ... }
```

---

## 9. Performance Optimization

### 9.1 Composite Indexes -- RESOLVED

All critical indexes now exist in `schema.go`:

```
memories: 9 indexes (session_id, domain, created_at, importance, access_scope, slug, parent, chunk_level)
relationships: 4 compound indexes (source+target, target+source, source+strength, target+strength)
```

### 9.2 Query Caching (P2 #25)

Not yet implemented. See Section 7.3.

---

## 10. Testing Improvements

### 10.1 Mock Interfaces (P1 #15)

```go
// internal/ai/mocks/mocks.go
type MockEmbeddingGenerator struct {
    GenerateFunc func(ctx context.Context, text string) ([]float64, error)
    Available    bool
}

func (m *MockEmbeddingGenerator) GenerateEmbedding(ctx context.Context, text string) ([]float64, error) {
    if m.GenerateFunc != nil {
        return m.GenerateFunc(ctx, text)
    }
    return make([]float64, 768), nil
}
```

### 10.2 Integration Tests

E2E testing framework already added (commit ad8b6b3). Expand with API full-cycle tests.

---

## 11. Security Hardening

### 11.1 Input Sanitization (Partially Implemented)

Content validation exists in MCP handlers. Still needed:

```go
// internal/validation/sanitize.go
const MaxContentLength = 1_000_000

func SanitizeContent(content string) (string, error) { ... }
func SanitizeTag(tag string) (string, error) { ... }
```

### 11.2 Error Message Sanitization (Resolved)

Error handling cleaned up in PR #66. Internal errors no longer leak via API responses.

---

## 12. Code Quality & Maintainability

### 12.1 Add Linting Configuration (New Issue N1)

No `.golangci.yml` exists. CI uses default settings. Create:

```yaml
# .golangci.yml
linters:
  enable:
    - errcheck
    - govet
    - ineffassign
    - staticcheck
    - unused
    - gosimple
    - gosec
    - gofmt
    - goimports
    - misspell

linters-settings:
  errcheck:
    check-type-assertions: true
  govet:
    check-shadowing: true
  misspell:
    locale: US

issues:
  max-issues-per-linter: 0
  max-same-issues: 0
```

### 12.2 Remove Dead Code (New Issue N2)

Old `GetGraph()` at operations.go:573 is superseded by `GetGraphOptimized()` at operations.go:674. Remove the legacy version once all callers are migrated.

---

## 13. Implementation Roadmap

### Phase 1: Structural Cleanup (High Impact, Low Risk)

| Task | Files | Status |
|------|-------|--------|
| Split operations.go into 9 domain files | database/ | **TODO** |
| Split handlers.go into tool-specific files | mcp/ | **TODO** |
| Consolidate Memory types to 1 + views | mcp/, api/ | **TODO** |
| Add `.golangci.yml` | project root | **TODO** |
| Remove legacy `GetGraph()` | operations.go | **TODO** |

### Phase 2: Reliability & Performance

| Task | Files | Status |
|------|-------|--------|
| Add MCP request timeouts | mcp/server.go | **TODO** |
| Add embedding cache | ai/cache.go (new) | **TODO** |
| Parallelize hybrid search | search/engine.go | **TODO** |
| Add retry logic for AI | ai/retry.go (new) | **TODO** |
| Add batch operations | database/batch_ops.go (new) | **TODO** |

### Phase 3: Testability & Quality

| Task | Files | Status |
|------|-------|--------|
| Define AI interfaces | ai/interfaces.go (new) | **TODO** |
| Create mock implementations | ai/mocks/ (new) | **TODO** |
| Add down migrations | database/migrations.go | **TODO** |
| Token-based chunking | memory/chunker.go | **TODO** |
| Restrict CORS origins | api/server.go, config | **TODO** |

### Phase 4: Architecture

| Task | Files | Status |
|------|-------|--------|
| Introduce error types package | errors/ (new) | **TODO** |
| Add metrics collection | metrics/ (new) | **TODO** |
| Input sanitization package | validation/ (new) | **TODO** |
| Query caching layer | database/cache.go (new) | **TODO** |
| Structured CLI output | cmd/*.go | **TODO** |

### Already Completed

| Task | Commit/PR |
|------|-----------|
| N+1 graph query fix (recursive CTE) | Pre-existing |
| Rate limiting package | Pre-existing |
| Composite database indexes | Pre-existing |
| Health endpoint | Pre-existing |
| Pagination | Pre-existing |
| FTS5 query escaping | Pre-existing |
| Graceful shutdown | Pre-existing |
| CORS configuration | Pre-existing |
| Search result merge/dedup | Pre-existing |
| golangci-lint cleanup (60+ errors) | PR #66 |
| CI pipeline optimization | PR #65 |
| E2E testing framework | Commit ad8b6b3 |

---

## 14. Conclusion

Of the original 54 issues, **18 have been resolved** (33%). The remaining 36 issues plus 3 new ones break down as:

| Priority | Remaining | Key Items |
|----------|-----------|-----------|
| P0 Critical | 3 | File splits, type consolidation |
| P1 High | 8 | Embedding cache, MCP timeout, retry logic, AI interfaces |
| P2 Medium | 16 | Query cache, CORS, structured output, migrations |
| P3 Low | 9 | Localization, token counting, duplicate detection |
| New | 3 | Lint config, dead code, ollama.go size |

### Success Metrics

| Metric | Original | Current | Target |
|--------|----------|---------|--------|
| Largest file | 1,687 LOC | 1,240 LOC | <500 LOC |
| Graph query | 100ms+ (N+1) | 4-10ms (CTE) | **DONE** |
| Lint errors | ~60 | 0 | **DONE** |
| Rate limiting | None | Full package | **DONE** |
| Test coverage | ~60% | ~60% | >80% |
| Memory types | 4+ | 4+ | 1 + views |

### Next Steps

1. Begin Phase 1: Split `operations.go` and `handlers.go`
2. Add `.golangci.yml` for consistent linting
3. Create GitHub issues for remaining items
4. Phase 2: Embedding cache + MCP timeouts for immediate performance/reliability wins

---

*Master Refactoring Document*
*MycelicMemory v1.2.0*
*Updated: 2026-01-28*
*Audited against: commit a6b01db (main)*
