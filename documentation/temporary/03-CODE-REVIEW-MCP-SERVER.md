# Code Review #2: MCP Server Layer

**Module**: `internal/mcp/`
**Files Analyzed**: `server.go`, `handlers.go`, `formatter.go`, `types.go`
**Total LOC**: ~3,400
**Review Date**: 2025-01-27

---

## Executive Summary

The MCP (Model Context Protocol) server is the primary interface for Claude integration. It implements JSON-RPC 2.0 over stdin/stdout and exposes all memory operations as tools. This is a critical module that must maintain compatibility with the MCP specification.

### Overall Assessment

| Aspect | Rating | Notes |
|--------|--------|-------|
| Code Quality | B | Functional but verbose |
| Error Handling | B+ | Consistent error responses |
| Protocol Compliance | A | Follows MCP spec |
| Performance | B- | Some inefficiencies |
| Maintainability | C+ | Very large files |
| Test Coverage | B- | Missing integration tests |
| Documentation | B+ | Good tool descriptions |

---

## File-by-File Analysis

### 1. `server.go` (902 LOC)

**Purpose**: MCP server core - initialization, request routing, tool definitions

#### Server Structure (Lines 23-47)

```go
type Server struct {
    db           *database.Database
    cfg          *config.Config
    aiManager    *ai.Manager
    memSvc       *memory.Service
    searchEng    *search.Engine
    relSvc       *relationships.Service
    benchmarkSvc *benchmark.Service
    formatter    *Formatter
    log          *logging.Logger

    stdin  io.Reader
    stdout io.Writer
    stderr io.Writer

    mu          sync.Mutex
    initialized bool
}
```

**Strengths**:
- Clean dependency injection
- Proper I/O abstraction (testable)
- Thread-safe initialization flag

**Issues**:

1. **Too Many Dependencies**:
```go
// 7 service dependencies make testing difficult
// Suggestion: Use interface groupings
type MemoryOperations interface {
    Store(*memory.StoreOptions) (*memory.StoreResult, error)
    Get(*memory.GetOptions) (*database.Memory, error)
    // ...
}
```

2. **No Context Cancellation Propagation**:
```go
func NewServer(db *database.Database, cfg *config.Config) *Server {
    // No context parameter - can't cancel during initialization
}
```

#### Main Loop (Lines 85-117)

```go
func (s *Server) Run(ctx context.Context) error {
    scanner := bufio.NewScanner(s.stdin)
    scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024)  // 10MB buffer

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
    return scanner.Err()
}
```

**Strengths**:
- Context cancellation support
- Large buffer for big requests
- Empty line handling

**Issues**:

1. **Blocking Read**:
```go
for scanner.Scan() {  // Blocks here
    select {
    case <-ctx.Done():  // Only checked after scan completes
```

Improvement:
```go
func (s *Server) Run(ctx context.Context) error {
    lines := make(chan string)
    errs := make(chan error, 1)

    go func() {
        scanner := bufio.NewScanner(s.stdin)
        for scanner.Scan() {
            lines <- scanner.Text()
        }
        errs <- scanner.Err()
        close(lines)
    }()

    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case line, ok := <-lines:
            if !ok {
                return <-errs
            }
            // Handle line
        }
    }
}
```

2. **No Request Timeout**:
```go
// No timeout on individual request processing
// A slow tool call blocks all subsequent requests
```

3. **No Concurrent Request Handling**:
```go
// Requests processed sequentially
// Could parallelize independent requests
```

#### Tool Definitions (Lines 450-901)

```go
func (s *Server) getToolDefinitions() []Tool {
    min1 := float64(1)
    max10 := float64(10)

    return []Tool{
        {
            Name:        "store_memory",
            Description: "Store a new memory with contextual information",
            InputSchema: InputSchema{
                Type: "object",
                Properties: map[string]Property{
                    "content": {
                        Type:        "string",
                        Description: "The memory content to store",
                    },
                    // ... 15 more tools defined inline
                },
            },
        },
    }
}
```

**Issues**:

1. **Massive Inline Definitions** (450 lines of tool definitions!):
```go
// Current: All tools defined in one giant function
// Better: Separate files or configuration

// tools/store_memory.go
var StoreMemoryTool = Tool{
    Name: "store_memory",
    // ...
}

// tools/registry.go
func GetAllTools() []Tool {
    return []Tool{
        StoreMemoryTool,
        SearchTool,
        // ...
    }
}
```

2. **Repeated Min/Max Variables**:
```go
min1 := float64(1)
max10 := float64(10)
min0 := float64(0)
max1 := float64(1)

// Better: Constants
const (
    MinImportance = 1.0
    MaxImportance = 10.0
    MinStrength   = 0.0
    MaxStrength   = 1.0
)
```

3. **No Tool Versioning**:
```go
// No way to deprecate or version tools
// Suggestion: Add version field
type Tool struct {
    Name        string
    Version     string  // "1.0", "2.0"
    Deprecated  bool
    Replacement string  // Tool name that replaces this
}
```

---

### 2. `handlers.go` (1,534 LOC)

**Purpose**: Tool request handlers - the largest and most complex file

#### Response Types (Lines 1-300)

```go
type StoreMemoryResponse struct {
    Success   bool   `json:"success"`
    MemoryID  string `json:"memory_id"`
    Content   string `json:"content"`
    CreatedAt string `json:"created_at"`
    SessionID string `json:"session_id"`
}

type SearchResponse struct {
    Count          int               `json:"count"`
    Optimization   *OptimizationInfo `json:"optimization,omitempty"`
    Results        []SearchResultLM  `json:"results"`
    SearchMetadata *SearchMetadata   `json:"search_metadata,omitempty"`
    SizeMetadata   *SizeMetadata     `json:"size_metadata,omitempty"`
}
// ... 30+ response types
```

**Issues**:

1. **Duplicate Type Definitions**:
```go
// MemoryFull defined in handlers.go
type MemoryFull struct {
    ID         string   `json:"id"`
    Content    string   `json:"content"`
    // ...
}

// Also exists: database.Memory, MemoryInfo, MemoryFullWithEmbed
// 4 different "Memory" types!
```

2. **Inconsistent Response Formats**:
```go
// Some handlers return wrapped response
return &StoreMemoryResponse{Success: true, ...}

// Others return flat objects
return &DomainFullLM{ID: domain.ID, ...}

// Inconsistent API design
```

#### Store Memory Handler (Lines 300-340)

```go
func (s *Server) handleStoreMemory(ctx context.Context, argsJSON []byte) (interface{}, error) {
    var params StoreMemoryParams
    if err := json.Unmarshal(argsJSON, &params); err != nil {
        return nil, fmt.Errorf("invalid parameters: %w", err)
    }

    if params.Content == "" {
        return nil, fmt.Errorf("content is required")
    }

    importance := params.Importance
    if importance == 0 {
        importance = 5  // Default
    }

    result, err := s.memSvc.Store(&memory.StoreOptions{
        Content:    params.Content,
        Importance: importance,
        Tags:       params.Tags,
        Domain:     params.Domain,
        Source:     params.Source,
    })
    // ...
}
```

**Strengths**:
- Clear parameter extraction
- Default value handling
- Delegation to service layer

**Issues**:

1. **No Input Sanitization**:
```go
// Content passed directly without sanitization
Content: params.Content,

// Potential issues:
// - Very long content (no limit)
// - Special characters
// - Unicode normalization
```

2. **Importance Validation Incomplete**:
```go
if importance == 0 {
    importance = 5
}
// Missing: Check importance > 10
```

#### Search Handler (Lines 342-431)

```go
func (s *Server) handleSearch(ctx context.Context, argsJSON []byte) (interface{}, error) {
    var params SearchParams
    json.Unmarshal(argsJSON, &params)

    limit := params.Limit
    if limit == 0 {
        limit = 10
    }

    searchType := params.SearchType
    if searchType == "" {
        searchType = "semantic"  // Default to semantic
    }

    results, err := s.searchEng.Search(&search.SearchOptions{
        Query:  params.Query,
        Limit:  limit,
        Domain: params.Domain,
        Tags:   params.Tags,
    })

    // Build response
    searchResults := make([]SearchResultLM, len(results))
    for i, r := range results {
        searchResults[i] = SearchResultLM{
            Memory: &MemoryFull{
                ID:      r.Memory.ID,
                Content: r.Memory.Content,
                // ... manual field copying
            },
        }
    }
}
```

**Issues**:

1. **Manual Field Mapping**:
```go
// 15+ fields copied manually
Memory: &MemoryFull{
    ID:         r.Memory.ID,
    Content:    r.Memory.Content,
    Importance: r.Memory.Importance,
    // ...
}

// Better: Use a mapper function
func toMemoryFull(m *database.Memory) *MemoryFull {
    return &MemoryFull{...}
}
```

2. **Token Estimation Hardcoded**:
```go
EstimatedTokens: estimatedChars / 4,  // rough estimate

// Better: Use actual tokenizer or configurable ratio
const CharsPerToken = 4  // GPT-4 average
```

3. **SearchType Not Used in Engine**:
```go
searchType := params.SearchType  // "semantic", "keyword", etc.
// But SearchOptions doesn't have SearchType field!
results, err := s.searchEng.Search(&search.SearchOptions{
    Query: params.Query,
    // Missing: SearchType: searchType
})
```

#### Analysis Handler (Lines 433-534)

```go
func (s *Server) handleAnalysis(ctx context.Context, argsJSON []byte) (interface{}, error) {
    // Check AI availability
    status := s.aiManager.GetStatus()
    if !status.OllamaAvailable {
        return nil, fmt.Errorf("AI analysis requires Ollama to be running")
    }

    opts := &ai.AnalysisOptions{
        Type:      analysisType,
        Question:  params.Question,
        Query:     params.Query,
        Timeframe: timeframe,
        Limit:     limit,
    }

    result, err := s.aiManager.Analyze(ctx, opts)
    // ...
}
```

**Strengths**:
- Pre-checks AI availability
- Delegates to AI manager

**Issues**:

1. **No Fallback for AI Unavailable**:
```go
if !status.OllamaAvailable {
    return nil, fmt.Errorf("AI analysis requires Ollama")
}

// Better: Provide degraded response
if !status.OllamaAvailable {
    return s.handleAnalysisWithoutAI(ctx, params)
}
```

2. **Hardcoded Confidence Values**:
```go
avgRelevance := 0.0
if len(result.SourceMemories) > 0 {
    avgRelevance = 0.61  // approximate - WHY 0.61?
}
```

#### Relationship Handlers (Lines 536-795)

```go
func (s *Server) handleRelationships(ctx context.Context, argsJSON []byte) (interface{}, error) {
    switch relType {
    case "find_related":
        return s.handleFindRelated(params)
    case "create":
        return s.handleCreateRelationship(params)
    case "map_graph":
        return s.handleMapGraph(params)
    case "discover":
        return s.handleDiscoverRelationships(ctx, params)
    default:
        return nil, fmt.Errorf("unknown relationship_type: %s", relType)
    }
}
```

**Strengths**:
- Clean switch dispatch
- Separate handler per operation

**Issues**:

1. **Inconsistent Context Passing**:
```go
case "find_related":
    return s.handleFindRelated(params)  // No ctx
case "discover":
    return s.handleDiscoverRelationships(ctx, params)  // Has ctx
```

2. **handleMapGraph N+1 Query**:
```go
func (s *Server) handleMapGraph(params RelationshipsParams) (interface{}, error) {
    // For each node in graph
    for i, n := range result.Nodes {
        mem, err := s.db.GetMemory(n.ID)  // DB call per node!
        rels, _ := s.db.GetRelationshipsForMemory(n.ID)  // Another DB call!
    }
}
```

#### Benchmark Handlers (Lines 1119-1533)

```go
func (s *Server) handleBenchmarkRun(ctx context.Context, argsJSON []byte) (interface{}, error) {
    // Check if bridge is available
    if err := s.benchmarkSvc.CheckBridge(); err != nil {
        return map[string]interface{}{
            "success": false,
            "error":   "Python benchmark bridge is not running",
        }, nil
    }
    // ...
}
```

**Issues**:

1. **Mixed Return Types**:
```go
// Returns map[string]interface{} instead of typed struct
return map[string]interface{}{
    "success": true,
    "run_id":  results.RunID,
}
```

2. **Long Functions**:
```go
func (s *Server) handleBenchmarkRun(...) (interface{}, error) {
    // 70 lines of code
}

func (s *Server) handleBenchmarkCompare(...) (interface{}, error) {
    // 60 lines of code
}
```

---

### 3. `formatter.go` (739 LOC)

**Purpose**: Format tool responses with rich markdown

```go
type Formatter struct {
    // Configuration could go here
}

func (f *Formatter) FormatToolResponse(toolName string, result interface{}, duration time.Duration) string {
    var sb strings.Builder

    // Header
    sb.WriteString(fmt.Sprintf("## %s\n\n", toolNameToTitle(toolName)))

    // Format based on tool type
    switch toolName {
    case "store_memory":
        f.formatStoreMemoryResponse(&sb, result)
    case "search":
        f.formatSearchResponse(&sb, result)
    // ... 15+ case statements
    }

    // Footer with timing
    sb.WriteString(fmt.Sprintf("\n---\n*Completed in %.2fms*\n", duration.Seconds()*1000))

    return sb.String()
}
```

**Strengths**:
- Consistent output format
- Rich markdown rendering
- Timing information included

**Issues**:

1. **Giant Switch Statement**:
```go
switch toolName {
    case "store_memory": ...
    case "search": ...
    case "analysis": ...
    // 15+ cases
}

// Better: Registry pattern
type ResponseFormatter interface {
    Format(result interface{}) string
}

var formatters = map[string]ResponseFormatter{
    "store_memory": &StoreMemoryFormatter{},
    "search":       &SearchFormatter{},
}
```

2. **Hardcoded Strings**:
```go
sb.WriteString("### Memory Stored\n\n")
sb.WriteString("| Field | Value |\n")
sb.WriteString("|-------|-------|\n")

// Better: Templates
const storeMemoryTemplate = `
### Memory Stored

| Field | Value |
|-------|-------|
| ID | {{.ID}} |
`
```

3. **No Localization Support**:
```go
// All strings in English, hardcoded
sb.WriteString("Memory not found")

// Future consideration: i18n support
```

---

### 4. `types.go` (200 LOC)

**Purpose**: MCP protocol type definitions

```go
// Request represents a JSON-RPC request
type Request struct {
    JSONRPC string          `json:"jsonrpc"`
    ID      interface{}     `json:"id,omitempty"`
    Method  string          `json:"method"`
    Params  json.RawMessage `json:"params,omitempty"`
}

// Response represents a JSON-RPC response
type Response struct {
    JSONRPC string      `json:"jsonrpc"`
    ID      interface{} `json:"id,omitempty"`
    Result  interface{} `json:"result,omitempty"`
    Error   *RPCError   `json:"error,omitempty"`
}
```

**Strengths**:
- Clean JSON-RPC implementation
- Proper omitempty tags
- RawMessage for flexible params

**Issues**:

1. **ID as interface{}**:
```go
ID interface{} `json:"id,omitempty"`

// JSON-RPC allows string, number, or null
// But interface{} allows anything
// Should validate at parse time
```

2. **Missing Request Validation**:
```go
// No validation method
type Request struct { ... }

// Should add:
func (r *Request) Validate() error {
    if r.JSONRPC != "2.0" {
        return ErrInvalidVersion
    }
    if r.Method == "" {
        return ErrMissingMethod
    }
    return nil
}
```

---

## Critical Issues Summary

### High Priority

| Issue | Location | Impact | Recommendation |
|-------|----------|--------|----------------|
| handlers.go too large | handlers.go (1534 LOC) | Maintainability | Split by tool category |
| N+1 queries in map_graph | handlers.go:644-717 | Performance | Batch fetch |
| 4 duplicate Memory types | handlers.go, types.go | Confusion | Consolidate types |
| Missing request timeout | server.go:85 | Reliability | Add per-request timeout |

### Medium Priority

| Issue | Location | Impact | Recommendation |
|-------|----------|--------|----------------|
| Tool definitions inline | server.go:450-901 | Maintainability | Extract to separate files |
| Manual field mapping | handlers.go | Error-prone | Use mapper functions |
| Hardcoded strings | formatter.go | Maintainability | Use templates |
| Inconsistent responses | handlers.go | API design | Standardize format |

### Low Priority

| Issue | Location | Impact | Recommendation |
|-------|----------|--------|----------------|
| No localization | formatter.go | International users | Add i18n support |
| ID type validation | types.go | Spec compliance | Validate ID type |
| Hardcoded constants | handlers.go | Flexibility | Extract to config |

---

## Recommendations

### Immediate Refactoring

1. **Split handlers.go by Domain**:
```
internal/mcp/
├── server.go
├── types.go
├── formatter.go
├── handlers/
│   ├── memory.go       # store, get, update, delete
│   ├── search.go       # search, intelligent_search
│   ├── relationships.go # relate, discover, map_graph
│   ├── organization.go  # categories, domains
│   ├── analysis.go      # analysis operations
│   └── benchmark.go     # benchmark tools
└── tools/
    ├── definitions.go   # Tool definitions
    └── registry.go      # Tool registration
```

2. **Add Request Timeout**:
```go
func (s *Server) handleRequest(ctx context.Context, line string) *Response {
    ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()
    // ...
}
```

3. **Consolidate Memory Types**:
```go
// Single canonical type in database package
type Memory struct { ... }

// MCP-specific view
type MemoryView struct {
    *database.Memory
    // Additional fields for API
}
```

4. **Use Templates for Formatting**:
```go
import "text/template"

var searchTemplate = template.Must(template.New("search").Parse(`
## Search Results

Found {{.Count}} memories matching "{{.Query}}"

{{range .Results}}
### {{.Memory.ID}}
{{.Memory.Content}}

Relevance: {{printf "%.2f" .RelevanceScore}}
{{end}}
`))
```

### Architecture Improvements

1. **Add Handler Interface**:
```go
type Handler interface {
    Handle(ctx context.Context, params json.RawMessage) (interface{}, error)
    GetToolDefinition() Tool
}

// Registration
handlers := map[string]Handler{
    "store_memory": &StoreMemoryHandler{memSvc: memSvc},
    "search":       &SearchHandler{searchEng: searchEng},
}
```

2. **Add Middleware Support**:
```go
type Middleware func(Handler) Handler

// Logging middleware
func LoggingMiddleware(h Handler) Handler {
    return HandlerFunc(func(ctx context.Context, params json.RawMessage) (interface{}, error) {
        start := time.Now()
        result, err := h.Handle(ctx, params)
        log.Info("tool call", "duration", time.Since(start))
        return result, err
    })
}
```

---

## Conclusion

The MCP server is functional and MCP-compliant but suffers from code organization issues. The main handlers.go file at 1,534 lines is difficult to maintain and navigate. The recommended refactoring would:

1. Improve maintainability through modular organization
2. Enable easier testing of individual handlers
3. Allow parallel development of different tool categories
4. Make the codebase more accessible to new contributors

**Overall Grade: B-**

---

*Review completed by Claude Code Analysis*
*Next: Code Review #3 - Memory Service Layer*
