# Code Review #1: Database Layer

**Module**: `internal/database/`
**Files Analyzed**: `database.go`, `models.go`, `schema.go`, `operations.go`, `migrations.go`
**Total LOC**: ~3,800
**Review Date**: 2025-01-27

---

## Executive Summary

The database layer is the foundation of MycelicMemory, implementing SQLite persistence with FTS5 full-text search. This is the largest module in the codebase (~1,687 LOC in operations.go alone) and handles all data access patterns.

### Overall Assessment

| Aspect | Rating | Notes |
|--------|--------|-------|
| Code Quality | B+ | Solid but verbose |
| Error Handling | B | Consistent but could be more specific |
| Performance | A- | Good indexing, some optimization opportunities |
| Maintainability | B | Large file could be split |
| Test Coverage | A | Comprehensive tests |
| Documentation | A- | Good inline docs |

---

## File-by-File Analysis

### 1. `database.go` (276 LOC)

**Purpose**: Connection management, initialization, and lifecycle

**Strengths**:

1. **Thread-Safe Design**:
```go
type Database struct {
    db   *sql.DB
    path string
    mu   sync.RWMutex  // Proper synchronization
}
```

2. **Correct SQLite Configuration**:
```go
dsn := fmt.Sprintf("%s?_foreign_keys=on&_journal_mode=WAL", path)
db.SetMaxOpenConns(1)  // Respects SQLite single-writer
```

3. **Graceful Initialization**:
```go
func (d *Database) Initialize() error {
    d.mu.Lock()
    defer d.mu.Unlock()
    // Creates tables, initializes FTS5
}
```

**Issues Identified**:

1. **Hardcoded WAL mode** (Line ~65):
```go
// Current
dsn := fmt.Sprintf("%s?_foreign_keys=on&_journal_mode=WAL", path)

// Issue: No option to disable WAL for certain use cases
// Suggestion: Make journal mode configurable
```

2. **No Connection Retry Logic** (Line ~70):
```go
db, err := sql.Open("sqlite3", dsn)
if err != nil {
    return nil, fmt.Errorf("failed to open database: %w", err)
}

// Missing: Retry logic for locked database scenarios
```

3. **Missing Health Check**:
```go
// Add: Periodic health checks for connection validity
func (d *Database) HealthCheck() error {
    return d.db.Ping()
}
```

---

### 2. `models.go` (351 LOC)

**Purpose**: Domain structures matching the SQLite schema

**Strengths**:

1. **Well-Documented Structures**:
```go
// Memory represents a stored memory
// VERIFIED: Matches memories table schema from Local Memory v1.2.0
type Memory struct {
    ID           string    `json:"id"`
    Content      string    `json:"content"`
    Importance   int       `json:"importance"` // 1-10 scale
    // ...
}
```

2. **Comprehensive Field Coverage**:
- All 20+ fields mapped
- JSON tags for serialization
- Proper Go types (time.Time for timestamps)

3. **Hierarchical Support**:
```go
// Chunk hierarchy fields
ParentMemoryID string `json:"parent_memory_id"`
ChunkLevel     int    `json:"chunk_level"`  // 0=full, 1=para, 2=atomic
ChunkIndex     int    `json:"chunk_index"`  // Position
```

**Issues Identified**:

1. **Tags Stored as JSON String** (Line ~35):
```go
Tags       []string  `json:"tags"`  // JSON array: ["tag1", "tag2"]

// Issue: Tags are stored as JSON string in SQLite, not a normalized table
// This prevents efficient tag-based queries and indexing
// Current queries: WHERE tags LIKE '%"tag"%'
```

2. **Embedding as []byte** (Line ~45):
```go
Embedding  []byte    `json:"embedding"`

// Issue: No type safety for 768-dimension float64 array
// Suggestion: Create EmbeddingVector type with validation
```

3. **Missing Validation Methods**:
```go
// Add: Model-level validation
func (m *Memory) Validate() error {
    if m.Content == "" {
        return ErrEmptyContent
    }
    if m.Importance < 1 || m.Importance > 10 {
        return ErrInvalidImportance
    }
    return nil
}
```

4. **Relationship Model Missing Validation**:
```go
type Relationship struct {
    Strength float64 `json:"strength"`  // No validation at model level
    // Should: 0.0 <= Strength <= 1.0
}
```

---

### 3. `schema.go` (428 LOC)

**Purpose**: SQL schema definitions as string constants

**Strengths**:

1. **Comprehensive Schema**:
```go
const CoreSchema = `
-- 16+ tables with proper constraints
CREATE TABLE IF NOT EXISTS memories (
    id TEXT PRIMARY KEY,
    content TEXT NOT NULL,
    importance INTEGER DEFAULT 5,
    -- CHECK constraints for data integrity
);
`
```

2. **FTS5 Integration**:
```go
const FTS5Schema = `
CREATE VIRTUAL TABLE IF NOT EXISTS memories_fts USING fts5(
    id UNINDEXED,
    content,
    source,
    tags
);

-- Automatic sync triggers
CREATE TRIGGER IF NOT EXISTS memories_fts_insert ...
CREATE TRIGGER IF NOT EXISTS memories_fts_delete ...
CREATE TRIGGER IF NOT EXISTS memories_fts_update ...
`
```

3. **Index Strategy**:
```go
CREATE INDEX IF NOT EXISTS idx_memories_session_id ON memories(session_id);
CREATE INDEX IF NOT EXISTS idx_memories_domain ON memories(domain);
CREATE INDEX IF NOT EXISTS idx_memories_created_at ON memories(created_at);
CREATE INDEX IF NOT EXISTS idx_memories_importance ON memories(importance);
// 9 indexes on memories table
```

**Issues Identified**:

1. **Schema as String Constants** (Anti-pattern):
```go
const CoreSchema = `...`

// Issue: SQL embedded in Go code is hard to maintain
// Suggestion: Use .sql files with embed directive
//go:embed sql/schema.sql
var SchemaSQL string
```

2. **No Partial Indexes for Common Queries**:
```sql
-- Current: Full index on domain
CREATE INDEX IF NOT EXISTS idx_memories_domain ON memories(domain);

-- Better: Partial index for non-null domains
CREATE INDEX IF NOT EXISTS idx_memories_domain_partial
ON memories(domain) WHERE domain IS NOT NULL;
```

3. **Missing Composite Indexes**:
```sql
-- Missing: Composite index for common query patterns
CREATE INDEX IF NOT EXISTS idx_memories_session_domain
ON memories(session_id, domain);

CREATE INDEX IF NOT EXISTS idx_memories_domain_importance
ON memories(domain, importance DESC);
```

4. **FTS5 Sync Triggers Not Transactional**:
```sql
-- Current: Separate triggers
CREATE TRIGGER memories_fts_insert AFTER INSERT ON memories BEGIN
    INSERT INTO memories_fts(...) VALUES (...);
END;

-- Issue: If main insert succeeds but FTS insert fails, data inconsistent
-- This is mitigated by SQLite's transaction handling, but worth noting
```

---

### 4. `operations.go` (1,687 LOC)

**Purpose**: All CRUD operations and queries

**This is the largest and most critical file. Analysis broken by function group.**

#### 4.1 Memory Operations (Lines 1-400)

**CreateMemory** (Line ~50):
```go
func (d *Database) CreateMemory(m *Memory) error {
    d.mu.Lock()
    defer d.mu.Unlock()

    if m.ID == "" {
        m.ID = uuid.New().String()
    }

    _, err := d.db.Exec(`
        INSERT INTO memories (id, content, source, importance, ...)
        VALUES (?, ?, ?, ?, ...)
    `, m.ID, m.Content, m.Source, m.Importance, ...)

    return err
}
```

**Strengths**:
- Proper mutex usage
- Auto-generated UUIDs
- Parameterized queries

**Issues**:

1. **Tags Serialization Every Call**:
```go
tagsJSON, _ := json.Marshal(m.Tags)  // Line ~75

// Issue: Serialization on every insert
// Optimization: Cache serialized tags if unchanged
```

2. **No Batch Insert**:
```go
// Current: One insert at a time
func (d *Database) CreateMemory(m *Memory) error

// Missing: Batch insert for bulk operations
func (d *Database) CreateMemories(memories []*Memory) error {
    tx, _ := d.db.Begin()
    stmt, _ := tx.Prepare("INSERT INTO memories ...")
    for _, m := range memories {
        stmt.Exec(...)
    }
    return tx.Commit()
}
```

3. **Error Messages Not Specific**:
```go
if err != nil {
    return fmt.Errorf("failed to create memory: %w", err)
}

// Better: Include context
return fmt.Errorf("failed to create memory (id=%s): %w", m.ID, err)
```

#### 4.2 GetMemory (Line ~120)

```go
func (d *Database) GetMemory(id string) (*Memory, error) {
    d.mu.RLock()
    defer d.mu.RUnlock()

    row := d.db.QueryRow(`
        SELECT id, content, source, importance, ...
        FROM memories WHERE id = ?
    `, id)

    var m Memory
    var tagsJSON string
    err := row.Scan(&m.ID, &m.Content, ...)
    // ...
}
```

**Strengths**:
- Uses RLock for read operations
- Single row query

**Issues**:

1. **No Caching**:
```go
// Every call hits the database
// Suggestion: Add LRU cache for frequently accessed memories
type Database struct {
    cache *lru.Cache  // github.com/hashicorp/golang-lru
}
```

2. **SELECT * Pattern Implicitly Used**:
```go
// Selecting all columns even when only ID/Content needed
// Suggestion: Add projection option
func (d *Database) GetMemory(id string, fields ...string) (*Memory, error)
```

#### 4.3 SearchFTS (Line ~400)

```go
func (d *Database) SearchFTS(query string, filters *SearchFilters) ([]*SearchResult, error) {
    // FTS5 query with ranking
    rows, err := d.db.Query(`
        SELECT m.*, bm25(memories_fts, 1.0, 0.75) as rank
        FROM memories m
        JOIN memories_fts fts ON m.id = fts.id
        WHERE memories_fts MATCH ?
        ORDER BY rank
        LIMIT ?
    `, escapedQuery, filters.Limit)
}
```

**Strengths**:
- BM25 ranking
- Proper FTS5 MATCH syntax
- Join with main table for full data

**Issues**:

1. **Query Escaping May Break Valid Queries**:
```go
func escapeFTS5Query(query string) string {
    words := strings.Fields(query)
    if len(words) > 1 {
        return strings.Join(words, " OR ")
    }
    return query
}

// Issue: Converts "exact phrase" to "exact OR phrase"
// Breaks phrase matching
```

2. **No Search Result Caching**:
```go
// Same query executed multiple times hits DB each time
// Add: Query result cache with TTL
```

3. **Filter Application After Query**:
```go
// Some filters applied in Go, not SQL
// This fetches more data than needed
```

#### 4.4 Relationship Operations (Lines 600-900)

**CreateRelationship** (Line ~620):
```go
func (d *Database) CreateRelationship(r *Relationship) error {
    if !IsValidRelationshipType(r.RelationshipType) {
        return fmt.Errorf("invalid relationship type")
    }
    // ...
}
```

**Issues**:

1. **No Duplicate Check**:
```go
// Can create same relationship twice
// Missing: UNIQUE constraint or upsert logic
```

2. **FindRelated Inefficient**:
```go
func (d *Database) FindRelated(memoryID string, filters *RelationshipFilters) ([]*Memory, error) {
    // Issue: Two separate queries (source and target)
    // Could use UNION or single query
}
```

#### 4.5 Graph Operations (Lines 900-1100)

**GetGraph** (Line ~920):
```go
func (d *Database) GetGraph(rootID string, depth int) (*Graph, error) {
    visited := make(map[string]int)  // memoryID -> distance
    queue := []string{rootID}

    // BFS traversal
    for len(queue) > 0 {
        current := queue[0]
        queue = queue[1:]
        // ... N+1 query problem
    }
}
```

**Issues**:

1. **N+1 Query Problem**:
```go
// Each node in the graph triggers a database query
// For depth=3, could be 100+ queries

// Better: Recursive CTE
const graphQuery = `
WITH RECURSIVE graph AS (
    SELECT id, content, 0 as distance
    FROM memories WHERE id = ?
    UNION ALL
    SELECT m.id, m.content, g.distance + 1
    FROM memories m
    JOIN memory_relationships r ON m.id = r.target_memory_id
    JOIN graph g ON r.source_memory_id = g.id
    WHERE g.distance < ?
)
SELECT * FROM graph;
`
```

#### 4.6 Category/Domain Operations (Lines 1100-1400)

Generally well-implemented but verbose.

**Issues**:

1. **Repetitive Code**:
```go
// ListCategories and ListDomains are nearly identical
// Could use generic list function with table parameter
```

---

### 5. `migrations.go` (162 LOC)

**Purpose**: Schema version upgrades

```go
func (d *Database) Migrate() error {
    currentVersion := d.getSchemaVersion()

    if currentVersion < 1 {
        d.migrateToV1()
    }
    if currentVersion < 2 {
        d.migrateToV2()
    }
    // ...
}
```

**Strengths**:
- Version tracking
- Incremental migrations
- Transaction-wrapped

**Issues**:

1. **No Down Migrations**:
```go
// Cannot rollback schema changes
// Missing: migrateDownFromV2()
```

2. **No Migration History Table**:
```go
// Only stores current version, not migration history
// Can't track when migrations were applied
```

3. **Manual Schema Version Management**:
```go
// Better: Use a migration framework like golang-migrate
```

---

## Critical Issues Summary

### High Priority

| Issue | Location | Impact | Recommendation |
|-------|----------|--------|----------------|
| N+1 queries in graph traversal | operations.go:920 | Performance | Use recursive CTE |
| Tags as JSON string | models.go:35 | Query efficiency | Normalize to tags table |
| No batch operations | operations.go | Performance | Add batch insert/update |
| Large monolithic file | operations.go (1687 LOC) | Maintainability | Split by domain |

### Medium Priority

| Issue | Location | Impact | Recommendation |
|-------|----------|--------|----------------|
| No query caching | operations.go | Performance | Add LRU cache |
| Missing composite indexes | schema.go | Query performance | Add strategic indexes |
| FTS5 query escaping | operations.go:400 | Search accuracy | Improve escaping logic |
| No connection retry | database.go:70 | Reliability | Add retry with backoff |

### Low Priority

| Issue | Location | Impact | Recommendation |
|-------|----------|--------|----------------|
| Schema as strings | schema.go | Maintainability | Use embedded SQL files |
| No down migrations | migrations.go | Operations | Add rollback capability |
| Generic error messages | operations.go | Debugging | Add context to errors |

---

## Recommendations

### Immediate Actions

1. **Split operations.go**:
```
internal/database/
├── database.go           # Connection management
├── models.go             # Domain structures
├── schema.go             # Table definitions
├── memory_ops.go         # Memory CRUD
├── relationship_ops.go   # Relationship CRUD
├── category_ops.go       # Category operations
├── search_ops.go         # Search operations
├── graph_ops.go          # Graph traversal
└── migrations.go         # Schema migrations
```

2. **Add Recursive CTE for Graph**:
```go
func (d *Database) GetGraphCTE(rootID string, depth int) (*Graph, error) {
    rows, err := d.db.Query(`
        WITH RECURSIVE graph(id, content, importance, distance) AS (
            SELECT id, content, importance, 0
            FROM memories WHERE id = ?
            UNION ALL
            SELECT m.id, m.content, m.importance, g.distance + 1
            FROM memories m
            JOIN memory_relationships r ON m.id = r.target_memory_id
            JOIN graph g ON r.source_memory_id = g.id
            WHERE g.distance < ?
        )
        SELECT DISTINCT * FROM graph
    `, rootID, depth)
    // ...
}
```

3. **Normalize Tags**:
```sql
CREATE TABLE tags (
    id TEXT PRIMARY KEY,
    name TEXT UNIQUE NOT NULL
);

CREATE TABLE memory_tags (
    memory_id TEXT REFERENCES memories(id) ON DELETE CASCADE,
    tag_id TEXT REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (memory_id, tag_id)
);

CREATE INDEX idx_memory_tags_tag ON memory_tags(tag_id);
```

### Future Improvements

1. **Add Read Replica Support** for high-read scenarios
2. **Implement Connection Pooling** for concurrent access
3. **Add Query Profiling** for performance monitoring
4. **Consider WAL2** when SQLite 3.37+ is widely available

---

## Conclusion

The database layer is functional and well-tested but has significant opportunities for optimization. The most impactful changes would be:

1. Fixing the N+1 query problem in graph traversal
2. Normalizing the tags storage
3. Splitting the monolithic operations.go

These changes would improve both performance and maintainability without breaking the existing API.

**Overall Grade: B+**

---

*Review completed by Claude Code Analysis*
*Next: Code Review #2 - MCP Server Layer*
