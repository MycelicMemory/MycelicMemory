# Ultrathink

**AI-powered persistent memory system for intelligent agents**

Ultrathink provides persistent, searchable, and intelligently connected memory storage for AI agents and humans alike. It solves the "context amnesia" problem by enabling memory storage and retrieval across sessions.

## Quick Start

### Installation (Development)

```bash
# Clone and enter the repository
git clone https://github.com/MycelicMemory/ultrathink.git
cd ultrathink

# Install dependencies
make deps

# Build and install globally
make dev-install
```

After installation, `ultrathink` is available system-wide:

```bash
# Verify installation
ultrathink --version

# Check system status
ultrathink doctor

# Store your first memory
ultrathink remember "Go channels are typed conduits for communication between goroutines"

# Search memories
ultrathink search "Go channels"

# AI-powered analysis (requires Ollama)
ultrathink analyze "What have I learned about Go?"
```

### Alternative Installation Methods

```bash
# Option 1: Symlink (auto-updates on rebuild)
make link

# Option 2: Install to GOPATH/bin
make install

# Option 3: Build only (run as ./ultrathink)
make build
```

### Prerequisites

| Requirement | Purpose | Installation |
|-------------|---------|--------------|
| **Go 1.21+** | Build from source | [golang.org](https://golang.org) |
| **SQLite 3** | Database storage | Pre-installed on macOS/Linux |
| **Ollama** (optional) | AI features | [ollama.ai](https://ollama.ai) |
| **Qdrant** (optional) | Semantic search | [qdrant.tech](https://qdrant.tech) |

#### Setting up Ollama (for AI features)

```bash
# Install Ollama
brew install ollama  # macOS
# or download from https://ollama.ai

# Start Ollama service
ollama serve

# Pull required models
ollama pull nomic-embed-text   # For embeddings (768-dim)
ollama pull qwen2.5:3b         # For analysis/chat
```

#### Setting up Qdrant (for semantic search)

```bash
# Run with Docker
docker run -p 6333:6333 qdrant/qdrant
```

---

## CLI Reference

### Memory Commands

#### `remember` - Store a memory

```bash
ultrathink remember <content> [flags]

# Examples
ultrathink remember "Go channels are like pipes between goroutines"
ultrathink remember "Important meeting notes" --importance 9 --tags meeting,work
ultrathink remember "Python async tip" --domain programming --source docs
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--importance` | `-i` | 5 | Importance level (1-10) |
| `--tags` | `-t` | | Tags (comma-separated) |
| `--domain` | `-d` | | Knowledge domain |
| `--source` | `-s` | | Source of the memory |

#### `search` - Search memories

```bash
ultrathink search <query> [flags]

# Examples
ultrathink search "concurrency patterns"
ultrathink search "golang" --limit 20
ultrathink search "api" --domain programming --tags backend
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--limit` | `-l` | 10 | Maximum results |
| `--domain` | `-d` | | Filter by domain |
| `--tags` | `-t` | | Filter by tags |

#### `get` - Get memory by ID

```bash
ultrathink get <memory-id>

# Example
ultrathink get 550e8400-e29b-41d4-a716-446655440000
```

#### `list` - List all memories

```bash
ultrathink list [flags]

# Examples
ultrathink list
ultrathink list --limit 20 --offset 10
ultrathink list --domain programming
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--limit` | `-l` | 50 | Maximum results |
| `--offset` | `-o` | 0 | Pagination offset |
| `--domain` | `-d` | | Filter by domain |

#### `update` - Update a memory

```bash
ultrathink update <memory-id> [flags]

# Examples
ultrathink update <id> --content "Updated content"
ultrathink update <id> --importance 9
ultrathink update <id> --tags newtag1,newtag2
```

| Flag | Short | Description |
|------|-------|-------------|
| `--content` | | New content |
| `--importance` | `-i` | New importance (1-10) |
| `--tags` | `-t` | New tags |
| `--domain` | `-d` | New domain |

#### `forget` - Delete a memory

```bash
ultrathink forget <memory-id>

# Example
ultrathink forget 550e8400-e29b-41d4-a716-446655440000
```

---

### Relationship Commands

#### `relate` - Create relationship between memories

```bash
ultrathink relate <source-id> <target-id> [flags]

# Examples
ultrathink relate <id1> <id2> --type similar
ultrathink relate <id1> <id2> --type enables --strength 0.9
ultrathink relate <id1> <id2> --type references --context "Both discuss async patterns"
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--type` | `-t` | similar | Relationship type |
| `--strength` | `-s` | 0.5 | Strength (0.0-1.0) |
| `--context` | `-c` | | Explanation context |

**Relationship Types:**
- `references` - One memory references another
- `contradicts` - Memories contradict each other
- `expands` - One expands on another
- `similar` - Memories are similar
- `sequential` - One follows another
- `causes` - One causes another
- `enables` - One enables another

#### `find_related` - Find related memories

```bash
ultrathink find_related <memory-id> [flags]

# Examples
ultrathink find_related <id>
ultrathink find_related <id> --limit 10 --min-strength 0.7
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--limit` | `-l` | 10 | Maximum results |
| `--min-strength` | | 0.0 | Minimum relationship strength |
| `--type` | `-t` | | Filter by relationship type |

#### `map_graph` - Visualize relationship graph

```bash
ultrathink map_graph <memory-id> [flags]

# Examples
ultrathink map_graph <id>
ultrathink map_graph <id> --depth 3
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--depth` | `-d` | 2 | Graph traversal depth (1-5) |

#### `discover` - AI relationship discovery

```bash
ultrathink discover [flags]

# Examples
ultrathink discover
ultrathink discover --limit 20 --min-strength 0.7
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--limit` | `-l` | 10 | Maximum relationships to discover |
| `--min-strength` | | 0.5 | Minimum strength threshold |

---

### AI Analysis Commands

#### `analyze` - AI-powered analysis

```bash
ultrathink analyze [question] [flags]

# Question answering
ultrathink analyze "What have I learned about Go?"
ultrathink analyze "How do channels relate to goroutines?"

# Summarization
ultrathink analyze --type summarize --timeframe week
ultrathink analyze --type summarize --timeframe month --domain programming

# Pattern detection
ultrathink analyze "concurrency" --type patterns
ultrathink analyze --type patterns --limit 50

# Temporal analysis (learning progression)
ultrathink analyze --type temporal --timeframe month
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--type` | `-t` | question | Analysis type |
| `--timeframe` | | all | Time period (today, week, month, all) |
| `--limit` | `-l` | 10 | Max memories to analyze |
| `--domain` | `-d` | | Filter by domain |

**Analysis Types:**
- `question` - Answer questions based on memories (requires query)
- `summarize` - Summarize memories over timeframe
- `patterns` - Find patterns in memories
- `temporal` - Analyze learning progression over time

---

### Organization Commands

#### Categories

```bash
# List all categories
ultrathink list_categories

# Create a category
ultrathink create_category <name> [flags]
ultrathink create_category "technical-docs" --description "Technical documentation"
ultrathink create_category "subtopic" --parent <parent-category-id>

# Category statistics
ultrathink category_stats

# Auto-categorize a memory using AI
ultrathink categorize <memory-id>
```

#### Domains

```bash
# List all domains
ultrathink list_domains

# Create a domain
ultrathink create_domain <name> [flags]
ultrathink create_domain "machine-learning" --description "ML and AI topics"

# Domain statistics
ultrathink domain_stats <domain-name>
```

#### Sessions

```bash
# List all sessions
ultrathink list_sessions

# Current session statistics
ultrathink session_stats
```

---

### Service Commands

#### `start` - Start the daemon

```bash
ultrathink start [flags]

# Examples
ultrathink start                    # Start with defaults
ultrathink start --port 3100        # Custom port
ultrathink start --background       # Run in background
ultrathink start --host 0.0.0.0     # Bind to all interfaces
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--port` | `-p` | 3099 | REST API port |
| `--host` | | localhost | Host to bind to |
| `--background` | `-b` | false | Run in background |

#### `stop` - Stop the daemon

```bash
ultrathink stop
```

#### `status` - Check daemon status

```bash
ultrathink status

# Output includes:
# - Running state and PID
# - Uptime
# - REST API endpoint
# - Database statistics
```

#### Process management

```bash
# List running Ultrathink processes
ultrathink ps

# Kill specific process
ultrathink kill <pid>

# Kill all Ultrathink processes
ultrathink kill_all
```

---

### System Commands

#### `doctor` - System health check

```bash
ultrathink doctor

# Checks:
# - Configuration validity
# - Database connectivity
# - Ollama availability and models
# - Qdrant connectivity
```

#### `setup` - Run setup wizard

```bash
ultrathink setup
```

#### `validate` - Validate installation

```bash
ultrathink validate
```

#### `install` - Install integrations

```bash
# Install MCP integration for Claude
ultrathink install mcp

# Install shell completions
ultrathink install shell
```

---

### Global Flags

These flags work with any command:

| Flag | Short | Description |
|------|-------|-------------|
| `--config` | `-c` | Config file path |
| `--log_level` | | Log level (debug, info, warn, error) |
| `--quiet` | | Suppress output |
| `--mcp` | | Run as MCP server |
| `--help` | `-h` | Show help |
| `--version` | `-v` | Show version |

---

## REST API Reference

Start the daemon to enable the REST API:

```bash
ultrathink start
# API available at http://localhost:3099
```

### Memory Endpoints

#### Create Memory
```http
POST /api/v1/memories
Content-Type: application/json

{
  "content": "Go channels are typed conduits",
  "importance": 8,
  "tags": ["go", "concurrency"],
  "domain": "programming",
  "source": "documentation"
}
```

#### List Memories
```http
GET /api/v1/memories?limit=20&offset=0&domain=programming
```

#### Get Memory
```http
GET /api/v1/memories/:id
```

#### Update Memory
```http
PUT /api/v1/memories/:id
Content-Type: application/json

{
  "content": "Updated content",
  "importance": 9,
  "tags": ["updated", "tags"]
}
```

#### Delete Memory
```http
DELETE /api/v1/memories/:id
```

#### Memory Statistics
```http
GET /api/v1/memories/stats
```

### Search Endpoints

#### Basic Search
```http
GET /api/v1/memories/search?query=concurrency&limit=10
```

```http
POST /api/v1/memories/search
Content-Type: application/json

{
  "query": "concurrency patterns",
  "limit": 10,
  "domain": "programming",
  "tags": ["go"]
}
```

#### Intelligent Search
```http
POST /api/v1/memories/search/intelligent
Content-Type: application/json

{
  "query": "how do channels work",
  "max_tokens": 500,
  "format": "intelligent"
}
```

#### Tag Search
```http
POST /api/v1/search/tags
Content-Type: application/json

{
  "tags": ["go", "concurrency"],
  "limit": 20
}
```

#### Date Range Search
```http
POST /api/v1/search/date-range
Content-Type: application/json

{
  "start_date": "2024-01-01",
  "end_date": "2024-12-31",
  "limit": 50
}
```

### Relationship Endpoints

#### Create Relationship
```http
POST /api/v1/relationships
Content-Type: application/json

{
  "source_id": "uuid-1",
  "target_id": "uuid-2",
  "type": "similar",
  "strength": 0.8,
  "context": "Both discuss async patterns"
}
```

#### Find Related Memories
```http
GET /api/v1/memories/:id/related?limit=10&min_strength=0.5
```

#### Get Relationship Graph
```http
GET /api/v1/memories/:id/graph?depth=2
```

#### Discover Relationships
```http
POST /api/v1/relationships/discover
Content-Type: application/json

{
  "limit": 10,
  "min_strength": 0.7
}
```

### AI Analysis Endpoint

```http
POST /api/v1/analyze
Content-Type: application/json

{
  "type": "question",
  "question": "What have I learned about Go?",
  "limit": 10,
  "domain": "programming"
}
```

**Analysis types:** `question`, `summarize`, `patterns`, `temporal`

### Category Endpoints

```http
# List categories
GET /api/v1/categories

# Create category
POST /api/v1/categories
{
  "name": "technical-docs",
  "description": "Technical documentation"
}

# Category statistics
GET /api/v1/categories/stats

# Auto-categorize memory
POST /api/v1/memories/:id/categorize
```

### Domain Endpoints

```http
# List domains
GET /api/v1/domains

# Create domain
POST /api/v1/domains
{
  "name": "machine-learning",
  "description": "ML and AI topics"
}

# Domain statistics
GET /api/v1/domains/:domain/stats
```

### Session Endpoints

```http
# List sessions
GET /api/v1/sessions

# Session statistics
GET /api/v1/sessions/stats
```

### System Endpoints

```http
# Health check
GET /api/v1/health

# System statistics
GET /api/v1/stats
```

---

## MCP Integration

Ultrathink works as an MCP (Model Context Protocol) server for AI assistants like Claude.

### Claude Desktop Setup

Add to your `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "ultrathink": {
      "command": "ultrathink",
      "args": ["--mcp"]
    }
  }
}
```

**Config file locations:**
- macOS: `~/Library/Application Support/Claude/claude_desktop_config.json`
- Windows: `%APPDATA%\Claude\claude_desktop_config.json`
- Linux: `~/.config/Claude/claude_desktop_config.json`

### Claude Code Setup

```bash
claude mcp add ultrathink --transport stdio -- ultrathink --mcp
```

### Available MCP Tools

Once connected, Claude can use these tools:

#### `store_memory`
Store a new memory with metadata.

```json
{
  "content": "Go channels are typed conduits",
  "importance": 8,
  "tags": ["go", "concurrency"],
  "domain": "programming"
}
```

#### `search`
Search memories with multiple modes.

```json
{
  "query": "concurrency patterns",
  "search_type": "semantic",
  "limit": 10,
  "domain": "programming",
  "response_format": "concise"
}
```

**Search types:** `semantic`, `keyword`, `tags`, `hybrid`
**Response formats:** `detailed`, `concise`, `ids_only`, `summary`

#### `analysis`
AI-powered analysis of memories.

```json
{
  "analysis_type": "question",
  "question": "What have I learned about Go?",
  "limit": 10
}
```

**Analysis types:** `question`, `summarize`, `analyze`, `temporal_patterns`

#### `relationships`
Manage memory relationships.

```json
{
  "relationship_type": "find_related",
  "memory_id": "uuid",
  "limit": 10
}
```

**Operations:** `find_related`, `discover`, `create`, `map_graph`

#### `categories`
Manage memory categories.

```json
{
  "categories_type": "list"
}
```

**Operations:** `list`, `create`, `categorize`

#### `domains`
Manage knowledge domains.

```json
{
  "domains_type": "list"
}
```

**Operations:** `list`, `create`, `stats`

#### `sessions`
Session management.

```json
{
  "sessions_type": "list"
}
```

**Operations:** `list`, `stats`

#### `stats`
Get system statistics.

```json
{
  "stats_type": "session"
}
```

**Types:** `session`, `domain`, `category`

#### `get_memory_by_id`
Retrieve a specific memory.

```json
{
  "id": "uuid"
}
```

#### `update_memory`
Update an existing memory.

```json
{
  "id": "uuid",
  "content": "Updated content",
  "importance": 9
}
```

#### `delete_memory`
Delete a memory.

```json
{
  "id": "uuid"
}
```

---

## Configuration

Configuration file: `~/.ultrathink/config.yaml`

```yaml
# Database settings
database:
  path: ~/.ultrathink/memories.db
  backup_interval: 24h
  max_backups: 7
  auto_migrate: true

# REST API settings
rest_api:
  enabled: true
  port: 3099
  host: localhost
  auto_port: true      # Find available port if 3099 is busy
  cors: true

# Session management
session:
  auto_generate: true
  strategy: git-directory  # or "manual"

# Logging
logging:
  level: info            # debug, info, warn, error
  format: console        # console, json

# Ollama AI settings
ollama:
  enabled: true
  auto_detect: true
  base_url: http://localhost:11434
  embedding_model: nomic-embed-text
  chat_model: qwen2.5:3b

# Qdrant vector database (optional)
qdrant:
  enabled: true
  auto_detect: true
  url: http://localhost:6333
```

### Session Strategies

**git-directory** (default): Session ID is derived from the git repository directory, providing automatic isolation per project.

**manual**: Use explicit session IDs for full control.

---

## Architecture

```
ultrathink/
├── cmd/ultrathink/          # CLI entry point
│   ├── main.go              # Main entry
│   ├── root.go              # Root command + MCP mode
│   ├── cmd_memory.go        # remember, search, get, list, update, forget
│   ├── cmd_relationships.go # relate, find_related, map_graph, discover
│   ├── cmd_analyze.go       # analyze (AI analysis)
│   ├── cmd_organization.go  # categories, domains, sessions
│   ├── cmd_service.go       # start, stop, status, ps, kill
│   └── cmd_doctor.go        # doctor, setup, validate
├── internal/
│   ├── api/                 # REST API server (Gin)
│   ├── mcp/                 # MCP server (JSON-RPC 2.0)
│   ├── memory/              # Core memory service
│   ├── database/            # SQLite + FTS5
│   ├── search/              # Search engine
│   ├── ai/                  # Ollama integration
│   ├── relationships/       # Graph algorithms
│   ├── vector/              # Qdrant client
│   └── daemon/              # Process management
├── pkg/config/              # Configuration
└── npm/                     # npm distribution package
```

---

## Development

### Building

```bash
# Build binary
make build

# Build for all platforms
make build-all

# Run tests
make test

# Run with coverage
make test-coverage

# Format code
make fmt

# Run linters
make lint
```

### Development Workflow

```bash
# Install globally for testing
make dev-install

# Or use symlink (auto-updates on rebuild)
make link

# After code changes
make build  # Binary updates automatically if using symlink

# Uninstall when done
make uninstall
```

### Running Tests

```bash
# All tests
make test

# With coverage report
make test-coverage

# Verbose output with race detection
make test-verbose

# Specific package
go test -tags fts5 ./internal/database/...
```

---

## Troubleshooting

### Common Issues

**"Ollama is not available"**
```bash
# Start Ollama service
ollama serve

# Verify it's running
curl http://localhost:11434/api/tags

# Pull required models
ollama pull nomic-embed-text
ollama pull qwen2.5:3b
```

**"Database locked"**
```bash
# Check for running processes
ultrathink ps

# Kill all processes
ultrathink kill_all
```

**"Port already in use"**
```bash
# Use different port
ultrathink start --port 3100

# Or enable auto_port in config
```

### Checking System Status

```bash
# Full system check
ultrathink doctor

# Check daemon status
ultrathink status

# List processes
ultrathink ps
```

---

## License

MIT License - See [LICENSE](LICENSE) file for details.

---

## Links

- **Repository**: [github.com/MycelicMemory/ultrathink](https://github.com/MycelicMemory/ultrathink)
- **Issues**: [GitHub Issues](https://github.com/MycelicMemory/ultrathink/issues)
