# Ultrathink

AI-powered persistent memory system for AI agents and humans.

Open source replica of local-memory with full MCP (Model Context Protocol) support.

## Installation

```bash
npm install -g ultrathink
```

## Quick Start

```bash
# Start the daemon
ultrathink start

# Store a memory
ultrathink remember "Go channels are typed conduits for communication between goroutines"

# Search memories
ultrathink search "concurrency"

# Ask questions about your memories
ultrathink analyze --question "What have I learned about Go?"

# Check status
ultrathink status
```

## MCP Integration

Add to your Claude Desktop config (`claude_desktop_config.json`):

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

Or for Claude Code:

```bash
claude mcp add ultrathink --transport stdio -- npx -y ultrathink --mcp
```

## Features

- **11 MCP Tools**: Full parity with local-memory v1.2.0
- **Semantic Search**: AI-powered search via Ollama embeddings
- **Knowledge Graph**: Relationship mapping between memories
- **AI Analysis**: Question answering and summarization
- **Session Management**: Automatic session detection via git
- **REST API**: Full HTTP API for programmatic access

## Requirements

- Node.js 16+
- SQLite (bundled)
- Ollama (optional, for AI features)
- Qdrant (optional, for vector search)

## Commands

| Command | Description |
|---------|-------------|
| `start` | Start the daemon |
| `stop` | Stop the daemon |
| `status` | Show daemon status |
| `remember <content>` | Store a new memory |
| `search <query>` | Search memories |
| `analyze` | AI-powered analysis |
| `relate <id1> <id2>` | Create memory relationship |
| `doctor` | Check system health |

## License

MIT
