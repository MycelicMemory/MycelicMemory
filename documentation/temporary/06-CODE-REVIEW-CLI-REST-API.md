# Code Review #5: CLI & REST API Layers

**Modules**: `cmd/mycelicmemory/` and `internal/api/`
**Files Analyzed**: `root.go`, `cmd_*.go`, `server.go`, `handlers_*.go`
**Total LOC**: ~3,500
**Review Date**: 2025-01-27

---

## Executive Summary

The CLI and REST API layers provide the user-facing interfaces to MycelicMemory. CLI uses Cobra for command handling, while REST API uses Gin framework. Both are functional but have opportunities for better code organization and error handling.

### Overall Assessment

| Aspect | Rating | Notes |
|--------|--------|-------|
| Code Quality | B | Functional but verbose |
| Error Handling | B- | Inconsistent patterns |
| Performance | B+ | Generally efficient |
| Maintainability | B- | Many command files |
| Test Coverage | C+ | Limited API tests |
| Documentation | B | Good command help |

---

## CLI Analysis

### 1. `root.go` (116 LOC)

**Purpose**: Root command, global flags, MCP mode

```go
var (
    cfgFile  string
    logLevel string
    mcpMode  bool
    quiet    bool
)

var rootCmd = &cobra.Command{
    Use:   "mycelicmemory",
    Short: "AI-powered persistent memory system",
    Long: `MyclicMemory provides persistent memory for AI agents.
Store, search, and analyze memories with semantic search and relationship mapping.`,
    Version: "1.2.0",
    PersistentPreRun: func(cmd *cobra.Command, args []string) {
        // Initialize logging
        // Load config
    },
    Run: func(cmd *cobra.Command, args []string) {
        if mcpMode {
            runMCPServer()
        } else {
            cmd.Help()
        }
    },
}
```

**Strengths**:
- Clear command structure
- MCP mode integration
- Version tracking

**Issues**:

1. **Global Variables**:
```go
var (
    cfgFile  string
    logLevel string
    mcpMode  bool
    quiet    bool
)

// Issue: Global state makes testing difficult
// Better: Pass through context or command struct
```

2. **Version Hardcoded**:
```go
Version: "1.2.0",

// Should use build-time variable
// go build -ldflags "-X main.Version=1.2.0"
Version: version,  // Injected at build time
```

3. **No Graceful Shutdown**:
```go
func runMCPServer() {
    server := mcp.NewServer(db, cfg)
    server.Run(context.Background())  // No signal handling
}

// Better:
func runMCPServer() {
    ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer cancel()

    server := mcp.NewServer(db, cfg)
    if err := server.Run(ctx); err != nil && err != context.Canceled {
        log.Fatal("server error", "error", err)
    }
}
```

---

### 2. `cmd_memory.go` (400 LOC)

**Purpose**: Memory CRUD commands (remember, search, get, list, update, forget)

#### Remember Command

```go
var rememberCmd = &cobra.Command{
    Use:   "remember [content]",
    Short: "Store a new memory",
    Long: `Store a new memory with optional metadata.

Examples:
  mycelicmemory remember "Go interfaces are implicit"
  mycelicmemory remember "Important decision" --importance 9 --tags decision,project`,
    Args: cobra.MinimumNArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        content := strings.Join(args, " ")

        importance, _ := cmd.Flags().GetInt("importance")
        tagsStr, _ := cmd.Flags().GetString("tags")
        domain, _ := cmd.Flags().GetString("domain")

        var tags []string
        if tagsStr != "" {
            tags = strings.Split(tagsStr, ",")
        }

        result, err := memSvc.Store(&memory.StoreOptions{
            Content:    content,
            Importance: importance,
            Tags:       tags,
            Domain:     domain,
        })
        if err != nil {
            return fmt.Errorf("failed to store memory: %w", err)
        }

        fmt.Printf("Memory stored: %s\n", result.Memory.ID)
        return nil
    },
}
```

**Strengths**:
- Good examples in help text
- Flag handling
- Error wrapping

**Issues**:

1. **Error Return vs Fatal**:
```go
if err != nil {
    return fmt.Errorf("failed to store memory: %w", err)
}

// Inconsistent - some commands use log.Fatal, others return error
// Standardize on returning errors
```

2. **Tags as Comma-Separated String**:
```go
tagsStr, _ := cmd.Flags().GetString("tags")
tags = strings.Split(tagsStr, ",")

// Better: Use StringSlice flag
cmd.Flags().StringSlice("tags", []string{}, "Tags for categorization")
// Usage: --tags foo --tags bar --tags baz
```

3. **No Input Validation**:
```go
content := strings.Join(args, " ")

// No validation of content length, encoding, etc.
if len(content) > MaxContentLength {
    return fmt.Errorf("content too long (max %d)", MaxContentLength)
}
```

#### Search Command

```go
var searchCmd = &cobra.Command{
    Use:   "search [query]",
    Short: "Search memories",
    Args:  cobra.MinimumNArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        query := strings.Join(args, " ")
        limit, _ := cmd.Flags().GetInt("limit")
        searchType, _ := cmd.Flags().GetString("type")

        results, err := searchEng.Search(&search.SearchOptions{
            Query:      query,
            SearchType: search.SearchType(searchType),
            Limit:      limit,
        })
        if err != nil {
            return err
        }

        // Display results
        for _, r := range results {
            fmt.Printf("[%s] (%.2f) %s\n", r.Memory.ID[:8], r.Relevance, truncate(r.Memory.Content, 80))
        }

        return nil
    },
}
```

**Issues**:

1. **No Structured Output Option**:
```go
fmt.Printf("[%s] (%.2f) %s\n", ...)

// Better: Support JSON output
output, _ := cmd.Flags().GetString("output")
switch output {
case "json":
    json.NewEncoder(os.Stdout).Encode(results)
case "table":
    printTable(results)
default:
    printSimple(results)
}
```

2. **Hardcoded Truncation**:
```go
truncate(r.Memory.Content, 80)

// Should be configurable
width, _ := cmd.Flags().GetInt("width")
```

---

### 3. `cmd_service.go` (200 LOC)

**Purpose**: Daemon management (start, stop, status)

```go
var startCmd = &cobra.Command{
    Use:   "start",
    Short: "Start the REST API server",
    RunE: func(cmd *cobra.Command, args []string) error {
        port, _ := cmd.Flags().GetInt("port")
        background, _ := cmd.Flags().GetBool("background")

        if background {
            return startBackground(port)
        }

        server := api.NewServer(db, cfg)
        return server.Start(port)
    },
}
```

**Issues**:

1. **Background Mode Implementation**:
```go
func startBackground(port int) error {
    // Uses exec.Command to start new process
    // Complex, OS-dependent

    // Better: Use proper daemonization library
    // Or: Recommend systemd/launchd for production
}
```

2. **No PID File**:
```go
// No tracking of running instance
// Can start multiple servers accidentally

// Add PID file
pidFile := filepath.Join(dataDir, "server.pid")
if _, err := os.Stat(pidFile); err == nil {
    return fmt.Errorf("server already running (PID file exists)")
}
```

---

### 4. `cmd_analyze.go` (150 LOC)

**Purpose**: AI analysis commands

```go
var analyzeCmd = &cobra.Command{
    Use:   "analyze [question]",
    Short: "Ask questions about your memories",
    RunE: func(cmd *cobra.Command, args []string) error {
        question := strings.Join(args, " ")
        limit, _ := cmd.Flags().GetInt("limit")
        timeframe, _ := cmd.Flags().GetString("timeframe")

        result, err := aiManager.Analyze(context.Background(), &ai.AnalysisOptions{
            Type:      "question",
            Question:  question,
            Timeframe: timeframe,
            Limit:     limit,
        })
        if err != nil {
            return err
        }

        fmt.Printf("Answer: %s\n", result.Answer)
        fmt.Printf("Confidence: %.2f\n", result.Confidence)
        fmt.Printf("Sources: %d memories\n", len(result.SourceMemories))

        return nil
    },
}
```

**Issues**:

1. **No AI Availability Check**:
```go
result, err := aiManager.Analyze(...)

// Should check first
status := aiManager.GetStatus()
if !status.OllamaAvailable {
    return fmt.Errorf("analysis requires Ollama (not running)")
}
```

2. **No Progress Indicator**:
```go
// Long-running AI operations show no progress
// Add spinner or progress indicator

fmt.Print("Analyzing...")
result, err := aiManager.Analyze(...)
fmt.Print("\r")  // Clear line
```

---

### 5. `cmd_doctor.go` (100 LOC)

**Purpose**: Health check diagnostics

```go
var doctorCmd = &cobra.Command{
    Use:   "doctor",
    Short: "Check system health",
    Run: func(cmd *cobra.Command, args []string) {
        fmt.Println("MycelicMemory Health Check")
        fmt.Println("==========================")

        // Check database
        checkDatabase()

        // Check Ollama
        checkOllama()

        // Check Qdrant
        checkQdrant()
    },
}

func checkDatabase() {
    fmt.Print("Database... ")
    if db != nil {
        stats, _ := db.GetStats()
        fmt.Printf("OK (%d memories)\n", stats.MemoryCount)
    } else {
        fmt.Println("NOT CONNECTED")
    }
}
```

**Strengths**:
- Clear output format
- Checks all services

**Issues**:

1. **No Exit Code**:
```go
// Always exits 0 even if checks fail
// Should exit non-zero for CI/CD

var hasErrors bool
if !checkDatabase() {
    hasErrors = true
}
// ...
if hasErrors {
    os.Exit(1)
}
```

2. **Hardcoded Output**:
```go
// No JSON/structured output for automation
// Add --json flag
```

---

## REST API Analysis

### 1. `server.go` (200 LOC)

**Purpose**: Gin server setup and middleware

```go
type Server struct {
    db        *database.Database
    config    *config.Config
    aiManager *ai.Manager
    memSvc    *memory.Service
    searchEng *search.Engine
    relSvc    *relationships.Service
    engine    *gin.Engine
    log       *logging.Logger
}

func NewServer(db *database.Database, cfg *config.Config) *Server {
    s := &Server{
        db:     db,
        config: cfg,
        engine: gin.New(),
    }

    s.setupMiddleware()
    s.setupRoutes()

    return s
}

func (s *Server) setupMiddleware() {
    s.engine.Use(gin.Recovery())
    s.engine.Use(cors.Default())
    s.engine.Use(s.loggingMiddleware())
}

func (s *Server) setupRoutes() {
    v1 := s.engine.Group("/api/v1")
    {
        // Memory routes
        v1.POST("/memories", s.createMemory)
        v1.GET("/memories", s.listMemories)
        v1.GET("/memories/:id", s.getMemory)
        v1.PUT("/memories/:id", s.updateMemory)
        v1.DELETE("/memories/:id", s.deleteMemory)
        v1.POST("/memories/search", s.searchMemories)

        // More routes...
    }
}
```

**Strengths**:
- Clean route organization
- API versioning (v1)
- Middleware setup

**Issues**:

1. **No Rate Limiting**:
```go
// No protection against abuse
// Add rate limiter
import "github.com/ulule/limiter/v3"

s.engine.Use(limiter.New(limiter.Config{
    Max:      100,
    Duration: time.Minute,
}))
```

2. **CORS Too Permissive**:
```go
s.engine.Use(cors.Default())

// cors.Default() allows all origins
// Better: Configure explicitly
s.engine.Use(cors.New(cors.Config{
    AllowOrigins:     []string{"http://localhost:3000"},
    AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
    AllowHeaders:     []string{"Origin", "Content-Type"},
    AllowCredentials: true,
}))
```

3. **No Health Check Endpoint**:
```go
// Missing /health for load balancers
v1.GET("/health", func(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{
        "status":   "healthy",
        "version":  version,
        "database": db.HealthCheck() == nil,
    })
})
```

---

### 2. `handlers_memory.go` (300 LOC)

**Purpose**: Memory CRUD handlers

```go
func (s *Server) createMemory(c *gin.Context) {
    var req CreateMemoryRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    result, err := s.memSvc.Store(&memory.StoreOptions{
        Content:    req.Content,
        Importance: req.Importance,
        Tags:       req.Tags,
        Domain:     req.Domain,
    })
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusCreated, MemoryResponse{
        ID:        result.Memory.ID,
        Content:   result.Memory.Content,
        CreatedAt: result.Memory.CreatedAt,
    })
}
```

**Strengths**:
- Proper status codes (201 for create)
- JSON binding
- Error responses

**Issues**:

1. **Error Messages Leak Implementation**:
```go
c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

// Leaks internal errors to client
// Better: Map to user-friendly messages
if errors.Is(err, database.ErrNotFound) {
    c.JSON(http.StatusNotFound, gin.H{"error": "Memory not found"})
    return
}
c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
s.log.Error("create memory failed", "error", err)
```

2. **No Request Validation**:
```go
type CreateMemoryRequest struct {
    Content    string   `json:"content"`
    Importance int      `json:"importance"`
    Tags       []string `json:"tags"`
    Domain     string   `json:"domain"`
}

// Missing validation tags
type CreateMemoryRequest struct {
    Content    string   `json:"content" binding:"required,min=1,max=1000000"`
    Importance int      `json:"importance" binding:"omitempty,min=1,max=10"`
    Tags       []string `json:"tags" binding:"omitempty,max=50,dive,min=1,max=100"`
    Domain     string   `json:"domain" binding:"omitempty,max=100"`
}
```

3. **Inconsistent Response Format**:
```go
// Some handlers return gin.H{}, others use typed structs
c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
c.JSON(http.StatusCreated, MemoryResponse{...})

// Standardize: Always use typed responses
type ErrorResponse struct {
    Error   string `json:"error"`
    Code    string `json:"code,omitempty"`
    Details any    `json:"details,omitempty"`
}
```

---

### 3. `handlers_search.go` (200 LOC)

**Purpose**: Search endpoint handlers

```go
func (s *Server) searchMemories(c *gin.Context) {
    var req SearchRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    results, err := s.searchEng.Search(&search.SearchOptions{
        Query:      req.Query,
        SearchType: search.SearchType(req.SearchType),
        Limit:      req.Limit,
        Tags:       req.Tags,
        Domain:     req.Domain,
    })
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, SearchResponse{
        Count:   len(results),
        Results: toSearchResults(results),
    })
}
```

**Issues**:

1. **No Pagination**:
```go
// Returns all results up to limit
// No cursor/offset pagination

type SearchRequest struct {
    Query  string `json:"query"`
    Limit  int    `json:"limit"`
    Offset int    `json:"offset"`  // Add offset
    Cursor string `json:"cursor"`  // Or cursor for large datasets
}

type SearchResponse struct {
    Count      int      `json:"count"`
    Results    []Result `json:"results"`
    NextCursor string   `json:"next_cursor,omitempty"`
    HasMore    bool     `json:"has_more"`
}
```

2. **No Request Timeout**:
```go
results, err := s.searchEng.Search(...)

// No timeout - could hang
ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
defer cancel()
results, err := s.searchEng.SearchWithContext(ctx, ...)
```

---

### 4. `handlers_analysis.go` (200 LOC)

**Purpose**: AI analysis endpoints

```go
func (s *Server) analyzeMemories(c *gin.Context) {
    var req AnalysisRequest
    c.ShouldBindJSON(&req)

    result, err := s.aiManager.Analyze(c.Request.Context(), &ai.AnalysisOptions{
        Type:      req.AnalysisType,
        Question:  req.Question,
        Query:     req.Query,
        Timeframe: req.Timeframe,
        Limit:     req.Limit,
    })
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, AnalysisResponse{
        Type:        result.Type,
        Answer:      result.Answer,
        Confidence:  result.Confidence,
        MemoryCount: result.MemoryCount,
    })
}
```

**Issues**:

1. **No AI Availability Check**:
```go
// Should check before trying
if s.aiManager == nil {
    c.JSON(http.StatusServiceUnavailable, gin.H{
        "error": "AI analysis not available",
    })
    return
}

status := s.aiManager.GetStatus()
if !status.OllamaAvailable {
    c.JSON(http.StatusServiceUnavailable, gin.H{
        "error": "Ollama is not running",
    })
    return
}
```

2. **Binding Error Ignored**:
```go
c.ShouldBindJSON(&req)  // Error not checked!

// Fix:
if err := c.ShouldBindJSON(&req); err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
    return
}
```

---

### 5. `response.go` (100 LOC)

**Purpose**: Response formatting utilities

```go
type MemoryResponse struct {
    ID         string    `json:"id"`
    Content    string    `json:"content"`
    Importance int       `json:"importance"`
    Tags       []string  `json:"tags"`
    Domain     string    `json:"domain,omitempty"`
    SessionID  string    `json:"session_id"`
    CreatedAt  time.Time `json:"created_at"`
    UpdatedAt  time.Time `json:"updated_at"`
}

func toMemoryResponse(m *database.Memory) MemoryResponse {
    return MemoryResponse{
        ID:         m.ID,
        Content:    m.Content,
        Importance: m.Importance,
        Tags:       m.Tags,
        Domain:     m.Domain,
        SessionID:  m.SessionID,
        CreatedAt:  m.CreatedAt,
        UpdatedAt:  m.UpdatedAt,
    }
}
```

**Strengths**:
- Clean separation
- Consistent formatting

**Issues**:

1. **Manual Field Mapping**:
```go
// Copying field by field is error-prone
// Consider: github.com/jinzhu/copier

import "github.com/jinzhu/copier"

func toMemoryResponse(m *database.Memory) MemoryResponse {
    var resp MemoryResponse
    copier.Copy(&resp, m)
    return resp
}
```

---

## Critical Issues Summary

### High Priority

| Issue | Location | Impact | Recommendation |
|-------|----------|--------|----------------|
| No rate limiting | server.go | Security | Add rate limiter |
| Binding error ignored | handlers_analysis.go | Bugs | Check all binding errors |
| Error message leakage | handlers_memory.go | Security | Map to safe messages |
| No pagination | handlers_search.go | Scalability | Add offset/cursor |

### Medium Priority

| Issue | Location | Impact | Recommendation |
|-------|----------|--------|----------------|
| Global variables | root.go | Testability | Use context/struct |
| No health endpoint | server.go | Operations | Add /health |
| CORS too permissive | server.go | Security | Configure explicitly |
| No structured output | cmd_*.go | Automation | Add --json flag |

### Low Priority

| Issue | Location | Impact | Recommendation |
|-------|----------|--------|----------------|
| Version hardcoded | root.go | Maintenance | Build-time injection |
| Manual field mapping | response.go | Maintenance | Use copier library |
| No progress indicator | cmd_analyze.go | UX | Add spinner |

---

## Recommendations

### Add Rate Limiting

```go
import (
    "github.com/ulule/limiter/v3"
    mgin "github.com/ulule/limiter/v3/drivers/middleware/gin"
    "github.com/ulule/limiter/v3/drivers/store/memory"
)

func (s *Server) setupMiddleware() {
    // Rate limiter: 100 requests per minute per IP
    rate := limiter.Rate{
        Period: time.Minute,
        Limit:  100,
    }
    store := memory.NewStore()
    instance := limiter.New(store, rate)
    middleware := mgin.NewMiddleware(instance)

    s.engine.Use(middleware)
    s.engine.Use(gin.Recovery())
    s.engine.Use(cors.New(s.corsConfig()))
}
```

### Add Pagination

```go
type PaginatedResponse[T any] struct {
    Data       []T    `json:"data"`
    Total      int    `json:"total"`
    Page       int    `json:"page"`
    PerPage    int    `json:"per_page"`
    TotalPages int    `json:"total_pages"`
    HasNext    bool   `json:"has_next"`
    HasPrev    bool   `json:"has_prev"`
}

func (s *Server) listMemories(c *gin.Context) {
    page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
    perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))

    if perPage > 100 {
        perPage = 100  // Max per page
    }

    offset := (page - 1) * perPage

    memories, total, err := s.memSvc.ListPaginated(perPage, offset)
    if err != nil {
        s.respondError(c, err)
        return
    }

    totalPages := (total + perPage - 1) / perPage

    c.JSON(http.StatusOK, PaginatedResponse[MemoryResponse]{
        Data:       toMemoryResponses(memories),
        Total:      total,
        Page:       page,
        PerPage:    perPage,
        TotalPages: totalPages,
        HasNext:    page < totalPages,
        HasPrev:    page > 1,
    })
}
```

### Standardize Error Responses

```go
type ErrorResponse struct {
    Error   string            `json:"error"`
    Code    string            `json:"code,omitempty"`
    Details map[string]string `json:"details,omitempty"`
}

var errorCodes = map[error]struct {
    Status int
    Code   string
    Msg    string
}{
    database.ErrNotFound:       {404, "NOT_FOUND", "Resource not found"},
    database.ErrDuplicate:      {409, "DUPLICATE", "Resource already exists"},
    memory.ErrInvalidContent:   {400, "INVALID_CONTENT", "Content is invalid"},
    ai.ErrServiceUnavailable:   {503, "AI_UNAVAILABLE", "AI service not available"},
}

func (s *Server) respondError(c *gin.Context, err error) {
    // Log the actual error
    s.log.Error("request failed", "error", err, "path", c.Request.URL.Path)

    // Find user-safe error
    for knownErr, info := range errorCodes {
        if errors.Is(err, knownErr) {
            c.JSON(info.Status, ErrorResponse{
                Error: info.Msg,
                Code:  info.Code,
            })
            return
        }
    }

    // Default: generic error
    c.JSON(http.StatusInternalServerError, ErrorResponse{
        Error: "Internal server error",
        Code:  "INTERNAL_ERROR",
    })
}
```

---

## Conclusion

The CLI and REST API layers are functional but need improvements in:

1. **Security**: Rate limiting, CORS configuration, error sanitization
2. **Scalability**: Pagination for list endpoints
3. **Operations**: Health endpoints, graceful shutdown
4. **Developer Experience**: Structured output options, validation

The Cobra and Gin frameworks are well-chosen, but the implementations need more attention to production-readiness.

**Overall Grade: B-**

---

*Review completed by Claude Code Analysis*
*All 5 code reviews complete*
