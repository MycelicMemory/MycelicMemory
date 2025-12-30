# MCP Server Package

## Purpose

Model Context Protocol (MCP) server implementing JSON-RPC 2.0 over stdio.

## Components

### server.go
- JSON-RPC 2.0 protocol implementation
- stdio transport layer
- Request/response handling
- Tool registration

### tools.go
- All 11 MCP tool implementations
- Parameter validation
- Response formatting

## Verified MCP Tools (11)

1. **store_memory** - Create memory (content, importance, tags, domain, source)
2. **get_memory_by_id** - Retrieve by UUID
3. **update_memory** - Modify (content, importance, tags)
4. **delete_memory** - Remove memory
5. **search** - Multi-mode search (semantic, tags, date, hybrid)
6. **analysis** - AI Q&A, summarization, patterns, temporal
7. **relationships** - Find, discover, create, map graph
8. **categories** - List, create, auto-categorize
9. **domains** - List, create, stats
10. **sessions** - List, stats
11. **stats** - Session, domain, category metrics

## JSON-RPC 2.0 Format

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "store_memory",
  "params": {
    "content": "Example memory",
    "importance": 8,
    "tags": ["example"]
  }
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "created_at": "2025-12-30T15:30:00Z"
  }
}
```

## Usage

Invoked by Claude Desktop/Code via stdio transport.

## Related Issues

- #18: Implement JSON-RPC 2.0 MCP server with 11 tools
