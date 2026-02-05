package api

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/MycelicMemory/mycelicmemory/internal/database"
)

// =============================================================================
// DATA SOURCE API TYPES
// =============================================================================

// CreateDataSourceRequest represents a request to register a new data source
type CreateDataSourceRequest struct {
	SourceType string `json:"source_type" binding:"required"`
	Name       string `json:"name" binding:"required"`
	Config     string `json:"config"` // JSON configuration
}

// UpdateDataSourceRequest represents a request to update a data source
type UpdateDataSourceRequest struct {
	Name   string `json:"name,omitempty"`
	Config string `json:"config,omitempty"`
	Status string `json:"status,omitempty"`
}

// IngestRequest represents a bulk ingestion request
type IngestRequest struct {
	Items      []IngestItemRequest `json:"items" binding:"required"`
	Checkpoint string              `json:"checkpoint,omitempty"`
}

// IngestItemRequest represents a single item to ingest
type IngestItemRequest struct {
	ExternalID  string                 `json:"external_id" binding:"required"`
	Content     string                 `json:"content" binding:"required"`
	ContentType string                 `json:"content_type,omitempty"`
	Timestamp   string                 `json:"timestamp,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// DataSourceResponse represents a data source in API responses
type DataSourceResponse struct {
	ID               string     `json:"id"`
	SourceType       string     `json:"source_type"`
	Name             string     `json:"name"`
	Config           string     `json:"config"`
	Status           string     `json:"status"`
	LastSyncAt       *time.Time `json:"last_sync_at,omitempty"`
	LastSyncPosition string     `json:"last_sync_position,omitempty"`
	ErrorMessage     string     `json:"error_message,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// SyncHistoryResponse represents a sync history entry in API responses
type SyncHistoryResponse struct {
	ID                string     `json:"id"`
	SourceID          string     `json:"source_id"`
	StartedAt         time.Time  `json:"started_at"`
	CompletedAt       *time.Time `json:"completed_at,omitempty"`
	ItemsProcessed    int        `json:"items_processed"`
	MemoriesCreated   int        `json:"memories_created"`
	DuplicatesSkipped int        `json:"duplicates_skipped"`
	Status            string     `json:"status"`
	Error             string     `json:"error,omitempty"`
}

// IngestResponse represents the result of an ingestion operation
type IngestResponse struct {
	Processed         int    `json:"processed"`
	MemoriesCreated   int    `json:"memories_created"`
	DuplicatesSkipped int    `json:"duplicates_skipped"`
	Checkpoint        string `json:"checkpoint"`
}

// =============================================================================
// DATA SOURCE ENDPOINTS
// =============================================================================

// createDataSource handles POST /api/v1/sources
func (s *Server) createDataSource(c *gin.Context) {
	var req CreateDataSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequestError(c, "Invalid request body: "+err.Error())
		return
	}

	// Validate source type
	if !database.IsValidDataSourceType(req.SourceType) {
		BadRequestError(c, fmt.Sprintf("Invalid source type: %s. Valid types: %v",
			req.SourceType, database.DataSourceTypes))
		return
	}

	// Set default config if not provided
	config := req.Config
	if config == "" {
		config = "{}"
	}

	ds := &database.DataSource{
		SourceType: req.SourceType,
		Name:       req.Name,
		Config:     config,
		Status:     "active",
	}

	if err := s.db.CreateDataSource(ds); err != nil {
		InternalError(c, "Failed to create data source: "+err.Error())
		return
	}

	CreatedResponse(c, "Data source created successfully", toDataSourceResponse(ds))
}

// listDataSources handles GET /api/v1/sources
func (s *Server) listDataSources(c *gin.Context) {
	filters := &database.DataSourceFilters{
		SourceType: c.Query("source_type"),
		Status:     c.Query("status"),
		Limit:      parseIntQuery(c, "limit", 50),
		Offset:     parseIntQuery(c, "offset", 0),
	}

	sources, err := s.db.ListDataSources(filters)
	if err != nil {
		InternalError(c, "Failed to list data sources: "+err.Error())
		return
	}

	// Convert to response format
	results := make([]*DataSourceResponse, len(sources))
	for i, ds := range sources {
		results[i] = toDataSourceResponse(ds)
	}

	SuccessResponse(c, fmt.Sprintf("Listed %d data sources", len(sources)), results)
}

// getDataSource handles GET /api/v1/sources/:id
func (s *Server) getDataSource(c *gin.Context) {
	id := c.Param("id")

	ds, err := s.db.GetDataSource(id)
	if err != nil {
		InternalError(c, "Failed to get data source: "+err.Error())
		return
	}
	if ds == nil {
		NotFoundError(c, "Data source not found")
		return
	}

	SuccessResponse(c, "Data source retrieved successfully", toDataSourceResponse(ds))
}

// updateDataSource handles PATCH /api/v1/sources/:id
func (s *Server) updateDataSource(c *gin.Context) {
	id := c.Param("id")

	var req UpdateDataSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequestError(c, "Invalid request body: "+err.Error())
		return
	}

	updates := &database.DataSourceUpdate{}
	if req.Name != "" {
		updates.Name = &req.Name
	}
	if req.Config != "" {
		updates.Config = &req.Config
	}
	if req.Status != "" {
		if !database.IsValidDataSourceStatus(req.Status) {
			BadRequestError(c, fmt.Sprintf("Invalid status: %s. Valid statuses: %v",
				req.Status, database.DataSourceStatuses))
			return
		}
		updates.Status = &req.Status
	}

	if err := s.db.UpdateDataSource(id, updates); err != nil {
		if err.Error() == fmt.Sprintf("data source not found: %s", id) {
			NotFoundError(c, "Data source not found")
			return
		}
		InternalError(c, "Failed to update data source: "+err.Error())
		return
	}

	// Fetch updated source
	ds, _ := s.db.GetDataSource(id)
	SuccessResponse(c, "Data source updated successfully", toDataSourceResponse(ds))
}

// deleteDataSource handles DELETE /api/v1/sources/:id
func (s *Server) deleteDataSource(c *gin.Context) {
	id := c.Param("id")

	err := s.db.DeleteDataSource(id)
	if err != nil {
		if err.Error() == fmt.Sprintf("data source not found: %s", id) {
			NotFoundError(c, "Data source not found")
			return
		}
		InternalError(c, "Failed to delete data source: "+err.Error())
		return
	}

	response := struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}{
		ID:     id,
		Status: "deleted",
	}

	SuccessResponse(c, "Data source deleted successfully", response)
}

// =============================================================================
// SOURCE CONTROL ENDPOINTS
// =============================================================================

// pauseDataSource handles POST /api/v1/sources/:id/pause
func (s *Server) pauseDataSource(c *gin.Context) {
	id := c.Param("id")

	status := "paused"
	err := s.db.UpdateDataSource(id, &database.DataSourceUpdate{Status: &status})
	if err != nil {
		if err.Error() == fmt.Sprintf("data source not found: %s", id) {
			NotFoundError(c, "Data source not found")
			return
		}
		InternalError(c, "Failed to pause data source: "+err.Error())
		return
	}

	ds, _ := s.db.GetDataSource(id)
	SuccessResponse(c, "Data source paused", toDataSourceResponse(ds))
}

// resumeDataSource handles POST /api/v1/sources/:id/resume
func (s *Server) resumeDataSource(c *gin.Context) {
	id := c.Param("id")

	status := "active"
	err := s.db.UpdateDataSource(id, &database.DataSourceUpdate{Status: &status})
	if err != nil {
		if err.Error() == fmt.Sprintf("data source not found: %s", id) {
			NotFoundError(c, "Data source not found")
			return
		}
		InternalError(c, "Failed to resume data source: "+err.Error())
		return
	}

	ds, _ := s.db.GetDataSource(id)
	SuccessResponse(c, "Data source resumed", toDataSourceResponse(ds))
}

// triggerSync handles POST /api/v1/sources/:id/sync
// Note: This creates a sync history entry. The actual sync is triggered externally.
func (s *Server) triggerSync(c *gin.Context) {
	id := c.Param("id")

	// Verify source exists
	ds, err := s.db.GetDataSource(id)
	if err != nil {
		InternalError(c, "Failed to get data source: "+err.Error())
		return
	}
	if ds == nil {
		NotFoundError(c, "Data source not found")
		return
	}

	if ds.Status == "paused" {
		BadRequestError(c, "Cannot sync a paused data source")
		return
	}

	// Create a sync history entry
	sh := &database.DataSourceSyncHistory{
		SourceID:  id,
		StartedAt: time.Now(),
		Status:    "running",
	}

	if err := s.db.CreateSyncHistory(sh); err != nil {
		InternalError(c, "Failed to create sync record: "+err.Error())
		return
	}

	SuccessResponse(c, "Sync triggered", toSyncHistoryResponse(sh))
}

// =============================================================================
// INGESTION ENDPOINT
// =============================================================================

// ingestItems handles POST /api/v1/sources/:id/ingest
func (s *Server) ingestItems(c *gin.Context) {
	id := c.Param("id")

	// Verify source exists and is active
	ds, err := s.db.GetDataSource(id)
	if err != nil {
		InternalError(c, "Failed to get data source: "+err.Error())
		return
	}
	if ds == nil {
		NotFoundError(c, "Data source not found")
		return
	}
	if ds.Status != "active" {
		BadRequestError(c, fmt.Sprintf("Data source is %s, cannot ingest", ds.Status))
		return
	}

	var req IngestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequestError(c, "Invalid request body: "+err.Error())
		return
	}

	if len(req.Items) == 0 {
		BadRequestError(c, "No items to ingest")
		return
	}

	// Process items
	var memoriesCreated, duplicatesSkipped int
	var lastCheckpoint string

	for _, item := range req.Items {
		ingestItem := toIngestItem(&item, ds.SourceType)

		_, created, err := s.db.IngestMemory(id, ingestItem)
		if err != nil {
			s.log.Warn("failed to ingest item", "external_id", item.ExternalID, "error", err)
			continue
		}

		if created {
			memoriesCreated++

			// Index for semantic search if AI is available
			if s.aiManager != nil && created {
				mem, _ := s.db.GetMemory(item.ExternalID)
				if mem != nil {
					ctx := c.Request.Context()
					_ = s.aiManager.IndexMemory(ctx, mem)
				}
			}
		} else {
			duplicatesSkipped++
		}

		lastCheckpoint = item.ExternalID
	}

	// Update checkpoint if provided or use last item's external_id
	checkpoint := req.Checkpoint
	if checkpoint == "" {
		checkpoint = lastCheckpoint
	}

	// Update source sync time
	_ = s.db.UpdateDataSourceSyncTime(id, time.Now(), checkpoint)

	response := &IngestResponse{
		Processed:         len(req.Items),
		MemoriesCreated:   memoriesCreated,
		DuplicatesSkipped: duplicatesSkipped,
		Checkpoint:        checkpoint,
	}

	SuccessResponse(c, fmt.Sprintf("Ingested %d items, created %d memories", len(req.Items), memoriesCreated), response)
}

// =============================================================================
// HISTORY & STATS ENDPOINTS
// =============================================================================

// getSyncHistory handles GET /api/v1/sources/:id/history
func (s *Server) getSyncHistory(c *gin.Context) {
	id := c.Param("id")
	limit := parseIntQuery(c, "limit", 20)

	// Verify source exists
	ds, err := s.db.GetDataSource(id)
	if err != nil || ds == nil {
		NotFoundError(c, "Data source not found")
		return
	}

	history, err := s.db.GetSyncHistory(id, limit)
	if err != nil {
		InternalError(c, "Failed to get sync history: "+err.Error())
		return
	}

	results := make([]*SyncHistoryResponse, len(history))
	for i, sh := range history {
		results[i] = toSyncHistoryResponse(sh)
	}

	SuccessResponse(c, fmt.Sprintf("Retrieved %d sync history entries", len(history)), results)
}

// getSourceStats handles GET /api/v1/sources/:id/stats
func (s *Server) getSourceStats(c *gin.Context) {
	id := c.Param("id")

	// Verify source exists
	ds, err := s.db.GetDataSource(id)
	if err != nil || ds == nil {
		NotFoundError(c, "Data source not found")
		return
	}

	stats, err := s.db.GetDataSourceStats(id)
	if err != nil {
		InternalError(c, "Failed to get source stats: "+err.Error())
		return
	}

	SuccessResponse(c, "Source statistics retrieved", stats)
}

// getSourceMemories handles GET /api/v1/sources/:id/memories
func (s *Server) getSourceMemories(c *gin.Context) {
	id := c.Param("id")
	limit := parseIntQuery(c, "limit", 50)
	offset := parseIntQuery(c, "offset", 0)

	// Verify source exists
	ds, err := s.db.GetDataSource(id)
	if err != nil || ds == nil {
		NotFoundError(c, "Data source not found")
		return
	}

	memories, err := s.db.GetMemoriesBySource(id, limit, offset)
	if err != nil {
		InternalError(c, "Failed to get memories: "+err.Error())
		return
	}

	// Convert to response format
	results := make([]*MemoryResponse, len(memories))
	for i, m := range memories {
		results[i] = toMemoryResponse(m, 1.0, nil)
	}

	SuccessResponse(c, fmt.Sprintf("Listed %d memories from source", len(memories)), results)
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

func toDataSourceResponse(ds *database.DataSource) *DataSourceResponse {
	if ds == nil {
		return nil
	}
	return &DataSourceResponse{
		ID:               ds.ID,
		SourceType:       ds.SourceType,
		Name:             ds.Name,
		Config:           ds.Config,
		Status:           ds.Status,
		LastSyncAt:       ds.LastSyncAt,
		LastSyncPosition: ds.LastSyncPosition,
		ErrorMessage:     ds.ErrorMessage,
		CreatedAt:        ds.CreatedAt,
		UpdatedAt:        ds.UpdatedAt,
	}
}

func toSyncHistoryResponse(sh *database.DataSourceSyncHistory) *SyncHistoryResponse {
	if sh == nil {
		return nil
	}
	return &SyncHistoryResponse{
		ID:                sh.ID,
		SourceID:          sh.SourceID,
		StartedAt:         sh.StartedAt,
		CompletedAt:       sh.CompletedAt,
		ItemsProcessed:    sh.ItemsProcessed,
		MemoriesCreated:   sh.MemoriesCreated,
		DuplicatesSkipped: sh.DuplicatesSkipped,
		Status:            sh.Status,
		Error:             sh.Error,
	}
}

func toIngestItem(req *IngestItemRequest, sourceType string) *database.IngestItem {
	var timestamp time.Time
	if req.Timestamp != "" {
		timestamp, _ = time.Parse(time.RFC3339, req.Timestamp)
	}
	if timestamp.IsZero() {
		timestamp = time.Now()
	}

	metadata := database.IngestMetadata{
		SourceType: sourceType,
	}

	if req.Metadata != nil {
		if author, ok := req.Metadata["author"].(string); ok {
			metadata.Author = author
		}
		if channel, ok := req.Metadata["channel"].(string); ok {
			metadata.Channel = channel
		}
		if threadID, ok := req.Metadata["thread_id"].(string); ok {
			metadata.ThreadID = threadID
		}
		if domain, ok := req.Metadata["domain"].(string); ok {
			metadata.Domain = domain
		}
		if importance, ok := req.Metadata["importance"].(float64); ok {
			metadata.Importance = int(importance)
		}
		if tags, ok := req.Metadata["tags"].([]interface{}); ok {
			for _, t := range tags {
				if tag, ok := t.(string); ok {
					metadata.Tags = append(metadata.Tags, tag)
				}
			}
		}
	}

	return &database.IngestItem{
		ExternalID:  req.ExternalID,
		Content:     req.Content,
		ContentType: req.ContentType,
		Timestamp:   timestamp,
		Metadata:    metadata,
	}
}
