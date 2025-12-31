package mcp

import "encoding/json"

// JSON-RPC 2.0 Protocol Types

// Request represents a JSON-RPC 2.0 request
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response represents a JSON-RPC 2.0 response
type Response struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

// RPCError represents a JSON-RPC 2.0 error
type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Standard JSON-RPC 2.0 error codes
const (
	ParseError     = -32700
	InvalidRequest = -32600
	MethodNotFound = -32601
	InvalidParams  = -32602
	InternalError  = -32603
)

// MCP Protocol Types

// ServerInfo represents MCP server information
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ServerCapabilities represents MCP server capabilities
type ServerCapabilities struct {
	Tools   *ToolsCapability   `json:"tools,omitempty"`
	Prompts *PromptsCapability `json:"prompts,omitempty"`
}

// ToolsCapability represents tools capability
type ToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// PromptsCapability represents prompts capability
type PromptsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// InitializeResult is the response to initialize request
type InitializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      ServerInfo         `json:"serverInfo"`
}

// Tool represents an MCP tool definition
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema InputSchema `json:"inputSchema"`
}

// InputSchema represents JSON Schema for tool input
type InputSchema struct {
	Type       string                 `json:"type"`
	Properties map[string]Property    `json:"properties,omitempty"`
	Required   []string               `json:"required,omitempty"`
}

// Property represents a JSON Schema property
type Property struct {
	Type        string   `json:"type"`
	Description string   `json:"description,omitempty"`
	Default     interface{} `json:"default,omitempty"`
	Enum        []string `json:"enum,omitempty"`
	Minimum     *float64 `json:"minimum,omitempty"`
	Maximum     *float64 `json:"maximum,omitempty"`
	Items       *Property `json:"items,omitempty"`
}

// ToolsListResult is the response to tools/list request
type ToolsListResult struct {
	Tools []Tool `json:"tools"`
}

// Prompt represents an MCP prompt definition
type Prompt struct {
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Arguments   []PromptArgument `json:"arguments,omitempty"`
}

// PromptArgument represents a prompt argument
type PromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

// PromptsListResult is the response to prompts/list request
type PromptsListResult struct {
	Prompts []Prompt `json:"prompts"`
}

// CallToolParams are the parameters for tools/call request
type CallToolParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// CallToolResult is the response to tools/call request
type CallToolResult struct {
	Content []ContentBlock `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

// ContentBlock represents a content block in tool response
type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// Tool-specific parameter types

// StoreMemoryParams for store_memory tool
type StoreMemoryParams struct {
	Content    string   `json:"content"`
	Importance int      `json:"importance,omitempty"`
	Tags       []string `json:"tags,omitempty"`
	Domain     string   `json:"domain,omitempty"`
	Source     string   `json:"source,omitempty"`
}

// SearchParams for search tool
type SearchParams struct {
	Query             string   `json:"query,omitempty"`
	SearchType        string   `json:"search_type,omitempty"`
	UseAI             bool     `json:"use_ai,omitempty"`
	Limit             int      `json:"limit,omitempty"`
	Tags              []string `json:"tags,omitempty"`
	Domain            string   `json:"domain,omitempty"`
	StartDate         string   `json:"start_date,omitempty"`
	EndDate           string   `json:"end_date,omitempty"`
	ResponseFormat    string   `json:"response_format,omitempty"`
	SessionFilterMode string   `json:"session_filter_mode,omitempty"`
}

// AnalysisParams for analysis tool
type AnalysisParams struct {
	AnalysisType         string `json:"analysis_type,omitempty"`
	Question             string `json:"question,omitempty"`
	Query                string `json:"query,omitempty"`
	Timeframe            string `json:"timeframe,omitempty"`
	Limit                int    `json:"limit,omitempty"`
	Concept              string `json:"concept,omitempty"`
	TemporalAnalysisType string `json:"temporal_analysis_type,omitempty"`
	ContextLimit         int    `json:"context_limit,omitempty"`
}

// RelationshipsParams for relationships tool
type RelationshipsParams struct {
	RelationshipType     string  `json:"relationship_type,omitempty"`
	MemoryID             string  `json:"memory_id,omitempty"`
	SourceMemoryID       string  `json:"source_memory_id,omitempty"`
	TargetMemoryID       string  `json:"target_memory_id,omitempty"`
	RelationshipTypeEnum string  `json:"relationship_type_enum,omitempty"`
	Strength             float64 `json:"strength,omitempty"`
	Context              string  `json:"context,omitempty"`
	Depth                int     `json:"depth,omitempty"`
	Limit                int     `json:"limit,omitempty"`
	MinStrength          float64 `json:"min_strength,omitempty"`
}

// CategoriesParams for categories tool
type CategoriesParams struct {
	CategoriesType      string  `json:"categories_type,omitempty"`
	Name                string  `json:"name,omitempty"`
	Description         string  `json:"description,omitempty"`
	ParentID            string  `json:"parent_id,omitempty"`
	MemoryID            string  `json:"memory_id,omitempty"`
	AutoCreate          bool    `json:"auto_create,omitempty"`
	ConfidenceThreshold float64 `json:"confidence_threshold,omitempty"`
}

// DomainsParams for domains tool
type DomainsParams struct {
	DomainsType string `json:"domains_type,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Domain      string `json:"domain,omitempty"`
}

// SessionsParams for sessions tool
type SessionsParams struct {
	SessionsType string `json:"sessions_type,omitempty"`
}

// StatsParams for stats tool
type StatsParams struct {
	StatsType  string `json:"stats_type,omitempty"`
	Domain     string `json:"domain,omitempty"`
	CategoryID string `json:"category_id,omitempty"`
}

// UpdateMemoryParams for update_memory tool
type UpdateMemoryParams struct {
	ID         string   `json:"id"`
	Content    string   `json:"content,omitempty"`
	Importance int      `json:"importance,omitempty"`
	Tags       []string `json:"tags,omitempty"`
}

// DeleteMemoryParams for delete_memory tool
type DeleteMemoryParams struct {
	ID string `json:"id"`
}

// GetMemoryParams for get_memory_by_id tool
type GetMemoryParams struct {
	ID string `json:"id"`
}

// PromptMessage represents a message in a prompt
type PromptMessage struct {
	Role    string       `json:"role"`
	Content ContentBlock `json:"content"`
}

// PromptGetResult is the response to prompts/get request
type PromptGetResult struct {
	Description string          `json:"description,omitempty"`
	Messages    []PromptMessage `json:"messages"`
}
