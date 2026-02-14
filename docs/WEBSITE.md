# MycelicMemory

**Persistent memory for Claude and AI agents. Store knowledge, search semantically, and build a personal knowledge graph — all running locally on your machine.**

Free and open source. No cloud accounts, no API keys, no subscriptions.

---

## What It Does

MycelicMemory gives Claude (and other AI tools) a long-term memory that persists across conversations. Instead of starting fresh every time, Claude can:

- **Remember** decisions, preferences, debugging solutions, and project context
- **Search** past knowledge using semantic similarity, keywords, or tags
- **Build connections** between related memories automatically
- **Analyze** patterns in what you've learned over time

It works through MCP (Model Context Protocol) — the standard way to give Claude access to external tools. Once connected, Claude stores and retrieves memories automatically as you work.

---

## Quick Start

### 1. Install

```bash
npm install -g mycelicmemory
```

The first run downloads the correct binary for your platform (macOS, Linux, or Windows).

Verify it works:

```bash
mycelicmemory --version
mycelicmemory doctor
```

### 2. Connect to Claude Code

Edit `~/.claude/mcp.json`:

**macOS / Linux:**
```json
{
  "mcpServers": {
    "mycelicmemory": {
      "command": "mycelicmemory",
      "args": ["--mcp"]
    }
  }
}
```

**Windows** (use the direct binary path to avoid pipe issues):
```json
{
  "mcpServers": {
    "mycelicmemory": {
      "command": "C:\\Users\\YOUR_USERNAME\\AppData\\Roaming\\npm\\node_modules\\mycelicmemory\\bin\\mycelicmemory-windows-x64.exe",
      "args": ["--mcp"]
    }
  }
}
```

Find your binary path with:
```powershell
$BinaryPath = Join-Path (npm root -g) "mycelicmemory\bin\mycelicmemory-windows-x64.exe"
Write-Host $BinaryPath
```

### 3. Connect to Claude Desktop (optional)

Add the same `mcpServers` block to your Claude Desktop config:

| Platform | Config File |
|----------|-------------|
| macOS | `~/Library/Application Support/Claude/claude_desktop_config.json` |
| Windows | `%APPDATA%\Claude\claude_desktop_config.json` |
| Linux | `~/.config/Claude/claude_desktop_config.json` |

### 4. Verify

Restart Claude Code, then run `/mcp`. You should see `mycelicmemory` listed. Try:

> "Remember that I prefer TypeScript over JavaScript for all new projects"

> "What do you remember about my preferences?"

---

## How It Works

MycelicMemory runs as a local service on your machine. Everything stays on your hardware — no data leaves your computer.

```
Claude Code / Desktop
        |
        | MCP Protocol (stdin/stdout)
        v
  MycelicMemory Binary
        |
        |--- SQLite Database (memories, relationships, sessions)
        |--- Ollama (optional: semantic search, AI analysis)
        |--- Qdrant (optional: vector similarity search)
```

**Without Ollama:** Keyword search, tag filtering, date-range queries. All core memory operations work.

**With Ollama:** Adds semantic search (find memories by meaning, not just keywords), AI-powered analysis, and automatic relationship discovery.

---

## Features

### Memory Storage

Store knowledge with metadata that makes it searchable later:

```
"Remember: We chose PostgreSQL over MongoDB because we need ACID transactions
for the payment service. Importance: 9, tags: architecture, database, decision"
```

Each memory can have:
- **Content** — the knowledge itself
- **Importance** (1-10) — how critical this information is
- **Tags** — searchable labels (e.g., `debugging`, `go`, `architecture`)
- **Domain** — knowledge area (e.g., `databases`, `frontend`)
- **Source** — where this came from

### Search

Multiple search strategies to find the right memory:

| Type | How It Works | Example |
|------|-------------|---------|
| **Semantic** | Finds memories by meaning using AI embeddings | "concurrency patterns" finds memories about goroutines, async/await, threading |
| **Keyword** | Traditional text matching | "PostgreSQL" finds exact mentions |
| **Tags** | Filter by labels | `tags: debugging, go` |
| **Hybrid** | Combines semantic + keyword | Best of both approaches |
| **Date range** | Filter by time period | "memories from this week" |

### Knowledge Graph

Memories don't exist in isolation. MycelicMemory tracks relationships between them:

- **References** — one memory cites another
- **Expands** — elaborates on a previous idea
- **Contradicts** — conflicts with existing knowledge
- **Similar** — related concepts
- **Causes / Enables** — causal chains
- **Sequential** — ordered steps or progressions

Claude can discover these relationships automatically, or you can create them explicitly.

### AI Analysis (requires Ollama)

Ask questions across your entire knowledge base:

- **Question answering** — "What have I learned about error handling in Go?"
- **Summarization** — "Summarize my memories from this week"
- **Pattern detection** — "What themes appear in my debugging notes?"
- **Temporal analysis** — "How has my understanding of React evolved?"

### Claude Code Chat History

Ingest and search your Claude Code conversation history:

- Import conversations from `~/.claude/` JSONL files
- Browse sessions by project
- Search across all past conversations
- Trace any memory back to the conversation that created it
- Generate summary memories from long sessions

### Auto-Memory Prompt

MycelicMemory includes a built-in MCP prompt (`auto-memory`) that instructs Claude to:
1. **Search first** — check for relevant memories at the start of each conversation
2. **Store continuously** — save decisions, debugging insights, preferences, and learnings automatically
3. **Build relationships** — connect new memories to existing ones

This means Claude manages your knowledge base proactively without you needing to say "remember this" every time.

### Data Source Ingestion

Ingest knowledge from external sources into your memory system:

- Register data sources with custom configurations
- Bulk import with deduplication
- Checkpoint-based resumable ingestion
- Sync history and status tracking

### Organization

Structure your knowledge:

- **Domains** — broad knowledge areas (databases, frontend, devops)
- **Categories** — hierarchical labels with AI-powered auto-categorization
- **Sessions** — automatic grouping by work session
- **Tags** — flexible cross-cutting labels

---

## Optional: AI-Powered Features

MycelicMemory works without any AI services. Keyword search, tag filtering, memory storage, and the knowledge graph all work out of the box.

For semantic search and AI analysis, install Ollama:

### Install Ollama

**macOS:**
```bash
brew install ollama
ollama serve
```

**Linux:**
```bash
curl -fsSL https://ollama.ai/install.sh | sh
ollama serve
```

**Windows:**
Download from [ollama.ai](https://ollama.ai/download)

### Pull Models

```bash
ollama pull nomic-embed-text   # Semantic search (embeddings)
ollama pull qwen2.5:3b         # AI analysis (chat)
```

### Verify

```bash
mycelicmemory doctor
# Should show: Ollama: Available
```

### Optional: Qdrant Vector Database

For high-performance semantic search on large memory collections, add Qdrant:

```bash
docker run -d --name qdrant -p 6333:6333 qdrant/qdrant
```

Without Qdrant, semantic search still works through Ollama — Qdrant adds faster similarity lookups for large datasets.

---

## Desktop App

A standalone Electron application for browsing and managing memories visually.

### Pages

- **Dashboard** — Memory stats, service health, quick actions, recent activity charts
- **Memory Browser** — Search, filter, edit, and delete memories with detail panels
- **Claude Sessions** — Browse Claude Code conversation history by project, view messages and tool calls
- **Knowledge Graph** — Interactive network visualization of memory relationships (vis-network)
- **Settings** — Configure API endpoints, Ollama models, MCP setup guide, Qdrant, theme

### Install

```bash
cd desktop
npm install
npm run dev
```

### Package for Distribution

```bash
npm run package:win    # Windows installer (.exe)
npm run package:mac    # macOS disk image (.dmg)
npm run package:linux  # Linux AppImage
```

The desktop app automatically starts and manages the MycelicMemory backend, Ollama, and Qdrant services.

---

## CLI Usage

MycelicMemory also works as a standalone command-line tool:

```bash
# Store a memory
mycelicmemory remember "React 18 strict mode runs useEffect twice in dev — this is intentional" \
  --importance 7 --tags debugging,react,gotcha

# Search memories
mycelicmemory search "concurrency patterns" --limit 5

# AI analysis
mycelicmemory analyze "What have I learned about testing?" --type question

# List memories
mycelicmemory list --domain databases --limit 20

# Relationship management
mycelicmemory relate <id1> <id2> --type expands --strength 0.8
mycelicmemory discover  # AI-powered relationship discovery

# Service management
mycelicmemory start    # Start the API daemon
mycelicmemory stop     # Stop the daemon
mycelicmemory doctor   # Health check and diagnostics
mycelicmemory status   # Show daemon status
```

---

## Automatic Memory Capture (Hooks)

Set up Claude Code hooks to capture knowledge automatically — no manual "remember this" needed.

### Install Hooks

```bash
mkdir -p ~/.claude/hooks

curl -o ~/.claude/hooks/mycelicmemory-memory-capture.py \
  https://raw.githubusercontent.com/MycelicMemory/mycelicmemory/main/hooks/mycelicmemory-memory-capture.py

curl -o ~/.claude/hooks/mycelicmemory-context-loader.py \
  https://raw.githubusercontent.com/MycelicMemory/mycelicmemory/main/hooks/mycelicmemory-context-loader.py

chmod +x ~/.claude/hooks/*.py
```

### What the Hooks Do

- **Memory Capture** — Automatically stores decisions, debugging insights, and learnings from your conversations
- **Context Loader** — Loads relevant memories at the start of new sessions so Claude has context from previous work

See the [Hooks Guide](docs/HOOKS.md) for configuration details.

---

## Configuration

Config file: `~/.mycelicmemory/config.yaml`

```yaml
database:
  path: ~/.mycelicmemory/memories.db

rest_api:
  port: 3099
  host: localhost

ollama:
  enabled: true
  base_url: http://localhost:11434
  embedding_model: nomic-embed-text
  chat_model: qwen2.5:3b

qdrant:
  enabled: false
  url: http://localhost:6333
```

All settings are also configurable through the Desktop App's Settings page.

---

## REST API

MycelicMemory exposes a full REST API at `http://localhost:3099/api/v1/`:

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/memories` | POST | Create a memory |
| `/memories` | GET | List memories (paginated) |
| `/memories/:id` | GET | Get memory by ID |
| `/memories/:id` | PUT | Update memory |
| `/memories/:id` | DELETE | Delete memory |
| `/memories/search` | POST | Search memories |
| `/memories/search/intelligent` | POST | AI-powered intelligent search |
| `/memories/:id/related` | GET | Find related memories |
| `/memories/:id/graph` | GET | Get knowledge graph |
| `/analyze` | POST | AI analysis (question, summarize, patterns) |
| `/relationships` | POST | Create relationship |
| `/relationships/discover` | POST | AI relationship discovery |
| `/categories` | GET/POST | List/create categories |
| `/domains` | GET/POST | List/create domains |
| `/sessions` | GET | List sessions |
| `/stats` | GET | System statistics |
| `/health` | GET | Health check |
| `/sources` | GET/POST | Data source management |
| `/chats/ingest` | POST | Ingest Claude Code conversations |
| `/chats` | GET | List chat sessions |
| `/chats/search` | GET | Search chat sessions |

Start the API server:
```bash
mycelicmemory start
```

---

## MCP Tools Reference

These are the tools Claude sees when MycelicMemory is connected:

| Tool | Description |
|------|-------------|
| `store_memory` | Save a memory with importance, tags, domain, and source |
| `search` | Search memories (semantic, keyword, tag, hybrid, date-range) |
| `get_memory_by_id` | Retrieve a specific memory by UUID |
| `update_memory` | Update content, importance, or tags |
| `delete_memory` | Remove a memory permanently |
| `analysis` | AI question answering, summarization, pattern detection, temporal analysis |
| `relationships` | Find related, discover connections, create links, map knowledge graph |
| `categories` | List, create, and auto-categorize memories |
| `domains` | Manage knowledge domains |
| `sessions` | List and query sessions |
| `stats` | System and domain statistics |
| `ingest_conversations` | Import Claude Code conversation history |
| `search_chats` | Search past Claude Code sessions |
| `get_chat` | Retrieve full conversation with messages and tool calls |
| `trace_source` | Trace a memory back to the conversation that created it |

---

## Troubleshooting

### "command not found: mycelicmemory"

Ensure npm global bin is in your PATH:
```bash
npm bin -g    # Shows the global bin directory — add this to PATH
```

### MCP Not Connecting

1. **Verify the binary works:** `mycelicmemory --version`
2. **Check your config:** `cat ~/.claude/mcp.json` (must be valid JSON)
3. **Test MCP mode directly:**
   ```bash
   echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}' | mycelicmemory --mcp
   ```
   You should see a JSON response with `"serverInfo"`.
4. **Restart Claude Code** — MCP config changes require a restart.

### Windows: npm Wrapper Fails in MCP Mode

Use the direct binary path instead of `mycelicmemory`. The npm wrapper routes through `cmd.exe -> node.js -> binary`, which can break stdin/stdout piping. See the [installation section](#2-connect-to-claude-code) for the direct path.

### macOS: Security Warning

Go to System Preferences > Security & Privacy > General and click "Allow Anyway" next to the mycelicmemory message.

### Ollama Not Detected

```bash
ollama serve          # Start Ollama
mycelicmemory doctor  # Verify detection
```

### Full Diagnostics

```bash
mycelicmemory doctor
```

This checks: binary version, database connectivity, Ollama availability, Qdrant connectivity, and model availability.

---

## Build from Source

Requires Go 1.23+ and a C compiler (for SQLite):

```bash
git clone https://github.com/MycelicMemory/mycelicmemory.git
cd mycelicmemory
make deps && make build && make install
```

### Desktop App from Source

```bash
cd desktop
npm install
npm run dev          # Development
npm run package      # Build installer for current platform
```

---

## Links

- **Repository:** [github.com/MycelicMemory/mycelicmemory](https://github.com/MycelicMemory/mycelicmemory)
- **npm:** [npmjs.com/package/mycelicmemory](https://npmjs.com/package/mycelicmemory)
- **Issues:** [GitHub Issues](https://github.com/MycelicMemory/mycelicmemory/issues)
- **Use Cases:** [15 detailed examples](docs/USE_CASES.md)
- **Hooks Guide:** [Automatic memory capture](docs/HOOKS.md)

---

MIT License. Free and open source.
