# Ultrathink

<p align="center">
  <strong>AI-powered persistent memory system for Claude and other AI agents</strong>
</p>

<p align="center">
  <a href="#quick-start">Quick Start</a> •
  <a href="#features">Features</a> •
  <a href="#use-cases">Use Cases</a> •
  <a href="#hooks">Auto-Memory Hooks</a> •
  <a href="docs/QUICKSTART.md">Full Setup Guide</a>
</p>

---

Ultrathink gives Claude persistent memory across conversations. Store knowledge, search semantically, build knowledge graphs, and get AI-powered insights from your memories.

**Free and open source** - no license keys, no subscriptions.

---

## Quick Start

### 1. Install

```bash
# Via npm (easiest)
npm install -g ultrathink

# Or build from source
git clone https://github.com/MycelicMemory/ultrathink.git
cd ultrathink && make dev-install
```

### 2. Connect to Claude Code

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

### 3. Restart Claude Code

```bash
claude
```

### 4. Start Using Memory

Ask Claude:
- "Remember that Go channels are typed conduits for goroutine communication"
- "What do I know about concurrency?"
- "Summarize what I learned this week"

---

## Features

### Core Memory Operations

| Tool | Description |
|------|-------------|
| `store_memory` | Save memories with importance (1-10), tags, and domain |
| `search` | Semantic, keyword, tag, or hybrid search |
| `get_memory_by_id` | Retrieve specific memory by UUID |
| `update_memory` | Modify content, importance, or tags |
| `delete_memory` | Remove a memory permanently |

### AI-Powered Analysis

| Tool | Description |
|------|-------------|
| `analysis(question)` | Ask natural language questions about your memories |
| `analysis(summarize)` | Generate summaries across timeframes |
| `analysis(analyze)` | Detect patterns and themes |
| `analysis(temporal_patterns)` | Track learning progression over time |

### Knowledge Organization

| Tool | Description |
|------|-------------|
| `relationships` | Create/discover connections between memories |
| `categories` | Organize memories hierarchically |
| `domains` | Group by knowledge domain (programming, devops, etc.) |
| `sessions` | Track memories by session/project |
| `stats` | View memory statistics and metrics |

---

## Use Cases

### For Developers

1. **Code Decision Journal** - Record architectural decisions with rationale
2. **Debugging Knowledge Base** - Store gotchas, bugs, and their solutions
3. **API Reference Cache** - Remember API patterns you frequently use
4. **Configuration Vault** - Track environment-specific settings
5. **Learning Log** - Document new concepts as you learn them

### For Teams

6. **Project Context** - Share project-specific knowledge across sessions
7. **Onboarding Assistant** - Build searchable codebase knowledge
8. **Best Practices Library** - Curate team coding standards
9. **Incident Postmortems** - Store and search past incidents

### For Research

10. **Literature Notes** - Store paper summaries with tags
11. **Concept Relationships** - Map how ideas connect
12. **Progress Tracking** - Monitor learning over time
13. **Citation Manager** - Track sources and references

### For Personal Knowledge

14. **Second Brain** - Build a searchable knowledge base
15. **Daily Learnings** - Capture TILs with automatic tagging

See [docs/USE_CASES.md](docs/USE_CASES.md) for detailed examples of each use case.

---

## Auto-Memory Hooks

Ultrathink can automatically capture knowledge from your Claude Code sessions using hooks.

### What Gets Captured

- **Code Decisions** - "Because...", "Decided to...", "Instead of..."
- **Bug Fixes** - "The bug was...", "Fixed by..."
- **Best Practices** - "Should always/never...", "Gotcha..."
- **Config Changes** - Edits to `.env`, `config.*`, `docker-compose.yml`, etc.

### Quick Hook Setup

```bash
# Create hooks directory
mkdir -p ~/.claude/hooks

# Download hook scripts
curl -o ~/.claude/hooks/ultrathink-memory-capture.py \
  https://raw.githubusercontent.com/MycelicMemory/ultrathink/main/hooks/ultrathink-memory-capture.py
curl -o ~/.claude/hooks/ultrathink-context-loader.py \
  https://raw.githubusercontent.com/MycelicMemory/ultrathink/main/hooks/ultrathink-context-loader.py

# Make executable
chmod +x ~/.claude/hooks/*.py
```

Add to `~/.claude/settings.json`:

```json
{
  "hooks": {
    "PostToolUse": [
      {
        "matcher": "Edit|Write",
        "hooks": [{"type": "command", "command": "python3 ~/.claude/hooks/ultrathink-memory-capture.py"}]
      }
    ],
    "Stop": [
      {
        "matcher": "",
        "hooks": [{"type": "command", "command": "python3 ~/.claude/hooks/ultrathink-memory-capture.py"}]
      }
    ],
    "SessionStart": [
      {
        "matcher": "",
        "hooks": [{"type": "command", "command": "python3 ~/.claude/hooks/ultrathink-context-loader.py"}]
      }
    ]
  }
}
```

See [docs/HOOKS.md](docs/HOOKS.md) for full hook documentation.

---

## CLI Usage

Ultrathink also works as a standalone CLI:

```bash
# Store memories
ultrathink remember "Go interfaces are satisfied implicitly"
ultrathink remember "Important insight" --importance 9 --tags learning,go

# Search
ultrathink search "concurrency patterns"
ultrathink search "golang" --domain programming

# Analyze
ultrathink analyze "What have I learned about testing?"
ultrathink analyze --type summarize --timeframe week

# Relationships
ultrathink relate <id1> <id2> --type similar
ultrathink find_related <id>

# Service management
ultrathink start    # Start REST API daemon
ultrathink stop     # Stop daemon
ultrathink doctor   # Health check
```

---

## REST API

Start the daemon for REST API access:

```bash
ultrathink start
# API at http://localhost:3099
```

### Key Endpoints

```http
POST   /api/v1/memories              # Create memory
GET    /api/v1/memories              # List memories
GET    /api/v1/memories/:id          # Get memory
PUT    /api/v1/memories/:id          # Update memory
DELETE /api/v1/memories/:id          # Delete memory
POST   /api/v1/memories/search       # Search memories
POST   /api/v1/analyze               # AI analysis
POST   /api/v1/relationships         # Create relationship
GET    /api/v1/memories/:id/graph    # Relationship graph
GET    /api/v1/health                # Health check
```

---

## Configuration

Config file: `~/.ultrathink/config.yaml`

```yaml
database:
  path: ~/.ultrathink/memories.db
  backup_interval: 24h

rest_api:
  port: 3099
  host: localhost

session:
  auto_generate: true
  strategy: git-directory

ollama:
  enabled: true
  base_url: http://localhost:11434
  embedding_model: nomic-embed-text
  chat_model: qwen2.5:3b

qdrant:
  enabled: true
  url: http://localhost:6333
```

---

## Optional Dependencies

| Component | Purpose | Installation |
|-----------|---------|--------------|
| **Ollama** | AI embeddings & analysis | `brew install ollama && ollama serve` |
| **Qdrant** | Vector semantic search | `docker run -p 6333:6333 qdrant/qdrant` |

### Setup Ollama

```bash
ollama serve
ollama pull nomic-embed-text   # Embeddings
ollama pull qwen2.5:3b         # Analysis
```

---

## Troubleshooting

### MCP Not Available

```bash
# Verify installation
which ultrathink && ultrathink --version

# Test MCP mode
echo '{"jsonrpc":"2.0","id":1,"method":"initialize"}' | ultrathink --mcp

# Validate config
cat ~/.claude/mcp.json | python3 -m json.tool
```

### Ollama Issues

```bash
ollama serve              # Start service
curl localhost:11434/api/tags  # Verify
ollama pull nomic-embed-text   # Get models
```

### Full Health Check

```bash
ultrathink doctor
```

---

## Architecture

```
ultrathink/
├── cmd/ultrathink/          # CLI entry point
├── internal/
│   ├── mcp/                 # MCP server (JSON-RPC 2.0)
│   ├── api/                 # REST API (Gin)
│   ├── memory/              # Core memory service
│   ├── database/            # SQLite + FTS5
│   ├── ai/                  # Ollama integration
│   ├── relationships/       # Graph algorithms
│   └── vector/              # Qdrant client
├── pkg/config/              # Configuration
├── hooks/                   # Claude Code hooks
└── npm/                     # npm distribution
```

---

## Documentation

- [Quick Start Guide](docs/QUICKSTART.md) - Get up and running in 5 minutes
- [Use Cases](docs/USE_CASES.md) - 15 detailed examples with code
- [Hooks Setup](docs/HOOKS.md) - Automatic memory capture
- [API Reference](docs/API.md) - Full REST API documentation
- [Contributing](CONTRIBUTING.md) - Development guide

---

## License

MIT License - Free and open source.

---

## Links

- **Repository**: [github.com/MycelicMemory/ultrathink](https://github.com/MycelicMemory/ultrathink)
- **Issues**: [GitHub Issues](https://github.com/MycelicMemory/ultrathink/issues)
- **npm**: [npmjs.com/package/ultrathink](https://npmjs.com/package/ultrathink)
