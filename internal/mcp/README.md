# MCP Server

Model Context Protocol (MCP) server implementing JSON-RPC 2.0 over stdio for Claude integration.

## Quick Start

### Claude Code Setup

1. Add to `~/.claude/mcp.json`:
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

2. Restart Claude Code

3. Verify tools are available (should see `mcp__ultrathink__*` tools)

### Claude Desktop Setup

Add to your Claude Desktop config:
- **macOS:** `~/Library/Application Support/Claude/claude_desktop_config.json`
- **Windows:** `%APPDATA%\Claude\claude_desktop_config.json`
- **Linux:** `~/.config/Claude/claude_desktop_config.json`

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

## Architecture

```
internal/mcp/
├── server.go      # JSON-RPC 2.0 protocol, stdio transport, request handling
├── types.go       # Request/response types, tool parameter structs
└── README.md      # This file
```

## Available Tools (11)

| Tool | Description | Key Parameters |
|------|-------------|----------------|
| `store_memory` | Create new memory | content, importance, tags, domain, source |
| `search` | Multi-mode search | query, search_type, use_ai, limit, tags |
| `analysis` | AI-powered analysis | analysis_type, question, timeframe, concept |
| `relationships` | Relationship operations | relationship_type, memory_id, strength |
| `categories` | Category management | categories_type, name, memory_id |
| `domains` | Domain management | domains_type, name, domain |
| `sessions` | Session management | sessions_type |
| `stats` | System statistics | stats_type, domain, category_id |
| `get_memory_by_id` | Retrieve memory | id |
| `update_memory` | Modify memory | id, content, importance, tags |
| `delete_memory` | Remove memory | id |

## JSON-RPC 2.0 Protocol

### Request Format

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "store_memory",
    "arguments": {
      "content": "Example memory content",
      "importance": 8,
      "tags": ["example", "test"]
    }
  }
}
```

### Response Format

**Success:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{\"id\":\"550e8400-e29b-41d4-a716-446655440000\",\"created_at\":\"2025-12-30T15:30:00Z\"}"
      }
    ]
  }
}
```

**Error:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "error": {
    "code": -32602,
    "message": "Invalid params: content is required"
  }
}
```

## Tool Details

### store_memory

Store a new memory with metadata.

```json
{
  "content": "Go channels are typed conduits for communication",
  "importance": 8,
  "tags": ["go", "concurrency"],
  "domain": "programming",
  "source": "documentation"
}
```

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| content | string | Yes | - | Memory content |
| importance | int | No | 5 | Importance 1-10 |
| tags | []string | No | [] | Tags for categorization |
| domain | string | No | "" | Knowledge domain |
| source | string | No | "" | Memory source |

### search

Search memories with multiple modes.

```json
{
  "query": "concurrency patterns",
  "search_type": "semantic",
  "use_ai": true,
  "limit": 10,
  "domain": "programming",
  "response_format": "concise"
}
```

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| query | string | Depends | - | Search query |
| search_type | string | No | "semantic" | semantic, tags, date_range, hybrid |
| use_ai | bool | No | false | Enable AI-powered search |
| limit | int | No | 10 | Max results |
| tags | []string | No | [] | Filter by tags |
| domain | string | No | "" | Filter by domain |
| response_format | string | No | "detailed" | detailed, concise, ids_only, summary |

### analysis

AI-powered memory analysis.

```json
{
  "analysis_type": "question",
  "question": "What have I learned about Go?",
  "limit": 10,
  "timeframe": "week"
}
```

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| analysis_type | string | No | "question" | question, summarize, analyze, temporal_patterns |
| question | string | Depends | - | Question to answer (for type=question) |
| query | string | No | "" | Filter memories |
| timeframe | string | No | "all" | today, week, month, all |
| limit | int | No | 10 | Max memories to analyze |
| concept | string | No | "" | Concept for temporal analysis |

### relationships

Manage memory relationships.

```json
{
  "relationship_type": "create",
  "source_memory_id": "uuid-1",
  "target_memory_id": "uuid-2",
  "relationship_type_enum": "similar",
  "strength": 0.8,
  "context": "Both discuss async patterns"
}
```

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| relationship_type | string | No | "find_related" | find_related, discover, create, map_graph |
| memory_id | string | Depends | - | Central memory for find/map |
| source_memory_id | string | Depends | - | Source for create |
| target_memory_id | string | Depends | - | Target for create |
| relationship_type_enum | string | No | "similar" | references, contradicts, expands, similar, sequential, causes, enables |
| strength | float | No | 0.5 | Relationship strength 0.0-1.0 |
| depth | int | No | 2 | Graph traversal depth 1-5 |

## Testing MCP Mode

### Manual Testing

```bash
# Test initialization
echo '{"jsonrpc":"2.0","id":1,"method":"initialize"}' | ultrathink --mcp

# Test tools/list
echo '{"jsonrpc":"2.0","id":2,"method":"tools/list"}' | ultrathink --mcp

# Test store_memory
echo '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"store_memory","arguments":{"content":"Test memory"}}}' | ultrathink --mcp
```

### Interactive Testing

```bash
ultrathink --mcp
# Then type JSON-RPC requests, one per line
```

## Error Codes

| Code | Meaning |
|------|---------|
| -32700 | Parse error |
| -32600 | Invalid request |
| -32601 | Method not found |
| -32602 | Invalid params |
| -32603 | Internal error |

## Files

### server.go

- `MCPServer` struct with database and config
- `Run()` - Main stdio loop
- `handleRequest()` - Route requests to handlers
- `handleToolsCall()` - Execute tool calls
- Tool handlers: `handleStoreMemory`, `handleSearch`, etc.

### types.go

- JSON-RPC types: `Request`, `Response`, `RPCError`
- MCP types: `ServerInfo`, `ServerCapabilities`, `Tool`, `InputSchema`
- Tool parameter structs: `StoreMemoryParams`, `SearchParams`, etc.
