# Ultrathink Quick Start Guide

Get Ultrathink running with Claude in under 5 minutes.

---

## Prerequisites Checklist

Before you begin, ensure you have:

- [ ] **Operating System**: macOS (arm64/amd64), Linux, or Windows with WSL2
- [ ] **Git**: For cloning the repository
- [ ] **One of the following** (choose your installation path):
  - **npm/Node.js 16+** (Path A - easiest, pre-built binaries)
  - **Go 1.23+ and C compiler** (Path B - build from source)

---

## Path A: npm Install (Easiest)

If you have Node.js 16+ installed, this is the fastest way to get started.

### Step 1: Install via npm

```bash
npm install -g ultrathink
```

This downloads a pre-built binary for your platform. No compilation needed.

### Step 2: Verify Installation

```bash
ultrathink --version
# ultrathink version 1.2.0

ultrathink doctor
# Shows system health status
```

**Skip to [Connect to Claude Code](#connect-to-claude-code)** below.

---

## Path B: Build from Source

Building from source gives you the latest code and is required for development.

### System Requirements

Ultrathink requires CGO (C bindings for Go) because it uses SQLite with FTS5 full-text search. This means you need both Go and a C compiler.

#### macOS

```bash
# Install Xcode Command Line Tools (includes C compiler)
xcode-select --install

# Install Go 1.23+ via Homebrew
brew install go

# Verify
go version   # Should show go1.23 or higher
gcc --version  # Should show Apple clang
```

#### Linux (Ubuntu/Debian)

```bash
# Install build tools and Go
sudo apt-get update
sudo apt-get install -y build-essential golang

# If Go version is too old, install manually:
# Download from https://go.dev/dl/
wget https://go.dev/dl/go1.23.4.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.23.4.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# Verify
go version   # Should show go1.23 or higher
gcc --version
```

#### Linux (Fedora/RHEL)

```bash
# Install build tools
sudo dnf groupinstall "Development Tools"
sudo dnf install golang

# Verify
go version
gcc --version
```

#### Windows

We recommend using WSL2 (Windows Subsystem for Linux) and following the Linux instructions above.

Alternatively, install:
- Go 1.23+ from https://go.dev/dl/
- MinGW-w64 for GCC: https://www.mingw-w64.org/

### Build Steps

#### Step 1: Clone the Repository

```bash
git clone https://github.com/MycelicMemory/ultrathink.git
cd ultrathink
```

#### Step 2: Download Dependencies

```bash
make deps
```

#### Step 3: Build the Binary

```bash
make build
```

This creates the `ultrathink` binary in the current directory.

#### Step 4: Install Globally

**Option A: Install to /usr/local/bin (requires sudo)**

```bash
make dev-install
```

**Option B: Install to GOPATH/bin (no sudo)**

```bash
make install
# Binary goes to $(go env GOPATH)/bin/ultrathink
# Ensure GOPATH/bin is in your PATH
```

**Option C: Create symlink (for development)**

```bash
make link
# Creates symlink - rebuilds take effect immediately
```

#### Step 5: Verify Installation

```bash
ultrathink --version
# ultrathink version 1.2.0

ultrathink doctor
# Shows system health status
```

---

## Connect to Claude Code

### Step 1: Configure MCP

Edit `~/.claude/mcp.json` (create it if it doesn't exist):

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

### Step 2: Restart Claude Code

```bash
claude
```

### Step 3: Test It Works

Ask Claude:

```
"Remember that ultrathink is now set up and working"
```

You should see Claude using the `mcp__ultrathink__store_memory` tool.

Then ask:

```
"What memories do I have?"
```

You should see Claude using the `mcp__ultrathink__search` tool.

---

## Connect to Claude Desktop

### Step 1: Find Your Config File

| Platform | Config Location |
|----------|-----------------|
| macOS | `~/Library/Application Support/Claude/claude_desktop_config.json` |
| Windows | `%APPDATA%\Claude\claude_desktop_config.json` |
| Linux | `~/.config/Claude/claude_desktop_config.json` |

### Step 2: Add Ultrathink Configuration

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

**Note:** If using npx instead of global install:

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

### Step 3: Restart Claude Desktop

Close and reopen Claude Desktop completely.

---

## Optional: Enable AI Features

Ultrathink works without AI services, but semantic search and AI-powered analysis require Ollama.

### Install Ollama

#### macOS

```bash
brew install ollama
ollama serve
```

#### Linux

```bash
curl -fsSL https://ollama.ai/install.sh | sh
ollama serve
```

### Download Required Models

```bash
# Embedding model for semantic search (768 dimensions)
ollama pull nomic-embed-text

# Chat model for AI analysis
ollama pull qwen2.5:3b
```

### Verify Ollama Integration

```bash
ultrathink doctor
# Should show: Ollama: Available
```

### Optional: Qdrant Vector Database

For high-performance vector search on large memory collections:

```bash
docker run -d -p 6333:6333 qdrant/qdrant
```

Verify with:

```bash
ultrathink doctor
# Should show: Qdrant: Available
```

---

## What You Can Do Now

### Store Memories

Ask Claude:
- "Remember that React useEffect runs after render"
- "Store this with importance 9: never commit .env files"
- "Save this debugging tip: always check network tab first"

### Search Memories

Ask Claude:
- "What do I know about React?"
- "Search my memories for debugging tips"
- "Find all high-importance memories"

### Analyze Memories

Ask Claude:
- "Summarize what I learned this week"
- "What patterns do you see in my notes?"
- "Answer this based on my memories: how should I structure Go projects?"

### Organize Knowledge

Ask Claude:
- "Create a 'Best Practices' category"
- "How are these two concepts related?"
- "Show me memories from the programming domain"

---

## CLI Reference

Ultrathink also works as a standalone CLI:

```bash
# Store a memory
ultrathink remember "Go interfaces are satisfied implicitly"
ultrathink remember "Important!" --importance 9 --tags go,patterns

# Search
ultrathink search "concurrency patterns"

# AI analysis
ultrathink analyze "What have I learned about testing?"

# Service management
ultrathink start    # Start REST API daemon
ultrathink stop     # Stop daemon
ultrathink doctor   # Health check
```

---

## Troubleshooting

### "command not found: ultrathink"

**npm install:** Ensure npm global bin is in PATH:
```bash
npm bin -g
# Add this directory to your PATH
```

**Built from source:** Verify installation location:
```bash
which ultrathink
# Should show /usr/local/bin/ultrathink or similar
```

### "MCP server not found" in Claude

1. Verify ultrathink is in PATH:
   ```bash
   which ultrathink
   ```

2. Test MCP mode directly:
   ```bash
   echo '{"jsonrpc":"2.0","id":1,"method":"initialize"}' | ultrathink --mcp
   ```

3. Validate config JSON:
   ```bash
   cat ~/.claude/mcp.json | python3 -m json.tool
   ```

4. Restart Claude Code completely

### CGO build errors

Ensure you have a C compiler:

```bash
# macOS
xcode-select --install

# Linux
sudo apt-get install build-essential  # Debian/Ubuntu
sudo dnf groupinstall "Development Tools"  # Fedora
```

### "Ollama not available"

```bash
# Start Ollama service
ollama serve

# Verify it's running
curl http://localhost:11434/api/tags

# Pull required models
ollama pull nomic-embed-text
ollama pull qwen2.5:3b
```

### Full diagnostics

```bash
ultrathink doctor
```

This shows the status of all components and common issues.

---

## Next Steps

1. **[Set up auto-capture hooks](HOOKS.md)** - Automatically save knowledge from coding sessions
2. **[Explore use cases](USE_CASES.md)** - 15 detailed examples with code
3. **[Run benchmarks](../benchmark/locomo/README.md)** - Test memory retrieval accuracy
4. **[Contribute](../CONTRIBUTING.md)** - Help improve Ultrathink

---

## Getting Help

- **GitHub Issues:** [github.com/MycelicMemory/ultrathink/issues](https://github.com/MycelicMemory/ultrathink/issues)
- **Documentation:** [Main README](../README.md)
