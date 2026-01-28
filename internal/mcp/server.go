package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/MycelicMemory/mycelicmemory/internal/ai"
	"github.com/MycelicMemory/mycelicmemory/internal/database"
	"github.com/MycelicMemory/mycelicmemory/internal/logging"
	"github.com/MycelicMemory/mycelicmemory/internal/memory"
	"github.com/MycelicMemory/mycelicmemory/internal/ratelimit"
	"github.com/MycelicMemory/mycelicmemory/internal/relationships"
	"github.com/MycelicMemory/mycelicmemory/internal/search"
	"github.com/MycelicMemory/mycelicmemory/pkg/config"
)

const (
	ProtocolVersion = "2024-11-05"
	ServerName      = "mycelicmemory"
	ServerVersion   = "1.2.0"
)

// Server implements the MCP server
type Server struct {
	db           *database.Database
	cfg          *config.Config
	aiManager    *ai.Manager
	memSvc      *memory.Service
	searchEng   *search.Engine
	relSvc      *relationships.Service
	rateLimiter *ratelimit.Limiter
	formatter    *Formatter
	log          *logging.Logger

	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer

	mu          sync.Mutex
	initialized bool
}

// NewServer creates a new MCP server instance
func NewServer(db *database.Database, cfg *config.Config) *Server {
	log := logging.GetLogger("mcp")
	log.Info("initializing MCP server", "version", ServerVersion, "protocol", ProtocolVersion)

	// Initialize rate limiter with config or defaults
	var rateLimiterInstance *ratelimit.Limiter
	if cfg.RateLimit.Enabled {
		rateLimiterInstance = ratelimit.NewLimiter(&ratelimit.Config{
			Enabled: cfg.RateLimit.Enabled,
			Global: ratelimit.LimitConfig{
				RequestsPerSecond: cfg.RateLimit.Global.RequestsPerSecond,
				BurstSize:         cfg.RateLimit.Global.BurstSize,
			},
			Tools: convertToolLimits(cfg.RateLimit.Tools),
		})
		log.Info("rate limiting enabled", "global_rps", cfg.RateLimit.Global.RequestsPerSecond)
	}

	return &Server{
		db:          db,
		cfg:         cfg,
		aiManager:   ai.NewManager(db, cfg),
		memSvc:      memory.NewService(db, cfg),
		searchEng:   search.NewEngine(db, cfg),
		relSvc:      relationships.NewService(db, cfg),
		rateLimiter: rateLimiterInstance,
		formatter:   NewFormatter(),
		log:         log,
		stdin:       os.Stdin,
		stdout:      os.Stdout,
		stderr:      os.Stderr,
	}
}

// convertToolLimits converts config tool limits to ratelimit package format
func convertToolLimits(tools []config.ToolLimitConfig) []ratelimit.ToolLimit {
	result := make([]ratelimit.ToolLimit, len(tools))
	for i, t := range tools {
		result[i] = ratelimit.ToolLimit{
			Name:              t.Name,
			RequestsPerSecond: t.RequestsPerSecond,
			BurstSize:         t.BurstSize,
		}
	}
	return result
}

// Run starts the MCP server main loop
func (s *Server) Run(ctx context.Context) error {
	s.log.Info("starting MCP server main loop")
	scanner := bufio.NewScanner(s.stdin)
	// Increase buffer size for large requests
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			s.log.Info("context cancelled, shutting down")
			return ctx.Err()
		default:
		}

		line := scanner.Text()
		if line == "" {
			continue
		}

		response := s.handleRequest(ctx, line)
		if response != nil {
			s.sendResponse(response)
		}
	}

	if err := scanner.Err(); err != nil {
		s.log.Error("scanner error", "error", err)
		return fmt.Errorf("scanner error: %w", err)
	}

	s.log.Info("MCP server shutdown complete")
	return nil
}

// handleRequest processes a single JSON-RPC request
func (s *Server) handleRequest(ctx context.Context, line string) *Response {
	var req Request
	if err := json.Unmarshal([]byte(line), &req); err != nil {
		s.log.Error("failed to parse request", "error", err)
		return &Response{
			JSONRPC: "2.0",
			Error: &RPCError{
				Code:    ParseError,
				Message: "Parse error",
				Data:    err.Error(),
			},
		}
	}

	s.log.Debug("received request", "method", req.Method, "id", req.ID)

	if req.JSONRPC != "2.0" {
		s.log.Warn("invalid jsonrpc version", "version", req.JSONRPC)
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &RPCError{
				Code:    InvalidRequest,
				Message: "Invalid Request",
				Data:    "jsonrpc must be '2.0'",
			},
		}
	}

	// Handle different methods
	switch req.Method {
	case "initialize":
		s.log.Info("handling initialize request")
		return s.handleInitialize(req)
	case "initialized":
		s.log.Debug("received initialized notification")
		// Notification, no response needed
		return nil
	case "tools/list":
		s.log.Debug("handling tools/list request")
		return s.handleToolsList(req)
	case "tools/call":
		return s.handleToolsCall(ctx, req)
	case "prompts/list":
		s.log.Debug("handling prompts/list request")
		return s.handlePromptsList(req)
	case "prompts/get":
		s.log.Debug("handling prompts/get request")
		return s.handlePromptsGet(req)
	case "ping":
		s.log.Debug("handling ping request")
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  map[string]interface{}{},
		}
	default:
		s.log.Warn("method not found", "method", req.Method)
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &RPCError{
				Code:    MethodNotFound,
				Message: "Method not found",
				Data:    req.Method,
			},
		}
	}
}

// handleInitialize handles the initialize request
func (s *Server) handleInitialize(req Request) *Response {
	s.mu.Lock()
	s.initialized = true
	s.mu.Unlock()

	return &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: InitializeResult{
			ProtocolVersion: ProtocolVersion,
			Capabilities: ServerCapabilities{
				Tools: &ToolsCapability{
					ListChanged: false,
				},
				// Prompts disabled for now to match local-memory compatibility
				// Prompts: &PromptsCapability{
				// 	ListChanged: false,
				// },
			},
			ServerInfo: ServerInfo{
				Name:        ServerName,
				Version:     ServerVersion,
				Description: "AI-powered persistent memory system with semantic search and relationship mapping",
			},
		},
	}
}

// handlePromptsList returns available prompts for automatic behavior
func (s *Server) handlePromptsList(req Request) *Response {
	prompts := []Prompt{
		{
			Name:        "auto-memory",
			Description: "Instructions for automatic memory storage and retrieval",
			Arguments:   []PromptArgument{},
		},
	}

	return &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: PromptsListResult{
			Prompts: prompts,
		},
	}
}

// handlePromptsGet returns the content of a specific prompt
func (s *Server) handlePromptsGet(req Request) *Response {
	var params struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &RPCError{
				Code:    InvalidParams,
				Message: "Invalid params",
				Data:    err.Error(),
			},
		}
	}

	if params.Name != "auto-memory" {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &RPCError{
				Code:    InvalidParams,
				Message: "Prompt not found",
				Data:    params.Name,
			},
		}
	}

	promptContent := `# MyclicMemory Automatic Memory System

You have access to persistent memory. Use it proactively to build and leverage a knowledge base.

## AUTOMATIC SEARCH (Do this first!)
At the START of conversations, search for relevant context:
- Search for memories related to the user's topic/question
- Check for past decisions, preferences, and learnings before answering

## AUTOMATIC STORAGE (Do this continuously)
Store memories when the user shares valuable information:

| Type | Example Trigger | Tags | Importance |
|------|-----------------|------|------------|
| Technical decision | "We chose X because..." | decision, <tech> | 8-9 |
| Debugging insight | "The bug was caused by..." | debugging, gotcha | 7-9 |
| Architecture | "This service handles..." | architecture | 8 |
| Preference | "I prefer X over Y" | preference | 7 |
| Learning | "TIL..." or "I learned..." | learning | 6-8 |
| Project context | Tech stack, conventions | project | 9-10 |

## TAGGING STRATEGY
Use consistent, searchable tags:
- decision, debugging, gotcha, preference, learning, architecture
- Language: go, python, typescript, rust, etc.
- Domain: frontend, backend, devops, database, etc.

## WHAT TO STORE
✅ Store: Future-useful info, debugging insights, project conventions, preferences
❌ Don't store: Generic knowledge, temporary info, sensitive data

## RELATIONSHIP BUILDING
When storing related concepts:
1. Search for existing related memories
2. Create relationships between connected memories using the relationships tool
3. Build a knowledge graph over time`

	return &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: PromptGetResult{
			Description: "Instructions for automatic memory storage and retrieval",
			Messages: []PromptMessage{
				{
					Role: "user",
					Content: ContentBlock{
						Type: "text",
						Text: promptContent,
					},
				},
			},
		},
	}
}

// handleToolsList returns the list of available tools
func (s *Server) handleToolsList(req Request) *Response {
	return &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: ToolsListResult{
			Tools: s.getToolDefinitions(),
		},
	}
}

// handleToolsCall handles tool invocation
func (s *Server) handleToolsCall(ctx context.Context, req Request) *Response {
	var params CallToolParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.log.Error("failed to parse tool params", "error", err)
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &RPCError{
				Code:    InvalidParams,
				Message: "Invalid params",
				Data:    err.Error(),
			},
		}
	}

	s.log.LogRequest("tools/call", "tool", params.Name)

	// Rate limit check
	if s.rateLimiter != nil {
		result := s.rateLimiter.Allow(params.Name)
		if !result.Allowed {
			s.log.Warn("rate limit exceeded", "tool", params.Name, "limit_type", result.LimitType, "retry_after_ms", result.RetryAfter.Milliseconds())
			return &Response{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: &RPCError{
					Code:    RateLimitExceeded,
					Message: "Rate limit exceeded",
					Data: RateLimitErrorData{
						RetryAfterMs: result.RetryAfter.Milliseconds(),
						LimitType:    result.LimitType,
						Message:      fmt.Sprintf("Rate limit exceeded for %s. Retry after %v.", result.LimitType, result.RetryAfter),
					},
				},
			}
		}
	}

	// Track execution time
	startTime := time.Now()

	result, err := s.callTool(ctx, params.Name, params.Arguments)
	if err != nil {
		duration := time.Since(startTime).Seconds() * 1000
		s.log.LogError("tool_call", err, "tool", params.Name, "duration_ms", duration)
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: CallToolResult{
				Content: []ContentBlock{
					{Type: "text", Text: fmt.Sprintf("❌ **Error**\n\n```\n%v\n```", err)},
				},
				IsError: true,
			},
		}
	}

	duration := time.Since(startTime)
	durationMs := duration.Seconds() * 1000
	s.log.LogResponse("tools/call", durationMs, "tool", params.Name)

	// Format the response with rich UX
	formattedOutput := s.formatter.FormatToolResponse(params.Name, result, duration)

	return &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: CallToolResult{
			Content: []ContentBlock{
				{Type: "text", Text: formattedOutput},
			},
		},
	}
}

// callTool dispatches to the appropriate tool handler
func (s *Server) callTool(ctx context.Context, name string, args map[string]interface{}) (interface{}, error) {
	// Convert args to JSON for parsing
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal arguments: %w", err)
	}

	switch name {
	case "store_memory":
		return s.handleStoreMemory(ctx, argsJSON)
	case "search":
		return s.handleSearch(ctx, argsJSON)
	case "analysis":
		return s.handleAnalysis(ctx, argsJSON)
	case "relationships":
		return s.handleRelationships(ctx, argsJSON)
	case "categories":
		return s.handleCategories(ctx, argsJSON)
	case "domains":
		return s.handleDomains(ctx, argsJSON)
	case "sessions":
		return s.handleSessions(ctx, argsJSON)
	case "stats":
		return s.handleStats(ctx, argsJSON)
	case "get_memory_by_id":
		return s.handleGetMemory(ctx, argsJSON)
	case "update_memory":
		return s.handleUpdateMemory(ctx, argsJSON)
	case "delete_memory":
		return s.handleDeleteMemory(ctx, argsJSON)
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

// sendResponse sends a JSON-RPC response to stdout
func (s *Server) sendResponse(resp *Response) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.Marshal(resp)
	if err != nil {
		s.log.Error("failed to marshal response", "error", err)
		return
	}

	fmt.Fprintln(s.stdout, string(data))
}

// getToolDefinitions returns all tool definitions
func (s *Server) getToolDefinitions() []Tool {
	min1 := float64(1)
	max10 := float64(10)
	min0 := float64(0)
	max1 := float64(1)

	return []Tool{
		{
			Name:        "store_memory",
			Description: "Store a new memory with contextual information for later retrieval",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"content": {
						Type:        "string",
						Description: "The memory content to store",
					},
					"importance": {
						Type:        "integer",
						Description: "Importance level (1-10)",
						Default:     5,
						Minimum:     &min1,
						Maximum:     &max10,
					},
					"tags": {
						Type:        "array",
						Description: "Tags for categorization",
						Items:       &Property{Type: "string"},
					},
					"domain": {
						Type:        "string",
						Description: "Knowledge domain",
					},
					"source": {
						Type:        "string",
						Description: "Source of the memory",
					},
				},
				Required: []string{"content"},
			},
		},
		{
			Name:        "search",
			Description: "Search memories using semantic, keyword, tag, or date-based queries",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"query": {
						Type:        "string",
						Description: "Search query text",
					},
					"search_type": {
						Type:        "string",
						Description: "Type of search",
						Enum:        []string{"semantic", "tags", "date_range", "hybrid"},
						Default:     "semantic",
					},
					"use_ai": {
						Type:        "boolean",
						Description: "Enable AI-powered semantic search",
						Default:     false,
					},
					"limit": {
						Type:        "integer",
						Description: "Maximum number of results",
						Default:     10,
					},
					"tags": {
						Type:        "array",
						Description: "Filter by tags",
						Items:       &Property{Type: "string"},
					},
					"domain": {
						Type:        "string",
						Description: "Filter by domain",
					},
					"response_format": {
						Type:        "string",
						Description: "Response format",
						Enum:        []string{"detailed", "concise", "ids_only", "summary"},
						Default:     "concise",
					},
				},
			},
		},
		{
			Name:        "analysis",
			Description: "AI-powered analysis: question answering, summarization, pattern detection, and temporal analysis",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"analysis_type": {
						Type:        "string",
						Description: "Type of analysis",
						Enum:        []string{"question", "summarize", "analyze", "temporal_patterns"},
						Default:     "question",
					},
					"question": {
						Type:        "string",
						Description: "Question to answer (for question type)",
					},
					"query": {
						Type:        "string",
						Description: "Filter query for memories",
					},
					"timeframe": {
						Type:        "string",
						Description: "Time period for analysis",
						Enum:        []string{"today", "week", "month", "all"},
						Default:     "all",
					},
					"limit": {
						Type:        "integer",
						Description: "Maximum memories to analyze",
						Default:     10,
					},
				},
			},
		},
		{
			Name:        "relationships",
			Description: "Manage memory relationships: find related, discover connections, create links, map graphs",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"relationship_type": {
						Type:        "string",
						Description: "Operation type",
						Enum:        []string{"find_related", "discover", "create", "map_graph"},
						Default:     "find_related",
					},
					"memory_id": {
						Type:        "string",
						Description: "Target memory ID",
					},
					"source_memory_id": {
						Type:        "string",
						Description: "Source memory ID (for create)",
					},
					"target_memory_id": {
						Type:        "string",
						Description: "Target memory ID (for create)",
					},
					"relationship_type_enum": {
						Type:        "string",
						Description: "Type of relationship",
						Enum:        []string{"references", "contradicts", "expands", "similar", "sequential", "causes", "enables"},
						Default:     "references",
					},
					"strength": {
						Type:        "number",
						Description: "Relationship strength (0-1)",
						Default:     0.5,
						Minimum:     &min0,
						Maximum:     &max1,
					},
					"depth": {
						Type:        "integer",
						Description: "Graph traversal depth",
						Default:     2,
					},
				},
			},
		},
		{
			Name:        "categories",
			Description: "Manage categories: list, create, and auto-categorize memories",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"categories_type": {
						Type:        "string",
						Description: "Operation type",
						Enum:        []string{"list", "create", "categorize"},
						Default:     "list",
					},
					"name": {
						Type:        "string",
						Description: "Category name (for create)",
					},
					"description": {
						Type:        "string",
						Description: "Category description",
					},
					"memory_id": {
						Type:        "string",
						Description: "Memory ID to categorize",
					},
				},
			},
		},
		{
			Name:        "domains",
			Description: "Manage knowledge domains: list, create, and get statistics",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"domains_type": {
						Type:        "string",
						Description: "Operation type",
						Enum:        []string{"list", "create", "stats"},
						Default:     "list",
					},
					"name": {
						Type:        "string",
						Description: "Domain name",
					},
					"description": {
						Type:        "string",
						Description: "Domain description",
					},
					"domain": {
						Type:        "string",
						Description: "Domain name for statistics",
					},
				},
			},
		},
		{
			Name:        "sessions",
			Description: "Manage sessions: list all sessions or get session statistics",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"sessions_type": {
						Type:        "string",
						Description: "Operation type",
						Enum:        []string{"list", "stats"},
						Default:     "list",
					},
				},
			},
		},
		{
			Name:        "stats",
			Description: "Get system statistics for sessions, domains, or categories",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"stats_type": {
						Type:        "string",
						Description: "Type of statistics",
						Enum:        []string{"session", "domain", "category"},
						Default:     "session",
					},
					"domain": {
						Type:        "string",
						Description: "Domain name for domain stats",
					},
					"category_id": {
						Type:        "string",
						Description: "Category ID for category stats",
					},
				},
			},
		},
		{
			Name:        "get_memory_by_id",
			Description: "Retrieve a specific memory by its UUID",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"id": {
						Type:        "string",
						Description: "Memory UUID",
					},
				},
				Required: []string{"id"},
			},
		},
		{
			Name:        "update_memory",
			Description: "Update an existing memory's content, importance, or tags",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"id": {
						Type:        "string",
						Description: "Memory UUID to update",
					},
					"content": {
						Type:        "string",
						Description: "New content",
					},
					"importance": {
						Type:        "integer",
						Description: "New importance level (1-10)",
						Minimum:     &min1,
						Maximum:     &max10,
					},
					"tags": {
						Type:        "array",
						Description: "New tags",
						Items:       &Property{Type: "string"},
					},
				},
				Required: []string{"id"},
			},
		},
		{
			Name:        "delete_memory",
			Description: "Delete a memory by its UUID",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"id": {
						Type:        "string",
						Description: "Memory UUID to delete",
					},
				},
				Required: []string{"id"},
			},
		},
	}
}
