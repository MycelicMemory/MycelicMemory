# MyclicMemory Quick Start Guide

Get MyclicMemory running with Claude in under 5 minutes.

---

## Step 1: Install Node.js (if needed)

MyclicMemory requires Node.js 16 or higher.

**Check if installed:**
```bash
node --version
# Should show v16.0.0 or higher
```

**Install if needed:**
- **macOS**: `brew install node` or download from [nodejs.org](https://nodejs.org)
- **Windows**: Download from [nodejs.org](https://nodejs.org)
- **Linux**: `sudo apt install nodejs npm` or use [nvm](https://github.com/nvm-sh/nvm)

---

## Step 2: Install MyclicMemory

```bash
npm install -g mycelicmemory
```

The installer automatically downloads the correct binary for your platform (macOS, Linux, or Windows).

**Verify:**
```bash
mycelicmemory --version
```

---

## Step 3: Connect to Claude Code

Edit `~/.claude/mcp.json` (create if it doesn't exist):

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

Restart Claude Code:

```bash
claude
```

---

## Step 4: Test It Works

Ask Claude:

```
"Remember that mycelicmemory is now set up and working"
```

You should see Claude using `mcp__mycelicmemory__store_memory`.

Then ask:

```
"What memories do I have?"
```

You should see Claude using `mcp__mycelicmemory__search`.

---

## Optional: Enable AI Features

For semantic search and AI-powered analysis, install Ollama:

**macOS:**
```bash
brew install ollama
ollama serve
ollama pull nomic-embed-text
ollama pull qwen2.5:3b
```

**Linux:**
```bash
curl -fsSL https://ollama.ai/install.sh | sh
ollama serve
ollama pull nomic-embed-text
ollama pull qwen2.5:3b
```

**Windows:**
Download from [ollama.ai/download](https://ollama.ai/download)

Verify with:
```bash
mycelicmemory doctor
```

---

## What You Can Do Now

### Store Memories
- "Remember that React useEffect runs after render"
- "Store this with importance 9: never commit .env files"

### Search Memories
- "What do I know about React?"
- "Search my memories for debugging tips"

### Analyze Memories (requires Ollama)
- "Summarize what I learned this week"
- "What patterns do you see in my notes?"

---

## Troubleshooting

### "command not found: mycelicmemory"

Ensure npm global bin is in your PATH:

```bash
npm bin -g
# Add this directory to your PATH
```

### MCP Not Available

```bash
# Verify
which mycelicmemory
mycelicmemory --version

# Test MCP
echo '{"jsonrpc":"2.0","id":1,"method":"initialize"}' | mycelicmemory --mcp
```

### Full Diagnostics

```bash
mycelicmemory doctor
```

---

## Next Steps

- [Hooks Setup](HOOKS.md) - Automatic memory capture
- [Use Cases](USE_CASES.md) - 15 detailed examples
- [Main README](../README.md) - Full documentation
