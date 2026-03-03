# MycelicMemory — Master Reference Document

**Version:** 1.3.0
**Last Updated:** March 2026
**Schema Version:** 6

---

## Table of Contents

1. [Product Overview](#1-product-overview)
2. [Architecture Overview](#2-architecture-overview)
3. [Installation & Configuration](#3-installation--configuration)
4. [CLI & Entry Points](#4-cli--entry-points)
5. [Database Layer](#5-database-layer)
6. [Schema & Data Model](#6-schema--data-model)
7. [Memory Service](#7-memory-service)
8. [Search Engine](#8-search-engine)
9. [Active Recall Engine](#9-active-recall-engine)
10. [MCP Server (Claude Integration)](#10-mcp-server-claude-integration)
11. [MCP Tools Reference](#11-mcp-tools-reference)
12. [REST API](#12-rest-api)
13. [REST API Endpoints Reference](#13-rest-api-endpoints-reference)
14. [AI Manager (Ollama & Qdrant)](#14-ai-manager-ollama--qdrant)
15. [Knowledge Graph & Relationships](#15-knowledge-graph--relationships)
16. [Universal Ingestion Pipeline](#16-universal-ingestion-pipeline)
17. [Claude Code Adapter](#17-claude-code-adapter)
18. [Slack Adapter](#18-slack-adapter)
19. [Desktop Application — Architecture](#19-desktop-application--architecture)
20. [Desktop Application — Pages & Features](#20-desktop-application--pages--features)
21. [Desktop Application — Components & UX](#21-desktop-application--components--ux)
22. [API Bridge & IPC Layer](#22-api-bridge--ipc-layer)
23. [Response Formatting & UX](#23-response-formatting--ux)
24. [Schema Migrations](#24-schema-migrations)
25. [Build, CI/CD & Deployment](#25-build-cicd--deployment)
26. [Development Guide & Conventions](#26-development-guide--conventions)

---

## 1. Product Overview

MycelicMemory is an AI-powered persistent memory system designed for Claude and other AI agents. It captures, organizes, and retrieves knowledge across sessions — turning ephemeral AI conversations into a durable, searchable knowledge base with a relationship graph.

### Core Value Proposition

AI assistants like Claude lose all context between sessions. MycelicMemory solves this by providing:

- **Persistent storage** — Memories survive across sessions, projects, and tools.
- **Semantic search** — Find relevant memories by meaning, not just keywords.
- **Knowledge graph** — Memories are connected through typed relationships, enabling graph traversal and discovery.
- **Multi-source ingestion** — Import knowledge from Claude Code sessions, Slack workspaces, and other sources through a universal pipeline.
- **Active recall** — A multi-signal scoring engine that proactively surfaces relevant memories based on the current working context.
- **Local-first** — All data stays on the user's machine. No cloud accounts or subscriptions required for core functionality.

### Execution Modes

MycelicMemory operates in four distinct modes:

| Mode | Transport | Use Case |
|------|-----------|----------|
| **MCP Server** | JSON-RPC 2.0 over stdin/stdout | Claude Desktop, Claude Code integration |
| **REST API Daemon** | HTTP on port 3002/3099 | Desktop app, browser testing, external tools |
| **CLI** | Terminal commands | Direct memory operations, scripting |
| **Desktop App** | Electron + React | Visual memory browser, knowledge graph, settings |

### Technology Stack

| Layer | Technology |
|-------|-----------|
| Language | Go 1.23 |
| Database | SQLite with FTS5 (full-text search) |
| Vector Store | Qdrant (optional, for semantic search) |
| AI Models | Ollama (local, optional) — nomic-embed-text, qwen2.5:3b |
| REST Framework | Gin |
| CLI Framework | Cobra + Viper |
| Desktop | Electron 28 + React 18 + Vite 5 + Tailwind CSS 3.4 |
| Graph Visualization | vis-network 9.1 |
| Charts | Recharts 2.10 |

---

## 2. Architecture Overview

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                         CLIENTS                                     │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌───────────────────┐  │
│  │  Claude   │  │  Claude  │  │   CLI    │  │  Desktop App      │  │
│  │  Desktop  │  │   Code   │  │ Commands │  │  (Electron+React) │  │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────────┬──────────┘  │
│       │              │             │                  │              │
│   JSON-RPC 2.0   JSON-RPC 2.0   Direct Call     REST / IPC        │
│   (stdin/stdout) (stdin/stdout)                                     │
└───────┼──────────────┼─────────────┼──────────────────┼─────────────┘
        │              │             │                  │
┌───────▼──────────────▼─────────────▼──────────────────▼─────────────┐
│                      SERVICE LAYER                                   │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────────────┐   │
│  │   MCP    │  │  REST    │  │  Memory  │  │  Recall Engine   │   │
│  │  Server  │  │   API    │  │  Service │  │  (Multi-signal)  │   │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────────┬─────────┘   │
│       │              │             │                  │              │
│  ┌────▼──────────────▼─────────────▼──────────────────▼──────────┐  │
│  │                    CORE SERVICES                               │  │
│  │  ┌──────────┐  ┌──────────────┐  ┌────────────────────────┐  │  │
│  │  │ Search   │  │ Relationship │  │  Pipeline (Ingestion)  │  │  │
│  │  │ Engine   │  │   Service    │  │  ┌─────┐ ┌─────────┐  │  │  │
│  │  └────┬─────┘  └──────┬───────┘  │  │Queue│ │Adapters │  │  │  │
│  │       │               │           │  └──┬──┘ └────┬────┘  │  │  │
│  │  ┌────▼───────────────▼───────────▼─────▼─────────▼─────┐ │  │  │
│  │  │              AI Manager                               │ │  │  │
│  │  │  ┌──────────┐    ┌──────────┐                        │ │  │  │
│  │  │  │  Ollama  │    │  Qdrant  │                        │ │  │  │
│  │  │  │  Client  │    │  Client  │                        │ │  │  │
│  │  │  └──────────┘    └──────────┘                        │ │  │  │
│  │  └──────────────────────────────────────────────────────┘ │  │  │
│  └───────────────────────────┬───────────────────────────────┘  │  │
└──────────────────────────────┼──────────────────────────────────┘  │
                               │                                      │
┌──────────────────────────────▼──────────────────────────────────────┐
│                      STORAGE LAYER                                   │
│  ┌────────────────────────┐  ┌──────────────┐  ┌────────────────┐  │
│  │  SQLite + FTS5         │  │ Qdrant Cloud │  │  Ollama Local  │  │
│  │  (~/.mycelicmemory/)   │  │  (optional)  │  │  (optional)    │  │
│  └────────────────────────┘  └──────────────┘  └────────────────┘  │
└─────────────────────────────────────────────────────────────────────┘
```

### Package Map

```
cmd/mycelicmemory/          CLI entry point (Cobra commands)
internal/
├── api/                    REST API (Gin framework)
│   ├── server.go           Route registration, middleware
│   ├── handlers_memory.go  Memory CRUD endpoints
│   ├── handlers_search.go  Search endpoints
│   ├── handlers_relationships.go  Graph endpoints
│   ├── handlers_chat.go    Conversation endpoints
│   ├── handlers_recall.go  Active recall endpoint
│   ├── handlers_analysis.go Analysis endpoints
│   ├── handlers_source.go  Data source management
│   ├── handlers_system.go  Health, stats
│   ├── middleware.go       CORS, auth, rate limiting
│   └── response.go         Response wrapper helpers
├── ai/                     AI model integration
│   ├── manager.go          Ollama + Qdrant orchestration
│   └── ollama.go           Embedding & chat client
├── claude/                 Claude Code integration
│   ├── reader.go           JSONL file parser
│   ├── adapter.go          Pipeline SourceAdapter
│   ├── ingestion.go        Session ingestion + linking
│   └── types.go            Conversation data types
├── database/               SQLite persistence
│   ├── database.go         Connection, lifecycle, WAL
│   ├── schema.go           16-table schema (v6)
│   ├── models.go           Go struct definitions
│   ├── operations.go       Core CRUD (~3500 lines)
│   ├── operations_chat.go  Conversation operations
│   ├── operations_source.go Data source operations
│   ├── operations_stats.go  Statistics queries
│   └── migrations.go       v1→v6 migration chain
├── mcp/                    MCP protocol server
│   ├── server.go           JSON-RPC 2.0 handler
│   ├── handlers.go         20+ tool implementations
│   ├── types.go            Request/response structs
│   └── formatter.go        Rich UX formatting
├── memory/                 Memory business logic
│   ├── service.go          Store, validate, enrich
│   ├── chunker.go          Hierarchical chunking
│   └── session.go          Session detection strategies
├── pipeline/               Universal ingestion
│   ├── queue.go            Concurrent job queue
│   ├── types.go            SourceAdapter interface
│   └── transformer.go      Item → Memory conversion
├── adapters/               Source adapters
│   └── slack/              Slack workspace export
│       ├── adapter.go      SourceAdapter implementation
│       └── types.go        Slack-specific types
├── recall/                 Active memory agent
│   ├── engine.go           Multi-signal recall
│   ├── scorer.go           Weighted scoring
│   └── context.go          Context extraction
├── relationships/          Knowledge graph
│   └── service.go          CRUD, discovery, clustering
├── search/                 Search engine
│   └── engine.go           Semantic, keyword, hybrid
├── vector/                 Vector operations
│   └── qdrant.go           Qdrant HTTP client
├── ratelimit/              Rate limiting
│   └── bucket.go           Token bucket algorithm
├── logging/                Structured logging
├── daemon/                 Process management
└── dependencies/           DI container
```

### Data Flow

**Memory Creation:**
```
Client → MCP/REST → Memory Service → Database (INSERT)
                                   → FTS5 Index (trigger)
                                   → Qdrant Vector (async)
                                   → Auto-relationships (goroutine)
```

**Memory Search:**
```
Client → MCP/REST → Search Engine → FTS5 (keyword)
                                  → Qdrant (semantic)
                                  → Scorer (merge + rank)
                                  → Client
```

**Active Recall:**
```
Client → MCP/REST → Recall Engine → Context Extraction
                                  → Semantic Search (Qdrant)
                                  → Keyword Search (FTS5)
                                  → Graph Expansion (BFS)
                                  → Multi-signal Scoring
                                  → Ranked Results → Client
```

**Source Ingestion:**
```
Trigger → Pipeline Queue → Source Adapter → ConversationItem stream
                                          → Transformer → Memory objects
                                          → Database (batch insert)
                                          → Summary + Relationships
```

---

## 3. Installation & Configuration

### Installation

```bash
# Via npm (recommended)
npm install -g mycelicmemory

# From source (requires Go 1.23+, CGO-enabled C compiler)
git clone https://github.com/MycelicMemory/mycelicmemory
cd mycelicmemory
make build
```

### Configuration File

Location: `~/.mycelicmemory/config.yaml`

```yaml
# Database
database:
  path: "~/.mycelicmemory/memories.db"
  backup_interval: "24h"
  max_backups: 7
  auto_migrate: true

# REST API
rest_api:
  enabled: true
  port: 3002            # Default port
  host: "localhost"
  cors_enabled: true
  api_key: ""           # Optional authentication
  auto_port: true       # Find available port

# Session Detection
session:
  strategy: "git-directory"  # git-directory | manual | hash
  auto_generate: true

# Ollama (AI Model Server — Optional)
ollama:
  base_url: "http://localhost:11434"
  embedding_model: "nomic-embed-text"  # 768-dim vectors
  chat_model: "qwen2.5:3b"
  auto_detect: true

# Qdrant (Vector Database — Optional)
qdrant:
  url: "http://localhost:6333"
  api_key: ""           # For Qdrant Cloud
  auto_detect: true

# Logging
logging:
  level: "info"         # debug | info | warn | error
  format: "console"     # console | json

# Rate Limiting
rate_limit:
  enabled: true
  requests_per_second: 10
  burst_size: 20
```

### MCP Configuration for Claude

**Claude Code** (`~/.claude/claude_code_config.json`):
```json
{
  "mcpServers": {
    "mycelicmemory": {
      "command": "mycelicmemory",
      "args": ["--mcp"],
      "env": {}
    }
  }
}
```

**Claude Desktop** (`~/Library/Application Support/Claude/claude_desktop_config.json`):
```json
{
  "mcpServers": {
    "mycelicmemory": {
      "command": "/usr/local/bin/mycelicmemory",
      "args": ["--mcp"]
    }
  }
}
```

### Service Dependencies

| Service | Required | Purpose | Default |
|---------|----------|---------|---------|
| SQLite + FTS5 | Yes | Storage + full-text search | Built-in via CGO |
| Ollama | No | Embeddings + AI analysis | Graceful degradation |
| Qdrant | No | High-performance vector search | Falls back to FTS5 |

When optional services are unavailable, the system degrades gracefully:

- **Full mode** (Ollama + Qdrant): Semantic + keyword + graph scoring
- **Keyword+Graph mode** (no Qdrant): FTS5 keyword + graph scoring
- **Keyword-only mode** (no Ollama or Qdrant): FTS5 keyword scoring only

---

## 4. CLI & Entry Points

### Root Command

**File:** `cmd/mycelicmemory/root.go` (116 lines)

The CLI is built with Cobra and supports two primary execution modes:

```bash
# Standard CLI mode
mycelicmemory [command] [flags]

# MCP Server mode (for Claude integration)
mycelicmemory --mcp
```

**Global Flags:**
- `--config` — Path to config file (default: `~/.mycelicmemory/config.yaml`)
- `--log_level` — Override log level
- `--mcp` — Run as MCP server (JSON-RPC 2.0 over stdin/stdout)
- `--quiet` — Suppress non-essential output

### Service Commands

**File:** `cmd/mycelicmemory/cmd_service.go` (328 lines)

| Command | Description |
|---------|-------------|
| `start` | Launch REST API daemon (foreground or background) |
| `stop` | Gracefully stop running daemon |
| `status` | Show daemon uptime, port, process info |
| `ps` | List all running MycelicMemory processes |
| `kill_all` | Terminate all processes |

**Start Flags:**
- `--port` — Override REST API port
- `--host` — Override bind address
- `--background` — Run as daemon (detach from terminal)

### Memory Commands

| Command | Description |
|---------|-------------|
| `remember <content>` | Store a memory with optional flags |
| `remember-file <path>` | Store file contents as memory |
| `search <query>` | Search memories (keyword/semantic) |
| `list` | List memories with filters |
| `forget <id>` | Delete a memory |
| `analyze` | AI-powered analysis (requires Ollama) |
| `relate` | Create memory relationships |
| `discover` | AI relationship discovery |
| `doctor` | System health check |

### Initialization Flow

```
main.go → Execute() → root.go
  ├── --mcp flag? → MCP Server (stdin/stdout JSON-RPC)
  │     ├── Initialize database, AI manager, services
  │     ├── Start JSON-RPC 2.0 message loop
  │     └── Handle tool calls until EOF
  │
  └── Subcommand? → Cobra dispatch
        ├── "start" → Launch Gin REST API server
        │     ├── Configure middleware (CORS, auth, rate limit)
        │     ├── Register routes
        │     └── ListenAndServe (graceful shutdown on SIGINT/SIGTERM)
        │
        └── Other → Execute command, exit
```

---

## 5. Database Layer

### Connection Management

**File:** `internal/database/database.go` (295 lines)

The database layer wraps SQLite with FTS5 support, providing thread-safe access through a sync.RWMutex.

```go
type Database struct {
    db     *sql.DB
    mu     sync.RWMutex
    dbPath string
}
```

**Initialization Sequence:**
1. Open SQLite connection with WAL mode and foreign keys enabled
2. Execute `CoreSchema` — 16 base tables
3. Execute `FTS5Schema` — Full-text search virtual table + sync triggers
4. Execute `DataSourceSchema` — Multi-source ingestion tables
5. Execute `ConversationSchema` — Conversation tracking tables
6. Run migrations (`v1→v6` chain)
7. Record schema version

**SQLite Pragmas:**
```sql
PRAGMA journal_mode = WAL;      -- Write-Ahead Logging for concurrency
PRAGMA foreign_keys = ON;       -- Enforce referential integrity
PRAGMA busy_timeout = 5000;     -- 5s retry on lock contention
```

### Core Operations

**File:** `internal/database/operations.go` (~3500 lines)

This is the largest file in the codebase, containing all CRUD operations.

**Memory Operations:**
| Function | Description |
|----------|-------------|
| `CreateMemory(m *Memory)` | Insert with UUID generation, timestamp |
| `GetMemory(id string)` | Retrieve by primary key |
| `UpdateMemory(id string, updates map[string]interface{})` | Partial update |
| `DeleteMemory(id string)` | Delete with CASCADE cleanup |
| `ListMemories(limit, offset int, domain string)` | Paginated listing |
| `SearchFTS(query string, limit int)` | BM25-ranked full-text search |
| `GetMemoriesByConversationID(convID string)` | All memories from a conversation |
| `GetMemoriesByDomainAndTags(domain string, tags []string, minShared, limit int)` | Tag intersection within domain |

**Relationship Operations:**
| Function | Description |
|----------|-------------|
| `CreateRelationship(rel *Relationship)` | Insert graph edge |
| `GetRelationships(memoryID string)` | Edges for a node |
| `GetAllRelationships(limit int, minStrength float64)` | Full graph edges |
| `GetGraph(memoryID string, depth int)` | BFS graph traversal |
| `GetGraphOptimized(memoryID string, depth int)` | Recursive CTE traversal |
| `GetGraphStats()` | Node/edge counts, avg degree, orphans, hubs |

**Chat/Conversation Operations** (`operations_chat.go`):
| Function | Description |
|----------|-------------|
| `UpsertConversation(conv *Conversation)` | Insert or update session |
| `GetConversation(id string)` | Retrieve full session |
| `ListConversations(limit, offset int, projectPath string)` | Paginated listing |
| `SearchConversations(query string, limit int)` | Full-text search over conversations |
| `InsertMessage(msg *ConversationMessage)` | Add message to session |
| `InsertAction(action *ConversationAction)` | Add tool call record |

**Data Source Operations** (`operations_source.go`):
| Function | Description |
|----------|-------------|
| `CreateDataSource(src *DataSource)` | Register new source |
| `UpdateDataSource(id string, updates map[string]interface{})` | Update source state |
| `ListDataSources(sourceType, status string)` | Filtered listing |
| `CreateSyncHistory(entry *SyncHistory)` | Record sync operation |
| `ListSyncHistory(sourceID string, limit int)` | Sync history for source |

---

## 6. Schema & Data Model

### Schema Version: 6

**File:** `internal/database/schema.go` (464 lines)

The schema is divided into four constant blocks executed in order during initialization:

1. **CoreSchema** — Base tables (memories, relationships, categories, domains, sessions, metrics)
2. **FTS5Schema** — Full-text search virtual table with auto-sync triggers
3. **DataSourceSchema** — Multi-source ingestion tables
4. **ConversationSchema** — Conversation tracking tables

### Entity-Relationship Diagram

```
┌─────────────┐       ┌─────────────────────┐       ┌──────────────┐
│  memories   │──────<│ memory_relationships │>──────│  memories    │
│             │  src   │                     │  tgt  │              │
│  id (PK)    │       │  id (PK)            │       │              │
│  content    │       │  source_memory_id    │       │              │
│  source     │       │  target_memory_id    │       │              │
│  importance │       │  relationship_type   │       │              │
│  tags       │       │  strength            │       │              │
│  domain     │       │  context             │       │              │
│  session_id │       │  auto_generated      │       │              │
│  embedding  │       │  last_accessed_at    │       │              │
│  created_at │       │  access_count        │       │              │
│  updated_at │       └─────────────────────┘       └──────────────┘
│  slug       │
│  parent_id  │──┐    ┌─────────────────────┐
│  chunk_level│  └───>│  memories (parent)   │
│  chunk_index│       └─────────────────────┘
│  convo_id   │──┐
└──────┬──────┘  │    ┌─────────────────────┐
       │         └───>│  conversations       │
       │              │  id (PK)             │
       │              │  session_id          │
       │              │  project_path        │
       │              │  model, title        │
       │              │  message_count       │
       │              │  summary_memory_id   │──> memories
       │              │  source_id           │──> data_sources
       │              └──────────┬───────────┘
       │                         │
       │              ┌──────────▼───────────┐
       │              │  messages             │
       │              │  id, session_id (FK)  │
       │              │  role, content        │
       │              │  sequence_index       │
       │              └──────────┬───────────┘
       │                         │
       │              ┌──────────▼───────────┐
       │              │  actions              │
       │              │  id, session_id (FK)  │
       │              │  tool_name, input     │
       │              │  result_text, success │
       │              └─────────────────────┘
       │
  ┌────▼─────────────────┐    ┌──────────────────┐
  │ memory_categorizations│───>│  categories       │
  │  memory_id (FK)      │    │  id (PK)          │
  │  category_id (FK)    │    │  name, description│
  │  confidence          │    │  parent_id (self) │
  └──────────────────────┘    └──────────────────┘

┌─────────────────┐    ┌───────────────────────────┐
│  data_sources   │───<│ data_source_sync_history   │
│  id (PK)        │    │  id (PK)                   │
│  source_type    │    │  source_id (FK)             │
│  name, config   │    │  items_processed            │
│  status         │    │  memories_created            │
│  last_sync_at   │    │  duplicates_skipped          │
└─────────────────┘    └───────────────────────────┘

┌──────────────┐  ┌──────────────────┐  ┌──────────────┐
│  domains     │  │  vector_metadata │  │ agent_sessions│
│  id, name    │  │  memory_id (FK)  │  │ session_id   │
│  description │  │  vector_index    │  │ agent_type   │
└──────────────┘  │  embedding_model │  │ is_active    │
                  └──────────────────┘  └──────────────┘
```

### Key Table Details

**memories** — The core content table:
- `id` TEXT PRIMARY KEY (UUID v4)
- `content` TEXT NOT NULL — The memory text
- `importance` INTEGER DEFAULT 5 — 1 (trivial) to 10 (critical)
- `tags` TEXT — JSON array: `["tag1", "tag2"]`
- `domain` TEXT — Knowledge partition (e.g., "frontend", "devops")
- `chunk_level` INTEGER — 0=full, 1=paragraph, 2=atomic
- `conversation_id` TEXT — Links memory to its source conversation
- 10 indexes for optimized queries

**memory_relationships** — Knowledge graph edges:
- 7 relationship types: `references`, `contradicts`, `expands`, `similar`, `sequential`, `causes`, `enables`
- `strength` REAL — 0.0 to 1.0 (normalized confidence)
- `auto_generated` BOOLEAN — AI-discovered vs manual
- `last_accessed_at` DATETIME — For strength decay (schema v6)
- `access_count` INTEGER — Usage tracking (schema v6)
- 8 indexes including compound indexes for optimized BFS traversal

**memories_fts** — FTS5 virtual table:
- Indexes: `content`, `source`, `tags` (searchable columns)
- Unindexed: `id`, `slug`, `session_id`, `domain` (metadata only)
- Auto-synchronized via INSERT/UPDATE/DELETE triggers

### Valid Source Types (12)

```
claude-stream, claude-code-local, slack, discord, telegram,
imessage, email, browser, notion, obsidian, github, custom
```

### Valid Relationship Types (7)

| Type | Meaning | Example |
|------|---------|---------|
| `references` | A cites B | "See authentication docs" → auth memory |
| `contradicts` | A conflicts with B | New finding invalidates old one |
| `expands` | A elaborates on B | Detailed explanation of a concept |
| `similar` | A resembles B | Two memories about the same topic |
| `sequential` | A follows B | Temporal or logical ordering |
| `causes` | A leads to B | Bug → crash, decision → outcome |
| `enables` | A is prerequisite for B | Setup step → feature use |

---

## 7. Memory Service

### Service Architecture

**File:** `internal/memory/service.go`

```go
type Service struct {
    db              *database.Database
    config          *config.Config
    sessionDetector *SessionDetector
    chunker         *Chunker
}
```

The memory service is the business logic layer between API handlers and the database. It handles validation, enrichment, session detection, and hierarchical chunking.

### Store Options

```go
type StoreOptions struct {
    Content         string   // Required — the memory text
    Importance      int      // 1-10, defaults to 5
    Tags            []string // Categorization labels
    Domain          string   // Knowledge partition
    Source          string   // Origin identifier
    SessionID       string   // Override auto-detection
    AgentType       string   // claude-desktop | claude-code | api | unknown
    AgentContext    string   // Additional context metadata
    AccessScope     string   // session | shared | global
    Slug            string   // Human-readable identifier
    ConversationID  string   // Link to source conversation
}
```

### Session Detection Strategies

**File:** `internal/memory/session.go`

| Strategy | Method |
|----------|--------|
| `git-directory` | Hash of git repository root path (default) |
| `manual` | Explicit session ID passed by client |
| `hash` | Content-based hash for deduplication |

### Hierarchical Chunking

**File:** `internal/memory/chunker.go`

Long memories are split into a parent-child hierarchy:

| Level | Name | Description |
|-------|------|-------------|
| 0 | Full | Complete original memory |
| 1 | Paragraph | Major sections/paragraphs |
| 2 | Atomic | Individual sentences/facts |

Child chunks reference their parent via `parent_memory_id` and are ordered by `chunk_index`. This enables both overview retrieval (level 0) and precise fact retrieval (level 2).

---

## 8. Search Engine

### Search Modes

**File:** `internal/search/engine.go`

The search engine supports multiple strategies that are selected based on the request and available services:

| Mode | Engine | Requirement |
|------|--------|-------------|
| `keyword` | FTS5 BM25 ranking | Always available |
| `semantic` | Qdrant vector similarity | Ollama + Qdrant |
| `hybrid` | Combined keyword + semantic | Ollama + Qdrant |
| `tags` | Tag intersection (AND/OR) | Always available |
| `date_range` | Temporal filtering | Always available |

### Search Options

```go
type SearchOptions struct {
    Query       string   // Search text
    SearchType  string   // keyword | semantic | hybrid | tags | date_range
    Tags        []string // For tag-based search
    TagOperator string   // "and" | "or" (default: "and")
    Domain      string   // Filter by domain
    Limit       int      // Max results (default: 10)
    MinScore    float64  // Minimum relevance threshold
    UseAI       bool     // Enable AI-powered search
    StartDate   string   // For date_range
    EndDate     string   // For date_range
}
```

### FTS5 Search

The FTS5 full-text search provides BM25-ranked results across `content`, `source`, and `tags` columns:

```sql
SELECT m.*, rank FROM memories m
JOIN memories_fts ON m.id = memories_fts.id
WHERE memories_fts MATCH ?
ORDER BY rank
LIMIT ?
```

### Semantic Search

When Ollama and Qdrant are available:
1. Generate embedding for query text via Ollama (`nomic-embed-text`, 768 dimensions)
2. Search Qdrant collection for nearest neighbors (cosine similarity)
3. Retrieve full memory records from SQLite by matched IDs
4. Score by vector similarity

---

## 9. Active Recall Engine

### Overview

**Files:** `internal/recall/engine.go`, `internal/recall/scorer.go`, `internal/recall/context.go`

The recall engine is the most sophisticated retrieval component. Unlike simple search, it combines multiple signals to proactively surface the most relevant memories for a given context.

### Request Structure

```go
type RecallRequest struct {
    Context string   // Current working context (task, file, conversation)
    Files   []string // Active file paths (for keyword extraction)
    Project string   // Project identifier
    Limit   int      // Max results (default: 10)
    Depth   int      // Graph traversal depth (default: 1)
}
```

### Multi-Signal Scoring

The recall engine applies weighted scoring across five signals. Weights adjust based on available services:

**Full Mode** (Ollama + Qdrant available):

| Signal | Weight | Source |
|--------|--------|--------|
| Semantic similarity | 0.40 | Qdrant cosine similarity |
| Keyword relevance | 0.20 | FTS5 BM25 score |
| Importance | 0.15 | Memory importance (1-10, normalized) |
| Recency | 0.15 | Exponential decay, 30-day half-life |
| Graph connectivity | 0.10 | Relationship traversal score |

**Keyword+Graph Mode** (no Qdrant):

| Signal | Weight |
|--------|--------|
| Keyword relevance | 0.60 |
| Importance | 0.15 |
| Recency | 0.15 |
| Graph connectivity | 0.10 |

**Keyword-Only Mode** (no Qdrant, no graph):

| Signal | Weight |
|--------|--------|
| Keyword relevance | 0.70 |
| Importance | 0.15 |
| Recency | 0.15 |

### Recall Pipeline

```
1. Context Extraction
   └── Parse input context, extract keywords from files

2. Candidate Generation (parallel)
   ├── Semantic search (Qdrant) → candidate set A
   └── Keyword search (FTS5)   → candidate set B

3. Graph Expansion
   └── BFS from top candidates → expand through relationships

4. Multi-Signal Scoring
   └── Apply weighted formula to all candidates

5. Re-ranking
   └── Sort by composite score, deduplicate, apply limit

6. Result Assembly
   └── Attach relation chains, timing metrics
```

### Response Structure

```go
type RecallResult struct {
    Memories      []RecallMemory // Ranked results
    TotalFound    int
    SearchMode    string         // "full" | "keyword_graph" | "keyword_only"
    GraphExpanded int            // Nodes traversed via BFS
    Timing        RecallTiming   // Per-stage timing breakdown
}

type RecallMemory struct {
    Memory         database.Memory
    Score          float64        // Composite score (0.0-1.0)
    MatchType      string         // "semantic" | "keyword" | "graph"
    RelationChain  []RelationLink // How this was discovered
}

type RecallTiming struct {
    EmbeddingMs int64
    SemanticMs  int64
    KeywordMs   int64
    GraphMs     int64
    RerankMs    int64
    TotalMs     int64
}
```

---

## 10. MCP Server (Claude Integration)

### Protocol

**File:** `internal/mcp/server.go`

MycelicMemory implements the Model Context Protocol (MCP) specification version `2024-11-05`. The server communicates via JSON-RPC 2.0 over stdin/stdout, making it compatible with Claude Desktop and Claude Code.

### Server Structure

```go
type Server struct {
    db             *database.Database
    cfg            *config.Config
    aiManager      *ai.Manager
    memSvc         *memory.Service
    searchEng      *search.Engine
    relSvc         *relationships.Service
    recallEng      *recall.Engine
    claudeIngester *claude.Ingester
    claudeReader   *claude.Reader
    pipelineQueue  *pipeline.Queue
    rateLimiter    *ratelimit.Limiter
    formatter      *Formatter
    log            *logging.Logger
}
```

### Message Flow

```
Claude Client                    MCP Server
    │                                │
    │── initialize ────────────────>│
    │<── capabilities ──────────────│
    │                                │
    │── tools/list ────────────────>│
    │<── tool definitions ──────────│
    │                                │
    │── tools/call ────────────────>│
    │   {name: "store_memory",      │
    │    arguments: {...}}          │
    │                                │── Execute handler
    │                                │── Format response
    │<── result ────────────────────│
    │                                │
```

### JSON-RPC Types

```go
type Request struct {
    JSONRPC string          `json:"jsonrpc"`  // Always "2.0"
    Method  string          `json:"method"`
    Params  json.RawMessage `json:"params"`
    ID      interface{}     `json:"id"`
}

type Response struct {
    JSONRPC string      `json:"jsonrpc"`
    Result  interface{} `json:"result,omitempty"`
    Error   *RPCError   `json:"error,omitempty"`
    ID      interface{} `json:"id"`
}
```

### Capabilities Advertised

```json
{
  "protocolVersion": "2024-11-05",
  "serverInfo": {
    "name": "mycelicmemory",
    "version": "1.2.0"
  },
  "capabilities": {
    "tools": {}
  }
}
```

---

## 11. MCP Tools Reference

### Memory Management

| Tool | Parameters | Description |
|------|-----------|-------------|
| `store_memory` | content, importance?, tags?, domain?, source? | Store a new memory |
| `search` | query, search_type?, tags?, tag_operator?, domain?, limit?, use_ai? | Search memories |
| `get_memory_by_id` | id | Retrieve by UUID |
| `update_memory` | id, content?, importance?, tags? | Update fields |
| `delete_memory` | id | Permanently delete |

### Analysis

| Tool | Parameters | Description |
|------|-----------|-------------|
| `analysis` | analysis_type, question?, query?, timeframe?, limit?, domain? | AI-powered analysis |

Analysis types:
- `question` — Natural language Q&A over memories
- `summarize` — Generate summaries (timeframes: today, week, month, all)
- `analyze` — Pattern detection across memories
- `temporal_patterns` — Knowledge evolution over time

### Relationships

| Tool | Parameters | Description |
|------|-----------|-------------|
| `relationships` | relationship_type, memory_id?, source_memory_id?, target_memory_id?, strength?, depth?, limit? | Graph operations |

Relationship sub-operations (via `relationship_type` parameter):
- `find_related` — Find related memories for a given ID
- `discover` — AI-powered relationship discovery
- `create` — Create a new relationship edge
- `map_graph` — BFS graph traversal from a node

### Organization

| Tool | Parameters | Description |
|------|-----------|-------------|
| `categories` | categories_type, name?, description?, memory_id? | Category management |
| `domains` | domains_type, name?, description?, domain? | Domain management |
| `sessions` | sessions_type | Session listing/stats |
| `stats` | stats_type, domain?, category_id? | System statistics |

### Ingestion & Recall

| Tool | Parameters | Description |
|------|-----------|-------------|
| `context_recall` | context, files?, project?, limit?, depth? | Multi-signal active recall |
| `reindex_memories` | — | Re-index all memories into Qdrant |
| `ingest_conversations` | project_path?, min_messages?, create_summaries? | Import Claude Code history |
| `search_chats` | query, project_path?, limit? | Search conversations |
| `get_chat` | session_id | Get full conversation |
| `trace_source` | memory_id | Trace memory to conversation |
| `ingest_source` | source_id, backfill? | Trigger pipeline ingestion |
| `pipeline_status` | — | Check ingestion progress |
| `list_sources` | source_type? | List registered adapters |
| `link_session_memories` | conversation_id | Link session memories to summary |

### Auto-Relationship on Store

When `store_memory` is called, the server automatically:
1. Stores the memory in the database
2. Indexes it in Qdrant (if available)
3. Launches a goroutine to find the 3 most similar existing memories via FTS5
4. Creates `similar` relationships with strength proportional to relevance score

This ensures the knowledge graph grows organically with every new memory.

---

## 12. REST API

### Server Setup

**File:** `internal/api/server.go`

The REST API is built on the Gin framework with structured middleware:

```go
type APIServer struct {
    db            *database.Database
    config        *config.Config
    aiManager     *ai.Manager
    memSvc        *memory.Service
    searchEng     *search.Engine
    relSvc        *relationships.Service
    recallEng     *recall.Engine
    claudeIngester *claude.Ingester
    pipelineQueue *pipeline.Queue
    router        *gin.Engine
}
```

### Middleware Stack

1. **CORS** — Configurable origins, methods, headers
2. **API Key Auth** — Optional bearer token authentication
3. **Rate Limiting** — Token bucket per-IP and global limits
4. **Request Logging** — Structured request/response logging
5. **Recovery** — Panic recovery with error response

### Response Format

All endpoints return a consistent wrapper:

```json
{
  "success": true,
  "message": "Operation completed",
  "data": { ... }
}
```

Error responses:
```json
{
  "success": false,
  "message": "Error description",
  "data": null
}
```

**Helper functions** (`internal/api/response.go`):
- `SuccessResponse(c, message, data)` — 200 with data
- `CreatedResponse(c, message, data)` — 201 with data
- `ErrorResponse(c, status, message)` — Error with HTTP status
- `NotFoundResponse(c, message)` — 404

---

## 13. REST API Endpoints Reference

### Memory CRUD

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/memories` | Create memory |
| `GET` | `/api/v1/memories` | List memories (paginated) |
| `GET` | `/api/v1/memories/:id` | Get memory by ID |
| `PUT` | `/api/v1/memories/:id` | Update memory |
| `DELETE` | `/api/v1/memories/:id` | Delete memory |

### Search

| Method | Path | Description |
|--------|------|-------------|
| `GET/POST` | `/api/v1/memories/search` | General search |
| `POST` | `/api/v1/memories/search/intelligent` | AI-powered search |
| `POST` | `/api/v1/search/tags` | Tag-based search (AND/OR) |
| `POST` | `/api/v1/search/date-range` | Date range filter |

### Recall

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/recall` | Active memory recall |

### Relationships & Graph

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/relationships` | All relationships (paginated) |
| `POST` | `/api/v1/relationships` | Create relationship |
| `POST` | `/api/v1/relationships/discover` | AI discovery |
| `POST` | `/api/v1/relationships/batch-discover` | Fast keyword-based discovery |
| `GET` | `/api/v1/memories/:id/related` | Related memories |
| `GET` | `/api/v1/memories/:id/graph` | Graph traversal |
| `GET` | `/api/v1/graph/stats` | Graph statistics |

### Analysis

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/analyze` | AI analysis |

### Organization

| Method | Path | Description |
|--------|------|-------------|
| `GET/POST` | `/api/v1/categories` | Category CRUD |
| `GET` | `/api/v1/categories/stats` | Category statistics |
| `POST` | `/api/v1/memories/:id/categorize` | Assign category |
| `GET/POST` | `/api/v1/domains` | Domain CRUD |
| `GET` | `/api/v1/domains/:name/stats` | Domain statistics |

### Chat History

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/chats/projects` | Claude Code projects |
| `GET` | `/api/v1/chats` | List conversations |
| `GET` | `/api/v1/chats/:id` | Get conversation |
| `GET` | `/api/v1/chats/:id/messages` | Get messages |
| `GET` | `/api/v1/chats/:id/tool-calls` | Get tool calls |
| `POST` | `/api/v1/chats/ingest` | Ingest Claude history |
| `GET` | `/api/v1/chats/search` | Search conversations |

### Data Sources

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/sources` | List sources |
| `POST` | `/api/v1/sources` | Create source |
| `GET` | `/api/v1/sources/:id` | Get source |
| `PATCH` | `/api/v1/sources/:id` | Update source |
| `DELETE` | `/api/v1/sources/:id` | Delete source |
| `POST` | `/api/v1/sources/:id/sync` | Trigger sync |
| `POST` | `/api/v1/sources/:id/pause` | Pause syncing |
| `POST` | `/api/v1/sources/:id/resume` | Resume syncing |
| `GET` | `/api/v1/sources/:id/history` | Sync history |
| `GET` | `/api/v1/sources/:id/stats` | Source statistics |
| `GET` | `/api/v1/sources/:id/memories` | Memories from source |

### System

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/health` | Health check (API, Ollama, Qdrant, DB) |
| `GET` | `/api/v1/stats` | System statistics |
| `GET` | `/api/v1/memories/:id/trace` | Trace memory to source |

---

## 14. AI Manager (Ollama & Qdrant)

### Manager Architecture

**File:** `internal/ai/manager.go`

```go
type Manager struct {
    ollama         *OllamaClient
    qdrant         *vector.QdrantClient
    db             *database.Database
    config         *config.Config
    initialized    bool
    cachedStatus   *Status
    cachedStatusAt time.Time
    statusCacheTTL time.Duration  // 60s
}
```

The AI Manager orchestrates two optional external services:

### Ollama Client

**File:** `internal/ai/ollama.go`

| Operation | Model | Dimensions | Timeout |
|-----------|-------|------------|---------|
| Embedding | `nomic-embed-text` | 768 | 60s |
| Chat/Analysis | `qwen2.5:3b` | — | 60s |

**Endpoints:**
- `POST /api/embeddings` — Generate vector embeddings
- `POST /api/chat` — AI chat completion
- `GET /api/tags` — List available models

### Qdrant Client

**File:** `internal/vector/qdrant.go`

| Setting | Value |
|---------|-------|
| Collection | `mycelicmemory-memories` |
| Distance | Cosine |
| Dimensions | 768 |
| HNSW m | 16 |
| HNSW ef_construct | 100 |

**Operations:**
- `Upsert(memoryID, vector, metadata)` — Index a memory
- `Search(vector, limit, minScore)` — Nearest neighbor search
- `Delete(memoryID)` — Remove from index
- `CollectionInfo()` — Collection statistics

**Qdrant Cloud Support:**
- `api_key` field in config for cloud authentication
- Sends `api-key` header on all HTTP requests
- URL can point to `*.qdrant.io` cloud instances

### Status Caching

The manager caches service availability for 60 seconds to minimize health check overhead:

```go
type Status struct {
    OllamaAvailable bool
    QdrantAvailable bool
    EmbeddingModel  string
    ChatModel       string
}
```

---

## 15. Knowledge Graph & Relationships

### Service Architecture

**File:** `internal/relationships/service.go`

```go
type Service struct {
    db     *database.Database
    config *config.Config
}
```

### Relationship CRUD

```go
type CreateOptions struct {
    SourceMemoryID   string
    TargetMemoryID   string
    RelationshipType string   // Must be one of 7 valid types
    Strength         float64  // 0.0 to 1.0
    Context          string   // Explanation text
    AutoGenerated    bool     // true = AI-discovered
}
```

Validation:
- Source and target memories must exist
- Relationship type must be valid
- Strength is clamped to [0.0, 1.0]
- No duplicate edges (same source, target, type)

### Discovery Methods

**AI-Powered Discovery** (slow, ~200s for 200 memories):
- Uses Ollama chat model to analyze memory pairs
- Generates relationship type, strength, and explanation
- Best for high-quality, nuanced connections

**Batch Keyword Discovery** (fast, ~5s for 200 memories):
- Uses FTS5 to find memories sharing significant keywords
- Extracts keywords from content (filters stopwords, short words)
- Creates `similar` relationships where BM25 overlap exceeds threshold
- Supports domain filtering

**Domain Clustering:**
- Finds memory pairs within the same domain sharing 2+ tags
- Creates `similar` relationships with strength based on tag overlap ratio
- Builds dense clusters within knowledge domains

### Graph Traversal

**BFS traversal** (`GetGraph`):
```
Given: start node, max depth
1. Queue ← {start node}
2. While queue not empty and depth < max:
   a. Dequeue node
   b. Fetch all relationships for node
   c. For each connected node not yet visited:
      - Add to results with distance
      - Enqueue for further expansion
3. Return nodes + edges
```

**Optimized CTE traversal** (`GetGraphOptimized`):
```sql
WITH RECURSIVE graph AS (
  SELECT source_memory_id, target_memory_id, 1 as depth
  FROM memory_relationships
  WHERE source_memory_id = ? OR target_memory_id = ?
  UNION ALL
  SELECT r.source_memory_id, r.target_memory_id, g.depth + 1
  FROM memory_relationships r
  JOIN graph g ON ...
  WHERE g.depth < ?
)
SELECT DISTINCT * FROM graph;
```

### Strength Decay (Schema v6)

Relationships that aren't traversed lose strength over time:

```go
func (s *Service) DecayRelationships(maxAgeDays int, decayFactor float64)
```

- `last_accessed_at` tracks when a relationship was last traversed
- `access_count` tracks total traversals
- `TouchRelationship(id)` updates both fields, strengthening the edge
- Unused relationships decay, keeping the graph clean

### Bridge Node Detection

```go
type BridgeNode struct {
    MemoryID    string
    Domains     []string  // Domains this memory connects
    Connections int       // Total edge count
}
```

Bridge nodes connect otherwise disconnected domains. They represent the most valuable cross-domain insights.

### Graph Statistics

```go
type GraphStats struct {
    TotalNodes           int
    TotalEdges           int
    AvgDegree            float64
    OrphanCount          int          // Nodes with no edges
    MostConnected        []NodeDegree // Top hubs
    RelationshipTypeDist map[string]int
}
```

---

## 16. Universal Ingestion Pipeline

### Architecture

**Files:** `internal/pipeline/queue.go`, `internal/pipeline/types.go`, `internal/pipeline/transformer.go`

The pipeline provides a universal framework for ingesting data from any source into the memory system.

### SourceAdapter Interface

```go
type SourceAdapter interface {
    Type() string                                              // "slack", "claude-code-local", etc.
    Configure(config json.RawMessage) error                    // Parse source-specific config
    ReadItems(ctx context.Context, checkpoint string) (        // Stream items
        <-chan ConversationItem, <-chan error)
    Checkpoint() string                                        // Current position for resume
    Validate() error                                           // Verify configuration
}
```

Any data source can be integrated by implementing this interface. The pipeline handles queuing, batching, transformation, and error recovery.

### Universal Data Format

```go
type ConversationItem struct {
    ExternalID     string            // Source system ID
    SourceType     string            // "slack", "discord", etc.
    ConversationID string            // Thread/channel grouping
    ProjectOrSpace string            // Workspace context
    Role           string            // "user", "assistant", "bot"
    Author         string            // Display name
    Content        string            // Message text
    ContentType    string            // "text", "markdown", "code"
    Timestamp      time.Time
    SequenceIndex  int
    Attachments    []Attachment      // Files, images, links
    Actions        []Action          // Tool calls, reactions
    Metadata       map[string]any    // Source-specific data
    ThreadID       string            // Thread parent
    ReplyToID      string            // Reply reference
}
```

### Queue Configuration

```go
type QueueConfig struct {
    MaxConcurrent  int           // Worker pool size (default: 4)
    BatchSize      int           // Items per batch (default: 100)
    RetryAttempts  int           // Max retries (default: 3)
    RetryDelay     time.Duration // Base delay, exponential backoff (default: 1s)
    BackfillMode   bool          // Full history vs incremental
    ProgressReport time.Duration // Update interval (default: 5s)
}
```

### Job Lifecycle

```
Created → Queued → Running → Completed
                         └─→ Failed (retries exhausted)
                         └─→ Cancelled (user action)
```

### Ingestion Flow

```
1. Source Registration
   └── Create data_source record with type + config JSON

2. Sync Trigger
   └── Create job → Queue → Worker picks up

3. Item Streaming
   └── Adapter.ReadItems() yields ConversationItem stream

4. Transformation
   └── ConversationItem → Memory objects (content, tags, domain)
   └── ConversationItem → Conversation + Message + Action records

5. Storage
   └── Batch INSERT into memories, conversations, messages, actions
   └── FTS5 triggers auto-index new content

6. Post-Processing
   └── Create summary memories for conversations
   └── Link session memories to summaries
   └── Create temporal chain relationships
   └── Scan tool calls for store_memory operations → create relationships

7. Checkpoint
   └── Update data_source.last_sync_position for incremental resume
```

### Registered Adapters

| Adapter | Source Type | Status |
|---------|-----------|--------|
| Claude Code | `claude-code-local` | Production |
| Slack | `slack` | Production |
| Discord | `discord` | Planned |
| Telegram | `telegram` | Planned |
| Email | `email` | Planned |
| GitHub | `github` | Planned |

---

## 17. Claude Code Adapter

### Architecture

**Files:** `internal/claude/reader.go`, `internal/claude/adapter.go`, `internal/claude/ingestion.go`, `internal/claude/types.go`

The Claude Code adapter reads local JSONL conversation files from `~/.claude/projects/*/sessions/` and converts them into the universal pipeline format.

### Configuration

```go
type AdapterConfig struct {
    ClaudeDir   string // Path to ~/.claude (default: auto-detect)
    ProjectPath string // Filter to specific project
    MinMessages int    // Skip sessions with fewer messages
}
```

### Reader

**File:** `internal/claude/reader.go`

Parses JSONL files where each line is a JSON object representing a message:

```
{"type":"human","message":{"content":"..."},"timestamp":"..."}
{"type":"assistant","message":{"content":[...],"model":"claude-sonnet-4-20250514"},"timestamp":"..."}
```

The reader handles:
- Multiple message content types (text, tool_use, tool_result)
- Model extraction from assistant messages
- Timestamp parsing across formats
- File enumeration across projects

### Ingestion Pipeline

**File:** `internal/claude/ingestion.go`

When a session is ingested, the following operations occur:

```
1. Parse JSONL → ConversationFile
2. Upsert Conversation record (dedup by project_hash + session_id)
3. Insert Messages (role, content, sequence_index, token_count)
4. Insert Actions (tool_name, input, result, success, filepath)
5. Create Summary Memory (if create_summaries=true)
   └── Content: title + first prompt + message/tool stats
   └── Domain: "conversations"
   └── Source: "claude-code-session"
6. Link Session Memories
   └── Find all memories with same conversation_id
   └── Create 'references' relationships from summary → each memory
7. Find Tool Call Memories
   └── Scan messages for store_memory tool calls
   └── Match against existing memories by content prefix
   └── Create 'references' relationships
8. Create Temporal Chain
   └── Order memories by created_at within conversation
   └── Create 'sequential' relationships in chronological order
```

### Statistics (as of March 2026)

- 75+ sessions ingested from local Claude Code history
- 20,000+ messages parsed
- 6,000+ tool calls recorded
- Summary memories created as graph nodes with domain="conversations"
- Sequential relationships auto-created between sessions in the same project

---

## 18. Slack Adapter

### Architecture

**Files:** `internal/adapters/slack/adapter.go`, `internal/adapters/slack/types.go`

The Slack adapter processes Slack workspace export archives (the ZIP export from Slack's data export feature).

### Configuration

```go
type Config struct {
    ExportPath     string // Required: path to extracted export directory
    IncludePrivate bool   // Include private channels (groups.json)
    IncludeDMs     bool   // Include direct messages (dms.json)
    MinMessages    int    // Skip channels with fewer messages
}
```

### Export Structure

```
slack-export/
├── channels.json       # Public channel metadata
├── users.json          # User ID → name mapping
├── groups.json         # Private channel metadata
├── dms.json            # Direct message metadata
├── general/            # Per-channel directories
│   ├── 2024-01-15.json # Messages by date
│   ├── 2024-01-16.json
│   └── ...
├── random/
│   └── ...
└── ...
```

### Parsing Pipeline

1. Load `users.json` → build ID-to-name mapping
2. Load `channels.json` → public channel list
3. Load `groups.json` → private channels (if enabled)
4. Load `dms.json` → DM groups (if enabled)
5. For each channel:
   a. Read date-stamped JSON files chronologically
   b. Parse messages with thread support
   c. Extract reactions, file attachments
   d. Convert to `ConversationItem` stream
6. Yield items through pipeline channel

### Features

- **Thread support** — Replies linked to parent messages via `ThreadID`
- **Reaction parsing** — Emoji reactions captured as metadata
- **File attachments** — File references preserved as `Attachment` objects
- **User resolution** — Slack user IDs replaced with display names
- **Incremental sync** — Checkpoint tracks last processed date per channel

---

## 19. Desktop Application — Architecture

### Technology Stack

| Component | Technology | Version |
|-----------|-----------|---------|
| Shell | Electron | 28.0.0 |
| UI Framework | React | 18.2.0 |
| Build Tool | Vite | 5.0.0 |
| Styling | Tailwind CSS | 3.4.0 |
| Language | TypeScript | 5.3.0 |
| Routing | react-router-dom | 6.21 |
| Icons | lucide-react | latest |
| Charts | Recharts | 2.10 |
| Graph | vis-network | 9.1.9 |
| Notifications | react-hot-toast | 2.6 |
| Settings | electron-store | 8.1 |

### Process Architecture

```
┌─────────────────────────────────────┐
│         Main Process (Node.js)       │
│  ┌─────────────┐  ┌──────────────┐  │
│  │ Window Mgr  │  │ IPC Handlers │  │
│  │ (index.ts)  │  │  memory.ipc  │  │
│  │             │  │  claude.ipc  │  │
│  │             │  │  config.ipc  │  │
│  └──────┬──────┘  └──────┬───────┘  │
│         │                │           │
│  ┌──────▼────────────────▼────────┐  │
│  │  MycelicMemoryClient (REST)    │  │
│  │  → http://127.0.0.1:3002      │  │
│  └────────────────────────────────┘  │
└──────────────┬───────────────────────┘
               │ contextBridge
┌──────────────▼───────────────────────┐
│        Preload Script                 │
│  window.mycelicMemory = {             │
│    memory, claude, stats, domains,    │
│    relationships, settings, services  │
│  }                                    │
└──────────────┬───────────────────────┘
               │
┌──────────────▼───────────────────────┐
│        Renderer Process (React)       │
│  ┌──────────┐  ┌──────────────────┐  │
│  │ App.tsx  │  │  api-bridge.ts   │  │
│  │ (Router) │  │  (Electron/HTTP) │  │
│  └──────────┘  └──────────────────┘  │
│  ┌──────────────────────────────────┐ │
│  │  Pages: Dashboard, Memories,     │ │
│  │  Sessions, Graph, Sources,       │ │
│  │  Settings                        │ │
│  └──────────────────────────────────┘ │
└───────────────────────────────────────┘
```

### Directory Structure

```
desktop/
├── src/
│   ├── main/                    # Electron main process
│   │   ├── index.ts             # Window management, app lifecycle
│   │   ├── preload.ts           # Context bridge (IPC → renderer)
│   │   ├── ipc/                 # IPC handler modules
│   │   │   ├── memory.ipc.ts
│   │   │   ├── claude.ipc.ts
│   │   │   └── config.ipc.ts
│   │   └── services/            # Backend service clients
│   │       └── mycelicmemory-client.ts  # REST API wrapper
│   │
│   ├── renderer/                # React frontend
│   │   ├── main.tsx             # Entry point (HashRouter)
│   │   ├── App.tsx              # Root component (sidebar + routes)
│   │   ├── api-bridge.ts        # Unified API (IPC or fetch)
│   │   ├── pages/               # Page components
│   │   │   ├── Dashboard.tsx
│   │   │   ├── MemoryBrowser.tsx
│   │   │   ├── ClaudeSessions.tsx
│   │   │   ├── KnowledgeGraph.tsx
│   │   │   ├── DataSources.tsx
│   │   │   └── Settings.tsx
│   │   ├── components/          # Shared components
│   │   │   ├── CommandPalette.tsx
│   │   │   ├── CreateMemoryModal.tsx
│   │   │   ├── ConfirmDialog.tsx
│   │   │   ├── ErrorBoundary.tsx
│   │   │   ├── Toast.tsx
│   │   │   ├── Badge.tsx
│   │   │   └── ...
│   │   └── styles/              # Tailwind + global CSS
│   │
│   └── shared/                  # Shared TypeScript types
│       └── types.ts
│
├── package.json
├── vite.config.ts
├── tailwind.config.ts
├── tsconfig.json
└── electron-builder.yml
```

---

## 20. Desktop Application — Pages & Features

### Dashboard (`/`)

**File:** `desktop/src/renderer/pages/Dashboard.tsx`

The landing page providing a system overview with live data.

**Layout:**
- 4-column stat card grid (Total Memories, Domains, Sessions, This Week)
- 2x2 quick action grid (Search, Add Memory, Knowledge Graph, Claude Sessions)
- Domain distribution pie chart (Recharts)
- Importance distribution bar chart
- Recent memories panel (last 5)
- Service control panel (API, Ollama, Qdrant status with start buttons)

**Data Sources:** `stats.dashboard()`, `stats.health()`, `domains.list()`, `memory.list()`

**Refresh:** Auto-polls every 30 seconds; 3-second polling when disconnected.

### Memory Browser (`/memories`)

**File:** `desktop/src/renderer/pages/MemoryBrowser.tsx`

Two-panel layout for browsing, searching, and editing memories.

**Left Panel (w-96):**
- Search bar with keyword/semantic toggle
- Filter panel (domain dropdown, importance range sliders)
- Paginated memory list (50 per page)
- MemoryCard components with preview, domain badge, importance indicator

**Right Panel (flex-1):**
- Full memory detail view
- Inline editing (content, importance)
- Tag display
- Delete with confirmation dialog
- Metadata (ID, created/updated dates, session)

**Search Modes:** Keyword search (FTS5) or semantic search (Qdrant).

### Claude Sessions (`/sessions`)

**File:** `desktop/src/renderer/pages/ClaudeSessions.tsx`

Three-column layout for exploring ingested Claude Code conversations.

**Left Column (w-64):** Project list with session counts, "Ingest Conversations" button
**Middle Column (w-80):** Session list with search, title/prompt preview, message counts
**Right Column (flex-1):** Message thread with user/assistant avatars, tool call display, expandable long messages

**Ingest:** Clicking "Ingest Conversations" triggers `claude.ingest()` which scans `~/.claude/projects/` and imports new sessions.

### Knowledge Graph (`/graph`)

**File:** `desktop/src/renderer/pages/KnowledgeGraph.tsx`

Interactive force-directed graph visualization using vis-network.

**Visualization:**
- Memory nodes: circles sized by importance, colored by domain
- Session summary nodes: orange diamonds
- Chat session nodes: colored stars (by project)
- Relationship edges: colored by type, width by strength
- Session trace edges: orange dashed lines

**Controls:**
- Domain filter, relationship type filter
- Show/hide chat sessions toggle
- Zoom in/out, fit to view
- Discover relationships button (triggers batch keyword discovery)
- Click node to see details in side panel

**Data Loading:** Single `relationships.getAll({ limit: 1000 })` call replaces the previous N per-memory fetching pattern. Fallback to per-memory approach if endpoint unavailable.

**Physics:** Barnes-Hut algorithm tuned for readability (gravitational constant: -3000, spring length: 120, damping: 0.15).

### Data Sources (`/sources`)

**File:** `desktop/src/renderer/pages/DataSources.tsx`

Two-panel layout for managing data source integrations.

**Left Panel:** Source list with status badges (active/paused/error), add source button
**Right Panel:** Source detail with stats grid, configuration JSON, sync history timeline

**Actions:** Sync Now, Pause/Resume, Delete, Add Source (modal with type dropdown, name, config JSON).

**Supported Source Types:** 12 types (claude-stream, claude-code-local, slack, discord, telegram, imessage, email, browser, notion, obsidian, github, custom).

### Settings (`/settings`)

**File:** `desktop/src/renderer/pages/Settings.tsx`

Configuration page with live connection testing.

**Sections:**
- Connection status grid (API, Ollama, Qdrant, Database)
- MCP setup guide with platform-specific instructions (Windows/macOS/Linux)
- API configuration (URL, port)
- Ollama configuration (URL, embedding model selector, chat model selector)
- Qdrant configuration (enable toggle, URL)
- Interface settings (theme, sidebar state)

---

## 21. Desktop Application — Components & UX

### Global Components

**CommandPalette** (`Ctrl+K`):
- Full-screen modal with search input
- Real-time search across memories, sessions, domains
- Keyboard navigation (Up/Down/Enter/Escape)
- 200ms debounce on input

**CreateMemoryModal** (`Ctrl+N`):
- Content textarea, domain input, importance slider (1-10), tag editor
- `Ctrl+Enter` to save
- Validation: content required

**ConfirmDialog:**
- Used for destructive actions (delete memory, delete source)
- Variant styling: danger (red), warning (amber)

**ErrorBoundary:**
- Wraps all routes
- Catches React rendering errors
- Shows retry button with error details

**Toast Notifications:**
- react-hot-toast, bottom-right position
- Success (3s), Error (5s) auto-dismiss
- Dark theme matching app design

### Design System

**Color Palette:**
- Background: `slate-900` (primary), `slate-800` (panels)
- Primary: `primary-500` (indigo)
- Status: green (success/active), amber (warning/paused), red (error/danger)
- Domain colors: indigo, green, amber, cyan, purple, pink, teal, orange

**Typography:** System fonts, monospace for code/IDs

**Domain Color Map:**
```typescript
{
  general: '#6366f1',     // indigo
  frontend: '#22c55e',    // green
  backend: '#f59e0b',     // amber
  database: '#06b6d4',    // cyan
  devops: '#8b5cf6',      // purple
  testing: '#ec4899',     // pink
  programming: '#6366f1', // indigo
  code: '#14b8a6',        // teal
  conversations: '#f97316' // orange
}
```

**Relationship Color Map:**
```typescript
{
  references: '#6366f1',  // indigo
  contradicts: '#ef4444', // red
  expands: '#22c55e',     // green
  similar: '#8b5cf6',     // purple
  sequential: '#f59e0b',  // amber
  causes: '#06b6d4',      // cyan
  enables: '#ec4899'      // pink
}
```

### Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| `Ctrl+K` / `Cmd+K` | Open command palette |
| `Ctrl+N` / `Cmd+N` | Create new memory |
| `Ctrl+B` / `Cmd+B` | Toggle sidebar |
| `Escape` | Close modals/palette |
| `Ctrl+Enter` | Save in create modal |

---

## 22. API Bridge & IPC Layer

### Dual-Environment API

**File:** `desktop/src/renderer/api-bridge.ts` (248 lines)

The API bridge provides a unified interface that works in both Electron (IPC) and browser (HTTP fetch) environments:

```typescript
// Detection
const isElectron = typeof window !== 'undefined' && window.mycelicMemory !== undefined;

// Export appropriate implementation
export const api = isElectron ? window.mycelicMemory : browserApi;
```

### Browser Fallback

When running in a browser (for development/testing), the bridge uses direct `fetch()` calls:

```typescript
async function fetchApi<T>(endpoint: string, options?: RequestInit): Promise<T> {
    const response = await fetch(`${API_BASE}${endpoint}`, { ...options });
    const data = await response.json();
    return data.data ?? data;  // Auto-unwrap {success, message, data} wrapper
}
```

### API Surface

The complete API surface exposed to the renderer:

```typescript
api.memory     .list / .get / .create / .store / .update / .delete / .search
api.stats      .dashboard / .health
api.domains    .list / .create / .stats
api.categories .list / .create / .stats / .categorize
api.relationships .getAll / .get / .create / .discover / .batchDiscover / .related / .graph
api.graph      .stats
api.analysis   .analyze
api.sources    .list / .get / .create / .update / .delete / .pause / .resume / .sync / .history / .stats / .memories
api.trace      .source
api.recall     .query
api.search     .tags / .dateRange / .intelligent
api.claude     .projects / .sessions / .session / .messages / .toolCalls / .ingest / .search
api.config     .get / .set / .getAll
api.services   .status / .startBackend / .startOllama / .startQdrant / .stopBackend / .onStatusUpdate
```

### IPC Security

The preload script uses Electron's `contextBridge` with context isolation enabled:
- No direct Node.js access from renderer
- All IPC calls go through typed channel handlers
- Renderer only sees the `window.mycelicMemory` API object

---

## 23. Response Formatting & UX

### MCP Formatter

**File:** `internal/mcp/formatter.go` (890 lines)

The formatter transforms raw tool responses into rich, human-readable Markdown for Claude's output:

```go
func (f *Formatter) FormatToolResponse(toolName string, result interface{}, duration time.Duration) string
```

### Format Structure

Every formatted response follows this template:

```markdown
💾 **Store Memory**
*Persisting knowledge for future recall*
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

[Tool-specific formatted content]

⚡ *Completed in 42ms*

💡 **Next Steps**
   → Use `search` to verify the memory was indexed
   → Use `relationships(discover)` to find connections

<details>
<summary>📋 Raw JSON Response</summary>
```json
{ ... }
```
</details>
```

### Tool-Specific Formatting

| Tool | Format Style |
|------|-------------|
| `store_memory` | Confirmation box with ID, timestamps |
| `search` | Numbered results with relevance bars, metadata YAML blocks |
| `get_memory_by_id` | Full detail card with importance bar, metadata table |
| `analysis` | Confidence bar, answer block, reasoning, sources |
| `relationships` | Connection diagrams with strength bars |
| `context_recall` | Score bars, domain summary, relation chains, timing breakdown |
| `stats` | ASCII table with visual metrics |

### Visual Elements

- **Progress bars:** `[████████░░]` (filled/empty blocks)
- **Emoji icons:** Tool-specific (💾 store, 🔍 search, 🧭 recall, etc.)
- **Performance indicators:** ⚡ (<100ms), 🚀 (<500ms), ✓ (<1s), 🐢 (>1s)
- **ID truncation:** UUIDs shortened to 8 chars with ellipsis
- **Time formatting:** ISO 8601 → "Jan 02, 2006 15:04"
- **Age calculation:** "today", "3 days", "2 months"

---

## 24. Schema Migrations

### Migration System

**File:** `internal/database/migrations.go`

Migrations run automatically on startup when `auto_migrate: true` in config. Each migration:
1. Checks current schema version
2. Applies pending migrations in order
3. Updates `schema_version` table
4. Logs results to `migration_log`

### Migration Chain

| Version | Name | Changes |
|---------|------|---------|
| v1→v2 | Initial setup | Base schema establishment |
| v2→v3 | Data sources | Added `data_sources`, `data_source_sync_history` tables |
| v3→v4 | Chat history | Added `cc_sessions`, `cc_messages`, `cc_tool_calls` tables |
| v4→v5 | Schema generalization | Renamed `cc_sessions`→`conversations`, `cc_messages`→`messages`, `cc_tool_calls`→`actions`, `memories.cc_session_id`→`conversation_id` |
| v5→v6 | Relationship decay | Added `last_accessed_at`, `access_count` to `memory_relationships` |

### v5→v6 Migration Detail

```sql
ALTER TABLE memory_relationships ADD COLUMN last_accessed_at DATETIME;
ALTER TABLE memory_relationships ADD COLUMN access_count INTEGER DEFAULT 0;
UPDATE memory_relationships SET last_accessed_at = created_at WHERE last_accessed_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_relationships_last_accessed ON memory_relationships(last_accessed_at);
```

### Safety Guarantees

- Migrations are idempotent (safe to re-run)
- Each migration runs in a transaction
- Failed migrations are logged with error details
- Schema version only advances on successful migration
- `IF NOT EXISTS` / `ADD COLUMN` patterns prevent duplicate application

---

## 25. Build, CI/CD & Deployment

### Go Binary Build

```bash
# Standard build (Linux/macOS)
make build

# Windows build (CRITICAL: requires strip flags)
CGO_ENABLED=1 go build -tags "fts5" -ldflags "-s -w" -o mycelicmemory.exe ./cmd/mycelicmemory
```

**Windows Note:** Without `-ldflags "-s -w"`, the binary produces a 57MB PE with 24 sections that Windows refuses to run. With strip flags: 25MB, 12 sections, works correctly. Build requires TDM-GCC 10.3.0+ for CGO/SQLite compilation.

### Desktop App Build

```bash
cd desktop

# Development
npm run dev           # Vite dev server + Electron

# Production
npm run build         # Vite + TypeScript compilation
npm run package       # Auto-detect platform
npm run package:win   # Windows NSIS installer
npm run package:mac   # macOS DMG
npm run package:linux # Linux AppImage/deb
```

### CI/CD Workflows

**Pull Request Validation** (`.github/workflows/ci.yml`):
- Go 1.23, Ubuntu latest
- `go test -race -coverprofile` — Tests with race detector
- `golangci-lint` v4 — Static analysis
- Coverage upload

**Release Pipeline** (`.github/workflows/release.yml`):
- Triggers on `v*` git tags
- Cross-platform builds via reusable workflow
- 5 target platforms: linux-x64, linux-arm64, macos-x64, macos-arm64, windows-x64
- Creates GitHub release with binaries + SHA256 checksums

**npm Publishing** (`.github/workflows/npm-publish.yml`):
- Publishes to npm registry
- Binary wrapper at `bin/mycelicmemory`

### Deployment Modes

| Mode | Command | Use Case |
|------|---------|----------|
| Foreground | `mycelicmemory start` | Development, debugging |
| Background | `mycelicmemory start --background` | Production daemon |
| MCP | `mycelicmemory --mcp` | Claude integration (stdin/stdout) |
| Desktop | `npm run dev` (dev) / installer (prod) | Visual interface |

### Health Monitoring

**Health endpoint:** `GET /api/v1/health`

```json
{
  "status": "ok",
  "database": true,
  "ollama": true,
  "qdrant": true,
  "version": "1.3.0",
  "uptime": "2h34m"
}
```

---

## 26. Development Guide & Conventions

### Prerequisites

| Requirement | Version | Purpose |
|-------------|---------|---------|
| Go | 1.23+ | Core binary |
| C Compiler | GCC/Clang | SQLite FTS5 (CGO) |
| Node.js | 16+ | Desktop app, npm wrapper |
| Git | 2.x | Version control |
| Ollama | Latest (optional) | AI features |
| Qdrant | Latest (optional) | Vector search |

### Development Workflow

```bash
# 1. Clone and setup
git clone https://github.com/MycelicMemory/mycelicmemory
cd mycelicmemory
go mod download

# 2. Build
make build                        # or CGO_ENABLED=1 go build ...

# 3. Run tests
make test                         # Standard tests
make test-coverage                # With coverage report
make test-verbose                 # Verbose with race detector

# 4. Start services
./mycelicmemory start             # REST API on port 3002

# 5. Desktop development
cd desktop && npm install && npm run dev

# 6. Lint
make lint                         # golangci-lint
make fmt                          # go fmt
make vet                          # go vet
```

### Code Organization Conventions

- **Package naming:** Lowercase, single-word (e.g., `database`, `memory`, `recall`)
- **File naming:** `operations_*.go` for domain-split operations files
- **Handler naming:** `handle<ToolName>` for MCP, `<verb><Resource>` for REST
- **Error handling:** Return errors up the chain; log at the boundary
- **Concurrency:** `sync.RWMutex` for database access; goroutines for async work
- **Configuration:** Viper for config, Cobra for CLI

### API Response Convention

All REST endpoints must use the response helpers:
```go
SuccessResponse(c, "message", data)    // 200
CreatedResponse(c, "message", data)    // 201
ErrorResponse(c, status, "message")    // 4xx/5xx
NotFoundResponse(c, "message")         // 404
```

### Testing Strategy

| Level | Tool | Coverage |
|-------|------|----------|
| Unit | `go test` | Service logic, database operations |
| Integration | `go test -tags integration` | API endpoints, pipeline |
| Race Detection | `go test -race` | Concurrency safety |
| E2E | GitHub Actions workflow | Full pipeline validation |

### Git Conventions

- **Branch naming:** `feat/`, `fix/`, `refactor/`, `docs/`
- **Commit style:** Conventional commits (`feat:`, `fix:`, `refactor:`, `docs:`)
- **PR workflow:** Branch from `main`, PR with description, CI must pass
- **Trunk-based:** Short-lived feature branches, frequent merges to `main`

### Key Files Quick Reference

| Purpose | File |
|---------|------|
| CLI entry | `cmd/mycelicmemory/main.go` |
| Config | `pkg/config/config.go` |
| Schema | `internal/database/schema.go` |
| Migrations | `internal/database/migrations.go` |
| Memory CRUD | `internal/database/operations.go` |
| MCP tools | `internal/mcp/handlers.go` |
| REST routes | `internal/api/server.go` |
| Recall engine | `internal/recall/engine.go` |
| Pipeline | `internal/pipeline/queue.go` |
| Desktop entry | `desktop/src/main/index.ts` |
| React app | `desktop/src/renderer/App.tsx` |
| API bridge | `desktop/src/renderer/api-bridge.ts` |
| Types | `desktop/src/shared/types.ts` |

---

## Appendix: TypeScript Type Definitions

### Core Types (desktop/src/shared/types.ts)

```typescript
interface Memory {
  id: string;
  content: string;
  domain?: string;
  source?: string;
  importance: number;         // 1-10
  tags?: string[];
  created_at: string;         // ISO 8601
  updated_at: string;
  session_id?: string;
  conversation_id?: string;
}

interface SearchOptions {
  query: string;
  search_type?: 'semantic' | 'keyword' | 'hybrid' | 'tags';
  domain?: string;
  tags?: string[];
  limit?: number;
  use_ai?: boolean;
}

interface MemoryRelationship {
  id: string;
  source_memory_id: string;
  target_memory_id: string;
  relationship_type: string;  // 7 valid types
  strength: number;           // 0.0-1.0
  context?: string;
  auto_generated: boolean;
  created_at: string;
}

interface ClaudeSession {
  id: string;
  session_id: string;
  project_path: string;
  project_hash: string;
  model?: string;
  title?: string;
  first_prompt?: string;
  summary?: string;
  created_at: string;
  message_count: number;
  tool_call_count: number;
  last_activity?: string;
}

interface HealthStatus {
  api: boolean;
  ollama: boolean;
  qdrant: boolean;
  database: boolean;
}

interface DashboardStats {
  memory_count: number;
  session_count: number;
  domain_count: number;
  this_week_count: number;
}
```

---

*This document is auto-generated from the MycelicMemory codebase at version 1.3.0. For the latest information, consult the source code and inline documentation.*
