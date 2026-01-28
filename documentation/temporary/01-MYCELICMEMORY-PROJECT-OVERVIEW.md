# MycelicMemory: Comprehensive Project Overview

**Version 1.2.2 | Deep Technical Analysis**

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Project Vision & Goals](#2-project-vision--goals)
3. [Architecture Overview](#3-architecture-overview)
4. [Technology Stack Analysis](#4-technology-stack-analysis)
5. [Core Module Deep Dive](#5-core-module-deep-dive)
6. [Data Architecture](#6-data-architecture)
7. [Interface Layer Analysis](#7-interface-layer-analysis)
8. [AI Integration Architecture](#8-ai-integration-architecture)
9. [Build & Deployment System](#9-build--deployment-system)
10. [Testing Infrastructure](#10-testing-infrastructure)
11. [Security Considerations](#11-security-considerations)
12. [Performance Characteristics](#12-performance-characteristics)
13. [Project Metrics & Statistics](#13-project-metrics--statistics)
14. [Development History & Roadmap](#14-development-history--roadmap)
15. [Appendix: File Index](#15-appendix-file-index)

---

## 1. Executive Summary

### What is MycelicMemory?

MycelicMemory is an **AI-powered persistent memory system** designed to give Claude and other AI agents the ability to maintain long-term memory across conversations. It addresses a fundamental limitation of large language models: they have no persistent memory between sessions.

### Core Value Proposition

| Feature | Description |
|---------|-------------|
| **Persistent Memory** | Store knowledge and insights that survive across sessions |
| **Semantic Search** | Find related memories using AI-powered vector similarity |
| **Relationship Mapping** | Build knowledge graphs connecting related concepts |
| **AI Analysis** | Get insights, summaries, and pattern detection |
| **Multi-Interface** | CLI, REST API, MCP server, Desktop app |

### Technical Highlights

- **Language**: Go 1.23.0 with CGO for SQLite bindings
- **Database**: SQLite with FTS5 full-text search
- **Vector Store**: Optional Qdrant integration for semantic search
- **AI Backend**: Ollama for embeddings and chat
- **Distribution**: npm package, native binaries, Electron installer
- **Protocol**: Model Context Protocol (MCP) for Claude integration

### Project Status

- **Version**: 1.2.2 (Production-ready)
- **License**: MIT (Free and open source)
- **Codebase Size**: ~21,000 lines of Go code
- **Test Coverage**: 10 test files, comprehensive coverage

---

## 2. Project Vision & Goals

### Problem Statement

Large Language Models (LLMs) like Claude operate in a stateless manner - each conversation starts fresh with no memory of previous interactions. This creates several challenges:

1. **Context Loss**: Valuable information shared in past sessions is forgotten
2. **Repeated Explanations**: Users must re-explain preferences and project context
3. **No Learning**: The AI cannot build on previous insights
4. **Fragmented Knowledge**: Information exists in scattered conversation histories

### Solution Architecture

MycelicMemory addresses these challenges through:

```
User/AI Agent
      |
      v
+-------------------+
|   MCP Protocol    |  <-- JSON-RPC over stdin/stdout
+-------------------+
      |
      v
+-------------------+
|   Memory Service  |  <-- Business logic layer
+-------------------+
      |
      +-------------+-------------+
      |             |             |
      v             v             v
+----------+  +-----------+  +----------+
| SQLite   |  | Qdrant    |  | Ollama   |
| FTS5     |  | Vectors   |  | AI       |
+----------+  +-----------+  +----------+
```

### Design Principles

1. **Graceful Degradation**: Works without AI services, enables enhanced features when available
2. **Zero Configuration**: Sensible defaults, auto-detection of services
3. **Multi-Interface**: Same functionality across CLI, API, MCP
4. **Compatibility**: Maintains schema compatibility with Local Memory project
5. **Performance**: Sub-10ms response times for common operations

---

## 3. Architecture Overview

### Layered Architecture

MycelicMemory follows a clean layered architecture pattern:

```
┌─────────────────────────────────────────────────────────────┐
│                    PRESENTATION LAYER                        │
├─────────────────┬───────────────┬───────────────────────────┤
│     CLI         │    REST API   │      MCP Server            │
│  (Cobra cmds)   │    (Gin)      │    (JSON-RPC)              │
├─────────────────┴───────────────┴───────────────────────────┤
│                    BUSINESS LOGIC LAYER                      │
├─────────────────┬───────────────┬───────────────────────────┤
│ Memory Service  │ Search Engine │ Relationships Service      │
│                 │               │                            │
│ - Store/Update  │ - Semantic    │ - Create/Query             │
│ - Get/Delete    │ - Keyword     │ - Discover (AI)            │
│ - List/Filter   │ - Hybrid      │ - Graph Traversal          │
│ - Chunking      │ - Tag-based   │                            │
├─────────────────┴───────────────┴───────────────────────────┤
│                    DATA ACCESS LAYER                         │
├─────────────────┬───────────────┬───────────────────────────┤
│   Database      │  Vector Store │    AI Manager              │
│   (SQLite)      │   (Qdrant)    │    (Ollama)                │
│                 │               │                            │
│ - CRUD ops      │ - Index       │ - Embeddings               │
│ - FTS5 search   │ - Search      │ - Chat/Analysis            │
│ - Transactions  │ - Filter      │ - Relationship discovery   │
└─────────────────┴───────────────┴───────────────────────────┘
```

### Package Structure

```
mycelicmemory/
├── cmd/mycelicmemory/          # Application entry points
│   ├── main.go                 # Minimal entry point
│   ├── root.go                 # Root command, MCP mode
│   ├── cmd_memory.go           # remember, search, get, list, update, forget
│   ├── cmd_service.go          # start, stop, status
│   ├── cmd_analyze.go          # AI analysis commands
│   ├── cmd_relationships.go    # relate, discover
│   ├── cmd_organization.go     # categories, domains
│   ├── cmd_doctor.go           # Health diagnostics
│   ├── cmd_ui.go               # Dashboard launcher
│   ├── cmd_benchmark.go        # LoCoMo benchmarking
│   └── cmd_extras.go           # Utility commands
│
├── internal/                   # Private packages
│   ├── database/               # SQLite + FTS5
│   ├── memory/                 # Memory business logic
│   ├── search/                 # Search engines
│   ├── relationships/          # Graph operations
│   ├── ai/                     # AI orchestration
│   ├── vector/                 # Qdrant client
│   ├── api/                    # REST API
│   ├── mcp/                    # MCP server
│   ├── daemon/                 # Service management
│   ├── benchmark/              # LoCoMo benchmarking
│   ├── logging/                # Structured logging
│   └── testutil/               # Test utilities
│
├── pkg/                        # Public packages
│   └── config/                 # Configuration management
│
├── dashboard/                  # React web UI
├── installer/                  # Electron desktop app
├── benchmark/                  # LoCoMo test suite
└── scripts/                    # Build and deployment
```

### Design Patterns Used

| Pattern | Location | Purpose |
|---------|----------|---------|
| **Service Pattern** | `internal/*/service.go` | Encapsulate business logic |
| **Repository Pattern** | `internal/database/operations.go` | Abstract data access |
| **Manager Pattern** | `internal/ai/manager.go` | Coordinate AI services |
| **Factory Pattern** | `mcp.NewServer()`, `ai.NewManager()` | Object creation |
| **Strategy Pattern** | `memory.SessionDetector` | Pluggable session detection |
| **Decorator Pattern** | `logging.Logger` | Add logging to operations |

---

## 4. Technology Stack Analysis

### Core Languages & Runtimes

| Technology | Version | Purpose |
|------------|---------|---------|
| **Go** | 1.23.0 | Primary development language |
| **CGO** | Required | SQLite C bindings |
| **Node.js** | 16+ | npm wrapper, dashboard build |

### Backend Dependencies

```go
// Core dependencies (from go.mod)
github.com/spf13/cobra v1.10.2      // CLI framework
github.com/spf13/viper v1.18.2      // Configuration
github.com/gin-gonic/gin v1.11.0    // Web framework
github.com/mattn/go-sqlite3 v1.14.19 // SQLite driver
github.com/google/uuid v1.5.0       // UUID generation
```

#### Dependency Analysis

**spf13/cobra** - CLI Framework
- Industry standard for Go CLI applications
- Used by: kubectl, hugo, gh, docker
- Provides: subcommands, flags, auto-help, bash completion

**spf13/viper** - Configuration
- Supports: YAML, JSON, TOML, environment variables
- Features: Live watching, defaults, aliases
- Used for: `~/.mycelicmemory/config.yaml`

**gin-gonic/gin** - Web Framework
- High-performance HTTP router
- Middleware support (CORS, logging)
- JSON serialization
- Used for: REST API on port 3099

**mattn/go-sqlite3** - Database Driver
- CGO-based SQLite bindings
- Supports: FTS5, WAL mode, foreign keys
- Build tag: `fts5` required

### Frontend Stack

```json
// dashboard/package.json
{
  "react": "^18.2.0",
  "react-dom": "^18.2.0",
  "react-router-dom": "^6.21.0",
  "recharts": "^2.10.3",
  "lucide-react": "^0.303.0",
  "tailwindcss": "^3.4.0",
  "vite": "^5.0.10"
}
```

### Desktop Application

```json
// installer/package.json
{
  "electron": "^28.0.0",
  "electron-builder": "^24.9.1",
  "electron-store": "^8.1.0"
}
```

### External Services

| Service | Purpose | Default URL | Model |
|---------|---------|-------------|-------|
| **Ollama** | Embeddings + Chat | localhost:11434 | nomic-embed-text, qwen2.5:3b |
| **Qdrant** | Vector database | localhost:6333 | HNSW algorithm |

---

## 5. Core Module Deep Dive

### 5.1 Database Layer (`internal/database/`)

**Purpose**: Persistent storage with full-text search capabilities

**Files**:
- `database.go` (276 LOC) - Connection management
- `models.go` (351 LOC) - Domain structures
- `schema.go` (428 LOC) - Table definitions
- `operations.go` (1,687 LOC) - CRUD operations
- `migrations.go` (162 LOC) - Schema upgrades

**Key Features**:

```go
// Database struct with connection pooling
type Database struct {
    db   *sql.DB
    path string
    mu   sync.RWMutex  // Thread-safe access
}

// Connection configuration
dsn := fmt.Sprintf("%s?_foreign_keys=on&_journal_mode=WAL", path)
db.SetMaxOpenConns(1)  // SQLite single-writer limitation
```

**Schema Tables** (20+ tables):

| Table | Purpose |
|-------|---------|
| `memories` | Primary content storage |
| `memory_relationships` | Graph edges |
| `categories` | Hierarchical organization |
| `memory_categorizations` | M2M junction |
| `domains` | Knowledge partitions |
| `vector_metadata` | Embedding tracking |
| `agent_sessions` | Session management |
| `memories_fts` | FTS5 virtual table |
| `benchmark_runs` | Performance tracking |

### 5.2 Memory Service (`internal/memory/`)

**Purpose**: Business logic for memory operations

**Files**:
- `service.go` (415 LOC) - Core operations
- `chunker.go` (231 LOC) - Content splitting
- `session.go` (182 LOC) - Session detection

**Chunking Strategy**:

```go
// Hierarchical chunking for large content
type ChunkConfig struct {
    MaxChunkSize int    // 1000 chars (~200 tokens)
    Overlap      int    // 100 chars for context
    MinChunkSize int    // 1500 chars threshold
}

// Chunk levels
// Level 0: Full content (root)
// Level 1: Paragraph-sized chunks
// Level 2: Atomic concept chunks
```

**Session Detection Strategies**:

1. **Git Directory** (default): Hash of `.git` path
2. **Manual**: User-specified ID
3. **Hash**: Custom formula

### 5.3 Search Engine (`internal/search/`)

**Purpose**: Unified search across multiple modes

**Search Types**:

| Type | Implementation | Use Case |
|------|----------------|----------|
| `semantic` | Ollama embeddings + Qdrant | Concept similarity |
| `keyword` | SQLite FTS5 + BM25 | Exact phrase matching |
| `tags` | JSON array matching | Category filtering |
| `date_range` | Timestamp filtering | Temporal queries |
| `hybrid` | Combined semantic + keyword | Best of both |

**FTS5 Query Processing**:

```go
// Convert multi-word queries to OR for better recall
func escapeFTS5Query(query string) string {
    words := strings.Fields(query)
    if len(words) > 1 {
        return strings.Join(words, " OR ")
    }
    return query
}
```

### 5.4 Relationships Service (`internal/relationships/`)

**Purpose**: Knowledge graph operations

**7 Relationship Types**:

| Type | Description |
|------|-------------|
| `references` | Memory references another |
| `contradicts` | Memory contradicts another |
| `expands` | Memory expands on another |
| `similar` | Memory is similar to another |
| `sequential` | Memory follows another in sequence |
| `causes` | Memory causes another |
| `enables` | Memory enables another |

**Graph Traversal**:

```go
// BFS traversal for map_graph
func (d *Database) GetGraph(rootID string, depth int) (*Graph, error) {
    visited := make(map[string]int)  // memoryID -> distance
    queue := []string{rootID}
    visited[rootID] = 0
    // ... BFS implementation
}
```

### 5.5 AI Manager (`internal/ai/`)

**Purpose**: Orchestrate external AI services

**Components**:

```go
type Manager struct {
    ollama      *OllamaClient          // Embeddings + chat
    qdrant      *vector.QdrantClient   // Vector storage
    db          *database.Database     // Memory access
    config      *config.Config         // Settings
    mu          sync.RWMutex
    initialized bool
}
```

**Analysis Operations**:

| Operation | Input | Output |
|-----------|-------|--------|
| `question` | Query string | Answer + confidence + sources |
| `summarize` | Timeframe | Summary + themes |
| `analyze` | Query | Patterns + insights |
| `temporal_patterns` | Concept | Learning progression |

---

## 6. Data Architecture

### Memory Model

```go
type Memory struct {
    ID           string    `json:"id"`           // UUID
    Content      string    `json:"content"`      // Main text
    Source       string    `json:"source"`       // Origin info
    Importance   int       `json:"importance"`   // 1-10 scale
    Tags         []string  `json:"tags"`         // JSON array
    SessionID    string    `json:"session_id"`   // Context
    Domain       string    `json:"domain"`       // Knowledge area
    Embedding    []byte    `json:"embedding"`    // 768-dim vector
    CreatedAt    time.Time `json:"created_at"`
    UpdatedAt    time.Time `json:"updated_at"`
    AgentType    string    `json:"agent_type"`   // claude-desktop, etc.
    AgentContext string    `json:"agent_context"`
    AccessScope  string    `json:"access_scope"` // session/shared/global

    // Hierarchical chunking
    ParentMemoryID string  `json:"parent_memory_id"`
    ChunkLevel     int     `json:"chunk_level"`  // 0=root, 1=para, 2=atomic
    ChunkIndex     int     `json:"chunk_index"`  // Position
}
```

### Data Flow: Store Operation

```
User Input: "Remember: Go channels are typed conduits"
        │
        ▼
┌───────────────────────────────────────┐
│ Input Validation                      │
│ - Content not empty                   │
│ - Importance 1-10 (default 5)        │
│ - Tags normalized (lowercase, trim)   │
└───────────────────────────────────────┘
        │
        ▼
┌───────────────────────────────────────┐
│ Session Detection                     │
│ - Git directory hash                  │
│ - Manual ID                           │
│ - Custom hash                         │
└───────────────────────────────────────┘
        │
        ▼
┌───────────────────────────────────────┐
│ Chunking Decision                     │
│ - Content > 1500 chars?               │
│   - Yes: Split hierarchically         │
│   - No: Store as single root          │
└───────────────────────────────────────┘
        │
        ▼
┌───────────────────────────────────────┐
│ AI Processing (if available)         │
│ - Generate embedding via Ollama       │
│ - Store in Qdrant                     │
└───────────────────────────────────────┘
        │
        ▼
┌───────────────────────────────────────┐
│ Database Write                        │
│ - INSERT INTO memories                │
│ - FTS5 trigger auto-indexes           │
└───────────────────────────────────────┘
        │
        ▼
Return: { id, content, created_at, session_id }
```

### Data Flow: Search Operation

```
User Query: "concurrency patterns"
        │
        ▼
┌───────────────────────────────────────┐
│ Route by Search Type                  │
├───────────────────────────────────────┤
│ semantic → Vector similarity          │
│ keyword  → FTS5 full-text             │
│ tags     → JSON array match           │
│ hybrid   → Both + merge               │
└───────────────────────────────────────┘
        │
        ├── SEMANTIC PATH ──────────────┐
        │   1. Generate query embedding │
        │   2. Qdrant similarity search │
        │   3. Return top-K by cosine   │
        │                               │
        ├── KEYWORD PATH ───────────────┤
        │   1. Parse FTS5 query         │
        │   2. Execute on virtual table │
        │   3. Rank by BM25 score       │
        │                               │
        └── HYBRID PATH ────────────────┤
            1. Parallel: semantic + kw  │
            2. Merge results            │
            3. Weighted combination     │
                                        │
        ◄───────────────────────────────┘
        │
        ▼
┌───────────────────────────────────────┐
│ Post-Processing                       │
│ - Deduplicate                         │
│ - Apply min_relevance threshold       │
│ - Limit result count                  │
│ - Sort by relevance                   │
└───────────────────────────────────────┘
        │
        ▼
Return: [{ memory, relevance_score }]
```

---

## 7. Interface Layer Analysis

### 7.1 CLI Interface (`cmd/mycelicmemory/`)

**Framework**: Cobra

**Command Groups**:

| Command | File | Description |
|---------|------|-------------|
| `remember` | cmd_memory.go | Store new memory |
| `search` | cmd_memory.go | Search memories |
| `get` | cmd_memory.go | Get by ID |
| `list` | cmd_memory.go | List all |
| `update` | cmd_memory.go | Modify memory |
| `forget` | cmd_memory.go | Delete memory |
| `relate` | cmd_relationships.go | Create relationship |
| `discover` | cmd_relationships.go | AI discover |
| `analyze` | cmd_analyze.go | AI analysis |
| `start/stop` | cmd_service.go | Daemon management |
| `doctor` | cmd_doctor.go | Health check |
| `benchmark` | cmd_benchmark.go | Run benchmarks |

**Example Usage**:

```bash
# Store with options
mycelicmemory remember "Go interfaces are implicit" \
  --importance 8 \
  --tags go,interfaces,learning \
  --domain programming

# Search with filters
mycelicmemory search "concurrency" \
  --limit 10 \
  --domain programming

# AI analysis
mycelicmemory analyze "What have I learned about testing?"
```

### 7.2 REST API (`internal/api/`)

**Framework**: Gin

**Base URL**: `http://localhost:3099/api/v1/`

**Endpoints**:

| Method | Path | Handler |
|--------|------|---------|
| POST | `/memories` | Create memory |
| GET | `/memories` | List memories |
| GET | `/memories/:id` | Get by ID |
| PUT | `/memories/:id` | Update memory |
| DELETE | `/memories/:id` | Delete memory |
| POST | `/memories/search` | Search |
| POST | `/memories/search/intelligent` | AI search |
| POST | `/relationships` | Create relationship |
| POST | `/relationships/discover` | AI discover |
| GET | `/categories` | List categories |
| POST | `/categories` | Create category |
| GET | `/domains` | List domains |
| POST | `/domains` | Create domain |
| GET | `/sessions` | List sessions |
| GET | `/stats` | System statistics |

### 7.3 MCP Server (`internal/mcp/`)

**Protocol**: JSON-RPC 2.0 over stdin/stdout

**Protocol Version**: 2024-11-05

**Available Tools**:

```json
{
  "tools": [
    "store_memory",
    "search",
    "analysis",
    "relationships",
    "categories",
    "domains",
    "sessions",
    "stats",
    "get_memory_by_id",
    "update_memory",
    "delete_memory",
    "benchmark_run",
    "benchmark_status",
    "benchmark_results",
    "benchmark_compare",
    "benchmark_improve"
  ]
}
```

**MCP Integration**:

```json
// ~/.claude/mcp.json
{
  "mcpServers": {
    "mycelicmemory": {
      "command": "mycelicmemory",
      "args": ["--mcp"]
    }
  }
}
```

---

## 8. AI Integration Architecture

### Ollama Integration

**Client**: `internal/ai/ollama.go`

**Models Used**:

| Model | Purpose | Dimensions |
|-------|---------|------------|
| `nomic-embed-text` | Embeddings | 768 |
| `qwen2.5:3b` | Chat/Analysis | N/A |

**Embedding Generation**:

```go
func (c *OllamaClient) GenerateEmbedding(ctx context.Context, text string) ([]float64, error) {
    req := EmbeddingRequest{
        Model:  c.embeddingModel,  // nomic-embed-text
        Prompt: text,
    }
    // POST to http://localhost:11434/api/embeddings
    // Returns 768-dimensional float64 array
}
```

### Qdrant Integration

**Client**: `internal/vector/qdrant.go`

**Collection Configuration**:

```go
// Collection: mycelicmemory-memories
// Algorithm: HNSW (Hierarchical Navigable Small World)
// Parameters: m=16, ef_construct=100
// Metric: Cosine similarity
```

**Operations**:

| Operation | Method |
|-----------|--------|
| Index | `Upsert(ctx, id, vector, payload)` |
| Search | `Search(ctx, vector, limit, filter)` |
| Delete | `Delete(ctx, ids)` |

### Graceful Degradation

The system works without AI services:

```go
func (e *Engine) semanticSearch(opts *SearchOptions) ([]*SearchResult, error) {
    // Check if AI manager is available
    if e.aiManager == nil {
        return e.keywordSearch(opts)  // Fallback
    }

    status := e.aiManager.GetStatus()
    if !status.OllamaAvailable || !status.QdrantAvailable {
        return e.keywordSearch(opts)  // Fallback
    }

    // Proceed with semantic search...
}
```

---

## 9. Build & Deployment System

### Makefile Targets

```makefile
# Development
make build         # Compile binary
make dev-install   # Build + install to /usr/local/bin
make link          # Symlink for rapid iteration
make dev           # Live reload with air

# Testing
make test          # Run all tests
make test-coverage # Generate coverage report
make test-verbose  # With race detector
make lint          # golangci-lint

# Distribution
make build-all     # Multi-platform builds
```

### Build Configuration

```makefile
BINARY_NAME=mycelicmemory
VERSION=1.2.0
BUILD_TAGS=-tags "fts5"
CGO_ENABLED=CGO_ENABLED=1
LDFLAGS=-ldflags "-s -w -X main.Version=$(VERSION)"
```

### Platform Support

| Platform | Architecture | Binary |
|----------|--------------|--------|
| macOS | arm64 | mycelicmemory-darwin-arm64 |
| macOS | amd64 | mycelicmemory-darwin-amd64 |
| Linux | amd64 | mycelicmemory-linux-amd64 |
| Linux | arm64 | mycelicmemory-linux-arm64 |
| Windows | amd64 | mycelicmemory-windows-amd64.exe |

### npm Distribution

```json
// package.json
{
  "name": "mycelicmemory",
  "version": "1.2.2",
  "bin": {
    "mycelicmemory": "bin/mycelicmemory"
  }
}
```

The npm package includes a shell wrapper that downloads the appropriate binary on first run.

### CI/CD Pipelines

| Workflow | File | Trigger |
|----------|------|---------|
| CI | ci.yml | Pull requests |
| E2E Tests | e2e-test.yml | Push to main |
| Installation Test | installation-test.yml | Push to main |
| Release | release.yml | Tag push |
| npm Publish | npm-publish.yml | Release |
| Installer Build | installer.yml | Release |
| Benchmark | benchmark.yml | Manual/scheduled |

---

## 10. Testing Infrastructure

### Test Organization

| Package | Test File | Coverage |
|---------|-----------|----------|
| database | database_test.go | 929 LOC |
| ai | manager_test.go, ollama_test.go | 839 LOC |
| memory | service_test.go, session_test.go | 642 LOC |
| relationships | service_test.go | 433 LOC |
| search | engine_test.go | 368 LOC |
| vector | qdrant_test.go | 339 LOC |
| config | config_test.go | 267 LOC |

### Test Patterns

**Table-Driven Tests**:

```go
func TestMemoryStore(t *testing.T) {
    tests := []struct {
        name    string
        input   Memory
        want    Memory
        wantErr bool
    }{
        {
            name:    "valid memory",
            input:   Memory{Content: "test", Importance: 5},
            wantErr: false,
        },
        // ... more test cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### E2E Test Suite

```bash
scripts/e2e-tests/
├── run-all-e2e.sh      # Complete test run
├── test-cli.sh         # CLI functionality
├── test-memory-ops.sh  # CRUD operations
└── test-rest-api.sh    # HTTP endpoints
```

---

## 11. Security Considerations

### Data Storage

- **Location**: `~/.mycelicmemory/memories.db`
- **Permissions**: User-only access (0600)
- **Encryption**: Not enabled by default (SQLite supports it via SEE)

### Input Validation

```go
// Content validation
if strings.TrimSpace(opts.Content) == "" {
    return nil, fmt.Errorf("content is required")
}

// Importance range
if importance < 1 { importance = 5 }
if importance > 10 { importance = 10 }

// Relationship type validation
if !database.IsValidRelationshipType(relType) {
    return nil, fmt.Errorf("invalid relationship type: %s", relType)
}
```

### SQL Injection Prevention

All database operations use parameterized queries:

```go
_, err := d.db.Exec(`
    INSERT INTO memories (id, content, importance, ...)
    VALUES (?, ?, ?, ...)
`, m.ID, m.Content, m.Importance, ...)
```

### API Security

- CORS enabled for browser access
- No authentication by default (local-only)
- Localhost binding recommended

---

## 12. Performance Characteristics

### Database Performance

| Operation | Target | Measured |
|-----------|--------|----------|
| Insert memory | <10ms | ~5ms |
| FTS5 search | <50ms | ~20ms |
| Get by ID | <5ms | ~2ms |
| List (50 items) | <20ms | ~10ms |
| Graph traversal | <10ms | ~4ms |

### Memory Footprint

| Component | Size |
|-----------|------|
| Binary | 15-25 MB |
| Database (1K memories) | ~2 MB |
| In-memory overhead | ~50 MB |

### Scalability Limits

| Metric | Recommended | Maximum |
|--------|-------------|---------|
| Memories | 100K | 1M+ |
| Tags per memory | 10 | 50 |
| Content size | 10KB | 1MB |
| Graph depth | 3 | 5 |

---

## 13. Project Metrics & Statistics

### Codebase Size

| Category | Lines | Files |
|----------|-------|-------|
| Go source | 21,041 | 40+ |
| Tests | 4,500+ | 10 |
| Total Go | ~26,000 | 50+ |
| JavaScript (dashboard) | ~1,000 | 10 |
| Markdown docs | ~2,000 | 15 |

### Package Distribution

| Package | LOC | Purpose |
|---------|-----|---------|
| cmd/mycelicmemory | 2,540 | CLI commands |
| internal/database | 2,900 | Data layer |
| internal/mcp | 2,700 | MCP server |
| internal/ai | 1,200 | AI integration |
| internal/memory | 800 | Memory service |
| internal/search | 450 | Search engine |
| internal/relationships | 400 | Graph operations |
| internal/api | 600 | REST API |
| internal/benchmark | 1,200 | Benchmarking |
| pkg/config | 400 | Configuration |

### Dependency Count

| Type | Count |
|------|-------|
| Direct | 6 |
| Transitive | 30+ |
| Total | ~40 |

---

## 14. Development History & Roadmap

### Project Evolution

| Phase | Description | Status |
|-------|-------------|--------|
| Phase 1 | Project Setup | Complete |
| Phase 2 | Database Layer | Complete |
| Phase 3 | Core Memory Logic | Complete |
| Phase 4 | AI Integration | Complete |
| Phase 5 | REST API | Complete |
| Phase 6 | CLI | Complete |
| Phase 7 | MCP Server | Complete |
| Phase 8 | Daemon Management | Complete |
| Phase 9 | npm Distribution | Complete |
| Phase 10 | Build & Deployment | Complete |

### Recent Commits

| Hash | Type | Description |
|------|------|-------------|
| e6f0c91 | Hotfix | Rename ultrathink -> mycelicmemory |
| ad8b6b3 | Feature | E2E testing framework |
| 6aab6b9 | Feature | Desktop installer + dashboard |
| b21e3db | CI | npm publish workflow |

### Future Roadmap

1. **Phase 11**: Enhanced AI features (summarization, Q&A)
2. **Phase 12**: Multi-user support
3. **Phase 13**: Cloud sync capabilities
4. **Phase 14**: Plugin system
5. **Phase 15**: Mobile applications

---

## 15. Appendix: File Index

### Core Source Files

```
cmd/mycelicmemory/
├── main.go                 (15 LOC)  Entry point
├── root.go                 (116 LOC) Root command
├── cmd_memory.go           (400 LOC) Memory commands
├── cmd_service.go          (200 LOC) Service commands
├── cmd_analyze.go          (150 LOC) Analysis commands
├── cmd_relationships.go    (180 LOC) Relationship commands
├── cmd_organization.go     (150 LOC) Organization commands
├── cmd_doctor.go           (100 LOC) Diagnostics
├── cmd_ui.go               (80 LOC)  Dashboard launcher
├── cmd_benchmark.go        (200 LOC) Benchmarking
└── cmd_extras.go           (100 LOC) Utilities

internal/database/
├── database.go             (276 LOC) Connection management
├── models.go               (351 LOC) Domain structures
├── schema.go               (428 LOC) Table definitions
├── operations.go           (1,687 LOC) CRUD operations
├── migrations.go           (162 LOC) Schema upgrades
└── database_test.go        (929 LOC) Tests

internal/mcp/
├── server.go               (902 LOC) MCP server
├── handlers.go             (1,534 LOC) Tool handlers
├── formatter.go            (739 LOC) Response formatting
└── types.go                (200 LOC) Data structures

internal/memory/
├── service.go              (415 LOC) Memory service
├── chunker.go              (231 LOC) Content chunking
├── session.go              (182 LOC) Session detection
└── service_test.go         (433 LOC) Tests

internal/ai/
├── manager.go              (509 LOC) AI orchestration
├── ollama.go               (400 LOC) Ollama client
└── manager_test.go         (489 LOC) Tests

internal/search/
├── engine.go               (450 LOC) Search engine
└── engine_test.go          (368 LOC) Tests

internal/relationships/
├── service.go              (324 LOC) Relationship service
└── service_test.go         (433 LOC) Tests
```

### Configuration Files

```
config.example.yaml         Configuration template
Makefile                    Build system
go.mod                      Go module definition
go.sum                      Dependency checksums
package.json                npm package
.github/workflows/          CI/CD pipelines
```

### Documentation

```
README.md                   Main documentation
CONTRIBUTING.md             Contribution guide
LICENSE                     MIT license
docs/QUICKSTART.md          Getting started
docs/USE_CASES.md           Usage examples
docs/HOOKS.md               Hook configuration
docs/SOP-E2E-TESTING.md     Testing procedures
```

---

*This document provides a comprehensive technical overview of the MycelicMemory project. For implementation details, refer to the individual source files and API documentation.*

**Document Version**: 1.0
**Last Updated**: 2025-01-27
**Total Pages**: 25+
