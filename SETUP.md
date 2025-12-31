# Ultrathink Setup Guide

Complete guide for installing Ultrathink and connecting it to Claude.

---

## Quick Setup (5 minutes)

### 1. Install Ultrathink

**Option A: npm (Recommended)**
```bash
npm install -g ultrathink
```

**Option B: Build from source**
```bash
git clone https://github.com/MycelicMemory/ultrathink.git
cd ultrathink
make deps
make dev-install
```

**Verify installation:**
```bash
ultrathink --version
ultrathink doctor
```

### 2. Connect to Claude Code

Edit `~/.claude/mcp.json` and add ultrathink to the `mcpServers` section:

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

If the file doesn't exist, create it with the content above.

### 3. Restart Claude Code

```bash
# Exit current session (Ctrl+C or /exit)
# Start new session
claude
```

### 4. Verify It Works

In Claude Code, you should now see ultrathink tools available:
- `mcp__ultrathink__store_memory`
- `mcp__ultrathink__search`
- `mcp__ultrathink__analysis`
- etc.

Try asking Claude: "Remember that ultrathink is now set up and working"

---

## Claude Desktop Setup

### macOS

Edit: `~/Library/Application Support/Claude/claude_desktop_config.json`

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

### Windows

Edit: `%APPDATA%\Claude\claude_desktop_config.json`

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

### Linux

Edit: `~/.config/Claude/claude_desktop_config.json`

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

Restart Claude Desktop after editing the config.

---

## Optional: Enable AI Features

### Install Ollama (for AI-powered search and analysis)

Ollama provides local AI models for semantic search and memory analysis.

**Install:**
```bash
# macOS
brew install ollama

# Linux
curl -fsSL https://ollama.ai/install.sh | sh

# Windows
# Download from https://ollama.ai
```

**Start Ollama:**
```bash
ollama serve
```

**Download required models:**
```bash
ollama pull nomic-embed-text   # Embeddings (768-dim vectors)
ollama pull qwen2.5:3b         # Chat/analysis model
```

**Verify:**
```bash
curl http://localhost:11434/api/tags
ultrathink doctor  # Should show Ollama as available
```

### Install Qdrant (for vector search)

Qdrant provides high-performance similarity search.

**Using Docker:**
```bash
docker run -d -p 6333:6333 qdrant/qdrant
```

**Verify:**
```bash
curl http://localhost:6333/health
ultrathink doctor  # Should show Qdrant as available
```

---

## Prerequisites Reference

| Component | Version | Purpose | Required? |
|-----------|---------|---------|-----------|
| Node.js | 16+ | npm installation | For npm install |
| Go | 1.21+ | Build from source | For source build |
| SQLite | 3.x | Database | Pre-installed on macOS/Linux |
| Ollama | latest | AI embeddings & analysis | Optional |
| Qdrant | latest | Vector similarity search | Optional |

---

## Build from Source (Developers)

### 1. Install Go

**macOS:**
```bash
brew install go
```

**Linux:**
```bash
# Ubuntu/Debian
sudo apt install golang-go

# Fedora
sudo dnf install golang
```

**Verify:**
```bash
go version  # Should show go1.21+
```

### 2. Set Go Environment

Add to `~/.zshrc` or `~/.bashrc`:
```bash
export GOPATH=$HOME/go
export PATH=$PATH:$GOPATH/bin
```

### 3. Clone and Build

```bash
git clone https://github.com/MycelicMemory/ultrathink.git
cd ultrathink

# Install dependencies
make deps

# Build and install globally
make dev-install

# Verify
ultrathink --version
```

### 4. Development Workflow

```bash
# Build only (creates ./ultrathink)
make build

# Symlink for development (auto-updates on rebuild)
make link

# Run tests
make test

# Format code
make fmt

# Lint
make lint
```

---

## Configuration

Ultrathink uses `~/.ultrathink/config.yaml` for configuration.

### Create Config File

```bash
mkdir -p ~/.ultrathink
cat > ~/.ultrathink/config.yaml << 'EOF'
database:
  path: ~/.ultrathink/memories.db
  backup_interval: 24h
  max_backups: 7

rest_api:
  enabled: true
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
EOF
```

### Configuration Options

| Setting | Default | Description |
|---------|---------|-------------|
| `database.path` | `~/.ultrathink/memories.db` | SQLite database location |
| `rest_api.port` | `3099` | REST API port |
| `session.strategy` | `git-directory` | Session ID generation (git-directory or manual) |
| `ollama.enabled` | `true` | Enable Ollama AI features |
| `qdrant.enabled` | `true` | Enable Qdrant vector search |

---

## Troubleshooting

### MCP Server Not Available

1. **Check installation:**
   ```bash
   which ultrathink
   ultrathink --version
   ```

2. **Validate MCP config:**
   ```bash
   cat ~/.claude/mcp.json | python3 -m json.tool
   ```

3. **Test MCP mode:**
   ```bash
   echo '{"jsonrpc":"2.0","id":1,"method":"initialize"}' | ultrathink --mcp
   ```

4. **Restart Claude Code/Desktop**

### Ollama Not Available

```bash
# Check if running
pgrep ollama

# Start service
ollama serve

# Check models
ollama list

# Pull models if missing
ollama pull nomic-embed-text
ollama pull qwen2.5:3b
```

### Database Locked

```bash
ultrathink ps
ultrathink kill_all
```

### Port Already in Use

```bash
# Check what's using the port
lsof -i :3099

# Use different port
ultrathink start --port 3100
```

### Full System Check

```bash
ultrathink doctor
```

This shows the status of all components (database, Ollama, Qdrant).

---

## Uninstall

### npm Installation

```bash
npm uninstall -g ultrathink
rm -rf ~/.ultrathink
```

### Source Installation

```bash
cd ultrathink
make uninstall
rm -rf ~/.ultrathink
```

### Remove MCP Config

Edit `~/.claude/mcp.json` and remove the `ultrathink` entry.

---

## Next Steps

1. Ask Claude to "Remember something important"
2. Search your memories: "What do I know about X?"
3. Explore the CLI: `ultrathink --help`
4. Check the [README](README.md) for full documentation
