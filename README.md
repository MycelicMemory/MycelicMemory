# MyclicMemory

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

MyclicMemory gives Claude persistent memory across conversations. Store knowledge, search semantically, and get AI-powered insights from your memories.

**Free and open source** - no license keys, no subscriptions.

---

## Installation

### Prerequisites

**Required:**
- **Node.js 18+** (includes npm)

**Optional (for enhanced AI features):**
- **Ollama** - For semantic search and AI analysis

### Install via npm

```bash
npm install -g mycelicmemory
```

**Important:** After installation, run `mycelicmemory` once to download the platform-specific binary:

```bash
mycelicmemory --version
```

The first run automatically downloads and caches the binary for your system (macOS, Linux, or Windows).

### Alternative: Install from GitHub

```bash
npm install -g github:MycelicMemory/mycelicmemory
```

### Verify Installation

```bash
mycelicmemory --version    # Should show version
mycelicmemory doctor       # Check system dependencies
```

---

## Connect to Claude

MCP (Model Context Protocol) lets Claude communicate with MycelicMemory directly. Configuration depends on your platform.

### Step 1: Locate the Binary

After `npm install -g mycelicmemory`, find the native binary path:

**macOS/Linux:**
```bash
# The binary is inside the npm global modules directory
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
# Typically located at:
# %APPDATA%\npm\node_modules\mycelicmemory\bin\mycelicmemory-windows-x64.exe
$BinaryPath = Join-Path (npm root -g) "mycelicmemory\bin\mycelicmemory-windows-x64.exe"
Write-Host $BinaryPath
```

> **Why the direct binary path?** On Windows, the npm wrapper script (`mycelicmemory.cmd`) goes through `cmd.exe -> node.js -> Go binary`, which can cause stdin/stdout pipe issues in MCP mode. Using the direct binary path avoids this.

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

Replace `YOUR_USERNAME` with your Windows username.

Restart Claude Code after editing:
```bash
claude
```

### Step 3: Configure Claude Desktop (Optional)

Edit your config file:

| Platform | Config File Location |
|----------|----------|
| macOS | `~/Library/Application Support/Claude/claude_desktop_config.json` |
| Windows | `%APPDATA%\Claude\claude_desktop_config.json` |
| Linux | `~/.config/Claude/claude_desktop_config.json` |

Use the same `mcpServers` block as above (matching your platform).

Restart Claude Desktop after editing.

### Step 4: Verify MCP Connection

In Claude Code, run:
```
/mcp
```

You should see `mycelicmemory` listed as connected. Then ask Claude:
- "Remember that mycelicmemory is now working"
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

MyclicMemory works without AI services, but semantic search and AI-powered analysis require **Ollama**.

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
mycelicmemory doctor
# Should show: Ollama: Available
```

---

## CLI Usage

MyclicMemory also works as a standalone CLI:

```bash
# Store memories
mycelicmemory remember "Go interfaces are satisfied implicitly"
mycelicmemory remember "Important!" --importance 9 --tags learning,go

# Search
mycelicmemory search "concurrency patterns"

# AI analysis (requires Ollama)
mycelicmemory analyze "What have I learned about testing?"

# Service management
mycelicmemory start    # Start REST API daemon
mycelicmemory stop     # Stop daemon
mycelicmemory doctor   # Health check
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
```

---

## Build from Source

If you prefer to build from source (requires Go 1.23+ and a C compiler):

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
npm root -g        # Shows the global node_modules directory
npm bin -g         # Shows the global bin directory (add to PATH)
```

**Windows (PowerShell):**
```powershell
npm root -g
# Typically: C:\Users\<user>\AppData\Roaming\npm\node_modules
```

### MCP Not Available in Claude

**Step 1: Verify the binary exists and runs:**

```bash
mycelicmemory --version
mycelicmemory doctor
```

**Step 2: Test MCP mode directly** (use the native binary, not the npm wrapper):

macOS/Linux:
```bash
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}' | "$(npm root -g)/mycelicmemory/bin/mycelicmemory-$(uname -s | tr A-Z a-z)-$(uname -m)" --mcp
```

Windows (PowerShell):
```powershell
'{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}' | & "$(npm root -g)\mycelicmemory\bin\mycelicmemory-windows-x64.exe" --mcp
```

You should see a JSON response containing `"serverInfo":{"name":"mycelicmemory"}`.

**Step 3: Validate your MCP config:**

macOS/Linux:
```bash
cat ~/.claude/mcp.json
python3 -m json.tool ~/.claude/mcp.json
```

Windows (PowerShell):
```powershell
Get-Content "$env:USERPROFILE\.claude\mcp.json"
```

**Step 4: Restart Claude Code** — MCP config changes require a full restart.

### Windows: npm Wrapper Not Working in MCP Mode

If `"command": "mycelicmemory"` fails on Windows, use the direct binary path instead. The npm wrapper routes through `cmd.exe -> node.js -> binary`, which can break stdin/stdout piping required by MCP. See the [Connect to Claude](#connect-to-claude) section for the direct path configuration.

### macOS Security Warning

If macOS blocks the binary:
1. Go to System Preferences > Security & Privacy > General
2. Click "Allow Anyway" next to the mycelicmemory message

### Full Diagnostics

```bash
mycelicmemory doctor
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

- **Repository**: [github.com/MycelicMemory/mycelicmemory](https://github.com/MycelicMemory/mycelicmemory)
- **Issues**: [GitHub Issues](https://github.com/MycelicMemory/mycelicmemory/issues)
- **npm**: [npmjs.com/package/mycelicmemory](https://npmjs.com/package/mycelicmemory)
