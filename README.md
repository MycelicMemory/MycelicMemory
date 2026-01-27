# Ultrathink

<p align="center">
  <strong>AI-powered persistent memory system for Claude and other AI agents</strong>
</p>

<p align="center">
  <a href="#installation">Installation</a> •
  <a href="#connect-to-claude">Connect to Claude</a> •
  <a href="#features">Features</a> •
  <a href="#optional-ai-features">AI Features</a>
</p>

---

Ultrathink gives Claude persistent memory across conversations. Store knowledge, search semantically, and get AI-powered insights from your memories.

**Free and open source** - no license keys, no subscriptions.

---

## Installation

### Prerequisites

**Required:**
- **Node.js 16+** (includes npm)

**Optional (for enhanced AI features):**
- **Ollama** - For semantic search and AI analysis

### Install via npm

```bash
npm install -g ultrathink
```

That's it. The installer automatically downloads the correct binary for your platform.

### Verify Installation

```bash
ultrathink --version
ultrathink doctor
```

---

## Connect to Claude

### Claude Code (CLI)

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

Restart Claude Code:

```bash
claude
```

### Claude Desktop

Edit your config file:

| Platform | Location |
|----------|----------|
| macOS | `~/Library/Application Support/Claude/claude_desktop_config.json` |
| Windows | `%APPDATA%\Claude\claude_desktop_config.json` |
| Linux | `~/.config/Claude/claude_desktop_config.json` |

Add:

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

Restart Claude Desktop.

### Test It Works

Ask Claude:
- "Remember that ultrathink is now working"
- "What memories do I have?"

---

## Features

### Memory Operations

| Tool | Description |
|------|-------------|
| `store_memory` | Save memories with importance (1-10), tags, and domain |
| `search` | Semantic, keyword, tag, or hybrid search |
| `get_memory_by_id` | Retrieve specific memory by UUID |
| `update_memory` | Modify content, importance, or tags |
| `delete_memory` | Remove a memory permanently |

### AI Analysis (requires Ollama)

| Tool | Description |
|------|-------------|
| `analysis(question)` | Ask natural language questions about your memories |
| `analysis(summarize)` | Generate summaries across timeframes |
| `analysis(analyze)` | Detect patterns and themes |

### Knowledge Organization

| Tool | Description |
|------|-------------|
| `relationships` | Create/discover connections between memories |
| `categories` | Organize memories hierarchically |
| `domains` | Group by knowledge domain |
| `sessions` | Track memories by session |

---

## Optional: AI Features

Ultrathink works without AI services, but semantic search and AI-powered analysis require **Ollama**.

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

### Download Models

```bash
ollama pull nomic-embed-text   # For semantic search
ollama pull qwen2.5:3b         # For AI analysis
```

### Verify

```bash
ultrathink doctor
# Should show: Ollama: Available
```

---

## CLI Usage

Ultrathink also works as a standalone CLI:

```bash
# Store memories
ultrathink remember "Go interfaces are satisfied implicitly"
ultrathink remember "Important!" --importance 9 --tags learning,go

# Search
ultrathink search "concurrency patterns"

# AI analysis (requires Ollama)
ultrathink analyze "What have I learned about testing?"

# Service management
ultrathink start    # Start REST API daemon
ultrathink stop     # Stop daemon
ultrathink doctor   # Health check
```

---

## Configuration

Config file: `~/.ultrathink/config.yaml`

```yaml
database:
  path: ~/.ultrathink/memories.db

rest_api:
  port: 3099
  host: localhost

ollama:
  enabled: true
  base_url: http://localhost:11434
  embedding_model: nomic-embed-text
  chat_model: qwen2.5:3b
```

---

## Build from Source

If you prefer to build from source (requires Go 1.23+ and a C compiler):

```bash
git clone https://github.com/MycelicMemory/ultrathink.git
cd ultrathink
make deps && make build && make install
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup.

---

## Troubleshooting

### "command not found: ultrathink"

Ensure npm global bin is in your PATH:

```bash
npm bin -g
# Add this directory to your PATH
```

### MCP Not Available in Claude

```bash
# Verify installation
which ultrathink
ultrathink --version

# Test MCP mode
echo '{"jsonrpc":"2.0","id":1,"method":"initialize"}' | ultrathink --mcp

# Validate config
cat ~/.claude/mcp.json | python3 -m json.tool
```

### macOS Security Warning

If macOS blocks the binary:
1. Go to System Preferences > Security & Privacy > General
2. Click "Allow Anyway" next to the ultrathink message

### Full Diagnostics

```bash
ultrathink doctor
```

---

## Documentation

- [Quick Start Guide](docs/QUICKSTART.md)
- [Use Cases](docs/USE_CASES.md) - 15 detailed examples
- [Hooks Setup](docs/HOOKS.md) - Automatic memory capture
- [Contributing](CONTRIBUTING.md)

---

## License

MIT License - Free and open source.

---

## Links

- **Repository**: [github.com/MycelicMemory/ultrathink](https://github.com/MycelicMemory/ultrathink)
- **Issues**: [GitHub Issues](https://github.com/MycelicMemory/ultrathink/issues)
- **npm**: [npmjs.com/package/ultrathink](https://npmjs.com/package/ultrathink)
