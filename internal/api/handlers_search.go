package api

import (
	"crypto/md5"
	"encoding/hex"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/MycelicMemory/mycelicmemory/internal/search"
)

// SearchResultItem matches local-memory's search result format
type SearchResultItem struct {
	ID             string   `json:"id"`
	Summary        string   `json:"summary"`
	RelevanceScore float64  `json:"relevance_score"`
	Tags           []string `json:"tags,omitempty"`
	Importance     int      `json:"importance"`
	CreatedAt      string   `json:"created_at"`
}

// PaginationMetadata matches local-memory's pagination format
type PaginationMetadata struct {
	HasNextPage     bool `json:"has_next_page"`
	HasPreviousPage bool `json:"has_previous_page"`
	TotalCount      int  `json:"total_count"`
	PageSize        int  `json:"page_size"`
	CurrentPage     int  `json:"current_page"`
}

// SearchInfo matches local-memory's search info format
type SearchInfo struct {
	Query            string `json:"query"`
	SearchType       string `json:"search_type"`
	TotalResults     int    `json:"total_results"`
	ProcessingTimeMs int    `json:"processing_time_ms"`
	HasMoreResults   bool   `json:"has_more_results"`
}

// SearchResponse matches local-memory's complete search response
type SearchResponse struct {
	Data               []SearchResultItem `json:"data"`
	PaginationMetadata PaginationMetadata `json:"pagination_metadata"`
	QueryHash          string             `json:"query_hash"`
	SearchInfo         SearchInfo         `json:"search_info"`
}

// SearchRequest represents a search request
type SearchRequest struct {
	Query             string   `json:"query"`
	SearchType        string   `json:"search_type"`
	UseAI             bool     `json:"use_ai"`
	Limit             int      `json:"limit"`
	Tags              []string `json:"tags"`
	Domain            string   `json:"domain"`
	SessionID         string   `json:"session_id"`
	SessionFilterMode string   `json:"session_filter_mode"`
	ResponseFormat    string   `json:"response_format"`
}

// TagSearchRequest represents a tag search request
type TagSearchRequest struct {
	Tags        []string `json:"tags" binding:"required"`
	TagOperator string   `json:"tag_operator"` // "AND" or "OR"
	Limit       int      `json:"limit"`
	SessionID   string   `json:"session_id"`
	Domain      string   `json:"domain"`
}

// DateRangeSearchRequest represents a date range search request
type DateRangeSearchRequest struct {
	StartDate string `json:"start_date"` // YYYY-MM-DD
	EndDate   string `json:"end_date"`   // YYYY-MM-DD
	Limit     int    `json:"limit"`
	SessionID string `json:"session_id"`
	Domain    string `json:"domain"`
}

// searchMemoriesGET handles GET /api/v1/memories/search
func (s *Server) searchMemoriesGET(c *gin.Context) {
	query := c.Query("query")
	if query == "" {
		// Return all memories if no query
		s.listMemories(c)
		return
	}

	if err := validateQuery(query); err != nil {
		BadRequestError(c, err.Error())
		return
	}

	limit := clampLimit(parseIntQuery(c, "limit", 10))
	domain := c.Query("domain")
	sessionID := c.Query("session_id")
	useAI := c.Query("use_ai") == "true"

	searchType := search.SearchTypeKeyword
	if useAI {
		searchType = search.SearchTypeHybrid
	}

	opts := &search.SearchOptions{
		Query:      query,
		SearchType: searchType,
		UseAI:      useAI,
		Limit:      limit,
		Domain:     domain,
		SessionID:  sessionID,
	}

	results, err := s.searchEngine.Search(opts)
	if err != nil {
		InternalError(c, "Search failed: "+err.Error())
		return
	}

	// Convert to response format
	response := make([]*MemoryResponse, len(results))
	for i, r := range results {
		var similarity *float64
		if r.MatchType == "semantic" {
			similarity = &r.Relevance
		}
		response[i] = toMemoryResponse(r.Memory, r.Relevance, similarity)
	}

	SuccessResponse(c, "Search completed successfully", response)
}

// searchMemoriesPOST handles POST /api/v1/memories/search
func (s *Server) searchMemoriesPOST(c *gin.Context) {
	var req SearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequestError(c, "Invalid request body: "+err.Error())
		return
	}

	if req.Query == "" && len(req.Tags) == 0 {
		BadRequestError(c, "Query or tags required for search")
		return
	}

	if req.Query != "" {
		if err := validateQuery(req.Query); err != nil {
			BadRequestError(c, err.Error())
			return
		}
	}
	if len(req.Tags) > 0 {
		if err := validateTags(req.Tags); err != nil {
			BadRequestError(c, err.Error())
			return
		}
	}

	// Determine search type
	searchType := search.SearchTypeKeyword
	switch req.SearchType {
	case "semantic":
		searchType = search.SearchTypeSemantic
	case "tags":
		searchType = search.SearchTypeTags
	case "date_range":
		searchType = search.SearchTypeDateRange
	case "hybrid":
		searchType = search.SearchTypeHybrid
	}

	if req.UseAI && searchType == search.SearchTypeKeyword {
		searchType = search.SearchTypeHybrid
	}

	limit := clampLimit(req.Limit)
	if limit <= 0 {
		limit = 10
	}

	opts := &search.SearchOptions{
		Query:             req.Query,
		SearchType:        searchType,
		UseAI:             req.UseAI,
		Limit:             limit,
		Tags:              req.Tags,
		Domain:            req.Domain,
		SessionID:         req.SessionID,
		SessionFilterMode: req.SessionFilterMode,
		ResponseFormat:    req.ResponseFormat,
	}

	results, err := s.searchEngine.Search(opts)
	if err != nil {
		InternalError(c, "Search failed: "+err.Error())
		return
	}

	// Convert to local-memory format
	items := make([]SearchResultItem, len(results))
	for i, r := range results {
		items[i] = SearchResultItem{
			ID:             r.Memory.ID,
			Summary:        r.Memory.Content,
			RelevanceScore: r.Relevance,
			Tags:           r.Memory.Tags,
			Importance:     r.Memory.Importance,
			CreatedAt:      "", // local-memory returns empty string
		}
	}

	// Generate query hash
	hash := md5.Sum([]byte(req.Query))
	queryHash := hex.EncodeToString(hash[:])[:8]

	// Determine search type string
	searchTypeStr := "enhanced_text"
	if req.UseAI {
		searchTypeStr = "semantic"
	}

	response := &SearchResponse{
		Data: items,
		PaginationMetadata: PaginationMetadata{
			HasNextPage:     false,
			HasPreviousPage: false,
			TotalCount:      len(results),
			PageSize:        limit,
			CurrentPage:     1,
		},
		QueryHash: queryHash,
		SearchInfo: SearchInfo{
			Query:            req.Query,
			SearchType:       searchTypeStr,
			TotalResults:     len(results),
			ProcessingTimeMs: 0,
			HasMoreResults:   false,
		},
	}

	c.JSON(200, response)
}

// intelligentSearch handles POST /api/v1/memories/search/intelligent
func (s *Server) intelligentSearch(c *gin.Context) {
	var req SearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequestError(c, "Invalid request body: "+err.Error())
		return
	}

	if req.Query == "" {
		BadRequestError(c, "Query required for intelligent search")
		return
	}

	if err := validateQuery(req.Query); err != nil {
		BadRequestError(c, err.Error())
		return
	}

	limit := clampLimit(req.Limit)
	if limit <= 0 {
		limit = 10
	}

	opts := &search.SearchOptions{
		Limit:             limit,
		Domain:            req.Domain,
		SessionID:         req.SessionID,
		SessionFilterMode: req.SessionFilterMode,
		ResponseFormat:    req.ResponseFormat,
	}

	results, err := s.searchEngine.IntelligentSearch(req.Query, opts)
	if err != nil {
		InternalError(c, "Intelligent search failed: "+err.Error())
		return
	}

	// Convert to response format
	response := make([]*MemoryResponse, len(results))
	for i, r := range results {
		var similarity *float64
		if r.MatchType == "semantic" {
			similarity = &r.Relevance
		}
		response[i] = toMemoryResponse(r.Memory, r.Relevance, similarity)
	}

	SuccessResponse(c, "Intelligent search completed successfully", response)
}

// searchByTags handles POST /api/v1/search/tags
func (s *Server) searchByTags(c *gin.Context) {
	var req TagSearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequestError(c, "Invalid request body: "+err.Error())
		return
	}

	if err := validateTags(req.Tags); err != nil {
		BadRequestError(c, err.Error())
		return
	}

	limit := clampLimit(req.Limit)
	if limit <= 0 {
		limit = 50
	}

	tagOperator := req.TagOperator
	if tagOperator == "" {
		tagOperator = "OR"
	}
	if tagOperator != "AND" && tagOperator != "OR" {
		BadRequestError(c, "tag_operator must be 'AND' or 'OR'")
		return
	}

	opts := &search.SearchOptions{
		SearchType:  search.SearchTypeTags,
		Tags:        req.Tags,
		TagOperator: tagOperator,
		Limit:       limit,
		SessionID:   req.SessionID,
		Domain:      req.Domain,
	}

	results, err := s.searchEngine.Search(opts)
	if err != nil {
		InternalError(c, "Tag search failed: "+err.Error())
		return
	}

	// Convert to response format
	response := make([]*MemoryResponse, len(results))
	for i, r := range results {
		response[i] = toMemoryResponse(r.Memory, r.Relevance, nil)
	}

	SuccessResponse(c, "Found "+intToString(len(results))+" memories matching tags", response)
}

// searchByDateRange handles POST /api/v1/search/date-range
func (s *Server) searchByDateRange(c *gin.Context) {
	var req DateRangeSearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequestError(c, "Invalid request body: "+err.Error())
		return
	}

	if req.StartDate == "" && req.EndDate == "" {
		BadRequestError(c, "At least start_date or end_date required")
		return
	}

	limit := clampLimit(req.Limit)
	if limit <= 0 {
		limit = 50
	}

	opts := &search.SearchOptions{
		SearchType: search.SearchTypeDateRange,
		Limit:      limit,
		SessionID:  req.SessionID,
		Domain:     req.Domain,
	}

	// Parse dates
	if req.StartDate != "" {
		startDate, err := time.Parse("2006-01-02", req.StartDate)
		if err != nil {
			BadRequestError(c, "Invalid start_date format, use YYYY-MM-DD")
			return
		}
		opts.StartDate = &startDate
	}

	if req.EndDate != "" {
		endDate, err := time.Parse("2006-01-02", req.EndDate)
		if err != nil {
			BadRequestError(c, "Invalid end_date format, use YYYY-MM-DD")
			return
		}
		// Set to end of day
		endDate = endDate.Add(24*time.Hour - time.Second)
		opts.EndDate = &endDate
	}

	results, err := s.searchEngine.Search(opts)
	if err != nil {
		InternalError(c, "Date range search failed: "+err.Error())
		return
	}

	// Convert to response format
	response := make([]*MemoryResponse, len(results))
	for i, r := range results {
		response[i] = toMemoryResponse(r.Memory, r.Relevance, nil)
	}

	SuccessResponse(c, "Found "+intToString(len(results))+" memories in date range", response)
}
