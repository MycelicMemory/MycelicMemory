# Code Review #4: Search & Memory Services

**Modules**: `internal/memory/` and `internal/search/`
**Files Analyzed**: `service.go`, `chunker.go`, `session.go`, `engine.go`
**Total LOC**: ~1,300
**Review Date**: 2025-01-27

---

## Executive Summary

The Memory and Search services form the core business logic layer. Memory service handles CRUD operations with validation and enrichment, while Search engine provides unified access to multiple search modes. Both are well-structured with clear responsibilities.

### Overall Assessment

| Aspect | Rating | Notes |
|--------|--------|-------|
| Code Quality | B+ | Clean, focused |
| Error Handling | B+ | Consistent patterns |
| Performance | B | Some inefficiencies |
| Maintainability | A- | Well-organized |
| Test Coverage | B+ | Good coverage |
| Documentation | A- | Clear inline docs |

---

## Memory Service Analysis

### 1. `service.go` (415 LOC)

**Purpose**: Business logic for memory CRUD operations

#### Service Structure (Lines 1-30)

```go
type Service struct {
    db     *database.Database
    config *config.Config
    aiMgr  *ai.Manager
}

func NewService(db *database.Database, cfg *config.Config) *Service {
    return &Service{
        db:     db,
        config: cfg,
    }
}

func (s *Service) SetAIManager(aiMgr *ai.Manager) {
    s.aiMgr = aiMgr
}
```

**Strengths**:
- Simple dependency injection
- Optional AI manager
- Clear initialization

**Issues**:

1. **Optional AI Manager is Mutable**:
```go
func (s *Service) SetAIManager(aiMgr *ai.Manager) {
    s.aiMgr = aiMgr  // Not thread-safe
}

// Better: Set at construction or use sync.Once
func NewService(db *database.Database, cfg *config.Config, opts ...ServiceOption) *Service {
    s := &Service{db: db, config: cfg}
    for _, opt := range opts {
        opt(s)
    }
    return s
}

func WithAIManager(aiMgr *ai.Manager) ServiceOption {
    return func(s *Service) {
        s.aiMgr = aiMgr
    }
}
```

#### StoreOptions and Store Method (Lines 32-120)

```go
type StoreOptions struct {
    Content    string
    Importance int
    Tags       []string
    Domain     string
    Source     string
    SessionID  string  // Optional, auto-detected if empty
}

func (s *Service) Store(opts *StoreOptions) (*StoreResult, error) {
    // Validate content
    if strings.TrimSpace(opts.Content) == "" {
        return nil, fmt.Errorf("content is required")
    }

    // Validate importance
    importance := opts.Importance
    if importance < 1 {
        importance = 5  // Default
    }
    if importance > 10 {
        importance = 10  // Cap
    }

    // Normalize tags
    normalizedTags := make([]string, len(opts.Tags))
    for i, tag := range opts.Tags {
        normalizedTags[i] = strings.ToLower(strings.TrimSpace(tag))
    }

    // Detect session
    sessionID := opts.SessionID
    if sessionID == "" {
        sessionID = s.detectSession()
    }

    // Create memory object
    memory := &database.Memory{
        Content:    opts.Content,
        Importance: importance,
        Tags:       normalizedTags,
        Domain:     opts.Domain,
        Source:     opts.Source,
        SessionID:  sessionID,
    }

    // Store in database
    if err := s.db.CreateMemory(memory); err != nil {
        return nil, fmt.Errorf("failed to store memory: %w", err)
    }

    // Index for AI search if available
    if s.aiMgr != nil {
        ctx := context.Background()
        s.aiMgr.IndexMemory(ctx, memory)  // Fire and forget
    }

    return &StoreResult{Memory: memory}, nil
}
```

**Strengths**:
- Clear validation logic
- Tag normalization
- Session auto-detection
- Optional AI indexing

**Issues**:

1. **No Content Length Validation**:
```go
if strings.TrimSpace(opts.Content) == "" {
    return nil, fmt.Errorf("content is required")
}

// Missing: Length check
const MaxContentLength = 1_000_000  // 1MB
if len(opts.Content) > MaxContentLength {
    return nil, fmt.Errorf("content exceeds maximum length of %d", MaxContentLength)
}
```

2. **Fire-and-Forget AI Indexing**:
```go
s.aiMgr.IndexMemory(ctx, memory)  // Error ignored!

// Better: Log errors at minimum
if err := s.aiMgr.IndexMemory(ctx, memory); err != nil {
    log.Warn("failed to index memory", "id", memory.ID, "error", err)
}
```

3. **No Duplicate Detection**:
```go
// Same content can be stored multiple times
// Consider: Content hash for deduplication
func (s *Service) Store(opts *StoreOptions) (*StoreResult, error) {
    contentHash := sha256.Sum256([]byte(opts.Content))

    existing, err := s.db.GetMemoryByHash(hex.EncodeToString(contentHash[:]))
    if existing != nil {
        return nil, fmt.Errorf("duplicate content detected (id=%s)", existing.ID)
    }
}
```

4. **Background Context for AI**:
```go
ctx := context.Background()
s.aiMgr.IndexMemory(ctx, memory)

// Issue: No timeout, no cancellation
// Better: Derive from request context or add timeout
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
```

#### Get Method (Lines 122-160)

```go
type GetOptions struct {
    ID string
}

func (s *Service) Get(opts *GetOptions) (*database.Memory, error) {
    if opts.ID == "" {
        return nil, fmt.Errorf("id is required")
    }

    memory, err := s.db.GetMemory(opts.ID)
    if err != nil {
        return nil, fmt.Errorf("failed to get memory: %w", err)
    }

    return memory, nil
}
```

**Strengths**:
- Simple and focused
- Proper error wrapping

**Issues**:

1. **No UUID Validation**:
```go
if opts.ID == "" {
    return nil, fmt.Errorf("id is required")
}

// Add: Validate UUID format
if _, err := uuid.Parse(opts.ID); err != nil {
    return nil, fmt.Errorf("invalid memory ID format: %w", err)
}
```

#### Update Method (Lines 162-220)

```go
type UpdateOptions struct {
    ID         string
    Content    *string  // Pointer for optional update
    Importance *int
    Tags       []string
}

func (s *Service) Update(opts *UpdateOptions) (*database.Memory, error) {
    // Get existing
    memory, err := s.db.GetMemory(opts.ID)
    if err != nil {
        return nil, err
    }
    if memory == nil {
        return nil, fmt.Errorf("memory not found: %s", opts.ID)
    }

    // Apply updates
    if opts.Content != nil {
        memory.Content = *opts.Content
    }
    if opts.Importance != nil {
        memory.Importance = *opts.Importance
    }
    if opts.Tags != nil {
        memory.Tags = opts.Tags
    }

    memory.UpdatedAt = time.Now()

    // Persist
    if err := s.db.UpdateMemory(memory); err != nil {
        return nil, fmt.Errorf("failed to update memory: %w", err)
    }

    return memory, nil
}
```

**Strengths**:
- Pointer types for optional fields
- Preserves unchanged fields
- Updates timestamp

**Issues**:

1. **No Change Detection**:
```go
// Always updates even if nothing changed
// Waste of I/O

func (s *Service) Update(opts *UpdateOptions) (*database.Memory, error) {
    hasChanges := false
    if opts.Content != nil && *opts.Content != memory.Content {
        memory.Content = *opts.Content
        hasChanges = true
    }
    // ... same for other fields

    if !hasChanges {
        return memory, nil  // No-op
    }
}
```

2. **Missing Re-indexing**:
```go
// If content changes, should re-index for AI search
if opts.Content != nil && s.aiMgr != nil {
    s.aiMgr.IndexMemory(ctx, memory)  // Re-generate embedding
}
```

---

### 2. `chunker.go` (231 LOC)

**Purpose**: Split large content into hierarchical chunks

#### Chunker Configuration

```go
const (
    MaxChunkSize = 1000   // characters (~200-250 tokens)
    Overlap      = 100    // characters for context
    MinChunkSize = 1500   // Only chunk if content larger
)
```

**Analysis**:
- Constants are reasonable for GPT-style tokenizers
- Overlap preserves context at boundaries
- Threshold prevents unnecessary chunking

#### Chunk Method (Lines 40-120)

```go
func (c *Chunker) Chunk(content string) ([]*Chunk, error) {
    // Skip if content is small
    if len(content) < MinChunkSize {
        return []*Chunk{{
            Content: content,
            Level:   0,
            Index:   0,
        }}, nil
    }

    var chunks []*Chunk

    // Split into paragraphs first
    paragraphs := splitParagraphs(content)

    for i, para := range paragraphs {
        if len(para) <= MaxChunkSize {
            chunks = append(chunks, &Chunk{
                Content: para,
                Level:   1,  // Paragraph level
                Index:   i,
            })
        } else {
            // Further split large paragraphs
            subChunks := splitSentences(para, MaxChunkSize, Overlap)
            for j, sub := range subChunks {
                chunks = append(chunks, &Chunk{
                    Content: sub,
                    Level:   2,  // Sentence level
                    Index:   i*1000 + j,  // Composite index
                })
            }
        }
    }

    return chunks, nil
}
```

**Strengths**:
- Hierarchical approach
- Preserves paragraph boundaries
- Progressive splitting

**Issues**:

1. **Character-Based, Not Token-Based**:
```go
const MaxChunkSize = 1000  // characters

// Issue: Token count varies by content type
// Code has high chars/token, prose has lower
// Better: Use tiktoken or similar for accurate counting
```

2. **No Unicode Awareness**:
```go
if len(content) < MinChunkSize {

// Issue: len() counts bytes, not characters
// "Hello" = 5, but "Hello" in Japanese might be 15 bytes

// Better:
if utf8.RuneCountInString(content) < MinChunkSize {
```

3. **Hardcoded Split Logic**:
```go
paragraphs := splitParagraphs(content)

// Issue: Assumes \n\n paragraph separator
// Code, markdown, and other formats differ

// Better: Configurable splitter
type SplitStrategy interface {
    SplitParagraphs(content string) []string
    SplitSentences(content string) []string
}
```

4. **Composite Index Math**:
```go
Index: i*1000 + j,  // Composite index

// Issue: Breaks if >1000 sub-chunks per paragraph
// Better: Use separate parent/child indexing
```

#### splitSentences Function (Lines 150-200)

```go
func splitSentences(content string, maxSize, overlap int) []string {
    var chunks []string
    var current strings.Builder
    words := strings.Fields(content)

    for _, word := range words {
        if current.Len()+len(word)+1 > maxSize && current.Len() > 0 {
            chunks = append(chunks, current.String())

            // Overlap: Keep last N characters
            overlapText := getLastNChars(current.String(), overlap)
            current.Reset()
            current.WriteString(overlapText)
        }
        if current.Len() > 0 {
            current.WriteString(" ")
        }
        current.WriteString(word)
    }

    if current.Len() > 0 {
        chunks = append(chunks, current.String())
    }

    return chunks
}
```

**Issues**:

1. **Ignores Sentence Boundaries**:
```go
// Despite name, splits on word count, not sentences
// "Dr. Smith said hello." might split after "Dr."

// Better: Use sentence detection
import "github.com/neurosnap/sentences"

func splitSentences(content string) []string {
    tokenizer, _ := english.NewSentenceTokenizer(nil)
    return tokenizer.Tokenize(content)
}
```

---

### 3. `session.go` (182 LOC)

**Purpose**: Automatic session detection

#### Session Strategies

```go
type SessionStrategy string

const (
    SessionStrategyGitDirectory SessionStrategy = "git-directory"
    SessionStrategyManual       SessionStrategy = "manual"
    SessionStrategyHash         SessionStrategy = "hash"
)
```

#### DetectSession Method

```go
func (d *SessionDetector) DetectSession() string {
    switch d.strategy {
    case SessionStrategyGitDirectory:
        return d.detectGitSession()
    case SessionStrategyManual:
        return d.config.Session.ManualID
    case SessionStrategyHash:
        return d.computeHash()
    default:
        return d.detectGitSession()  // Default
    }
}

func (d *SessionDetector) detectGitSession() string {
    // Walk up directories looking for .git
    dir, _ := os.Getwd()
    for {
        gitPath := filepath.Join(dir, ".git")
        if _, err := os.Stat(gitPath); err == nil {
            // Hash the git directory path
            hash := sha256.Sum256([]byte(gitPath))
            return hex.EncodeToString(hash[:8])  // First 8 bytes
        }

        parent := filepath.Dir(dir)
        if parent == dir {
            break  // Reached root
        }
        dir = parent
    }

    // No .git found, use current directory
    hash := sha256.Sum256([]byte(dir))
    return hex.EncodeToString(hash[:8])
}
```

**Strengths**:
- Multiple strategy support
- Git-based grouping
- Deterministic hashes

**Issues**:

1. **No Caching**:
```go
func (d *SessionDetector) detectGitSession() string {
    // Walks filesystem every call!
}

// Better: Cache result
func (d *SessionDetector) detectGitSession() string {
    d.mu.Lock()
    defer d.mu.Unlock()

    if d.cachedSession != "" {
        return d.cachedSession
    }

    session := d.doDetect()
    d.cachedSession = session
    return session
}
```

2. **8-Byte Hash Collision Risk**:
```go
return hex.EncodeToString(hash[:8])

// 8 bytes = 64 bits = potential collisions at ~2^32 directories
// For a single user, unlikely, but consider full hash
```

3. **No Validation of ManualID**:
```go
case SessionStrategyManual:
    return d.config.Session.ManualID

// No validation - could be empty or invalid
if d.config.Session.ManualID == "" {
    return d.detectGitSession()  // Fallback
}
```

---

## Search Engine Analysis

### 4. `engine.go` (450 LOC)

**Purpose**: Unified search across multiple modes

#### Engine Structure (Lines 34-50)

```go
type Engine struct {
    db        *database.Database
    config    *config.Config
    aiManager *ai.Manager
}

func NewEngine(db *database.Database, cfg *config.Config) *Engine {
    return &Engine{
        db:     db,
        config: cfg,
    }
}

func NewEngineWithAI(db *database.Database, cfg *config.Config, aiMgr *ai.Manager) *Engine {
    return &Engine{
        db:        db,
        config:    cfg,
        aiManager: aiMgr,
    }
}
```

**Issues**:

1. **Two Constructors**:
```go
// NewEngine and NewEngineWithAI have overlapping purposes
// Better: Single constructor with options
func NewEngine(db *database.Database, cfg *config.Config, opts ...EngineOption) *Engine {
    e := &Engine{db: db, config: cfg}
    for _, opt := range opts {
        opt(e)
    }
    return e
}

func WithAI(aiMgr *ai.Manager) EngineOption {
    return func(e *Engine) { e.aiManager = aiMgr }
}
```

#### Search Method (Lines 94-120)

```go
func (e *Engine) Search(opts *SearchOptions) ([]*SearchResult, error) {
    if err := e.validateOptions(opts); err != nil {
        return nil, err
    }

    switch opts.SearchType {
    case SearchTypeSemantic:
        return e.semanticSearch(opts)
    case SearchTypeKeyword:
        return e.keywordSearch(opts)
    case SearchTypeTags:
        return e.tagSearch(opts)
    case SearchTypeDateRange:
        return e.dateRangeSearch(opts)
    case SearchTypeHybrid:
        return e.hybridSearch(opts)
    default:
        if opts.Query != "" {
            return e.keywordSearch(opts)
        }
        return e.listSearch(opts)
    }
}
```

**Strengths**:
- Clean dispatch by type
- Sensible defaults
- Validation before search

#### semanticSearch Method (Lines 188-244)

```go
func (e *Engine) semanticSearch(opts *SearchOptions) ([]*SearchResult, error) {
    // Check if AI manager is available
    if e.aiManager == nil {
        return e.keywordSearch(opts)  // Fallback
    }

    status := e.aiManager.GetStatus()
    if !status.OllamaEnabled || !status.QdrantEnabled {
        return e.keywordSearch(opts)
    }
    if !status.OllamaAvailable || !status.QdrantAvailable {
        return e.keywordSearch(opts)
    }

    // Perform semantic search
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    semanticResults, err := e.aiManager.SemanticSearch(ctx, &ai.SemanticSearchOptions{
        Query:     opts.Query,
        Limit:     limit,
        MinScore:  opts.MinRelevance,
        SessionID: opts.SessionID,
        Domain:    opts.Domain,
    })
    if err != nil {
        return e.keywordSearch(opts)  // Fallback on error
    }

    // Fetch full memory data
    var results []*SearchResult
    for _, sr := range semanticResults {
        mem, err := e.db.GetMemory(sr.MemoryID)
        if err != nil || mem == nil {
            continue
        }
        results = append(results, &SearchResult{
            Memory:    mem,
            Relevance: sr.Score,
            MatchType: "semantic",
        })
    }

    return results, nil
}
```

**Strengths**:
- Multiple fallback paths
- Timeout for AI operations
- Full memory fetch

**Issues**:

1. **N+1 Query Pattern**:
```go
for _, sr := range semanticResults {
    mem, err := e.db.GetMemory(sr.MemoryID)  // DB call per result!
}

// Better: Batch fetch
ids := make([]string, len(semanticResults))
for i, sr := range semanticResults {
    ids[i] = sr.MemoryID
}
memories, _ := e.db.GetMemoriesByIDs(ids)  // Single query
```

2. **Silent Skip on Error**:
```go
if err != nil || mem == nil {
    continue  // No logging
}

// Better: Log for debugging
if err != nil {
    log.Debug("failed to fetch memory", "id", sr.MemoryID, "error", err)
    continue
}
```

#### keywordSearch Method (Lines 153-185)

```go
func (e *Engine) keywordSearch(opts *SearchOptions) ([]*SearchResult, error) {
    filters := &database.SearchFilters{
        Query:     opts.Query,
        SessionID: opts.SessionID,
        Domain:    opts.Domain,
        Tags:      opts.Tags,
        UseAI:     false,
        Limit:     opts.Limit,
    }

    results, err := e.db.SearchFTS(opts.Query, filters)
    if err != nil {
        return nil, fmt.Errorf("keyword search failed: %w", err)
    }

    var output []*SearchResult
    for _, r := range results {
        if r.Relevance >= opts.MinRelevance {
            output = append(output, &SearchResult{
                Memory:    r.Memory,
                Relevance: r.Relevance,
                MatchType: "keyword",
            })
        }
    }

    return output, nil
}
```

**Issues**:

1. **Filtering After Query**:
```go
for _, r := range results {
    if r.Relevance >= opts.MinRelevance {  // Post-filter

// Better: Pass to database layer
filters := &database.SearchFilters{
    MinRelevance: opts.MinRelevance,
    // ...
}
```

#### hybridSearch Method (Lines 324-344)

```go
func (e *Engine) hybridSearch(opts *SearchOptions) ([]*SearchResult, error) {
    // Perform both searches
    keywordResults, err := e.keywordSearch(opts)
    if err != nil {
        return nil, err
    }

    // If AI is available, merge with semantic
    if e.aiManager != nil {
        status := e.aiManager.GetStatus()
        if status.OllamaAvailable && status.QdrantAvailable {
            semanticResults, err := e.semanticSearch(opts)
            if err == nil {
                keywordResults = mergeResults(keywordResults, semanticResults)
            }
        }
    }

    return keywordResults, nil
}
```

**Issues**:

1. **Sequential Search**:
```go
keywordResults, err := e.keywordSearch(opts)
semanticResults, err := e.semanticSearch(opts)

// Better: Parallel execution
var keywordResults, semanticResults []*SearchResult
var wg sync.WaitGroup

wg.Add(2)
go func() {
    defer wg.Done()
    keywordResults, _ = e.keywordSearch(opts)
}()
go func() {
    defer wg.Done()
    semanticResults, _ = e.semanticSearch(opts)
}()
wg.Wait()
```

2. **No Weighted Merge**:
```go
keywordResults = mergeResults(keywordResults, semanticResults)

// Current: Just deduplicates and keeps higher relevance
// Better: Weighted combination
func mergeResults(keyword, semantic []*SearchResult, keywordWeight, semanticWeight float64) []*SearchResult {
    // Combine scores: final = kw_weight * kw_score + sem_weight * sem_score
}
```

#### mergeResults Function (Lines 403-430)

```go
func mergeResults(a, b []*SearchResult) []*SearchResult {
    seen := make(map[string]*SearchResult)

    for _, r := range a {
        seen[r.Memory.ID] = r
    }

    for _, r := range b {
        if existing, ok := seen[r.Memory.ID]; ok {
            if r.Relevance > existing.Relevance {
                seen[r.Memory.ID] = r
            }
        } else {
            seen[r.Memory.ID] = r
        }
    }

    var results []*SearchResult
    for _, r := range seen {
        results = append(results, r)
    }

    return results
}
```

**Issues**:

1. **No Sorting**:
```go
// Results are in map iteration order (random)
for _, r := range seen {
    results = append(results, r)
}

// Better: Sort by relevance
sort.Slice(results, func(i, j int) bool {
    return results[i].Relevance > results[j].Relevance
})
```

2. **Loses Match Type Info**:
```go
if r.Relevance > existing.Relevance {
    seen[r.Memory.ID] = r
}

// If semantic replaces keyword, we lose that it matched both
// Could be useful info: "matched by both keyword AND semantic"
```

---

## Critical Issues Summary

### High Priority

| Issue | Location | Impact | Recommendation |
|-------|----------|--------|----------------|
| N+1 queries in semanticSearch | engine.go:228 | Performance | Batch fetch memories |
| Sequential hybrid search | engine.go:324 | Performance | Parallel execution |
| Character-based chunking | chunker.go | Accuracy | Token-based splitting |
| No merge sorting | engine.go:427 | UX | Sort by relevance |

### Medium Priority

| Issue | Location | Impact | Recommendation |
|-------|----------|--------|----------------|
| Fire-and-forget indexing | service.go:115 | Reliability | Log errors |
| Session detection no cache | session.go | Performance | Cache result |
| No content length limit | service.go:50 | Security | Add max length |
| Post-query filtering | engine.go:175 | Performance | Push to DB layer |

### Low Priority

| Issue | Location | Impact | Recommendation |
|-------|----------|--------|----------------|
| Two constructors | engine.go | Consistency | Use options pattern |
| Hardcoded split logic | chunker.go | Flexibility | Configurable splitter |
| 8-byte hash | session.go | Collision risk | Consider full hash |

---

## Recommendations

### Batch Memory Fetching

```go
// Add to database package
func (d *Database) GetMemoriesByIDs(ids []string) (map[string]*Memory, error) {
    placeholders := make([]string, len(ids))
    args := make([]interface{}, len(ids))
    for i, id := range ids {
        placeholders[i] = "?"
        args[i] = id
    }

    query := fmt.Sprintf(`
        SELECT * FROM memories
        WHERE id IN (%s)
    `, strings.Join(placeholders, ","))

    rows, err := d.db.Query(query, args...)
    // ... scan into map
}

// Use in search
func (e *Engine) semanticSearch(opts *SearchOptions) ([]*SearchResult, error) {
    // ...
    ids := extractIDs(semanticResults)
    memories, err := e.db.GetMemoriesByIDs(ids)

    for _, sr := range semanticResults {
        if mem, ok := memories[sr.MemoryID]; ok {
            results = append(results, &SearchResult{
                Memory:    mem,
                Relevance: sr.Score,
            })
        }
    }
}
```

### Parallel Hybrid Search

```go
func (e *Engine) hybridSearch(opts *SearchOptions) ([]*SearchResult, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    var keywordResults, semanticResults []*SearchResult
    var keywordErr, semanticErr error

    var wg sync.WaitGroup
    wg.Add(2)

    go func() {
        defer wg.Done()
        keywordResults, keywordErr = e.keywordSearch(opts)
    }()

    go func() {
        defer wg.Done()
        if e.hasAI() {
            semanticResults, semanticErr = e.semanticSearch(opts)
        }
    }()

    // Wait with context timeout
    done := make(chan struct{})
    go func() {
        wg.Wait()
        close(done)
    }()

    select {
    case <-done:
    case <-ctx.Done():
        return nil, ctx.Err()
    }

    // Merge available results
    if keywordErr == nil && semanticErr == nil {
        return mergeResults(keywordResults, semanticResults), nil
    }
    if keywordErr == nil {
        return keywordResults, nil
    }
    return nil, keywordErr
}
```

### Token-Based Chunking

```go
import "github.com/pkoukk/tiktoken-go"

type TokenChunker struct {
    encoding  *tiktoken.Tiktoken
    maxTokens int
    overlap   int
}

func NewTokenChunker(maxTokens, overlap int) *TokenChunker {
    enc, _ := tiktoken.GetEncoding("cl100k_base")  // GPT-4 encoding
    return &TokenChunker{
        encoding:  enc,
        maxTokens: maxTokens,
        overlap:   overlap,
    }
}

func (c *TokenChunker) Chunk(content string) ([]*Chunk, error) {
    tokens := c.encoding.Encode(content, nil, nil)

    if len(tokens) <= c.maxTokens {
        return []*Chunk{{Content: content, Level: 0}}, nil
    }

    var chunks []*Chunk
    for i := 0; i < len(tokens); i += c.maxTokens - c.overlap {
        end := min(i+c.maxTokens, len(tokens))
        chunkTokens := tokens[i:end]
        chunkText := c.encoding.Decode(chunkTokens)

        chunks = append(chunks, &Chunk{
            Content: chunkText,
            Level:   1,
            Index:   len(chunks),
        })

        if end >= len(tokens) {
            break
        }
    }

    return chunks, nil
}
```

---

## Conclusion

The Memory and Search services are well-structured with clear responsibilities. The main improvements needed are:

1. **Performance**: Batch fetching and parallel search
2. **Accuracy**: Token-based chunking instead of character-based
3. **Reliability**: Better error handling and logging

The graceful degradation for AI features is well-implemented and should be preserved.

**Overall Grade: B+**

---

*Review completed by Claude Code Analysis*
*Next: Code Review #5 - CLI & REST API Layers*
