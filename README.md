# Ultrathink

**AI-powered persistent memory system for Claude and other AI agents**

Ultrathink gives Claude persistent memory across conversations. Store knowledge, search semantically, build knowledge graphs, and get AI-powered insights from your memories.

**Free and open source** - no license keys, no subscriptions.

---

## Quick Start: Add Memory to Claude

### Option 1: Claude Code (Recommended)

Add ultrathink to your Claude Code MCP configuration:

**Step 1: Install ultrathink**
```bash
# Via npm (easiest)
npm install -g ultrathink

# Or build from source
git clone https://github.com/MycelicMemory/ultrathink.git
cd ultrathink && make dev-install
```

**Step 2: Add to Claude Code MCP config**

Edit `~/.claude/mcp.json`:
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

**Step 3: Restart Claude Code**
```bash
# Exit current session and start fresh
claude
```

You'll now have access to memory tools like `store_memory`, `search`, `analysis`, and more.

### Option 2: Claude Desktop

Add to your Claude Desktop config:

**macOS:** `~/Library/Application Support/Claude/claude_desktop_config.json`
**Windows:** `%APPDATA%\Claude\claude_desktop_config.json`
**Linux:** `~/.config/Claude/claude_desktop_config.json`

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

Restart Claude Desktop to enable the memory tools.

---

## What Can Claude Do With Memory?

Once connected, Claude can:

**Store memories:**
> "Remember that Go channels are typed conduits for communication between goroutines"

**Search memories:**
> "What do I know about concurrency patterns?"

**Analyze and summarize:**
> "Summarize what I've learned this week about Rust"

**Build knowledge graphs:**
> "How does this concept relate to what I learned yesterday?"

---

## Available MCP Tools

| Tool | Description |
|------|-------------|
| `store_memory` | Save new memories with tags, importance, and domain |
| `search` | Semantic, keyword, tag, or hybrid search |
| `analysis` | AI-powered Q&A, summarization, and pattern detection |
| `relationships` | Create and explore memory connections |
| `categories` | Organize memories into categories |
| `domains` | Group memories by knowledge domain |
| `sessions` | Track memory sessions |
| `stats` | View memory statistics |
| `get_memory_by_id` | Retrieve specific memory |
| `update_memory` | Modify existing memory |
| `delete_memory` | Remove a memory |

---

## Installation Options

### npm (Recommended for most users)

```bash
npm install -g ultrathink
ultrathink --version
```

### Build from Source

```bash
# Clone repository
git clone https://github.com/MycelicMemory/ultrathink.git
cd ultrathink

# Install dependencies and build
make deps
make dev-install

# Verify
ultrathink --version
```

### Alternative Install Methods

```bash
# Symlink (auto-updates on rebuild)
make link

# Install to GOPATH/bin
make install

# Build only (run as ./ultrathink)
make build
```

---

## Prerequisites

| Requirement | Purpose | Required? |
|-------------|---------|-----------|
| **Node.js 16+** | npm installation | For npm install |
| **Go 1.21+** | Build from source | For source build |
| **SQLite 3** | Database | Pre-installed on macOS/Linux |
| **Ollama** | AI embeddings & analysis | Optional but recommended |
| **Qdrant** | Vector semantic search | Optional |

### Setting Up Ollama (Recommended)

Ollama enables AI-powered semantic search and analysis:

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

### Setting Up Qdrant (Optional)

Qdrant provides high-performance vector search:

```bash
# Run with Docker
docker run -p 6333:6333 qdrant/qdrant
```

---

## CLI Usage

Ultrathink also works as a standalone CLI tool:

### Store Memories

```bash
ultrathink remember "Go channels are like pipes between goroutines"
ultrathink remember "Important concept" --importance 9 --tags learning,go
ultrathink remember "API design tip" --domain programming
```

### Search Memories

```bash
ultrathink search "concurrency"
ultrathink search "golang" --limit 20
ultrathink search "api" --domain programming
```

### AI Analysis

```bash
# Ask questions
ultrathink analyze "What have I learned about Go?"

# Summarize memories
ultrathink analyze --type summarize --timeframe week

# Find patterns
ultrathink analyze --type patterns --domain programming
```

### Manage Relationships

```bash
ultrathink relate <id1> <id2> --type similar
ultrathink find_related <id>
ultrathink map_graph <id> --depth 3
```

### Service Management

```bash
ultrathink start           # Start REST API daemon
ultrathink stop            # Stop daemon
ultrathink status          # Check status
ultrathink doctor          # System health check
```

---

## REST API

Start the daemon to enable the REST API:

```bash
ultrathink start
# API available at http://localhost:3099
```

### Key Endpoints

```http
POST   /api/v1/memories              # Create memory
GET    /api/v1/memories              # List memories
GET    /api/v1/memories/:id          # Get memory
PUT    /api/v1/memories/:id          # Update memory
DELETE /api/v1/memories/:id          # Delete memory
GET    /api/v1/memories/search       # Search memories
POST   /api/v1/analyze               # AI analysis
POST   /api/v1/relationships         # Create relationship
GET    /api/v1/memories/:id/graph    # Get relationship graph
GET    /api/v1/health                # Health check
```

See [API Documentation](#rest-api-reference) below for full details.

---

## Configuration

Configuration file: `~/.ultrathink/config.yaml`

```yaml
# Database
database:
  path: ~/.ultrathink/memories.db
  backup_interval: 24h
  max_backups: 7

# REST API
rest_api:
  enabled: true
  port: 3099
  host: localhost

# Session management
session:
  auto_generate: true
  strategy: git-directory  # Auto-detect project from git

# Ollama AI
ollama:
  enabled: true
  base_url: http://localhost:11434
  embedding_model: nomic-embed-text
  chat_model: qwen2.5:3b

# Qdrant vector search
qdrant:
  enabled: true
  url: http://localhost:6333
```

---

## Troubleshooting

### MCP Server Not Showing in Claude

1. **Verify installation:**
   ```bash
   which ultrathink
   ultrathink --version
   ```

2. **Check MCP config syntax:**
   ```bash
   # For Claude Code
   cat ~/.claude/mcp.json | python3 -m json.tool
   ```

3. **Restart Claude Code/Desktop** after config changes

4. **Test MCP mode manually:**
   ```bash
   echo '{"jsonrpc":"2.0","id":1,"method":"initialize"}' | ultrathink --mcp
   ```

### "Ollama is not available"

```bash
# Start Ollama service
ollama serve

# Verify it's running
curl http://localhost:11434/api/tags

# Pull required models
ollama pull nomic-embed-text
ollama pull qwen2.5:3b
```

### "Database locked"

```bash
ultrathink ps        # Check for running processes
ultrathink kill_all  # Kill all processes
```

### "Port already in use"

```bash
ultrathink start --port 3100  # Use different port
```

### System Health Check

```bash
ultrathink doctor
```

---

## REST API Reference

### Memory Endpoints

#### Create Memory
```http
POST /api/v1/memories
Content-Type: application/json

{
  "content": "Go channels are typed conduits",
  "importance": 8,
  "tags": ["go", "concurrency"],
  "domain": "programming"
}
```

#### Search Memories
```http
POST /api/v1/memories/search
Content-Type: application/json

{
  "query": "concurrency patterns",
  "limit": 10,
  "domain": "programming"
}
```

#### AI Analysis
```http
POST /api/v1/analyze
Content-Type: application/json

{
  "type": "question",
  "question": "What have I learned about Go?",
  "limit": 10
}
```

#### Create Relationship
```http
POST /api/v1/relationships
Content-Type: application/json

{
  "source_id": "uuid-1",
  "target_id": "uuid-2",
  "type": "similar",
  "strength": 0.8
}
```

#### Get Relationship Graph
```http
GET /api/v1/memories/:id/graph?depth=2
```

---

## Architecture

```
ultrathink/
├── cmd/ultrathink/          # CLI entry point
├── internal/
│   ├── api/                 # REST API (Gin)
│   ├── mcp/                 # MCP server (JSON-RPC 2.0)
│   ├── memory/              # Core memory service
│   ├── database/            # SQLite + FTS5
│   ├── ai/                  # Ollama integration
│   ├── relationships/       # Graph algorithms
│   └── vector/              # Qdrant client
├── pkg/config/              # Configuration
└── npm/                     # npm distribution
```

---

## Development

```bash
# Clone
git clone https://github.com/MycelicMemory/ultrathink.git
cd ultrathink

# Install dependencies
make deps

# Build
make build

# Run tests
make test

# Install for development (symlink)
make link
```

---

## License

MIT License - Free and open source.

---

## Links

- **Repository**: [github.com/MycelicMemory/ultrathink](https://github.com/MycelicMemory/ultrathink)
- **Issues**: [GitHub Issues](https://github.com/MycelicMemory/ultrathink/issues)
