# MycelicMemory

<p align="center">
  <strong>Persistent memory for Claude and AI agents</strong>
</p>

<p align="center">
  <a href="#quick-start">Quick Start</a> &bull;
  <a href="#connect-to-claude">Connect to Claude</a> &bull;
  <a href="#features">Features</a> &bull;
  <a href="#desktop-app">Desktop App</a> &bull;
  <a href="#cli-usage">CLI</a> &bull;
  <a href="#rest-api">REST API</a>
</p>

---

MycelicMemory gives Claude persistent memory across conversations. Store knowledge, search semantically, build a knowledge graph, and get AI-powered insights — all running locally on your machine.

**Free and open source** — no cloud accounts, no API keys, no subscriptions.

---

## Quick Start

### Prerequisites

- **Node.js 18+** (includes npm)
- **Ollama** (optional — enables semantic search and AI analysis)

### Install

```bash
npm install -g mycelicmemory
```

First run downloads the platform-specific binary automatically:

```bash
mycelicmemory --version    # Download binary + show version
mycelicmemory doctor       # Check system dependencies
```

### Alternative: Install from GitHub

```bash
npm install -g github:MycelicMemory/mycelicmemory
```

---

## Connect to Claude

MCP (Model Context Protocol) lets Claude communicate with MycelicMemory directly.

### Step 1: Locate the Binary

After `npm install -g mycelicmemory`, find the native binary path:

**macOS/Linux:**
```bash
BINARY_PATH="$(npm root -g)/mycelicmemory/bin/$(node -e "
  const os = require('os');
  const p = os.platform(), a = os.arch();
  const names = {
    'darwin-arm64': 'mycelicmemory-macos-arm64',
    'darwin-x64': 'mycelicmemory-macos-x64',
    'linux-arm64': 'mycelicmemory-linux-arm64',
    'linux-x64': 'mycelicmemory-linux-x64'
  };
  console.log(names[p+'-'+a]);
")"
echo $BINARY_PATH
```

**Windows (PowerShell):**
```powershell
$BinaryPath = Join-Path (npm root -g) "mycelicmemory\bin\mycelicmemory-windows-x64.exe"
Write-Host $BinaryPath
```

> **Why the direct binary path?** On Windows, the npm wrapper script goes through `cmd.exe -> node.js -> Go binary`, which can cause stdin/stdout pipe issues in MCP mode. Using the direct binary avoids this.

### Step 2: Configure Claude Code (CLI)

Edit `~/.claude/mcp.json`:

**macOS/Linux** (npm wrapper works reliably):
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

**Windows** (use direct binary path):
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

### Step 3: Configure Claude Desktop (Optional)

Edit your config file:

| Platform | Config File Location |
|----------|----------|
| macOS | `~/Library/Application Support/Claude/claude_desktop_config.json` |
| Windows | `%APPDATA%\Claude\claude_desktop_config.json` |
| Linux | `~/.config/Claude/claude_desktop_config.json` |

Use the same `mcpServers` block as above.

### Step 4: Verify MCP Connection

In Claude Code, run `/mcp`. You should see `mycelicmemory` listed. Then try:

- "Remember that mycelicmemory is now working"
- "What memories do I have?"

---

## Features

### Memory Operations

| Tool | Description |
|------|-------------|
| `store_memory` | Save memories with importance (1-10), tags, domain, and source |
| `search` | Semantic, keyword, tag, hybrid, or date-range search |
| `get_memory_by_id` | Retrieve specific memory by UUID |
| `update_memory` | Modify content, importance, or tags |
| `delete_memory` | Remove a memory permanently |

### AI Analysis (requires Ollama)

| Tool | Description |
|------|-------------|
| `analysis(question)` | Ask natural language questions about your memories |
| `analysis(summarize)` | Generate summaries across timeframes (today, week, month, all) |
| `analysis(analyze)` | Detect patterns and themes |
| `analysis(temporal_patterns)` | Analyze how your knowledge evolved over time |

### Knowledge Graph

Build and explore connections between memories:

| Tool | Description |
|------|-------------|
| `relationships(find_related)` | Find memories related to a given memory |
| `relationships(discover)` | AI-powered automatic relationship discovery |
| `relationships(create)` | Create explicit connections (references, expands, contradicts, similar, causes, enables, sequential) |
| `relationships(map_graph)` | Generate a graph visualization with configurable depth |

### Claude Code Chat History

Ingest and search your Claude Code conversation history:

| Tool | Description |
|------|-------------|
| `ingest_conversations` | Import sessions from `~/.claude/` JSONL files with optional summary generation |
| `search_chats` | Search across all past conversations by title, prompt, or content |
| `get_chat` | Retrieve full conversation with messages, tool calls, and linked memories |
| `trace_source` | Trace any memory back to the exact conversation that created it |

### Knowledge Organization

| Tool | Description |
|------|-------------|
| `categories` | Create, list, and auto-categorize memories with AI |
| `domains` | Group memories by knowledge domain with statistics |
| `sessions` | Track memories by work session |

### Auto-Memory Prompt

MycelicMemory includes a built-in MCP prompt that instructs Claude to:
1. **Search first** — check for relevant context at the start of conversations
2. **Store continuously** — save decisions, debugging insights, preferences, and learnings
3. **Build relationships** — connect new memories to existing ones

Claude manages your knowledge base proactively without needing explicit "remember this" instructions.

### Data Source Ingestion

Register external data sources and ingest knowledge through the REST API:

- Bulk import with deduplication (skip duplicate external IDs)
- Checkpoint-based resumable ingestion
- Sync history tracking and status monitoring
- Per-source statistics and error logging

---

## Desktop App

A standalone Electron application for visual memory management.

**Pages:**
- **Dashboard** — Memory stats, service health indicators, quick actions, activity charts
- **Memory Browser** — Search, filter, edit, and delete memories with detail panels
- **Claude Sessions** — Browse conversation history by project, view messages and tool calls
- **Knowledge Graph** — Interactive network visualization of memory relationships
- **Settings** — Configure API, Ollama models (with dropdown picker), MCP setup guide, Qdrant, theme

**Service Management:**
The desktop app automatically discovers and manages the MycelicMemory backend, Ollama, and Qdrant services.

**Quick start:**
```bash
cd desktop
npm install
npm run dev
```

**Package for distribution:**
```bash
npm run package:win    # Windows (.exe)
npm run package:mac    # macOS (.dmg)
npm run package:linux  # Linux (AppImage)
```

See [`desktop/README.md`](desktop/README.md) for architecture details.

---

## CLI Usage

```bash
# Store memories
mycelicmemory remember "Go interfaces are satisfied implicitly" \
  --importance 8 --tags learning,go --domain programming

# Search
mycelicmemory search "concurrency patterns" --limit 5

# AI analysis (requires Ollama)
mycelicmemory analyze "What have I learned about testing?" --type question

# Relationships
mycelicmemory relate <id1> <id2> --type expands --strength 0.8
mycelicmemory discover                     # AI relationship discovery
mycelicmemory map_graph <id> --depth 3     # Graph visualization

# Organization
mycelicmemory list --domain databases
mycelicmemory categorize <id> --auto-create

# Service management
mycelicmemory start    # Start REST API daemon
mycelicmemory stop     # Stop daemon
mycelicmemory status   # Show daemon status
mycelicmemory doctor   # Health check
```

---

## REST API

The REST API runs at `http://localhost:3099/api/v1/` when the daemon is started.

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Health check |
| `/memories` | POST | Create memory |
| `/memories` | GET | List memories (paginated) |
| `/memories/:id` | GET / PUT / DELETE | CRUD by ID |
| `/memories/search` | GET / POST | Search memories |
| `/memories/search/intelligent` | POST | AI-powered intelligent search |
| `/memories/:id/related` | GET | Find related memories |
| `/memories/:id/graph` | GET | Knowledge graph for memory |
| `/analyze` | POST | AI analysis |
| `/relationships` | POST | Create relationship |
| `/relationships/discover` | POST | AI relationship discovery |
| `/categories` | GET / POST | Manage categories |
| `/domains` | GET / POST | Manage domains |
| `/sessions` | GET | List sessions |
| `/stats` | GET | System statistics |
| `/search/tags` | POST | Tag-based search (AND/OR) |
| `/search/date-range` | POST | Date range search |
| `/sources` | GET / POST | Data source management |
| `/sources/:id/sync` | POST | Trigger sync |
| `/sources/:id/ingest` | POST | Bulk ingest |
| `/chats/ingest` | POST | Ingest Claude Code conversations |
| `/chats` | GET | List chat sessions |
| `/chats/search` | GET | Search conversations |
| `/chats/:id` | GET | Get session with messages |

Start the API:
```bash
mycelicmemory start --port 3099
```

---

## Automatic Memory Hooks

Capture knowledge automatically from Claude Code sessions — no manual "remember this" needed.

```bash
mkdir -p ~/.claude/hooks

curl -o ~/.claude/hooks/mycelicmemory-memory-capture.py \
  https://raw.githubusercontent.com/MycelicMemory/mycelicmemory/main/hooks/mycelicmemory-memory-capture.py

curl -o ~/.claude/hooks/mycelicmemory-context-loader.py \
  https://raw.githubusercontent.com/MycelicMemory/mycelicmemory/main/hooks/mycelicmemory-context-loader.py

chmod +x ~/.claude/hooks/*.py
```

- **Memory Capture** — Stores decisions, debugging insights, and learnings automatically
- **Context Loader** — Loads relevant memories at the start of new sessions

See [Hooks Guide](docs/HOOKS.md) for configuration.

---

## Optional: AI Features

MycelicMemory works without AI services (keyword search, tags, storage, knowledge graph). For semantic search and AI analysis, add Ollama.

### Install Ollama

**macOS:** `brew install ollama && ollama serve`

**Linux:** `curl -fsSL https://ollama.ai/install.sh | sh && ollama serve`

**Windows:** Download from [ollama.ai](https://ollama.ai/download)

### Download Models

```bash
ollama pull nomic-embed-text   # Semantic search embeddings
ollama pull qwen2.5:3b         # AI analysis
```

### Optional: Qdrant Vector Database

For high-performance similarity search on large collections:

```bash
docker run -d --name qdrant -p 6333:6333 qdrant/qdrant
```

### Verify

```bash
mycelicmemory doctor
```

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

---

## Build from Source

Requires Go 1.23+ and a C compiler:

```bash
git clone https://github.com/MycelicMemory/mycelicmemory.git
cd mycelicmemory
make deps && make build && make install
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup.

---

## Troubleshooting

### "command not found: mycelicmemory"

Ensure npm global bin is in your PATH:
```bash
npm bin -g    # Shows the directory to add to PATH
```

### MCP Not Available in Claude

1. **Verify:** `mycelicmemory --version && mycelicmemory doctor`
2. **Test MCP directly:**
   ```bash
   echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}' | mycelicmemory --mcp
   ```
3. **Check config:** Validate `~/.claude/mcp.json` is valid JSON
4. **Restart Claude Code** after config changes

### Windows: npm Wrapper Not Working

Use the direct binary path instead of `mycelicmemory`. See [Connect to Claude](#step-2-configure-claude-code-cli).

### macOS Security Warning

System Preferences > Security & Privacy > General > "Allow Anyway"

### Full Diagnostics

```bash
mycelicmemory doctor
```

---

## Documentation

- [Quick Start Guide](docs/QUICKSTART.md)
- [Use Cases](docs/USE_CASES.md) — 15 detailed examples
- [Hooks Setup](docs/HOOKS.md) — Automatic memory capture
- [Website / Full Guide](docs/WEBSITE.md) — Comprehensive documentation
- [Contributing](CONTRIBUTING.md)

---

## License

MIT License — Free and open source.

---

## Links

- **Repository**: [github.com/MycelicMemory/mycelicmemory](https://github.com/MycelicMemory/mycelicmemory)
- **Issues**: [GitHub Issues](https://github.com/MycelicMemory/mycelicmemory/issues)
- **npm**: [npmjs.com/package/mycelicmemory](https://npmjs.com/package/mycelicmemory)
