# Ultrathink Installation Guide

Complete installation guide for users starting from scratch with no dependencies.

## Table of Contents

- [Quick Install (npm)](#quick-install-npm)
- [Build from Source](#build-from-source)
- [Post-Installation Setup](#post-installation-setup)
- [Optional Features](#optional-features)
- [Troubleshooting](#troubleshooting)

---

## Quick Install (npm)

The fastest way to install Ultrathink. Requires Node.js 16+.

```bash
npm install -g github:MycelicMemory/ultrathink
```

Verify installation:
```bash
ultrathink --version
ultrathink doctor
```

**Note:** macOS only for npm install. Linux/Windows users should [build from source](#build-from-source).

**Done!** Skip to [Post-Installation Setup](#post-installation-setup).

---

## Build from Source

For developers or if you don't have Node.js.

### Prerequisites

Ultrathink requires **Go 1.23+** and a **C compiler** (for SQLite FTS5 full-text search).

### macOS

```bash
# Install Xcode Command Line Tools (C compiler)
xcode-select --install

# Install Go via Homebrew
brew install go

# Verify installations
go version    # Should show go1.23 or higher
gcc --version # Should show Apple clang
```

### Linux (Ubuntu/Debian)

```bash
# Update package list
sudo apt-get update

# Install C compiler and build tools
sudo apt-get install -y build-essential

# Install Go (method 1: apt - may be older version)
sudo apt-get install -y golang

# Install Go (method 2: official binary - recommended)
wget https://go.dev/dl/go1.23.4.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.23.4.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# Verify
go version
gcc --version
```

### Linux (Fedora/RHEL)

```bash
# Install development tools
sudo dnf groupinstall "Development Tools"

# Install Go
sudo dnf install golang

# Verify
go version
gcc --version
```

### Windows

Windows builds require WSL2 (recommended) or MinGW-w64.

**Option 1: WSL2 (Recommended)**
```powershell
# In PowerShell as Administrator
wsl --install

# Then follow Linux instructions inside WSL
```

**Option 2: MinGW-w64**
1. Install Go from https://go.dev/dl/
2. Install MinGW-w64 from https://www.mingw-w64.org/
3. Add MinGW bin directory to PATH
4. Set `CGO_ENABLED=1` environment variable

### Step 1: Clone Repository

```bash
git clone https://github.com/MycelicMemory/ultrathink.git
cd ultrathink
```

### Step 2: Download Dependencies

```bash
make deps
```

Or manually:
```bash
go mod download
go mod tidy
```

### Step 3: Build Binary

```bash
make build
```

This creates the `./ultrathink` binary in the project root.

Or manually:
```bash
CGO_ENABLED=1 go build -tags "fts5" -o ultrathink ./cmd/ultrathink
```

### Step 4: Install Binary

Choose one installation method:

**Option A: System-wide install (recommended)**
```bash
sudo make dev-install
# Installs to /usr/local/bin/ultrathink
```

**Option B: User install (no sudo)**
```bash
make install
# Installs to $(go env GOPATH)/bin/ultrathink
# Ensure $GOPATH/bin is in your PATH:
echo 'export PATH=$PATH:$(go env GOPATH)/bin' >> ~/.bashrc
source ~/.bashrc
```

**Option C: Symlink (for development)**
```bash
make link
# Creates symlink - rebuilds take effect immediately
```

### Step 5: Verify Installation

```bash
ultrathink --version
ultrathink doctor
```

---

## Post-Installation Setup

### Configure Claude Code

Add Ultrathink as an MCP server to give Claude persistent memory.

**1. Create/edit the MCP configuration file:**

```bash
# Create config directory if needed
mkdir -p ~/.claude

# Create or edit mcp.json
nano ~/.claude/mcp.json
```

**2. Add the Ultrathink server:**

```json
{
  "mcpServers": {
    "local-memory": {
      "command": "ultrathink",
      "args": ["--mcp"]
    }
  }
}
```

**3. Restart Claude Code** to load the new configuration.

**4. Verify connection:**
```bash
# Test MCP mode directly
echo '{"jsonrpc":"2.0","id":1,"method":"initialize"}' | ultrathink --mcp
```

### Configure Claude Desktop

**1. Find your config file:**
- **macOS:** `~/Library/Application Support/Claude/claude_desktop_config.json`
- **Windows:** `%APPDATA%\Claude\claude_desktop_config.json`
- **Linux:** `~/.config/Claude/claude_desktop_config.json`

**2. Add Ultrathink:**

```json
{
  "mcpServers": {
    "local-memory": {
      "command": "ultrathink",
      "args": ["--mcp"]
    }
  }
}
```

**3. Restart Claude Desktop.**

### Create Configuration File (Optional)

```bash
# Create config directory
mkdir -p ~/.ultrathink

# Copy example configuration
cp config.example.yaml ~/.ultrathink/config.yaml

# Edit as needed
nano ~/.ultrathink/config.yaml
```

### Verify Everything Works

```bash
# Full system diagnostics
ultrathink doctor

# Store a test memory
ultrathink remember "Installation complete!"

# Search for it
ultrathink search "installation"
```

---

## Optional Features

These features enhance Ultrathink but are not required for basic operation.

### Ollama (AI-Powered Features)

Ollama enables semantic search and AI analysis capabilities.

**Install Ollama:**

macOS:
```bash
brew install ollama
```

Linux:
```bash
curl -fsSL https://ollama.ai/install.sh | sh
```

**Start Ollama service:**
```bash
ollama serve
```

**Download required models:**
```bash
# Embedding model (required for semantic search)
ollama pull nomic-embed-text

# Chat model (required for AI analysis)
ollama pull qwen2.5:3b
```

**Verify:**
```bash
ultrathink doctor
# Should show: Ollama: Available
```

### Qdrant (Vector Database)

For large-scale semantic search with thousands of memories.

**Run via Docker:**
```bash
docker run -d -p 6333:6333 qdrant/qdrant
```

**Verify:**
```bash
ultrathink doctor
# Should show: Qdrant: Available
```

### Auto-Memory Hooks

Automatically capture knowledge from Claude Code sessions.

**1. Create hooks directory:**
```bash
mkdir -p ~/.claude/hooks
```

**2. Download hook scripts:**
```bash
curl -o ~/.claude/hooks/ultrathink-memory-capture.py \
  https://raw.githubusercontent.com/MycelicMemory/ultrathink/main/hooks/ultrathink-memory-capture.py

curl -o ~/.claude/hooks/ultrathink-context-loader.py \
  https://raw.githubusercontent.com/MycelicMemory/ultrathink/main/hooks/ultrathink-context-loader.py

chmod +x ~/.claude/hooks/*.py
```

**3. Configure Claude Code hooks in `~/.claude/settings.json`:**
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
    ]
  }
}
```

---

## Troubleshooting

### "command not found: ultrathink"

**Cause:** Binary not in PATH.

**Solutions:**
```bash
# Check where it's installed
which ultrathink
ls -la /usr/local/bin/ultrathink
ls -la $(go env GOPATH)/bin/ultrathink

# If using GOPATH, add to PATH
echo 'export PATH=$PATH:$(go env GOPATH)/bin' >> ~/.bashrc
source ~/.bashrc
```

### CGO/Compilation Errors

**Cause:** C compiler not found or misconfigured.

**Solutions:**

macOS:
```bash
xcode-select --install
```

Linux:
```bash
sudo apt-get install build-essential
```

Verify CGO is enabled:
```bash
go env CGO_ENABLED  # Should be 1
```

### MCP Connection Issues

**Cause:** Configuration file syntax error or wrong path.

**Solutions:**
```bash
# Validate JSON syntax
cat ~/.claude/mcp.json | python3 -m json.tool

# Test MCP mode directly
echo '{"jsonrpc":"2.0","id":1,"method":"initialize"}' | ultrathink --mcp

# Check if ultrathink is in PATH
which ultrathink

# Use absolute path in config if needed
{
  "mcpServers": {
    "local-memory": {
      "command": "/usr/local/bin/ultrathink",
      "args": ["--mcp"]
    }
  }
}
```

### Ollama Connection Failed

**Cause:** Ollama service not running or models not downloaded.

**Solutions:**
```bash
# Start Ollama
ollama serve

# Check if models are available
ollama list

# Pull missing models
ollama pull nomic-embed-text
ollama pull qwen2.5:3b

# Test connection
curl http://localhost:11434/api/tags
```

### Database Errors

**Cause:** Corrupted database or permissions issue.

**Solutions:**
```bash
# Check database location
ls -la ~/.ultrathink/

# Reset database (WARNING: deletes all memories)
rm ~/.ultrathink/memories.db
ultrathink doctor  # Recreates database
```

---

## Next Steps

See the [Quickstart Guide](QUICKSTART.md) to learn how to use Ultrathink effectively.
