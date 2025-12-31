package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/MycelicMemory/ultrathink/internal/ai"
	"github.com/MycelicMemory/ultrathink/internal/database"
	"github.com/MycelicMemory/ultrathink/internal/memory"
	"github.com/MycelicMemory/ultrathink/internal/relationships"
	"github.com/MycelicMemory/ultrathink/internal/search"
	"github.com/MycelicMemory/ultrathink/pkg/config"
)

const (
	ProtocolVersion = "2024-11-05"
	ServerName      = "ultrathink"
	ServerVersion   = "1.2.0"
)

// Server implements the MCP server
type Server struct {
	db        *database.Database
	cfg       *config.Config
	aiManager *ai.Manager
	memSvc    *memory.Service
	searchEng *search.Engine
	relSvc    *relationships.Service

	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer

	mu          sync.Mutex
	initialized bool
}

// NewServer creates a new MCP server instance
func NewServer(db *database.Database, cfg *config.Config) *Server {
	return &Server{
		db:        db,
		cfg:       cfg,
		aiManager: ai.NewManager(db, cfg),
		memSvc:    memory.NewService(db, cfg),
		searchEng: search.NewEngine(db, cfg),
		relSvc:    relationships.NewService(db, cfg),
		stdin:     os.Stdin,
		stdout:    os.Stdout,
		stderr:    os.Stderr,
	}
}

// Run starts the MCP server main loop
func (s *Server) Run(ctx context.Context) error {
	scanner := bufio.NewScanner(s.stdin)
	// Increase buffer size for large requests
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
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
		return fmt.Errorf("scanner error: %w", err)
	}

	return nil
}

// handleRequest processes a single JSON-RPC request
func (s *Server) handleRequest(ctx context.Context, line string) *Response {
	var req Request
	if err := json.Unmarshal([]byte(line), &req); err != nil {
		return &Response{
			JSONRPC: "2.0",
			Error: &RPCError{
				Code:    ParseError,
				Message: "Parse error",
				Data:    err.Error(),
			},
		}
	}

	if req.JSONRPC != "2.0" {
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
		return s.handleInitialize(req)
	case "initialized":
		// Notification, no response needed
		return nil
	case "tools/list":
		return s.handleToolsList(req)
	case "tools/call":
		return s.handleToolsCall(ctx, req)
	case "ping":
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  map[string]interface{}{},
		}
	default:
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
			},
			ServerInfo: ServerInfo{
				Name:    ServerName,
				Version: ServerVersion,
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

	result, err := s.callTool(ctx, params.Name, params.Arguments)
	if err != nil {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: CallToolResult{
				Content: []ContentBlock{
					{Type: "text", Text: fmt.Sprintf("Error: %v", err)},
				},
				IsError: true,
			},
		}
	}

	// Convert result to JSON for text response
	resultJSON, _ := json.MarshalIndent(result, "", "  ")

	return &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: CallToolResult{
			Content: []ContentBlock{
				{Type: "text", Text: string(resultJSON)},
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
		fmt.Fprintf(s.stderr, "Error marshaling response: %v\n", err)
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
