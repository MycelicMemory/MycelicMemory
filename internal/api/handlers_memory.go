package api

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/MycelicMemory/mycelicmemory/internal/database"
	"github.com/MycelicMemory/mycelicmemory/internal/memory"
)

// MemoryResponse matches local-memory format exactly
type MemoryResponse struct {
	Memory          *MemoryData `json:"memory"`
	RelevanceScore  float64     `json:"relevance_score"`
	SimilarityScore *float64    `json:"similarity_score"`
}

// MemoryData represents memory data in responses
type MemoryData struct {
	ID          string    `json:"id"`
	Content     string    `json:"content"`
	Source      *string   `json:"source"`
	Slug        *string   `json:"slug"`
	Importance  int       `json:"importance"`
	Tags        []string  `json:"tags"`
	SessionID   string    `json:"session_id"`
	Domain      *string   `json:"domain"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	CCSessionID *string   `json:"cc_session_id,omitempty"`
}

// CreateMemoryRequest represents a memory creation request
type CreateMemoryRequest struct {
	Content    string   `json:"content" binding:"required"`
	Importance int      `json:"importance"`
	Tags       []string `json:"tags"`
	Domain     string   `json:"domain"`
	Source     string   `json:"source"`
}

// UpdateMemoryRequest represents a memory update request
type UpdateMemoryRequest struct {
	Content    string   `json:"content"`
	Importance int      `json:"importance"`
	Tags       []string `json:"tags"`
	Domain     string   `json:"domain"`
	Source     string   `json:"source"`
}

// toMemoryData converts database.Memory to MemoryData
func toMemoryData(m *database.Memory) *MemoryData {
	data := &MemoryData{
		ID:         m.ID,
		Content:    m.Content,
		Importance: m.Importance,
		Tags:       m.Tags,
		SessionID:  m.SessionID,
		CreatedAt:  m.CreatedAt,
		UpdatedAt:  m.UpdatedAt,
	}

	// Handle nullable fields
	if m.Source != "" {
		data.Source = &m.Source
	}
	if m.Slug != "" {
		data.Slug = &m.Slug
	}
	if m.Domain != "" {
		data.Domain = &m.Domain
	}
	if m.CCSessionID != "" {
		data.CCSessionID = &m.CCSessionID
	}

	// Ensure tags is not nil
	if data.Tags == nil {
		data.Tags = []string{}
	}

	return data
}

// toMemoryResponse converts database.Memory to MemoryResponse
func toMemoryResponse(m *database.Memory, relevance float64, similarity *float64) *MemoryResponse {
	return &MemoryResponse{
		Memory:          toMemoryData(m),
		RelevanceScore:  relevance,
		SimilarityScore: similarity,
	}
}

// createMemory handles POST /api/v1/memories
func (s *Server) createMemory(c *gin.Context) {
	var req CreateMemoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequestError(c, "Invalid request body: "+err.Error())
		return
	}

	result, err := s.memoryService.Store(&memory.StoreOptions{
		Content:    req.Content,
		Importance: req.Importance,
		Tags:       req.Tags,
		Domain:     req.Domain,
		Source:     req.Source,
	})
	if err != nil {
		InternalError(c, "Failed to store memory: "+err.Error())
		return
	}

	// Index for semantic search if AI is available
	if s.aiManager != nil {
		ctx := c.Request.Context()
		_ = s.aiManager.IndexMemory(ctx, result.Memory)
	}

	// Return flat memory data to match local-memory format
	CreatedResponse(c, "Memory stored successfully", toMemoryData(result.Memory))
}

// getMemory handles GET /api/v1/memories/:id
func (s *Server) getMemory(c *gin.Context) {
	id := c.Param("id")

	mem, err := s.memoryService.Get(&memory.GetOptions{ID: id})
	if err != nil || mem == nil {
		// Use local-memory format for not found
		NotFoundErrorWithID(c, id)
		return
	}

	// Return flat memory data to match local-memory format
	SuccessResponse(c, "Memory retrieved successfully", toMemoryData(mem))
}

// listMemories handles GET /api/v1/memories
func (s *Server) listMemories(c *gin.Context) {
	// Parse query parameters
	limit := parseIntQuery(c, "limit", 50)
	offset := parseIntQuery(c, "offset", 0)
	sessionID := c.Query("session_id")
	domain := c.Query("domain")

	filters := &database.MemoryFilters{
		SessionID: sessionID,
		Domain:    domain,
		Limit:     limit,
		Offset:    offset,
	}

	memories, err := s.db.ListMemories(filters)
	if err != nil {
		InternalError(c, "Failed to list memories: "+err.Error())
		return
	}

	// Convert to response format - wrapped format to match local-memory list
	results := make([]*MemoryResponse, len(memories))
	for i, m := range memories {
		results[i] = toMemoryResponse(m, 1.0, nil)
	}

	SuccessResponse(c, "Listed "+intToString(len(memories))+" memories", results)
}

// updateMemory handles PUT /api/v1/memories/:id
func (s *Server) updateMemory(c *gin.Context) {
	id := c.Param("id")

	var req UpdateMemoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequestError(c, "Invalid request body: "+err.Error())
		return
	}

	opts := &memory.UpdateOptions{
		ID:   id,
		Tags: req.Tags,
	}

	// Set optional fields only if provided
	if req.Content != "" {
		opts.Content = &req.Content
	}
	if req.Importance > 0 {
		opts.Importance = &req.Importance
	}
	if req.Domain != "" {
		opts.Domain = &req.Domain
	}
	if req.Source != "" {
		opts.Source = &req.Source
	}

	mem, err := s.memoryService.Update(opts)
	if err != nil {
		if err.Error() == "memory not found" {
			NotFoundError(c, "Memory not found")
			return
		}
		InternalError(c, "Failed to update memory: "+err.Error())
		return
	}

	// Re-index for semantic search if AI is available
	if s.aiManager != nil {
		ctx := c.Request.Context()
		_ = s.aiManager.IndexMemory(ctx, mem)
	}

	// Return flat memory data to match local-memory format
	SuccessResponse(c, "Memory updated successfully", toMemoryData(mem))
}

// deleteMemory handles DELETE /api/v1/memories/:id
func (s *Server) deleteMemory(c *gin.Context) {
	id := c.Param("id")

	err := s.memoryService.Delete(id)
	if err != nil {
		if err.Error() == "memory not found" {
			NotFoundError(c, "Memory not found")
			return
		}
		InternalError(c, "Failed to delete memory: "+err.Error())
		return
	}

	// Remove from vector index if AI is available
	if s.aiManager != nil {
		ctx := c.Request.Context()
		_ = s.aiManager.DeleteMemoryIndex(ctx, id)
	}

	// Return deleted ID and status to match local-memory format
	response := struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}{
		ID:     id,
		Status: "deleted",
	}

	SuccessResponse(c, "Memory deleted successfully", response)
}

// memoryStats handles GET /api/v1/memories/stats
func (s *Server) memoryStats(c *gin.Context) {
	// This is an alias for systemStats focusing on memory stats
	s.systemStats(c)
}

// Helper function to parse int query parameters
func parseIntQuery(c *gin.Context, key string, defaultVal int) int {
	val := c.Query(key)
	if val == "" {
		return defaultVal
	}
	var result int
	if _, err := parseIntString(val, &result); err != nil {
		return defaultVal
	}
	return result
}

func parseIntString(s string, result *int) (bool, error) {
	var n int
	for _, c := range s {
		if c < '0' || c > '9' {
			return false, nil
		}
		n = n*10 + int(c-'0')
	}
	*result = n
	return true, nil
}
