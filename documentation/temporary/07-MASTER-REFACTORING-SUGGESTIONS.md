# Master Refactoring Document

**MycelicMemory v1.2.2 | Comprehensive Refactoring Plan**

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
14. [Migration Guide](#14-migration-guide)
15. [Conclusion](#15-conclusion)

---

## 1. Executive Summary

This document consolidates findings from 5 comprehensive code reviews covering all major modules of MycelicMemory. It provides a prioritized list of 50+ refactoring suggestions organized by impact and effort.

### Key Findings

| Metric | Value |
|--------|-------|
| Total Issues Identified | 54 |
| Critical (Must Fix) | 8 |
| High Priority | 16 |
| Medium Priority | 20 |
| Low Priority | 10 |

### Top 5 Immediate Actions

1. **Split operations.go** (1,687 LOC) into domain-specific files
2. **Fix N+1 query** in graph traversal using recursive CTE
3. **Add embedding cache** for AI operations
4. **Consolidate Memory types** (4 duplicates)
5. **Add rate limiting** to REST API

### Estimated Effort

| Category | Effort |
|----------|--------|
| Critical fixes | 2-3 days |
| High priority | 1 week |
| Medium priority | 2 weeks |
| Architecture improvements | 1 month |

---

## 2. Priority Matrix

### Critical (P0) - Must Fix

| # | Issue | Location | Impact | Effort |
|---|-------|----------|--------|--------|
| 1 | N+1 queries in graph | operations.go:920 | Performance degradation | Medium |
| 2 | handlers.go too large | mcp/handlers.go | Unmaintainable | Medium |
| 3 | operations.go too large | database/operations.go | Unmaintainable | Medium |
| 4 | 4 duplicate Memory types | mcp/handlers.go | Confusion, bugs | Low |
| 5 | Tags as JSON string | database/models.go | Query inefficiency | High |
| 6 | No rate limiting | api/server.go | Security risk | Low |
| 7 | Fire-and-forget indexing | memory/service.go | Silent failures | Low |
| 8 | Binding error ignored | api/handlers.go | Bug source | Low |

### High Priority (P1)

| # | Issue | Location | Impact | Effort |
|---|-------|----------|--------|--------|
| 9 | No embedding cache | ai/manager.go | Performance | Medium |
| 10 | Sequential hybrid search | search/engine.go | Performance | Medium |
| 11 | No batch operations | database/operations.go | Performance | Medium |
| 12 | No request timeout | mcp/server.go | Reliability | Low |
| 13 | Character-based chunking | memory/chunker.go | Accuracy | Medium |
| 14 | Tool definitions inline | mcp/server.go | Maintainability | Medium |
| 15 | Missing interfaces | ai/manager.go | Testability | Medium |
| 16 | No retry logic | ai/ollama.go | Reliability | Low |
| 17 | No status caching | ai/manager.go | Performance | Low |
| 18 | N+1 in semanticSearch | search/engine.go | Performance | Medium |
| 19 | Error message leakage | api/handlers.go | Security | Low |
| 20 | No pagination | api/handlers.go | Scalability | Medium |
| 21 | No merge sorting | search/engine.go | UX | Low |
| 22 | Session detection cache | memory/session.go | Performance | Low |
| 23 | No content length limit | memory/service.go | Security | Low |
| 24 | Composite index missing | database/schema.go | Performance | Low |

### Medium Priority (P2)

| # | Issue | Location | Impact | Effort |
|---|-------|----------|--------|--------|
| 25 | Query caching missing | database/operations.go | Performance | Medium |
| 26 | FTS5 escaping issues | database/operations.go | Search accuracy | Medium |
| 27 | No health endpoint | api/server.go | Operations | Low |
| 28 | CORS too permissive | api/server.go | Security | Low |
| 29 | No structured output | cmd/*.go | Automation | Medium |
| 30 | Two constructors | search/engine.go | Consistency | Low |
| 31 | Manual field mapping | mcp/handlers.go | Error-prone | Medium |
| 32 | Schema as strings | database/schema.go | Maintainability | Medium |
| 33 | No connection retry | database/database.go | Reliability | Low |
| 34 | Generic error messages | database/operations.go | Debugging | Low |
| 35 | No down migrations | database/migrations.go | Operations | Medium |
| 36 | Global variables | cmd/root.go | Testability | Medium |
| 37 | Version hardcoded | cmd/root.go | Maintenance | Low |
| 38 | No AI availability check | cmd/cmd_analyze.go | UX | Low |
| 39 | O(n^2) relationship | ai/manager.go | Scalability | High |
| 40 | No streaming | ai/ollama.go | UX | Medium |
| 41 | Verbose filter building | ai/manager.go | Readability | Low |
| 42 | Embedding not persisted | ai/manager.go | Data loss | Low |
| 43 | No change detection | memory/service.go | Efficiency | Low |
| 44 | Post-query filtering | search/engine.go | Performance | Low |

### Low Priority (P3)

| # | Issue | Location | Impact | Effort |
|---|-------|----------|--------|--------|
| 45 | No localization | mcp/formatter.go | International | High |
| 46 | ID type validation | mcp/types.go | Spec compliance | Low |
| 47 | 8-byte hash | memory/session.go | Collision risk | Low |
| 48 | No token counting | ai/ollama.go | Cost tracking | Medium |
| 49 | Hardcoded split logic | memory/chunker.go | Flexibility | Medium |
| 50 | Manual field mapping | api/response.go | Maintenance | Low |
| 51 | No progress indicator | cmd/*.go | UX | Low |
| 52 | No PID file | cmd/cmd_service.go | Operations | Low |
| 53 | No graceful shutdown | cmd/root.go | Reliability | Low |
| 54 | No duplicate detection | memory/service.go | Data quality | Medium |

---

## 3. Critical Refactoring: Database Layer

### 3.1 Split operations.go

**Current State**: 1,687 LOC in single file

**Target Structure**:

```
internal/database/
├── database.go          # Connection management (existing)
├── models.go            # Domain structures (existing)
├── schema.go            # Table definitions (existing)
├── migrations.go        # Schema upgrades (existing)
├── memory_ops.go        # NEW: Memory CRUD
├── relationship_ops.go  # NEW: Relationship CRUD
├── category_ops.go      # NEW: Category operations
├── domain_ops.go        # NEW: Domain operations
├── session_ops.go       # NEW: Session operations
├── search_ops.go        # NEW: Search operations
├── graph_ops.go         # NEW: Graph traversal
├── benchmark_ops.go     # NEW: Benchmark operations
└── stats_ops.go         # NEW: Statistics
```

**Implementation**:

```go
// memory_ops.go
package database

// Memory CRUD operations

func (d *Database) CreateMemory(m *Memory) error { ... }
func (d *Database) GetMemory(id string) (*Memory, error) { ... }
func (d *Database) UpdateMemory(m *Memory) error { ... }
func (d *Database) DeleteMemory(id string) error { ... }
func (d *Database) ListMemories(filters *MemoryFilters) ([]*Memory, error) { ... }
func (d *Database) GetMemoriesByIDs(ids []string) (map[string]*Memory, error) { ... }
```

### 3.2 Fix N+1 Query in Graph Traversal

**Current Code** (operations.go:920):

```go
func (d *Database) GetGraph(rootID string, depth int) (*Graph, error) {
    visited := make(map[string]int)
    queue := []string{rootID}

    for len(queue) > 0 {
        current := queue[0]
        queue = queue[1:]

        // N+1: One query per node!
        relationships, _ := d.GetRelationshipsForMemory(current)
        for _, rel := range relationships {
            // Add to queue...
        }
    }
}
```

**Refactored Code**:

```go
func (d *Database) GetGraph(rootID string, depth int) (*Graph, error) {
    const graphQuery = `
        WITH RECURSIVE graph_traverse(
            id, content, importance, domain, tags, session_id,
            distance, path
        ) AS (
            -- Base case: start node
            SELECT
                m.id, m.content, m.importance, m.domain, m.tags, m.session_id,
                0 as distance,
                m.id as path
            FROM memories m
            WHERE m.id = ?

            UNION ALL

            -- Recursive case: traverse relationships
            SELECT
                m.id, m.content, m.importance, m.domain, m.tags, m.session_id,
                g.distance + 1,
                g.path || ',' || m.id
            FROM memories m
            INNER JOIN memory_relationships r ON (
                (r.source_memory_id = g.id AND r.target_memory_id = m.id) OR
                (r.target_memory_id = g.id AND r.source_memory_id = m.id)
            )
            INNER JOIN graph_traverse g ON 1=1
            WHERE g.distance < ?
            AND g.path NOT LIKE '%' || m.id || '%'  -- Prevent cycles
        )
        SELECT DISTINCT
            id, content, importance, domain, tags, session_id, distance
        FROM graph_traverse
        ORDER BY distance, id;
    `

    rows, err := d.db.QueryContext(ctx, graphQuery, rootID, depth)
    if err != nil {
        return nil, fmt.Errorf("graph query failed: %w", err)
    }
    defer rows.Close()

    // Build graph from single query result
    graph := &Graph{
        Nodes: make([]NodeInfo, 0),
        Edges: make([]EdgeInfo, 0),
    }

    for rows.Next() {
        var node NodeInfo
        var tagsJSON string
        err := rows.Scan(&node.ID, &node.Content, &node.Importance,
            &node.Domain, &tagsJSON, &node.SessionID, &node.Distance)
        if err != nil {
            continue
        }
        json.Unmarshal([]byte(tagsJSON), &node.Tags)
        graph.Nodes = append(graph.Nodes, node)
    }

    // Get edges in single query
    graph.Edges, _ = d.getGraphEdges(rootID, depth)

    return graph, nil
}
```

### 3.3 Normalize Tags Table

**Current**: Tags stored as JSON array in `memories.tags` column

**New Schema**:

```sql
-- tags.sql
CREATE TABLE IF NOT EXISTS tags (
    id TEXT PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS memory_tags (
    memory_id TEXT NOT NULL REFERENCES memories(id) ON DELETE CASCADE,
    tag_id TEXT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (memory_id, tag_id)
);

CREATE INDEX idx_memory_tags_tag ON memory_tags(tag_id);
CREATE INDEX idx_memory_tags_memory ON memory_tags(memory_id);
```

**Migration**:

```go
func (d *Database) migrateTagsToNormalized() error {
    // 1. Create new tables
    d.db.Exec(tagsSchema)

    // 2. Extract unique tags from existing memories
    rows, _ := d.db.Query("SELECT DISTINCT tags FROM memories WHERE tags != '[]'")
    for rows.Next() {
        var tagsJSON string
        rows.Scan(&tagsJSON)
        var tags []string
        json.Unmarshal([]byte(tagsJSON), &tags)
        for _, tag := range tags {
            d.db.Exec("INSERT OR IGNORE INTO tags (id, name) VALUES (?, ?)",
                uuid.New().String(), strings.ToLower(tag))
        }
    }

    // 3. Create memory_tags relationships
    memRows, _ := d.db.Query("SELECT id, tags FROM memories WHERE tags != '[]'")
    for memRows.Next() {
        var memID, tagsJSON string
        memRows.Scan(&memID, &tagsJSON)
        var tags []string
        json.Unmarshal([]byte(tagsJSON), &tags)
        for _, tag := range tags {
            var tagID string
            d.db.QueryRow("SELECT id FROM tags WHERE name = ?", strings.ToLower(tag)).Scan(&tagID)
            d.db.Exec("INSERT INTO memory_tags (memory_id, tag_id) VALUES (?, ?)", memID, tagID)
        }
    }

    // 4. Update Memory model and operations to use new structure
    return nil
}
```

### 3.4 Add Batch Operations

```go
// batch_ops.go
package database

// BatchCreateMemories creates multiple memories in a single transaction
func (d *Database) BatchCreateMemories(memories []*Memory) error {
    d.mu.Lock()
    defer d.mu.Unlock()

    tx, err := d.db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()

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
        _, err := stmt.Exec(m.ID, m.Content, m.Importance, string(tagsJSON),
            m.Domain, m.SessionID, m.CreatedAt, m.UpdatedAt)
        if err != nil {
            return fmt.Errorf("failed to create memory %s: %w", m.ID, err)
        }
    }

    return tx.Commit()
}

// GetMemoriesByIDs fetches multiple memories in single query
func (d *Database) GetMemoriesByIDs(ids []string) (map[string]*Memory, error) {
    if len(ids) == 0 {
        return make(map[string]*Memory), nil
    }

    d.mu.RLock()
    defer d.mu.RUnlock()

    // Build placeholders
    placeholders := make([]string, len(ids))
    args := make([]interface{}, len(ids))
    for i, id := range ids {
        placeholders[i] = "?"
        args[i] = id
    }

    query := fmt.Sprintf(`
        SELECT id, content, source, importance, tags, session_id, domain,
               embedding, created_at, updated_at
        FROM memories
        WHERE id IN (%s)
    `, strings.Join(placeholders, ","))

    rows, err := d.db.Query(query, args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    result := make(map[string]*Memory)
    for rows.Next() {
        m := &Memory{}
        var tagsJSON string
        err := rows.Scan(&m.ID, &m.Content, &m.Source, &m.Importance,
            &tagsJSON, &m.SessionID, &m.Domain, &m.Embedding,
            &m.CreatedAt, &m.UpdatedAt)
        if err != nil {
            continue
        }
        json.Unmarshal([]byte(tagsJSON), &m.Tags)
        result[m.ID] = m
    }

    return result, nil
}
```

---

## 4. Critical Refactoring: MCP Server

### 4.1 Split handlers.go

**Current State**: 1,534 LOC in single file with 30+ response types

**Target Structure**:

```
internal/mcp/
├── server.go              # Core server (existing, trimmed)
├── types.go               # Protocol types (existing)
├── formatter.go           # Response formatting (existing)
├── tools/
│   ├── registry.go        # Tool registration
│   ├── definitions.go     # Tool schemas
│   ├── memory.go          # Memory tools
│   ├── search.go          # Search tools
│   ├── relationships.go   # Relationship tools
│   ├── analysis.go        # Analysis tools
│   ├── organization.go    # Categories/domains
│   └── benchmark.go       # Benchmark tools
└── responses/
    ├── memory.go          # Memory response types
    ├── search.go          # Search response types
    ├── relationships.go   # Relationship response types
    └── analysis.go        # Analysis response types
```

**Tool Handler Interface**:

```go
// tools/handler.go
package tools

// Handler defines the interface for tool handlers
type Handler interface {
    // GetDefinition returns the tool's JSON Schema definition
    GetDefinition() Tool

    // Handle executes the tool with given arguments
    Handle(ctx context.Context, args json.RawMessage) (interface{}, error)
}

// Registry manages tool registration
type Registry struct {
    handlers map[string]Handler
    mu       sync.RWMutex
}

func NewRegistry() *Registry {
    return &Registry{
        handlers: make(map[string]Handler),
    }
}

func (r *Registry) Register(name string, handler Handler) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.handlers[name] = handler
}

func (r *Registry) Get(name string) (Handler, bool) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    h, ok := r.handlers[name]
    return h, ok
}

func (r *Registry) GetDefinitions() []Tool {
    r.mu.RLock()
    defer r.mu.RUnlock()

    tools := make([]Tool, 0, len(r.handlers))
    for _, h := range r.handlers {
        tools = append(tools, h.GetDefinition())
    }
    return tools
}
```

**Example Handler**:

```go
// tools/memory.go
package tools

type StoreMemoryHandler struct {
    memSvc *memory.Service
    log    *logging.Logger
}

func NewStoreMemoryHandler(memSvc *memory.Service) *StoreMemoryHandler {
    return &StoreMemoryHandler{
        memSvc: memSvc,
        log:    logging.GetLogger("mcp.tools.store_memory"),
    }
}

func (h *StoreMemoryHandler) GetDefinition() Tool {
    return Tool{
        Name:        "store_memory",
        Description: "Store a new memory with contextual information",
        InputSchema: InputSchema{
            Type: "object",
            Properties: map[string]Property{
                "content": {
                    Type:        "string",
                    Description: "The memory content to store",
                },
                "importance": {
                    Type:        "integer",
                    Description: "Importance level (1-10)",
                    Default:     5,
                    Minimum:     ptr(1.0),
                    Maximum:     ptr(10.0),
                },
                // ...
            },
            Required: []string{"content"},
        },
    }
}

func (h *StoreMemoryHandler) Handle(ctx context.Context, args json.RawMessage) (interface{}, error) {
    var params StoreMemoryParams
    if err := json.Unmarshal(args, &params); err != nil {
        return nil, fmt.Errorf("invalid parameters: %w", err)
    }

    if params.Content == "" {
        return nil, fmt.Errorf("content is required")
    }

    importance := params.Importance
    if importance == 0 {
        importance = 5
    }

    result, err := h.memSvc.Store(&memory.StoreOptions{
        Content:    params.Content,
        Importance: importance,
        Tags:       params.Tags,
        Domain:     params.Domain,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to store memory: %w", err)
    }

    return &responses.StoreMemoryResponse{
        Success:   true,
        MemoryID:  result.Memory.ID,
        Content:   result.Memory.Content,
        CreatedAt: result.Memory.CreatedAt.Format(time.RFC3339),
        SessionID: result.Memory.SessionID,
    }, nil
}
```

### 4.2 Consolidate Memory Types

**Current**: 4 different Memory types exist:

1. `database.Memory` - Database model
2. `mcp.MemoryFull` - MCP response
3. `mcp.MemoryInfo` - Simplified response
4. `mcp.MemoryFullWithEmbed` - With embedding array

**Solution**: Single source of truth with views

```go
// database/models.go (Single canonical type)
type Memory struct {
    ID           string    `json:"id"`
    Content      string    `json:"content"`
    Importance   int       `json:"importance"`
    Tags         []string  `json:"tags"`
    SessionID    string    `json:"session_id"`
    Domain       string    `json:"domain,omitempty"`
    Source       string    `json:"source,omitempty"`
    Embedding    []byte    `json:"-"`  // Excluded from JSON by default
    CreatedAt    time.Time `json:"created_at"`
    UpdatedAt    time.Time `json:"updated_at"`
    // ...
}

// mcp/responses/memory.go (View types using embedding)
type MemoryView struct {
    *database.Memory
    EmbeddingArray []float64 `json:"embedding,omitempty"`
}

func ToMemoryView(m *database.Memory, includeEmbedding bool) *MemoryView {
    v := &MemoryView{Memory: m}
    if includeEmbedding && len(m.Embedding) > 0 {
        json.Unmarshal(m.Embedding, &v.EmbeddingArray)
    }
    return v
}

// Simplified view for lists
type MemorySummary struct {
    ID         string `json:"id"`
    Content    string `json:"content"`
    Importance int    `json:"importance"`
}

func ToMemorySummary(m *database.Memory) MemorySummary {
    return MemorySummary{
        ID:         m.ID,
        Content:    truncate(m.Content, 200),
        Importance: m.Importance,
    }
}
```

### 4.3 Add Request Timeout

```go
// server.go
func (s *Server) handleToolsCall(ctx context.Context, req Request) *Response {
    var params CallToolParams
    if err := json.Unmarshal(req.Params, &params); err != nil {
        return errorResponse(req.ID, InvalidParams, "Invalid params", err.Error())
    }

    // Add timeout for tool execution
    toolCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()

    startTime := time.Now()

    // Get handler from registry
    handler, ok := s.registry.Get(params.Name)
    if !ok {
        return errorResponse(req.ID, MethodNotFound, "Unknown tool", params.Name)
    }

    // Execute with timeout
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
        duration := time.Since(startTime)
        formatted := s.formatter.FormatToolResponse(params.Name, result, duration)
        return successResponse(req.ID, formatted)

    case err := <-errCh:
        return errorResponse(req.ID, InternalError, "Tool failed", err.Error())

    case <-toolCtx.Done():
        return errorResponse(req.ID, InternalError, "Tool timeout", "execution exceeded 30s")
    }
}
```

---

## 5. High Priority: AI Integration

### 5.1 Add Embedding Cache

```go
// ai/cache.go
package ai

import (
    "crypto/sha256"
    "encoding/hex"
    "sync"
    "time"

    lru "github.com/hashicorp/golang-lru/v2"
)

type EmbeddingCache struct {
    cache *lru.Cache[string, cachedEmbedding]
    mu    sync.RWMutex
    ttl   time.Duration
}

type cachedEmbedding struct {
    embedding []float64
    cachedAt  time.Time
}

func NewEmbeddingCache(size int, ttl time.Duration) *EmbeddingCache {
    cache, _ := lru.New[string, cachedEmbedding](size)
    return &EmbeddingCache{
        cache: cache,
        ttl:   ttl,
    }
}

func (c *EmbeddingCache) key(text string) string {
    // Normalize and hash
    normalized := strings.ToLower(strings.TrimSpace(text))
    hash := sha256.Sum256([]byte(normalized))
    return hex.EncodeToString(hash[:])
}

func (c *EmbeddingCache) Get(text string) ([]float64, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()

    k := c.key(text)
    if cached, ok := c.cache.Get(k); ok {
        if time.Since(cached.cachedAt) < c.ttl {
            return cached.embedding, true
        }
        c.cache.Remove(k)
    }
    return nil, false
}

func (c *EmbeddingCache) Set(text string, embedding []float64) {
    c.mu.Lock()
    defer c.mu.Unlock()

    k := c.key(text)
    c.cache.Add(k, cachedEmbedding{
        embedding: embedding,
        cachedAt:  time.Now(),
    })
}

// Usage in Manager
type Manager struct {
    // ...
    embeddingCache *EmbeddingCache
}

func NewManager(db *database.Database, cfg *config.Config) *Manager {
    return &Manager{
        // ...
        embeddingCache: NewEmbeddingCache(1000, 24*time.Hour),
    }
}

func (m *Manager) getEmbedding(ctx context.Context, text string) ([]float64, error) {
    // Check cache first
    if emb, ok := m.embeddingCache.Get(text); ok {
        return emb, nil
    }

    // Generate new embedding
    emb, err := m.ollama.GenerateEmbedding(ctx, text)
    if err != nil {
        return nil, err
    }

    // Cache for future use
    m.embeddingCache.Set(text, emb)
    return emb, nil
}
```

### 5.2 Add Retry Logic

```go
// ai/retry.go
package ai

import (
    "context"
    "errors"
    "time"
)

var (
    ErrMaxRetriesExceeded = errors.New("max retries exceeded")
)

type RetryConfig struct {
    MaxAttempts int
    InitialWait time.Duration
    MaxWait     time.Duration
    Multiplier  float64
}

var DefaultRetryConfig = RetryConfig{
    MaxAttempts: 3,
    InitialWait: 1 * time.Second,
    MaxWait:     10 * time.Second,
    Multiplier:  2.0,
}

func WithRetry[T any](ctx context.Context, cfg RetryConfig, fn func() (T, error)) (T, error) {
    var result T
    var lastErr error

    wait := cfg.InitialWait

    for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
        result, lastErr = fn()
        if lastErr == nil {
            return result, nil
        }

        // Don't retry certain errors
        if errors.Is(lastErr, context.Canceled) ||
            errors.Is(lastErr, context.DeadlineExceeded) {
            return result, lastErr
        }

        // Wait before retry (unless last attempt)
        if attempt < cfg.MaxAttempts-1 {
            select {
            case <-ctx.Done():
                return result, ctx.Err()
            case <-time.After(wait):
            }

            // Exponential backoff
            wait = time.Duration(float64(wait) * cfg.Multiplier)
            if wait > cfg.MaxWait {
                wait = cfg.MaxWait
            }
        }
    }

    return result, fmt.Errorf("%w: %v", ErrMaxRetriesExceeded, lastErr)
}

// Usage in OllamaClient
func (c *OllamaClient) GenerateEmbedding(ctx context.Context, text string) ([]float64, error) {
    return WithRetry(ctx, DefaultRetryConfig, func() ([]float64, error) {
        return c.doGenerateEmbedding(ctx, text)
    })
}
```

### 5.3 Define Interfaces for Testing

```go
// ai/interfaces.go
package ai

import "context"

// EmbeddingGenerator generates embeddings from text
type EmbeddingGenerator interface {
    GenerateEmbedding(ctx context.Context, text string) ([]float64, error)
    IsAvailable() bool
    IsEnabled() bool
}

// VectorStore stores and searches vectors
type VectorStore interface {
    Search(ctx context.Context, opts *SearchOptions) ([]*SearchResult, error)
    Upsert(ctx context.Context, id string, vector []float64, payload map[string]interface{}) error
    Delete(ctx context.Context, ids []string) error
    IsAvailable() bool
    IsEnabled() bool
}

// ChatProvider provides chat completions
type ChatProvider interface {
    Chat(ctx context.Context, messages []ChatMessage) (string, error)
    IsAvailable() bool
}

// Analyzer performs AI analysis
type Analyzer interface {
    AnswerQuestion(ctx context.Context, question string, context []string) (*AnswerResult, error)
    Summarize(ctx context.Context, contents []string, timeframe string) (*SummaryResult, error)
    AnalyzePatterns(ctx context.Context, contents []string, query string) (*PatternsResult, error)
}

// Ensure implementations satisfy interfaces
var _ EmbeddingGenerator = (*OllamaClient)(nil)
var _ VectorStore = (*QdrantClient)(nil)
var _ ChatProvider = (*OllamaClient)(nil)
```

---

## 6. High Priority: Search & Memory

### 6.1 Parallel Hybrid Search

```go
// search/engine.go
func (e *Engine) hybridSearch(opts *SearchOptions) ([]*SearchResult, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    type searchResult struct {
        results []*SearchResult
        err     error
        source  string
    }

    resultCh := make(chan searchResult, 2)

    // Launch keyword search
    go func() {
        results, err := e.keywordSearch(opts)
        resultCh <- searchResult{results, err, "keyword"}
    }()

    // Launch semantic search if AI available
    go func() {
        if e.hasAI() {
            results, err := e.semanticSearch(opts)
            resultCh <- searchResult{results, err, "semantic"}
        } else {
            resultCh <- searchResult{nil, nil, "semantic"}
        }
    }()

    // Collect results
    var keywordResults, semanticResults []*SearchResult
    var keywordErr, semanticErr error

    for i := 0; i < 2; i++ {
        select {
        case r := <-resultCh:
            switch r.source {
            case "keyword":
                keywordResults, keywordErr = r.results, r.err
            case "semantic":
                semanticResults, semanticErr = r.results, r.err
            }
        case <-ctx.Done():
            return nil, ctx.Err()
        }
    }

    // Return best available results
    if keywordErr != nil && semanticErr != nil {
        return nil, keywordErr
    }

    if semanticErr != nil || len(semanticResults) == 0 {
        return keywordResults, nil
    }

    if keywordErr != nil || len(keywordResults) == 0 {
        return semanticResults, nil
    }

    // Merge and sort
    return e.mergeAndSortResults(keywordResults, semanticResults, opts.Limit)
}

func (e *Engine) mergeAndSortResults(keyword, semantic []*SearchResult, limit int) []*SearchResult {
    seen := make(map[string]*SearchResult)

    // Weight: 40% keyword, 60% semantic
    for _, r := range keyword {
        r.Relevance *= 0.4
        r.MatchType = "keyword"
        seen[r.Memory.ID] = r
    }

    for _, r := range semantic {
        r.Relevance *= 0.6
        if existing, ok := seen[r.Memory.ID]; ok {
            // Matched both - boost score
            existing.Relevance += r.Relevance
            existing.MatchType = "hybrid"
        } else {
            r.MatchType = "semantic"
            seen[r.Memory.ID] = r
        }
    }

    // Convert to slice and sort
    results := make([]*SearchResult, 0, len(seen))
    for _, r := range seen {
        results = append(results, r)
    }

    sort.Slice(results, func(i, j int) bool {
        return results[i].Relevance > results[j].Relevance
    })

    if len(results) > limit {
        results = results[:limit]
    }

    return results
}
```

### 6.2 Token-Based Chunking

```go
// memory/chunker.go
import "github.com/pkoukk/tiktoken-go"

type TokenChunker struct {
    encoding     *tiktoken.Tiktoken
    maxTokens    int
    overlapRatio float64
}

func NewTokenChunker(maxTokens int, overlapRatio float64) (*TokenChunker, error) {
    enc, err := tiktoken.GetEncoding("cl100k_base")
    if err != nil {
        return nil, fmt.Errorf("failed to get encoding: %w", err)
    }

    return &TokenChunker{
        encoding:     enc,
        maxTokens:    maxTokens,
        overlapRatio: overlapRatio,
    }, nil
}

func (c *TokenChunker) CountTokens(text string) int {
    return len(c.encoding.Encode(text, nil, nil))
}

func (c *TokenChunker) Chunk(content string) ([]*Chunk, error) {
    tokens := c.encoding.Encode(content, nil, nil)
    totalTokens := len(tokens)

    if totalTokens <= c.maxTokens {
        return []*Chunk{{
            Content:     content,
            Level:       0,
            Index:       0,
            TokenCount:  totalTokens,
        }}, nil
    }

    overlap := int(float64(c.maxTokens) * c.overlapRatio)
    step := c.maxTokens - overlap

    var chunks []*Chunk
    for i := 0; i < totalTokens; i += step {
        end := min(i+c.maxTokens, totalTokens)
        chunkTokens := tokens[i:end]
        chunkText := c.encoding.Decode(chunkTokens)

        chunks = append(chunks, &Chunk{
            Content:    chunkText,
            Level:      1,
            Index:      len(chunks),
            TokenCount: len(chunkTokens),
            StartToken: i,
            EndToken:   end,
        })

        if end >= totalTokens {
            break
        }
    }

    return chunks, nil
}
```

---

## 7. Medium Priority: CLI & API

### 7.1 Add Rate Limiting

```go
// api/middleware.go
import (
    "net/http"
    "sync"
    "time"

    "golang.org/x/time/rate"
)

type RateLimiter struct {
    limiters map[string]*rate.Limiter
    mu       sync.Mutex
    r        rate.Limit
    b        int
}

func NewRateLimiter(requestsPerSecond float64, burst int) *RateLimiter {
    return &RateLimiter{
        limiters: make(map[string]*rate.Limiter),
        r:        rate.Limit(requestsPerSecond),
        b:        burst,
    }
}

func (rl *RateLimiter) getLimiter(ip string) *rate.Limiter {
    rl.mu.Lock()
    defer rl.mu.Unlock()

    limiter, exists := rl.limiters[ip]
    if !exists {
        limiter = rate.NewLimiter(rl.r, rl.b)
        rl.limiters[ip] = limiter
    }

    return limiter
}

func (rl *RateLimiter) Middleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        ip := c.ClientIP()
        limiter := rl.getLimiter(ip)

        if !limiter.Allow() {
            c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
                "error": "Rate limit exceeded",
                "code":  "RATE_LIMITED",
            })
            return
        }

        c.Next()
    }
}

// Usage
func (s *Server) setupMiddleware() {
    limiter := NewRateLimiter(10.0, 20)  // 10 req/sec, burst 20
    s.engine.Use(limiter.Middleware())
}
```

### 7.2 Add Pagination

```go
// api/pagination.go
type PaginationParams struct {
    Page    int `form:"page" binding:"omitempty,min=1"`
    PerPage int `form:"per_page" binding:"omitempty,min=1,max=100"`
}

func (p *PaginationParams) Normalize() {
    if p.Page < 1 {
        p.Page = 1
    }
    if p.PerPage < 1 {
        p.PerPage = 20
    }
    if p.PerPage > 100 {
        p.PerPage = 100
    }
}

func (p *PaginationParams) Offset() int {
    return (p.Page - 1) * p.PerPage
}

type PaginatedResponse[T any] struct {
    Data       []T `json:"data"`
    Pagination struct {
        Page       int  `json:"page"`
        PerPage    int  `json:"per_page"`
        Total      int  `json:"total"`
        TotalPages int  `json:"total_pages"`
        HasNext    bool `json:"has_next"`
        HasPrev    bool `json:"has_prev"`
    } `json:"pagination"`
}

func NewPaginatedResponse[T any](data []T, page, perPage, total int) PaginatedResponse[T] {
    totalPages := (total + perPage - 1) / perPage
    return PaginatedResponse[T]{
        Data: data,
        Pagination: struct {
            Page       int  `json:"page"`
            PerPage    int  `json:"per_page"`
            Total      int  `json:"total"`
            TotalPages int  `json:"total_pages"`
            HasNext    bool `json:"has_next"`
            HasPrev    bool `json:"has_prev"`
        }{
            Page:       page,
            PerPage:    perPage,
            Total:      total,
            TotalPages: totalPages,
            HasNext:    page < totalPages,
            HasPrev:    page > 1,
        },
    }
}
```

### 7.3 Add Health Endpoint

```go
// api/handlers_health.go
func (s *Server) setupHealthRoutes() {
    s.engine.GET("/health", s.healthCheck)
    s.engine.GET("/ready", s.readinessCheck)
}

func (s *Server) healthCheck(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{
        "status":  "healthy",
        "version": version,
        "uptime":  time.Since(startTime).String(),
    })
}

func (s *Server) readinessCheck(c *gin.Context) {
    checks := map[string]bool{
        "database": false,
        "ollama":   false,
        "qdrant":   false,
    }

    // Check database
    if err := s.db.Ping(); err == nil {
        checks["database"] = true
    }

    // Check AI services
    if s.aiManager != nil {
        status := s.aiManager.GetStatus()
        checks["ollama"] = status.OllamaAvailable
        checks["qdrant"] = status.QdrantAvailable
    }

    // Must have database to be ready
    if !checks["database"] {
        c.JSON(http.StatusServiceUnavailable, gin.H{
            "status": "not ready",
            "checks": checks,
        })
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "status": "ready",
        "checks": checks,
    })
}
```

---

## 8. Architecture Improvements

### 8.1 Introduce Error Types

```go
// errors/errors.go
package errors

import (
    "errors"
    "fmt"
)

// Sentinel errors
var (
    ErrNotFound       = errors.New("not found")
    ErrDuplicate      = errors.New("duplicate entry")
    ErrInvalidInput   = errors.New("invalid input")
    ErrUnauthorized   = errors.New("unauthorized")
    ErrServiceDown    = errors.New("service unavailable")
    ErrTimeout        = errors.New("operation timed out")
)

// Typed errors with context
type MemoryError struct {
    Op       string // Operation
    MemoryID string
    Err      error
}

func (e *MemoryError) Error() string {
    if e.MemoryID != "" {
        return fmt.Sprintf("%s memory %s: %v", e.Op, e.MemoryID, e.Err)
    }
    return fmt.Sprintf("%s memory: %v", e.Op, e.Err)
}

func (e *MemoryError) Unwrap() error {
    return e.Err
}

// Helper constructors
func MemoryNotFound(id string) error {
    return &MemoryError{Op: "get", MemoryID: id, Err: ErrNotFound}
}

func MemoryCreateFailed(err error) error {
    return &MemoryError{Op: "create", Err: err}
}
```

### 8.2 Add Metrics Collection

```go
// metrics/metrics.go
package metrics

import (
    "sync/atomic"
    "time"
)

type Metrics struct {
    MemoriesCreated  atomic.Int64
    MemoriesDeleted  atomic.Int64
    SearchesExecuted atomic.Int64
    SearchLatencyMs  atomic.Int64
    AICallsSuccess   atomic.Int64
    AICallsFailed    atomic.Int64
    ErrorCount       atomic.Int64
}

var Default = &Metrics{}

func (m *Metrics) IncrementMemoriesCreated() {
    m.MemoriesCreated.Add(1)
}

func (m *Metrics) RecordSearchLatency(d time.Duration) {
    m.SearchesExecuted.Add(1)
    m.SearchLatencyMs.Add(d.Milliseconds())
}

func (m *Metrics) Snapshot() map[string]int64 {
    searches := m.SearchesExecuted.Load()
    avgLatency := int64(0)
    if searches > 0 {
        avgLatency = m.SearchLatencyMs.Load() / searches
    }

    return map[string]int64{
        "memories_created":   m.MemoriesCreated.Load(),
        "memories_deleted":   m.MemoriesDeleted.Load(),
        "searches_executed":  searches,
        "avg_search_latency": avgLatency,
        "ai_calls_success":   m.AICallsSuccess.Load(),
        "ai_calls_failed":    m.AICallsFailed.Load(),
        "error_count":        m.ErrorCount.Load(),
    }
}
```

---

## 9. Performance Optimization

### 9.1 Add Query Caching

```go
// database/cache.go
package database

import (
    "sync"
    "time"

    lru "github.com/hashicorp/golang-lru/v2"
)

type QueryCache struct {
    cache *lru.Cache[string, cacheEntry]
    ttl   time.Duration
    mu    sync.RWMutex
}

type cacheEntry struct {
    data      interface{}
    timestamp time.Time
}

func NewQueryCache(size int, ttl time.Duration) *QueryCache {
    cache, _ := lru.New[string, cacheEntry](size)
    return &QueryCache{
        cache: cache,
        ttl:   ttl,
    }
}

func (c *QueryCache) Get(key string) (interface{}, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()

    if entry, ok := c.cache.Get(key); ok {
        if time.Since(entry.timestamp) < c.ttl {
            return entry.data, true
        }
        c.cache.Remove(key)
    }
    return nil, false
}

func (c *QueryCache) Set(key string, data interface{}) {
    c.mu.Lock()
    defer c.mu.Unlock()

    c.cache.Add(key, cacheEntry{
        data:      data,
        timestamp: time.Now(),
    })
}

func (c *QueryCache) Invalidate(pattern string) {
    c.mu.Lock()
    defer c.mu.Unlock()

    // Remove all keys matching pattern
    keys := c.cache.Keys()
    for _, key := range keys {
        if strings.Contains(key, pattern) {
            c.cache.Remove(key)
        }
    }
}
```

### 9.2 Add Composite Indexes

```sql
-- schema.go additions
CREATE INDEX IF NOT EXISTS idx_memories_session_domain
ON memories(session_id, domain);

CREATE INDEX IF NOT EXISTS idx_memories_domain_importance
ON memories(domain, importance DESC);

CREATE INDEX IF NOT EXISTS idx_memories_created_importance
ON memories(created_at DESC, importance DESC);

CREATE INDEX IF NOT EXISTS idx_relationships_source_type
ON memory_relationships(source_memory_id, relationship_type);
```

---

## 10. Testing Improvements

### 10.1 Add Integration Tests

```go
// tests/integration/api_test.go
package integration

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestMemoryAPI_FullCycle(t *testing.T) {
    // Setup
    server := setupTestServer(t)
    defer server.Close()

    // Create memory
    createReq := map[string]interface{}{
        "content":    "Test memory content",
        "importance": 8,
        "tags":       []string{"test", "integration"},
    }
    body, _ := json.Marshal(createReq)

    resp, err := http.Post(server.URL+"/api/v1/memories", "application/json", bytes.NewReader(body))
    require.NoError(t, err)
    assert.Equal(t, http.StatusCreated, resp.StatusCode)

    var createResult struct {
        ID string `json:"id"`
    }
    json.NewDecoder(resp.Body).Decode(&createResult)
    resp.Body.Close()

    memoryID := createResult.ID
    assert.NotEmpty(t, memoryID)

    // Get memory
    resp, err = http.Get(server.URL + "/api/v1/memories/" + memoryID)
    require.NoError(t, err)
    assert.Equal(t, http.StatusOK, resp.StatusCode)
    resp.Body.Close()

    // Search memory
    searchReq := map[string]interface{}{
        "query": "Test memory",
        "limit": 10,
    }
    body, _ = json.Marshal(searchReq)
    resp, err = http.Post(server.URL+"/api/v1/memories/search", "application/json", bytes.NewReader(body))
    require.NoError(t, err)
    assert.Equal(t, http.StatusOK, resp.StatusCode)

    var searchResult struct {
        Count int `json:"count"`
    }
    json.NewDecoder(resp.Body).Decode(&searchResult)
    resp.Body.Close()
    assert.GreaterOrEqual(t, searchResult.Count, 1)

    // Delete memory
    req, _ := http.NewRequest(http.MethodDelete, server.URL+"/api/v1/memories/"+memoryID, nil)
    resp, err = http.DefaultClient.Do(req)
    require.NoError(t, err)
    assert.Equal(t, http.StatusOK, resp.StatusCode)
    resp.Body.Close()

    // Verify deleted
    resp, err = http.Get(server.URL + "/api/v1/memories/" + memoryID)
    require.NoError(t, err)
    assert.Equal(t, http.StatusNotFound, resp.StatusCode)
    resp.Body.Close()
}
```

### 10.2 Add Mock Interfaces

```go
// internal/ai/mocks/mocks.go
package mocks

import "context"

type MockEmbeddingGenerator struct {
    GenerateFunc func(ctx context.Context, text string) ([]float64, error)
    Available    bool
}

func (m *MockEmbeddingGenerator) GenerateEmbedding(ctx context.Context, text string) ([]float64, error) {
    if m.GenerateFunc != nil {
        return m.GenerateFunc(ctx, text)
    }
    // Return deterministic mock embedding
    return make([]float64, 768), nil
}

func (m *MockEmbeddingGenerator) IsAvailable() bool {
    return m.Available
}

func (m *MockEmbeddingGenerator) IsEnabled() bool {
    return true
}
```

---

## 11. Security Hardening

### 11.1 Input Sanitization

```go
// internal/validation/sanitize.go
package validation

import (
    "regexp"
    "strings"
    "unicode"
)

// MaxContentLength is the maximum allowed content length
const MaxContentLength = 1_000_000  // 1MB

// SanitizeContent cleans and validates memory content
func SanitizeContent(content string) (string, error) {
    // Trim whitespace
    content = strings.TrimSpace(content)

    if content == "" {
        return "", fmt.Errorf("content cannot be empty")
    }

    if len(content) > MaxContentLength {
        return "", fmt.Errorf("content exceeds maximum length of %d bytes", MaxContentLength)
    }

    // Remove null bytes
    content = strings.ReplaceAll(content, "\x00", "")

    // Normalize unicode
    content = strings.ToValidUTF8(content, "")

    return content, nil
}

// SanitizeTag cleans and validates a tag
func SanitizeTag(tag string) (string, error) {
    tag = strings.TrimSpace(strings.ToLower(tag))

    if tag == "" {
        return "", fmt.Errorf("tag cannot be empty")
    }

    if len(tag) > 100 {
        return "", fmt.Errorf("tag exceeds maximum length of 100 characters")
    }

    // Only allow alphanumeric, dash, underscore
    validTag := regexp.MustCompile(`^[a-z0-9_-]+$`)
    if !validTag.MatchString(tag) {
        return "", fmt.Errorf("tag contains invalid characters")
    }

    return tag, nil
}
```

### 11.2 Error Message Sanitization

```go
// api/errors.go
package api

// UserSafeError maps internal errors to user-safe messages
func UserSafeError(err error) (int, string, string) {
    // Map known errors
    switch {
    case errors.Is(err, database.ErrNotFound):
        return http.StatusNotFound, "NOT_FOUND", "The requested resource was not found"

    case errors.Is(err, database.ErrDuplicate):
        return http.StatusConflict, "DUPLICATE", "A resource with this identifier already exists"

    case errors.Is(err, validation.ErrInvalidInput):
        return http.StatusBadRequest, "INVALID_INPUT", "The request contains invalid data"

    case errors.Is(err, ai.ErrServiceUnavailable):
        return http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "AI services are currently unavailable"

    case errors.Is(err, context.DeadlineExceeded):
        return http.StatusGatewayTimeout, "TIMEOUT", "The operation timed out"

    default:
        // Log the actual error for debugging
        log.Error("internal error", "error", err)
        // Return generic message to user
        return http.StatusInternalServerError, "INTERNAL_ERROR", "An internal error occurred"
    }
}
```

---

## 12. Code Quality & Maintainability

### 12.1 Add Documentation Comments

```go
// Package memory provides the core business logic for memory operations.
//
// The memory package is responsible for:
//   - Storing and retrieving memories
//   - Validating memory content and metadata
//   - Chunking large content into smaller pieces
//   - Session detection and management
//   - Coordinating with AI services for indexing
//
// Example usage:
//
//     svc := memory.NewService(db, config)
//     result, err := svc.Store(&memory.StoreOptions{
//         Content:    "My memory content",
//         Importance: 8,
//         Tags:       []string{"important", "work"},
//     })
//
package memory

// Service handles all memory-related operations.
// It coordinates between the database layer and AI services
// to provide a unified interface for memory management.
type Service struct {
    // ...
}

// Store creates a new memory with the given options.
// It performs validation, session detection, and optional AI indexing.
//
// Returns a StoreResult containing the created memory, or an error
// if validation fails or the database operation fails.
func (s *Service) Store(opts *StoreOptions) (*StoreResult, error) {
    // ...
}
```

### 12.2 Add Linting Configuration

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
    - structcheck
    - varcheck
    - deadcode
    - typecheck
    - gosec
    - prealloc
    - gofmt
    - goimports
    - misspell

linters-settings:
  errcheck:
    check-type-assertions: true
    check-blank: true
  govet:
    check-shadowing: true
  gosec:
    excludes:
      - G104  # Unhandled errors
  misspell:
    locale: US

issues:
  max-issues-per-linter: 0
  max-same-issues: 0
```

---

## 13. Implementation Roadmap

### Phase 1: Critical Fixes (Week 1)

| Day | Task |
|-----|------|
| 1 | Split operations.go into domain files |
| 2 | Split handlers.go into tool-specific files |
| 3 | Implement recursive CTE for graph |
| 4 | Consolidate Memory types |
| 5 | Add rate limiting |

### Phase 2: High Priority (Week 2-3)

| Day | Task |
|-----|------|
| 1-2 | Implement embedding cache |
| 3-4 | Parallel hybrid search |
| 5-6 | Batch database operations |
| 7-8 | Add retry logic for AI |
| 9-10 | Token-based chunking |

### Phase 3: Medium Priority (Week 4-5)

| Day | Task |
|-----|------|
| 1-2 | Add pagination |
| 3-4 | Standardize error handling |
| 5-6 | Add health endpoints |
| 7-8 | Add query caching |
| 9-10 | Improve test coverage |

### Phase 4: Architecture (Week 6-8)

| Week | Task |
|------|------|
| 6 | Define interfaces, add DI |
| 7 | Add metrics collection |
| 8 | Security hardening |

---

## 14. Migration Guide

### Breaking Changes

1. **Database Schema**: Tags normalization requires migration
2. **API Responses**: Standardized error format
3. **Memory Types**: Single canonical type

### Migration Steps

```bash
# 1. Backup database
cp ~/.mycelicmemory/memories.db ~/.mycelicmemory/memories.db.backup

# 2. Run migrations
mycelicmemory migrate --to=v2

# 3. Verify
mycelicmemory doctor

# 4. Test
mycelicmemory search "test query"
```

### Rollback

```bash
# If issues occur
mycelicmemory migrate --down
cp ~/.mycelicmemory/memories.db.backup ~/.mycelicmemory/memories.db
```

---

## 15. Conclusion

This refactoring plan addresses 54 identified issues across 5 modules. The most impactful changes are:

1. **File Organization**: Splitting large files improves maintainability
2. **Performance**: Caching and batch operations reduce latency
3. **Reliability**: Retry logic and timeouts improve resilience
4. **Security**: Rate limiting and input validation protect the system
5. **Testing**: Interfaces enable better test coverage

The estimated total effort is 6-8 weeks for full implementation. However, critical fixes can be completed in 1 week, providing immediate value.

### Success Metrics

| Metric | Current | Target |
|--------|---------|--------|
| Largest file | 1,687 LOC | <500 LOC |
| Graph query latency | 100ms+ | <10ms |
| Search with AI | 500ms | <200ms |
| Test coverage | ~60% | >80% |
| Memory types | 4 | 1 |

### Next Steps

1. Review and prioritize with team
2. Create GitHub issues for tracking
3. Begin Phase 1 implementation
4. Schedule weekly progress reviews

---

*Master Refactoring Document*
*MycelicMemory v1.2.2*
*Generated: 2025-01-27*
*Total Pages: 20+*
