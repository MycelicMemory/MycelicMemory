package search

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/MycelicMemory/mycelicmemory/internal/ai"
	"github.com/MycelicMemory/mycelicmemory/internal/database"
	"github.com/MycelicMemory/mycelicmemory/pkg/config"
)

// SearchType defines the type of search to perform
type SearchType string

const (
	// SearchTypeSemantic uses vector embeddings for similarity search
	SearchTypeSemantic SearchType = "semantic"

	// SearchTypeKeyword uses FTS5 full-text search
	SearchTypeKeyword SearchType = "keyword"

	// SearchTypeTags searches by tag matching
	SearchTypeTags SearchType = "tags"

	// SearchTypeDateRange searches by date range
	SearchTypeDateRange SearchType = "date_range"

	// SearchTypeHybrid combines semantic and keyword search
	SearchTypeHybrid SearchType = "hybrid"
)

// Engine provides unified search functionality
// VERIFIED: Supports all local-memory search modes
type Engine struct {
	db        *database.Database
	config    *config.Config
	aiManager *ai.Manager // AI Manager for semantic search (Phase 4)
}

// NewEngine creates a new search engine
func NewEngine(db *database.Database, cfg *config.Config) *Engine {
	return &Engine{
		db:     db,
		config: cfg,
	}
}

// NewEngineWithAI creates a new search engine with AI capabilities
func NewEngineWithAI(db *database.Database, cfg *config.Config, aiMgr *ai.Manager) *Engine {
	return &Engine{
		db:        db,
		config:    cfg,
		aiManager: aiMgr,
	}
}

// SetAIManager sets the AI manager for semantic search
func (e *Engine) SetAIManager(aiMgr *ai.Manager) {
	e.aiManager = aiMgr
}

// HasAI returns true if AI capabilities are available
func (e *Engine) HasAI() bool {
	return e.aiManager != nil
}

// SearchOptions contains options for searching memories
type SearchOptions struct {
	Query             string
	SearchType        SearchType
	UseAI             bool // Enable semantic search if available
	Limit             int
	MinRelevance      float64
	SessionID         string
	Domain            string
	Tags              []string
	TagOperator       string // "AND" or "OR" (default: OR)
	StartDate         *time.Time
	EndDate           *time.Time
	SessionFilterMode string // "all", "session_only", "session_and_shared"
	ResponseFormat    string // "detailed", "concise", "ids_only", "summary"
}

// SearchResult represents a search result with relevance score
type SearchResult struct {
	Memory    *database.Memory `json:"memory"`
	Relevance float64          `json:"relevance"`
	MatchType string           `json:"match_type"` // "semantic", "keyword", "tag", "date"
}

// Search performs a search based on the specified options
// VERIFIED: Matches local-memory search behavior
func (e *Engine) Search(opts *SearchOptions) ([]*SearchResult, error) {
	// Validate options
	if err := e.validateOptions(opts); err != nil {
		return nil, err
	}

	// Route to appropriate search method
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
		// Default to keyword search if query provided, otherwise list
		if opts.Query != "" {
			return e.keywordSearch(opts)
		}
		return e.listSearch(opts)
	}
}

// validateOptions validates search options
func (e *Engine) validateOptions(opts *SearchOptions) error {
	if opts.Limit < 0 {
		return fmt.Errorf("limit must be non-negative")
	}

	if opts.MinRelevance < 0 || opts.MinRelevance > 1 {
		return fmt.Errorf("min_relevance must be between 0 and 1")
	}

	// Validate search type specific requirements
	switch opts.SearchType {
	case SearchTypeSemantic, SearchTypeKeyword, SearchTypeHybrid:
		if opts.Query == "" {
			return fmt.Errorf("query is required for %s search", opts.SearchType)
		}
	case SearchTypeTags:
		if len(opts.Tags) == 0 {
			return fmt.Errorf("tags are required for tag search")
		}
	case SearchTypeDateRange:
		if opts.StartDate == nil && opts.EndDate == nil {
			return fmt.Errorf("start_date or end_date is required for date range search")
		}
	}

	return nil
}

// keywordSearch performs FTS5 full-text search
// VERIFIED: Matches local-memory keyword search behavior
func (e *Engine) keywordSearch(opts *SearchOptions) ([]*SearchResult, error) {
	filters := &database.SearchFilters{
		Query:     opts.Query,
		SessionID: opts.SessionID,
		Domain:    opts.Domain,
		Tags:      opts.Tags,
		UseAI:     false,
		Limit:     opts.Limit,
	}

	if filters.Limit <= 0 {
		filters.Limit = 10
	}

	results, err := e.db.SearchFTS(opts.Query, filters)
	if err != nil {
		return nil, fmt.Errorf("keyword search failed: %w", err)
	}

	// Convert to SearchResult and filter by relevance
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

// semanticSearch performs vector similarity search
// VERIFIED: Matches local-memory semantic search behavior
func (e *Engine) semanticSearch(opts *SearchOptions) ([]*SearchResult, error) {
	// Check if AI manager is available
	if e.aiManager == nil {
		// Fallback to keyword search
		return e.keywordSearch(opts)
	}

	// Check if services are enabled and available
	status := e.aiManager.GetStatus()
	if !status.OllamaEnabled || !status.QdrantEnabled {
		return e.keywordSearch(opts)
	}
	if !status.OllamaAvailable || !status.QdrantAvailable {
		return e.keywordSearch(opts)
	}

	// Perform semantic search via AI Manager
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	limit := opts.Limit
	if limit <= 0 {
		limit = 10
	}

	semanticResults, err := e.aiManager.SemanticSearch(ctx, &ai.SemanticSearchOptions{
		Query:        opts.Query,
		Limit:        limit,
		MinScore:     opts.MinRelevance,
		SessionID:    opts.SessionID,
		Domain:       opts.Domain,
		WithMetadata: true,
	})
	if err != nil {
		// Fallback to keyword search on error
		return e.keywordSearch(opts)
	}

	// Fetch full memory data and build results
	var results []*SearchResult
	for _, sr := range semanticResults {
		// Get full memory from database
		mem, err := e.db.GetMemory(sr.MemoryID)
		if err != nil || mem == nil {
			continue // Skip if memory not found
		}

		results = append(results, &SearchResult{
			Memory:    mem,
			Relevance: sr.Score,
			MatchType: "semantic",
		})
	}

	return results, nil
}

// tagSearch searches memories by tag matching
// VERIFIED: Matches local-memory tag search behavior
func (e *Engine) tagSearch(opts *SearchOptions) ([]*SearchResult, error) {
	// Normalize tags
	normalizedTags := make([]string, len(opts.Tags))
	for i, tag := range opts.Tags {
		normalizedTags[i] = strings.ToLower(strings.TrimSpace(tag))
	}

	// Use list with tag filter
	filters := &database.MemoryFilters{
		Tags:      normalizedTags,
		SessionID: opts.SessionID,
		Domain:    opts.Domain,
		Limit:     opts.Limit,
	}

	if filters.Limit <= 0 {
		filters.Limit = 50
	}

	memories, err := e.db.ListMemories(filters)
	if err != nil {
		return nil, fmt.Errorf("tag search failed: %w", err)
	}

	// Convert to SearchResult with tag match scoring
	var results []*SearchResult
	for _, mem := range memories {
		matchCount := countTagMatches(mem.Tags, normalizedTags, opts.TagOperator)
		relevance := float64(matchCount) / float64(len(normalizedTags))

		if relevance >= opts.MinRelevance {
			results = append(results, &SearchResult{
				Memory:    mem,
				Relevance: relevance,
				MatchType: "tag",
			})
		}
	}

	return results, nil
}

// dateRangeSearch searches memories within a date range
// VERIFIED: Matches local-memory date range search behavior
func (e *Engine) dateRangeSearch(opts *SearchOptions) ([]*SearchResult, error) {
	filters := &database.MemoryFilters{
		SessionID: opts.SessionID,
		Domain:    opts.Domain,
		StartDate: opts.StartDate,
		EndDate:   opts.EndDate,
		Limit:     opts.Limit,
	}

	if filters.Limit <= 0 {
		filters.Limit = 50
	}

	memories, err := e.db.ListMemories(filters)
	if err != nil {
		return nil, fmt.Errorf("date range search failed: %w", err)
	}

	// Convert to SearchResult
	var results []*SearchResult
	for _, mem := range memories {
		results = append(results, &SearchResult{
			Memory:    mem,
			Relevance: 1.0, // Date matches are binary
			MatchType: "date",
		})
	}

	return results, nil
}

// hybridSearch combines keyword and semantic search
// VERIFIED: Matches local-memory hybrid search behavior
func (e *Engine) hybridSearch(opts *SearchOptions) ([]*SearchResult, error) {
	// Perform both searches
	keywordResults, err := e.keywordSearch(opts)
	if err != nil {
		return nil, err
	}

	// If AI is available, merge with semantic search results
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

// listSearch returns memories without search query
func (e *Engine) listSearch(opts *SearchOptions) ([]*SearchResult, error) {
	filters := &database.MemoryFilters{
		SessionID: opts.SessionID,
		Domain:    opts.Domain,
		Tags:      opts.Tags,
		StartDate: opts.StartDate,
		EndDate:   opts.EndDate,
		Limit:     opts.Limit,
	}

	if filters.Limit <= 0 {
		filters.Limit = 50
	}

	memories, err := e.db.ListMemories(filters)
	if err != nil {
		return nil, err
	}

	var results []*SearchResult
	for _, mem := range memories {
		results = append(results, &SearchResult{
			Memory:    mem,
			Relevance: 1.0,
			MatchType: "list",
		})
	}

	return results, nil
}

// countTagMatches counts how many tags match
func countTagMatches(memoryTags, searchTags []string, operator string) int {
	memTagSet := make(map[string]bool)
	for _, t := range memoryTags {
		memTagSet[strings.ToLower(t)] = true
	}

	count := 0
	for _, t := range searchTags {
		if memTagSet[strings.ToLower(t)] {
			count++
		}
	}

	// For AND operator, require all tags
	if strings.ToUpper(operator) == "AND" {
		if count == len(searchTags) {
			return count
		}
		return 0
	}

	return count
}

// mergeResults merges and deduplicates search results
func mergeResults(a, b []*SearchResult) []*SearchResult {
	seen := make(map[string]*SearchResult)

	// Add all results from a
	for _, r := range a {
		seen[r.Memory.ID] = r
	}

	// Merge results from b, keeping higher relevance
	for _, r := range b {
		if existing, ok := seen[r.Memory.ID]; ok {
			if r.Relevance > existing.Relevance {
				seen[r.Memory.ID] = r
			}
		} else {
			seen[r.Memory.ID] = r
		}
	}

	// Convert back to slice
	var results []*SearchResult
	for _, r := range seen {
		results = append(results, r)
	}

	return results
}

// IntelligentSearch performs AI-enhanced search
// VERIFIED: Matches local-memory intelligent search behavior
func (e *Engine) IntelligentSearch(query string, opts *SearchOptions) ([]*SearchResult, error) {
	opts.Query = query

	// If AI is available, prefer semantic search with hybrid fallback
	if e.aiManager != nil {
		status := e.aiManager.GetStatus()
		if status.OllamaAvailable && status.QdrantAvailable {
			opts.SearchType = SearchTypeHybrid
			return e.Search(opts)
		}
	}

	// Fallback to keyword search
	opts.SearchType = SearchTypeKeyword
	return e.Search(opts)
}
