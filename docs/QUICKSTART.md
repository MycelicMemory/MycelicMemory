# Ultrathink Quick Start Guide

Get Ultrathink running with Claude in under 5 minutes.

---

## Step 1: Install Ultrathink

### Option A: npm (Recommended)

```bash
npm install -g ultrathink
```

### Option B: Build from Source

```bash
git clone https://github.com/MycelicMemory/ultrathink.git
cd ultrathink
make deps
make dev-install
```

### Verify Installation

```bash
ultrathink --version
# ultrathink version 1.0.0

ultrathink doctor
# Shows system status
```

---

## Step 2: Connect to Claude

### For Claude Code

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

### For Claude Desktop

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

---

## Step 3: Restart Claude

Close and reopen Claude Code or Claude Desktop.

---

## Step 4: Verify It Works

Ask Claude any of these:

```
"Remember that ultrathink is now set up and working"
```

```
"Store this: Go channels are typed conduits for goroutine communication"
```

```
"What memories do I have?"
```

You should see Claude using `mcp__ultrathink__store_memory` and `mcp__ultrathink__search` tools.

---

## Step 5: Enable AI Features (Optional but Recommended)

### Install Ollama

Ollama provides AI-powered semantic search and analysis.

```bash
# macOS
brew install ollama

# Linux
curl -fsSL https://ollama.ai/install.sh | sh

# Start Ollama
ollama serve

# Download models
ollama pull nomic-embed-text   # For embeddings
ollama pull qwen2.5:3b         # For analysis
```

### Verify Ollama

```bash
ultrathink doctor
# Should show: Ollama: Available
```

---

## What You Can Do Now

### Store Memories

Ask Claude:
- "Remember that React useEffect runs after render"
- "Store this debugging tip: always check network tab first"
- "Save this with high importance: never commit .env files"

### Search Memories

Ask Claude:
- "What do I know about React?"
- "Search my memories for debugging tips"
- "Find all high-importance memories"

### Analyze Memories

Ask Claude:
- "Summarize what I learned this week"
- "What patterns do you see in my programming notes?"
- "How do my React notes relate to my Vue notes?"

### Organize Knowledge

Ask Claude:
- "Create a category for 'Best Practices'"
- "Add this memory to the debugging domain"
- "How are these two concepts related?"

---

## Next Steps

1. **[Set up auto-capture hooks](HOOKS.md)** - Automatically save knowledge
2. **[Explore use cases](USE_CASES.md)** - 15 detailed examples
3. **[CLI reference](../README.md#cli-usage)** - Command-line usage
4. **[API documentation](API.md)** - REST API reference

---

## Troubleshooting

### "MCP server not found"

```bash
# Check ultrathink is in PATH
which ultrathink

# Test MCP mode directly
echo '{"jsonrpc":"2.0","id":1,"method":"initialize"}' | ultrathink --mcp
```

### "Ollama not available"

```bash
# Start Ollama service
ollama serve

# Check it's running
curl http://localhost:11434/api/tags
```

### Full diagnostics

```bash
ultrathink doctor
```

---

## Getting Help

- **Issues:** [GitHub Issues](https://github.com/MycelicMemory/ultrathink/issues)
- **Docs:** [Full Documentation](../README.md)
