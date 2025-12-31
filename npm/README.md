# Ultrathink

**AI-powered persistent memory for Claude and other AI agents**

Give Claude persistent memory across conversations. Store knowledge, search semantically, build knowledge graphs, and get AI-powered insights.

**Free and open source** - no license keys, no subscriptions.

## Installation

```bash
npm install -g ultrathink
```

## Connect to Claude Code

**Step 1:** Edit `~/.claude/mcp.json`:

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

**Step 2:** Restart Claude Code

**Step 3:** Ask Claude to remember something!

## Connect to Claude Desktop

Add to your Claude Desktop config:

- **macOS:** `~/Library/Application Support/Claude/claude_desktop_config.json`
- **Windows:** `%APPDATA%\Claude\claude_desktop_config.json`
- **Linux:** `~/.config/Claude/claude_desktop_config.json`

```json
{
  "mcpServers": {
    "ultrathink": {
      "command": "npx",
      "args": ["-y", "ultrathink", "--mcp"]
    }
  }
}
```

Restart Claude Desktop after editing.

## What Claude Can Do

Once connected, Claude can:

- **Store memories:** "Remember that Go channels are typed conduits"
- **Search memories:** "What do I know about concurrency?"
- **Analyze:** "Summarize what I learned this week"
- **Build graphs:** "How does this relate to yesterday's topic?"

## Available MCP Tools

| Tool | Description |
|------|-------------|
| `store_memory` | Save memories with tags, importance, domain |
| `search` | Semantic, keyword, tag, or hybrid search |
| `analysis` | AI Q&A, summarization, pattern detection |
| `relationships` | Create and explore memory connections |
| `categories` | Organize memories into categories |
| `domains` | Group by knowledge domain |
| `sessions` | Track memory sessions |
| `stats` | View statistics |
| `get_memory_by_id` | Retrieve specific memory |
| `update_memory` | Modify memory |
| `delete_memory` | Remove memory |

## CLI Usage

```bash
# Store a memory
ultrathink remember "Go channels are like pipes between goroutines"

# Search
ultrathink search "concurrency"

# AI analysis
ultrathink analyze "What have I learned about Go?"

# Check status
ultrathink doctor
```

## Optional: AI Features

For semantic search and AI analysis, install Ollama:

```bash
# Install
brew install ollama  # macOS

# Start
ollama serve

# Download models
ollama pull nomic-embed-text
ollama pull qwen2.5:3b
```

## Requirements

- Node.js 16+
- SQLite (bundled on macOS/Linux)
- Ollama (optional, for AI features)
- Qdrant (optional, for vector search)

## Troubleshooting

### MCP not showing in Claude

1. Verify installation: `ultrathink --version`
2. Check config syntax: `cat ~/.claude/mcp.json | python3 -m json.tool`
3. Restart Claude Code/Desktop
4. Test MCP: `echo '{"jsonrpc":"2.0","id":1,"method":"initialize"}' | ultrathink --mcp`

### System check

```bash
ultrathink doctor
```

## Links

- **Repository:** [github.com/MycelicMemory/ultrathink](https://github.com/MycelicMemory/ultrathink)
- **Documentation:** [SETUP.md](https://github.com/MycelicMemory/ultrathink/blob/main/SETUP.md)
- **Issues:** [GitHub Issues](https://github.com/MycelicMemory/ultrathink/issues)

## License

MIT - Free and open source.
