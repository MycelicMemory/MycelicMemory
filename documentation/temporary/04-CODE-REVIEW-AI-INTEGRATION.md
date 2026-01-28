# Code Review #3: AI Integration Layer

**Module**: `internal/ai/` and `internal/vector/`
**Files Analyzed**: `manager.go`, `ollama.go`, `qdrant.go`
**Total LOC**: ~1,700
**Review Date**: 2025-01-27

---

## Executive Summary

The AI integration layer orchestrates external services (Ollama for embeddings/chat, Qdrant for vector storage) and provides graceful degradation when services are unavailable. This is well-designed for optional AI enhancement of core functionality.

### Overall Assessment

| Aspect | Rating | Notes |
|--------|--------|-------|
| Code Quality | B+ | Clean abstractions |
| Error Handling | A- | Good fallbacks |
| Performance | B | Some optimization opportunities |
| Maintainability | B+ | Clear separation |
| Test Coverage | B | Mocked external services |
| Documentation | B+ | Good inline docs |

---

## File-by-File Analysis

### 1. `manager.go` (509 LOC)

**Purpose**: Central orchestrator for all AI operations

#### Manager Structure (Lines 15-27)

```go
type Manager struct {
    ollama      *OllamaClient
    qdrant      *vector.QdrantClient
    db          *database.Database
    config      *config.Config
    mu          sync.RWMutex
    initialized bool
}

func NewManager(db *database.Database, cfg *config.Config) *Manager {
    return &Manager{
        ollama: NewOllamaClient(&cfg.Ollama),
        qdrant: vector.NewQdrantClient(&cfg.Qdrant),
        db:     db,
        config: cfg,
    }
}
```

**Strengths**:
- Clean dependency injection
- Thread-safe with RWMutex
- Lazy initialization supported

**Issues**:

1. **No Interface for Testing**:
```go
// Current: Concrete types
ollama *OllamaClient

// Better: Interface for mocking
type EmbeddingProvider interface {
    GenerateEmbedding(ctx context.Context, text string) ([]float64, error)
    IsAvailable() bool
}

type ChatProvider interface {
    Chat(ctx context.Context, prompt string) (string, error)
}
```

2. **Config Coupling**:
```go
ollama: NewOllamaClient(&cfg.Ollama)

// Issue: Manager knows about config structure
// Better: Accept already-configured clients
func NewManager(ollama EmbeddingProvider, qdrant VectorStore, db *database.Database) *Manager
```

#### Initialize Method (Lines 40-71)

```go
func (m *Manager) Initialize(ctx context.Context) error {
    log.Info("initializing AI services")

    m.mu.Lock()
    defer m.mu.Unlock()

    if m.initialized {
        return nil
    }

    // Initialize Qdrant collection if enabled
    if m.qdrant.IsEnabled() && m.qdrant.IsAvailable() {
        if err := m.qdrant.InitCollection(ctx); err != nil {
            log.Warn("failed to initialize Qdrant collection", "error", err)
            // Continue - not fatal
        }
    }

    m.initialized = true
    return nil
}
```

**Strengths**:
- Idempotent initialization
- Non-fatal errors for optional services
- Context support

**Issues**:

1. **No Initialization Timeout**:
```go
// Could hang if Qdrant is slow to respond
if err := m.qdrant.InitCollection(ctx); err != nil {

// Better:
ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
defer cancel()
```

2. **Swallowed Error on Init Failure**:
```go
if err := m.qdrant.InitCollection(ctx); err != nil {
    log.Warn("failed to initialize Qdrant collection", "error", err)
    // Error lost - should track
}
```

#### GetStatus Method (Lines 74-105)

```go
func (m *Manager) GetStatus() *Status {
    status := &Status{
        OllamaEnabled:   m.ollama.IsEnabled(),
        OllamaAvailable: m.ollama.IsAvailable(),
        QdrantEnabled:   m.qdrant.IsEnabled(),
        QdrantAvailable: m.qdrant.IsAvailable(),
    }

    if status.OllamaAvailable {
        status.OllamaModel = m.ollama.ChatModel()
    }

    if status.QdrantAvailable {
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
        if info, err := m.qdrant.GetCollectionInfo(ctx); err == nil {
            status.VectorCount = info.VectorCount
        }
    }

    return status
}
```

**Strengths**:
- Timeout for vector count query
- Complete status information

**Issues**:

1. **No Caching**:
```go
// GetStatus called frequently, but makes network calls each time
if info, err := m.qdrant.GetCollectionInfo(ctx); err == nil {

// Better: Cache status with TTL
type Manager struct {
    statusCache *Status
    statusCacheTime time.Time
}

func (m *Manager) GetStatus() *Status {
    if time.Since(m.statusCacheTime) < 10*time.Second {
        return m.statusCache
    }
    // ... refresh status
}
```

#### SemanticSearch Method (Lines 127-204)

```go
func (m *Manager) SemanticSearch(ctx context.Context, opts *SemanticSearchOptions) ([]SemanticSearchResult, error) {
    if !m.ollama.IsEnabled() || !m.qdrant.IsEnabled() {
        return nil, fmt.Errorf("semantic search requires Ollama and Qdrant")
    }

    // Generate query embedding
    embedding, err := m.ollama.GenerateEmbedding(ctx, opts.Query)
    if err != nil {
        return nil, fmt.Errorf("failed to generate query embedding: %w", err)
    }

    // Build filter if needed
    var filter map[string]interface{}
    if opts.SessionID != "" || opts.Domain != "" {
        must := []map[string]interface{}{}
        if opts.SessionID != "" {
            must = append(must, map[string]interface{}{
                "key":   "session_id",
                "match": map[string]interface{}{"value": opts.SessionID},
            })
        }
        // ...
    }

    // Search Qdrant
    searchResults, err := m.qdrant.Search(ctx, &vector.SearchOptions{
        Vector:      embedding,
        Limit:       limit,
        MinScore:    opts.MinScore,
        Filter:      filter,
        WithPayload: true,
    })
}
```

**Strengths**:
- Pre-check availability
- Proper error wrapping
- Filter construction

**Issues**:

1. **No Query Embedding Cache**:
```go
// Same query generates embedding every time
embedding, err := m.ollama.GenerateEmbedding(ctx, opts.Query)

// Better: LRU cache for recent queries
type Manager struct {
    embeddingCache *lru.Cache // query -> []float64
}

func (m *Manager) getOrCreateEmbedding(ctx context.Context, text string) ([]float64, error) {
    if cached, ok := m.embeddingCache.Get(text); ok {
        return cached.([]float64), nil
    }
    embedding, err := m.ollama.GenerateEmbedding(ctx, text)
    if err == nil {
        m.embeddingCache.Add(text, embedding)
    }
    return embedding, err
}
```

2. **Filter Building is Verbose**:
```go
// 20+ lines to build a simple filter
must := []map[string]interface{}{}
if opts.SessionID != "" {
    must = append(must, map[string]interface{}{...})
}

// Better: Builder pattern
filter := NewFilterBuilder().
    Where("session_id", opts.SessionID).
    Where("domain", opts.Domain).
    Build()
```

#### IndexMemory Method (Lines 206-239)

```go
func (m *Manager) IndexMemory(ctx context.Context, memory *database.Memory) error {
    if !m.ollama.IsEnabled() || !m.qdrant.IsEnabled() {
        return nil  // Silently skip
    }

    // Generate embedding
    embedding, err := m.ollama.GenerateEmbedding(ctx, memory.Content)
    if err != nil {
        return fmt.Errorf("failed to generate embedding: %w", err)
    }

    // Store in Qdrant
    payload := map[string]interface{}{
        "content":    memory.Content,
        "session_id": memory.SessionID,
        "domain":     memory.Domain,
        "importance": memory.Importance,
        "created_at": memory.CreatedAt.Format(time.RFC3339),
    }

    if err := m.qdrant.Upsert(ctx, memory.ID, embedding, payload); err != nil {
        return fmt.Errorf("failed to store vector: %w", err)
    }

    // Store embedding in database
    memory.Embedding = float64SliceToBytes(embedding)

    return nil
}
```

**Strengths**:
- Silent skip when disabled (graceful degradation)
- Stores embedding in both Qdrant and SQLite

**Issues**:

1. **No Batch Indexing**:
```go
// For bulk imports, this is inefficient
func (m *Manager) IndexMemory(ctx context.Context, memory *database.Memory) error

// Better: Batch support
func (m *Manager) IndexMemories(ctx context.Context, memories []*database.Memory) error {
    embeddings := make([][]float64, len(memories))

    // Batch embedding generation
    for i, mem := range memories {
        embeddings[i], _ = m.ollama.GenerateEmbedding(ctx, mem.Content)
    }

    // Batch upsert
    return m.qdrant.BatchUpsert(ctx, memories, embeddings)
}
```

2. **Embedding Stored But Not Updated in DB**:
```go
memory.Embedding = float64SliceToBytes(embedding)
// But: No d.db.UpdateMemory() call!
// The embedding is set on the struct but not persisted
```

#### Analyze Method (Lines 278-330)

```go
func (m *Manager) Analyze(ctx context.Context, opts *AnalysisOptions) (*AnalysisResponse, error) {
    // Get relevant memories
    memories, err := m.getMemoriesForAnalysis(ctx, opts)
    if err != nil {
        return nil, fmt.Errorf("failed to get memories: %w", err)
    }

    if len(memories) == 0 {
        return &AnalysisResponse{
            Type:        opts.Type,
            Answer:      "No memories found matching the criteria.",
            MemoryCount: 0,
        }, nil
    }

    // Extract content
    contents := make([]string, len(memories))
    for i, mem := range memories {
        contents[i] = mem.Content
    }

    switch opts.Type {
    case "question":
        return m.answerQuestion(ctx, opts.Question, contents)
    case "summarize":
        return m.summarize(ctx, contents, opts.Timeframe)
    case "analyze":
        return m.analyzePatterns(ctx, contents, opts.Query)
    case "temporal_patterns":
        return m.analyzeTemporalPatterns(ctx, memories, opts.Query)
    }
}
```

**Strengths**:
- Clean dispatch by analysis type
- Handles empty results gracefully

**Issues**:

1. **Content Extraction Loses Metadata**:
```go
for i, mem := range memories {
    contents[i] = mem.Content  // Only content, no tags/importance
}

// Better: Include relevant metadata
for i, mem := range memories {
    contents[i] = fmt.Sprintf("[%s, importance=%d] %s",
        strings.Join(mem.Tags, ","), mem.Importance, mem.Content)
}
```

2. **No Analysis Caching**:
```go
// Same analysis query runs full LLM call each time
// Consider caching for expensive operations
```

#### DiscoverRelationships Method (Lines 439-484)

```go
func (m *Manager) DiscoverRelationships(ctx context.Context, limit int) ([]RelationshipSuggestion, error) {
    // Get recent memories
    memories, err := m.db.ListMemories(&database.MemoryFilters{Limit: limit * 2})

    var suggestions []RelationshipSuggestion

    // Compare pairs
    maxPairs := limit
    pairCount := 0
    for i := 0; i < len(memories) && pairCount < maxPairs; i++ {
        for j := i + 1; j < len(memories) && pairCount < maxPairs; j++ {
            suggestion, err := m.ollama.SuggestRelationships(
                ctx,
                memories[i].Content,
                memories[j].Content,
                memories[i].ID,
                memories[j].ID,
            )
            if suggestion != nil {
                suggestions = append(suggestions, *suggestion)
            }
            pairCount++
        }
    }
}
```

**Issues**:

1. **O(n^2) Comparisons**:
```go
// Comparing all pairs is expensive
for i := 0; i < len(memories); i++ {
    for j := i + 1; j < len(memories); j++ {

// Better: Use embedding similarity to pre-filter
// Only compare pairs with cosine similarity > 0.5
```

2. **Sequential API Calls**:
```go
for ... {
    suggestion, err := m.ollama.SuggestRelationships(...)
}

// Better: Parallel with concurrency limit
sem := make(chan struct{}, 5)  // 5 concurrent calls
var wg sync.WaitGroup
for ... {
    wg.Add(1)
    go func(i, j int) {
        sem <- struct{}{}
        defer func() { <-sem; wg.Done() }()
        // API call
    }(i, j)
}
wg.Wait()
```

---

### 2. `ollama.go` (400 LOC)

**Purpose**: Ollama API client for embeddings and chat

#### OllamaClient Structure

```go
type OllamaClient struct {
    config         *config.OllamaConfig
    httpClient     *http.Client
    available      bool
    availableCheck time.Time
    mu             sync.RWMutex
}
```

**Strengths**:
- Caches availability check
- Thread-safe

#### GenerateEmbedding Method

```go
func (c *OllamaClient) GenerateEmbedding(ctx context.Context, text string) ([]float64, error) {
    req := EmbeddingRequest{
        Model:  c.config.EmbeddingModel,
        Prompt: text,
    }

    body, _ := json.Marshal(req)
    httpReq, _ := http.NewRequestWithContext(ctx, "POST",
        c.config.BaseURL+"/api/embeddings", bytes.NewReader(body))
    httpReq.Header.Set("Content-Type", "application/json")

    resp, err := c.httpClient.Do(httpReq)
    if err != nil {
        return nil, fmt.Errorf("embedding request failed: %w", err)
    }
    defer resp.Body.Close()

    var result EmbeddingResponse
    json.NewDecoder(resp.Body).Decode(&result)

    return result.Embedding, nil
}
```

**Issues**:

1. **No Response Status Check**:
```go
resp, err := c.httpClient.Do(httpReq)
// Missing:
if resp.StatusCode != http.StatusOK {
    body, _ := io.ReadAll(resp.Body)
    return nil, fmt.Errorf("embedding failed: %s", body)
}
```

2. **No Request Timeout**:
```go
// Relies on context, but no default timeout
// If caller doesn't set timeout, could hang

func (c *OllamaClient) GenerateEmbedding(ctx context.Context, text string) ([]float64, error) {
    ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()
    // ...
}
```

3. **No Retry Logic**:
```go
// Single attempt, no retry on transient failures
resp, err := c.httpClient.Do(httpReq)

// Better: Exponential backoff
for attempt := 0; attempt < 3; attempt++ {
    resp, err = c.httpClient.Do(httpReq)
    if err == nil && resp.StatusCode == http.StatusOK {
        break
    }
    time.Sleep(time.Duration(1<<attempt) * time.Second)
}
```

#### Chat Method

```go
func (c *OllamaClient) Chat(ctx context.Context, messages []ChatMessage) (string, error) {
    req := ChatRequest{
        Model:    c.config.ChatModel,
        Messages: messages,
        Stream:   false,
    }

    body, _ := json.Marshal(req)
    httpReq, _ := http.NewRequestWithContext(ctx, "POST",
        c.config.BaseURL+"/api/chat", bytes.NewReader(body))

    resp, err := c.httpClient.Do(httpReq)
    // ...

    var result ChatResponse
    json.NewDecoder(resp.Body).Decode(&result)

    return result.Message.Content, nil
}
```

**Issues**:

1. **No Streaming Support**:
```go
Stream: false,

// For long responses, streaming improves UX
// Add streaming option for chat operations
```

2. **No Token Counting**:
```go
// No way to know how many tokens used
// Important for cost tracking and context limits
```

---

### 3. `qdrant.go` (339 LOC in `internal/vector/`)

**Purpose**: Qdrant vector database client

#### QdrantClient Structure

```go
type QdrantClient struct {
    config    *config.QdrantConfig
    client    *http.Client
    available bool
    mu        sync.RWMutex
}

const (
    CollectionName    = "mycelicmemory-memories"
    EmbeddingDimension = 768  // nomic-embed-text
)
```

**Strengths**:
- Constants for configuration
- HTTP client abstraction

#### Search Method

```go
func (c *QdrantClient) Search(ctx context.Context, opts *SearchOptions) ([]*SearchResult, error) {
    reqBody := map[string]interface{}{
        "vector":       opts.Vector,
        "limit":        opts.Limit,
        "with_payload": opts.WithPayload,
        "score_threshold": opts.MinScore,
    }

    if opts.Filter != nil {
        reqBody["filter"] = opts.Filter
    }

    body, _ := json.Marshal(reqBody)
    url := fmt.Sprintf("%s/collections/%s/points/search", c.config.URL, CollectionName)
    req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))

    resp, err := c.client.Do(req)
    // ...
}
```

**Issues**:

1. **Manual JSON Building**:
```go
reqBody := map[string]interface{}{...}

// Better: Use typed structs
type SearchRequest struct {
    Vector       []float64              `json:"vector"`
    Limit        int                    `json:"limit"`
    WithPayload  bool                   `json:"with_payload"`
    ScoreThreshold float64              `json:"score_threshold,omitempty"`
    Filter       *Filter                `json:"filter,omitempty"`
}
```

2. **No Connection Pooling Configuration**:
```go
// Using default http.Client
// Should configure for high-throughput scenarios
client := &http.Client{
    Transport: &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 100,
        IdleConnTimeout:     90 * time.Second,
    },
}
```

---

## Critical Issues Summary

### High Priority

| Issue | Location | Impact | Recommendation |
|-------|----------|--------|----------------|
| No embedding cache | manager.go:127 | Performance | Add LRU cache |
| O(n^2) relationship discovery | manager.go:439 | Scalability | Use similarity pre-filter |
| No retry logic | ollama.go | Reliability | Add exponential backoff |
| Embedding not persisted | manager.go:237 | Data loss | Call UpdateMemory |

### Medium Priority

| Issue | Location | Impact | Recommendation |
|-------|----------|--------|----------------|
| No status caching | manager.go:74 | Performance | Cache with TTL |
| Sequential API calls | manager.go:460 | Performance | Parallelize |
| No batch indexing | manager.go:206 | Performance | Add batch method |
| Missing interfaces | manager.go | Testability | Define interfaces |

### Low Priority

| Issue | Location | Impact | Recommendation |
|-------|----------|--------|----------------|
| No streaming | ollama.go | UX | Add streaming option |
| No token counting | ollama.go | Cost tracking | Track usage |
| Verbose filter building | manager.go:150 | Readability | Use builder pattern |

---

## Recommendations

### Add Embedding Cache

```go
type Manager struct {
    embeddingCache *lru.Cache
}

func NewManager(...) *Manager {
    cache, _ := lru.New(1000)  // 1000 recent queries
    return &Manager{
        embeddingCache: cache,
    }
}

func (m *Manager) getEmbedding(ctx context.Context, text string) ([]float64, error) {
    // Normalize text for cache key
    key := strings.TrimSpace(strings.ToLower(text))

    if cached, ok := m.embeddingCache.Get(key); ok {
        return cached.([]float64), nil
    }

    embedding, err := m.ollama.GenerateEmbedding(ctx, text)
    if err == nil {
        m.embeddingCache.Add(key, embedding)
    }
    return embedding, err
}
```

### Add Retry Logic

```go
func withRetry[T any](ctx context.Context, maxAttempts int, fn func() (T, error)) (T, error) {
    var result T
    var err error

    for attempt := 0; attempt < maxAttempts; attempt++ {
        result, err = fn()
        if err == nil {
            return result, nil
        }

        // Exponential backoff
        select {
        case <-ctx.Done():
            return result, ctx.Err()
        case <-time.After(time.Duration(1<<attempt) * time.Second):
        }
    }

    return result, fmt.Errorf("after %d attempts: %w", maxAttempts, err)
}

// Usage
embedding, err := withRetry(ctx, 3, func() ([]float64, error) {
    return c.ollama.GenerateEmbedding(ctx, text)
})
```

### Define Interfaces for Testing

```go
// interfaces.go
type EmbeddingGenerator interface {
    GenerateEmbedding(ctx context.Context, text string) ([]float64, error)
    IsAvailable() bool
}

type VectorStore interface {
    Search(ctx context.Context, opts *SearchOptions) ([]*SearchResult, error)
    Upsert(ctx context.Context, id string, vector []float64, payload map[string]interface{}) error
    Delete(ctx context.Context, ids []string) error
}

type ChatProvider interface {
    Chat(ctx context.Context, messages []ChatMessage) (string, error)
}

// Manager uses interfaces
type Manager struct {
    embedder    EmbeddingGenerator
    vectorStore VectorStore
    chat        ChatProvider
    // ...
}
```

---

## Conclusion

The AI integration layer is well-designed with good separation of concerns and graceful degradation. The main improvements needed are:

1. **Performance**: Embedding caching and batch operations
2. **Reliability**: Retry logic for transient failures
3. **Testability**: Interface definitions

The graceful degradation pattern is excellent and should be preserved - the system works without AI services but gains enhanced capabilities when they're available.

**Overall Grade: B+**

---

*Review completed by Claude Code Analysis*
*Next: Code Review #4 - Search & Memory Services*
