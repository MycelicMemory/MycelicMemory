package api

import (
	"github.com/gin-gonic/gin"

	"github.com/MycelicMemory/mycelicmemory/internal/memory"
)

// seedMemory is a convenience struct for defining seed data
type seedMemory struct {
	Content    string
	Domain     string
	Importance int
	Tags       []string
	Source     string
}

// seedMemories handles POST /api/v1/seed
func (s *Server) seedMemories(c *gin.Context) {
	seeds := getSeedData()

	created := 0
	for _, seed := range seeds {
		opts := &memory.StoreOptions{
			Content:    seed.Content,
			Domain:     seed.Domain,
			Importance: seed.Importance,
			Tags:       seed.Tags,
			Source:     seed.Source,
		}

		if _, err := s.memoryService.Store(opts); err != nil {
			s.log.Warn("Failed to store seed memory", "error", err, "content_prefix", seed.Content[:min(50, len(seed.Content))])
			continue
		}
		created++
	}

	// Run batch keyword discovery to auto-create relationships
	if s.relService != nil {
		discovered, err := s.relService.BatchDiscoverKeyword(200, 0.3, "")
		if err != nil {
			s.log.Warn("Seed relationship discovery failed", "error", err)
		} else {
			s.log.Info("Seed relationships discovered", "count", discovered)
		}
	}

	SuccessResponse(c, "Seed memories created", map[string]interface{}{
		"memories_created": created,
		"total_seeds":      len(seeds),
	})
}

func getSeedData() []seedMemory {
	return []seedMemory{
		// Architecture
		{
			Content:    "MycelicMemory is an AI-powered persistent memory system. It stores memories in SQLite with FTS5 full-text search, uses Ollama for embeddings/AI analysis, and Qdrant for vector similarity search. The MCP server communicates over JSON-RPC 2.0 via stdin/stdout.",
			Domain:     "architecture",
			Importance: 10,
			Tags:       []string{"overview", "architecture", "sqlite", "ollama", "qdrant", "mcp"},
			Source:     "seed",
		},
		{
			Content:    "The Go backend is organized into internal packages: api (REST/Gin), mcp (JSON-RPC server), database (SQLite+FTS5), memory (business logic), ai (Ollama client), vector (Qdrant client), relationships (graph service), search (unified search), recall (active memory agent), pipeline (universal ingestion), claude (JSONL parser), and adapters/ (source-specific adapters like Slack).",
			Domain:     "architecture",
			Importance: 9,
			Tags:       []string{"packages", "go", "structure", "backend"},
			Source:     "seed",
		},
		{
			Content:    "The desktop app is an Electron application with React/TypeScript renderer. Communication flows: Renderer -> IPC (preload.ts) -> Main process (memory.ipc.ts) -> MycelicMemoryClient (REST) -> Go backend. Browser mode uses api-bridge.ts for direct fetch fallback.",
			Domain:     "architecture",
			Importance: 9,
			Tags:       []string{"desktop", "electron", "ipc", "react", "typescript"},
			Source:     "seed",
		},
		{
			Content:    "All REST API responses use a wrapped format: {success: bool, message: string, data: <payload>}. Desktop clients must unwrap response.data from all endpoints. The api-bridge.ts fetchApi() auto-unwraps data.data ?? data.",
			Domain:     "api",
			Importance: 9,
			Tags:       []string{"api", "response-format", "rest", "unwrapping"},
			Source:     "seed",
		},

		// Schema
		{
			Content:    "Database schema is at version 5. Migration v4->v5 renamed cc_sessions->conversations, cc_messages->messages, cc_tool_calls->actions, and memories.cc_session_id->conversation_id. Go types have backwards-compatible aliases (CCSession=Conversation, etc).",
			Domain:     "schema",
			Importance: 8,
			Tags:       []string{"schema", "migration", "v5", "database"},
			Source:     "seed",
		},
		{
			Content:    "Core database tables: memories (content, domain, importance, tags, source), relationships (source_memory_id, target_memory_id, type, strength), conversations (session_id, project), messages (role, content, sequence), actions (tool_name, input, result), data_sources (type, config, status).",
			Domain:     "schema",
			Importance: 8,
			Tags:       []string{"tables", "schema", "database", "sqlite"},
			Source:     "seed",
		},

		// Build
		{
			Content:    "CRITICAL: On Windows with TDM-GCC, must use -ldflags '-s -w' when building. Without strip flags, the 57MB binary has 24 PE sections and Windows refuses to run it. With flags: 25MB, 12 sections, works fine. Build: CGO_ENABLED=1 go build -tags 'fts5' -ldflags '-s -w' -o mycelicmemory.exe ./cmd/mycelicmemory",
			Domain:     "build",
			Importance: 10,
			Tags:       []string{"build", "windows", "gcc", "ldflags", "critical"},
			Source:     "seed",
		},
		{
			Content:    "The FTS5 build tag is required for SQLite full-text search. Without it, FTS5 tables fail to create. The C compiler must be TDM-GCC 10.3.0 at /c/TDM-GCC-64/bin/gcc on Windows.",
			Domain:     "build",
			Importance: 8,
			Tags:       []string{"build", "fts5", "sqlite", "cgo"},
			Source:     "seed",
		},

		// MCP
		{
			Content:    "The MCP server exposes 20+ tools including: store_memory, search (semantic/keyword/hybrid/tags), get_memory_by_id, update_memory, delete_memory, relationships (find_related, discover, create, map_graph), analysis (question, summarize, analyze, temporal_patterns), categories, domains, sessions, ingest_conversations, search_chats, get_chat, trace_source, ingest_source, pipeline_status, list_sources, context_recall, reindex_memories.",
			Domain:     "mcp",
			Importance: 9,
			Tags:       []string{"mcp", "tools", "json-rpc", "api"},
			Source:     "seed",
		},
		{
			Content:    "MCP search supports 4 search types: semantic (vector similarity via Qdrant), keyword (FTS5 full-text), hybrid (both combined), and tags (tag-based filtering). The use_ai flag enables AI-powered semantic search when available.",
			Domain:     "mcp",
			Importance: 8,
			Tags:       []string{"search", "mcp", "fts5", "semantic", "qdrant"},
			Source:     "seed",
		},

		// Pipeline
		{
			Content:    "The universal ingestion pipeline (internal/pipeline/) uses the SourceAdapter interface. Queue processes adapters with backfill (full history) and incremental (checkpoint-based) modes. Registered adapters: claude-code-local (JSONL files), slack (workspace exports). Valid source types include: claude-code-local, slack, discord, telegram, imessage, email, browser, notion, obsidian, github, custom.",
			Domain:     "pipeline",
			Importance: 8,
			Tags:       []string{"pipeline", "ingestion", "adapters", "sources"},
			Source:     "seed",
		},
		{
			Content:    "The Claude adapter (internal/claude/adapter.go) implements SourceAdapter for parsing ~/.claude/projects/*/JSONL files. It creates Conversation records with messages and actions, and generates summary memories as graph nodes. 323+ sessions ingested with deduplication.",
			Domain:     "pipeline",
			Importance: 7,
			Tags:       []string{"claude", "adapter", "jsonl", "ingestion"},
			Source:     "seed",
		},
		{
			Content:    "The Slack adapter (internal/adapters/slack/) parses Slack workspace exports: channels.json, users.json, groups.json, dms.json, per-channel YYYY-MM-DD.json. Converts messages, threads, reactions, file attachments to ConversationItem. Supports include_private and include_dms config options.",
			Domain:     "pipeline",
			Importance: 7,
			Tags:       []string{"slack", "adapter", "ingestion", "export"},
			Source:     "seed",
		},

		// Recall
		{
			Content:    "The Active Memory Agent (internal/recall/) provides multi-signal context recall. Scoring weights: 0.40 semantic + 0.20 keyword + 0.15 importance + 0.15 recency + 0.10 graph. Graceful degradation: full (Ollama+Qdrant) -> keyword_graph -> keyword_only.",
			Domain:     "recall",
			Importance: 9,
			Tags:       []string{"recall", "scoring", "agent", "semantic", "degradation"},
			Source:     "seed",
		},
		{
			Content:    "Recall context_recall MCP tool takes: context (current working context), files (active file paths), project (project identifier), limit (max results), depth (graph traversal depth). Returns scored memories ranked by multi-signal relevance.",
			Domain:     "recall",
			Importance: 8,
			Tags:       []string{"recall", "mcp", "context", "tool"},
			Source:     "seed",
		},

		// Relationships
		{
			Content:    "Relationship types: references, contradicts, expands, similar, sequential, causes, enables. Each has a strength (0-1). Auto-discovery uses keyword-based batch matching (fast, no AI) or pairwise Ollama analysis (slow, AI-powered). The graph service supports depth-first traversal with MapGraphOptimized (2 queries instead of N+1).",
			Domain:     "relationships",
			Importance: 8,
			Tags:       []string{"relationships", "graph", "types", "discovery"},
			Source:     "seed",
		},
		{
			Content:    "BatchDiscoverKeyword scans all memories for shared tags, domains, and content keywords to create relationships without AI. Much faster than Ollama-based discovery (seconds vs minutes). Default threshold: 0.3 min_score.",
			Domain:     "relationships",
			Importance: 7,
			Tags:       []string{"discovery", "keyword", "batch", "performance"},
			Source:     "seed",
		},

		// Desktop
		{
			Content:    "Desktop UX features: Toast notifications (react-hot-toast), ErrorBoundary, ConfirmDialog, CreateMemoryModal (Ctrl+N), CommandPalette (Ctrl+K), collapsible sidebar (Ctrl+B), pagination on MemoryBrowser (50 per page).",
			Domain:     "desktop",
			Importance: 7,
			Tags:       []string{"desktop", "ux", "keyboard", "modal"},
			Source:     "seed",
		},
		{
			Content:    "Desktop IPC: preload.ts exposes sections: memory, claude, stats, domains, relationships, databases, settings, claudeStream, services, shell, app. Each maps to IPC handlers in memory.ipc.ts that call MycelicMemoryClient REST methods.",
			Domain:     "desktop",
			Importance: 8,
			Tags:       []string{"desktop", "ipc", "preload", "electron"},
			Source:     "seed",
		},
		{
			Content:    "The Knowledge Graph page (KnowledgeGraph.tsx) uses vis-network to visualize memories as nodes and relationships as edges. Memory nodes colored by domain, sized by importance. Session summary nodes shown as diamonds, chat sessions as stars. Discover button triggers keyword-based relationship discovery.",
			Domain:     "desktop",
			Importance: 7,
			Tags:       []string{"knowledge-graph", "visualization", "vis-network", "ui"},
			Source:     "seed",
		},
		{
			Content:    "The Data Sources page (/sources route) shows configured data sources with list+detail panels. Supports sync/pause/resume/delete operations. triggerSync uses pipeline queue for async ingestion. api-bridge.ts has 40+ endpoints for sources CRUD, categories, domains, relationships, analysis, search.",
			Domain:     "desktop",
			Importance: 7,
			Tags:       []string{"data-sources", "ui", "sync", "pipeline"},
			Source:     "seed",
		},

		// Config
		{
			Content:    "Configuration loaded from config.yaml (searched in: ./config.yaml, ~/.mycelicmemory/config.yaml, /etc/mycelicmemory/config.yaml). Key sections: database, rest_api, session, ollama, qdrant, rate_limit. Supports multiple database profiles with active_database selector.",
			Domain:     "config",
			Importance: 8,
			Tags:       []string{"config", "yaml", "profiles", "database"},
			Source:     "seed",
		},
		{
			Content:    "Qdrant Cloud support: QdrantConfig accepts api_key field. The QdrantClient sends api-key header on all requests. Configure via qdrant.api_key in config.yaml. Supports both local and cloud Qdrant instances.",
			Domain:     "config",
			Importance: 7,
			Tags:       []string{"qdrant", "cloud", "api-key", "config"},
			Source:     "seed",
		},

		// API Endpoints
		{
			Content:    "REST API endpoints organized as: /api/v1/memories (CRUD, search, stats, related, graph, trace, reindex), /api/v1/relationships (CRUD, discover, batch-discover), /api/v1/graph/stats, /api/v1/categories, /api/v1/domains, /api/v1/sessions, /api/v1/stats, /api/v1/sources (CRUD, sync, history), /api/v1/chats (ingest, search, sessions, messages), /api/v1/databases (CRUD, switch, archive), /api/v1/models, /api/v1/config, /api/v1/recall, /api/v1/analyze, /api/v1/seed.",
			Domain:     "api",
			Importance: 9,
			Tags:       []string{"api", "rest", "endpoints", "routes"},
			Source:     "seed",
		},
		{
			Content:    "Stats endpoint (GET /api/v1/stats) returns total_memories (not memory_count). Client maps this. Memory list returns [{memory: {...}, relevance_score}] — client extracts .memory. Health endpoint checks api, ollama, qdrant, database connectivity.",
			Domain:     "api",
			Importance: 7,
			Tags:       []string{"api", "stats", "health", "format"},
			Source:     "seed",
		},

		// Windows/Platform
		{
			Content:    "Windows/Git Bash gotchas: Git Bash mangles /F flags as paths — use MSYS_NO_PATHCONV=1 prefix. Can't run .exe directly from Git Bash — use cmd.exe /c or powershell.exe -Command. PowerShell $_ gets mangled by bash. Start-Process needs .cmd extension for npm.",
			Domain:     "platform",
			Importance: 7,
			Tags:       []string{"windows", "git-bash", "gotchas", "platform"},
			Source:     "seed",
		},

		// Running services
		{
			Content:    "Running services: Backend via 'mycelicmemory start' (or .exe with powershell Start-Process), Desktop via 'npm run dev' in desktop/. REST API on localhost:3002 by default. Health check: curl http://localhost:3002/api/v1/health. Daemon supports background mode with PID tracking.",
			Domain:     "operations",
			Importance: 8,
			Tags:       []string{"running", "services", "daemon", "startup"},
			Source:     "seed",
		},

		// Multi-database
		{
			Content:    "Multi-database management: CLI commands 'db list/create/switch/delete/archive/import/export/info'. REST endpoints under /api/v1/databases. Databases stored in ~/.mycelicmemory/databases/<name>.db. Default database stays at configured path. Archives go to ~/.mycelicmemory/backups/.",
			Domain:     "database",
			Importance: 8,
			Tags:       []string{"multi-database", "management", "archive", "cli"},
			Source:     "seed",
		},

		// Chat History
		{
			Content:    "Chat history integration: 75+ sessions, 20k+ messages, 6k+ tool calls ingested from local Claude history. MCP tools: ingest_conversations, search_chats, get_chat, trace_source. Summary memories created as graph nodes with domain='conversations'. Sequential relationships auto-created between sessions in same project.",
			Domain:     "chat-history",
			Importance: 8,
			Tags:       []string{"chat-history", "claude", "sessions", "ingestion"},
			Source:     "seed",
		},

		// Search
		{
			Content:    "Search endpoints: POST /memories/search (semantic, keyword, hybrid, tags), POST /search/tags (tag-based with AND/OR operator), POST /search/date-range, POST /memories/search/intelligent (AI-powered). FTS5 search uses content, tags, domain columns. Semantic search requires Ollama+Qdrant.",
			Domain:     "search",
			Importance: 8,
			Tags:       []string{"search", "fts5", "semantic", "endpoints"},
			Source:     "seed",
		},
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
