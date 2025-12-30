# Database Package

## Purpose

SQLite database layer with FTS5 full-text search implementation.

## Components

### schema.go
- Complete 16-table schema definition
- FTS5 virtual table configuration
- CHECK constraints (relationship types, agent types, ranges)
- FOREIGN KEY constraints with CASCADE deletion
- All 13 verified indexes

### database.go
- Database connection management
- Connection pooling
- Transaction support
- CRUD operation interfaces

### operations.go
- Memory CRUD operations
- Search operations (FTS5, tags, date range)
- Relationship operations
- Category/domain operations
- Session operations

### migrations.go
- Schema versioning system
- Migration execution
- Rollback support

## Verified Schema Tables

**Core Tables (7):**
1. memories - Primary content storage
2. memory_relationships - Graph edges (7 types)
3. categories - Hierarchical organization
4. memory_categorizations - M2M with confidence
5. domains - Knowledge partitions
6. vector_metadata - 768-dim embeddings
7. agent_sessions - Session management (4 types)

**FTS5 Tables (5):**
- memories_fts, memories_fts_data, memories_fts_idx, memories_fts_docsize, memories_fts_config

**Metadata Tables (4):**
- performance_metrics, migration_log, schema_version, sqlite_sequence

## Usage Example

```go
db, err := database.Open(config.Database.Path)
if err != nil {
    log.Fatal(err)
}
defer db.Close()

// Create memory
mem := &Memory{
    Content: "Example memory",
    Importance: 8,
    Tags: []string{"example", "test"},
}
id, err := db.CreateMemory(mem)
```

## Performance Targets

- Search: <5ms for keyword queries
- Graph traversal: 4ms target
- CRUD operations: <10ms

## Related Issues

- #5: Implement complete SQLite schema
- #6: Implement FTS5 full-text search
- #7: Implement database layer interface
- #8: Implement memories table
- #9: Implement memory_relationships table
