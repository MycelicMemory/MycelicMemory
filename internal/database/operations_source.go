package database

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// DATA SOURCE OPERATIONS
// =============================================================================

// CreateDataSource creates a new data source in the registry
func (d *Database) CreateDataSource(ds *DataSource) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Validate source type
	if !IsValidDataSourceType(ds.SourceType) {
		return fmt.Errorf("invalid source type: %s", ds.SourceType)
	}

	// Generate UUID if not provided
	if ds.ID == "" {
		ds.ID = uuid.New().String()
	}

	// Set timestamps
	now := time.Now()
	if ds.CreatedAt.IsZero() {
		ds.CreatedAt = now
	}
	ds.UpdatedAt = now

	// Default values
	if ds.Status == "" {
		ds.Status = "active"
	}
	if ds.Config == "" {
		ds.Config = "{}"
	}

	_, err := d.db.Exec(`
		INSERT INTO data_sources (
			id, source_type, name, config, status, last_sync_at,
			last_sync_position, error_message, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		ds.ID, ds.SourceType, ds.Name, ds.Config, ds.Status,
		nullTimePtr(ds.LastSyncAt), nullString(ds.LastSyncPosition),
		nullString(ds.ErrorMessage), ds.CreatedAt, ds.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create data source: %w", err)
	}

	return nil
}

// GetDataSource retrieves a data source by ID
func (d *Database) GetDataSource(id string) (*DataSource, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var ds DataSource
	var lastSyncAt sql.NullTime
	var lastSyncPosition, errorMessage sql.NullString

	err := d.db.QueryRow(`
		SELECT id, source_type, name, config, status, last_sync_at,
		       last_sync_position, error_message, created_at, updated_at
		FROM data_sources WHERE id = ?
	`, id).Scan(
		&ds.ID, &ds.SourceType, &ds.Name, &ds.Config, &ds.Status,
		&lastSyncAt, &lastSyncPosition, &errorMessage,
		&ds.CreatedAt, &ds.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get data source: %w", err)
	}

	// Handle nullable fields
	if lastSyncAt.Valid {
		ds.LastSyncAt = &lastSyncAt.Time
	}
	ds.LastSyncPosition = lastSyncPosition.String
	ds.ErrorMessage = errorMessage.String

	return &ds, nil
}

// ListDataSources retrieves all data sources with optional filters
func (d *Database) ListDataSources(filters *DataSourceFilters) ([]*DataSource, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var whereClauses []string
	var args []interface{}

	if filters.SourceType != "" {
		whereClauses = append(whereClauses, "source_type = ?")
		args = append(args, filters.SourceType)
	}
	if filters.Status != "" {
		whereClauses = append(whereClauses, "status = ?")
		args = append(args, filters.Status)
	}

	query := `
		SELECT id, source_type, name, config, status, last_sync_at,
		       last_sync_position, error_message, created_at, updated_at
		FROM data_sources
	`

	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	query += " ORDER BY created_at DESC"

	// Apply pagination
	limit := filters.Limit
	if limit <= 0 {
		limit = 50
	}
	query += fmt.Sprintf(" LIMIT %d", limit)

	if filters.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", filters.Offset)
	}

	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list data sources: %w", err)
	}
	defer rows.Close()

	return scanDataSources(rows)
}

// UpdateDataSource updates an existing data source
func (d *Database) UpdateDataSource(id string, updates *DataSourceUpdate) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Build dynamic update query
	var setClauses []string
	var args []interface{}

	if updates.Name != nil {
		setClauses = append(setClauses, "name = ?")
		args = append(args, *updates.Name)
	}
	if updates.Config != nil {
		setClauses = append(setClauses, "config = ?")
		args = append(args, *updates.Config)
	}
	if updates.Status != nil {
		if !IsValidDataSourceStatus(*updates.Status) {
			return fmt.Errorf("invalid status: %s", *updates.Status)
		}
		setClauses = append(setClauses, "status = ?")
		args = append(args, *updates.Status)
	}
	if updates.LastSyncPosition != nil {
		setClauses = append(setClauses, "last_sync_position = ?")
		args = append(args, *updates.LastSyncPosition)
	}
	if updates.ErrorMessage != nil {
		setClauses = append(setClauses, "error_message = ?")
		args = append(args, *updates.ErrorMessage)
	}

	if len(setClauses) == 0 {
		return nil // No updates to apply
	}

	// Always update updated_at
	setClauses = append(setClauses, "updated_at = ?")
	args = append(args, time.Now())

	// Add WHERE clause
	args = append(args, id)

	query := fmt.Sprintf("UPDATE data_sources SET %s WHERE id = ?", strings.Join(setClauses, ", "))

	result, err := d.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update data source: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("data source not found: %s", id)
	}

	return nil
}

// DeleteDataSource removes a data source by ID
// Note: CASCADE delete will remove associated sync history
func (d *Database) DeleteDataSource(id string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	result, err := d.db.Exec("DELETE FROM data_sources WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete data source: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("data source not found: %s", id)
	}

	return nil
}

// UpdateDataSourceSyncTime updates the last sync timestamp
func (d *Database) UpdateDataSourceSyncTime(id string, syncTime time.Time, position string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	result, err := d.db.Exec(`
		UPDATE data_sources
		SET last_sync_at = ?, last_sync_position = ?, updated_at = ?, error_message = NULL, status = 'active'
		WHERE id = ?
	`, syncTime, position, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update sync time: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("data source not found: %s", id)
	}

	return nil
}

// SetDataSourceError sets the error state for a data source
func (d *Database) SetDataSourceError(id string, errorMsg string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	result, err := d.db.Exec(`
		UPDATE data_sources
		SET status = 'error', error_message = ?, updated_at = ?
		WHERE id = ?
	`, errorMsg, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to set error: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("data source not found: %s", id)
	}

	return nil
}

// =============================================================================
// SYNC HISTORY OPERATIONS
// =============================================================================

// CreateSyncHistory creates a new sync history record
func (d *Database) CreateSyncHistory(sh *DataSourceSyncHistory) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if sh.ID == "" {
		sh.ID = uuid.New().String()
	}

	if sh.StartedAt.IsZero() {
		sh.StartedAt = time.Now()
	}

	if sh.Status == "" {
		sh.Status = "running"
	}

	_, err := d.db.Exec(`
		INSERT INTO data_source_sync_history (
			id, source_id, started_at, completed_at, items_processed,
			memories_created, duplicates_skipped, status, error
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		sh.ID, sh.SourceID, sh.StartedAt, nullTimePtr(sh.CompletedAt),
		sh.ItemsProcessed, sh.MemoriesCreated, sh.DuplicatesSkipped,
		sh.Status, nullString(sh.Error),
	)

	if err != nil {
		return fmt.Errorf("failed to create sync history: %w", err)
	}

	return nil
}

// UpdateSyncHistory updates a sync history record
func (d *Database) UpdateSyncHistory(id string, itemsProcessed, memoriesCreated, duplicatesSkipped int, status, errorMsg string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	var completedAt interface{}
	if status == "completed" || status == "failed" {
		completedAt = time.Now()
	}

	_, err := d.db.Exec(`
		UPDATE data_source_sync_history
		SET items_processed = ?, memories_created = ?, duplicates_skipped = ?,
		    status = ?, error = ?, completed_at = ?
		WHERE id = ?
	`, itemsProcessed, memoriesCreated, duplicatesSkipped, status, nullString(errorMsg), completedAt, id)

	if err != nil {
		return fmt.Errorf("failed to update sync history: %w", err)
	}

	return nil
}

// GetSyncHistory retrieves sync history for a data source
func (d *Database) GetSyncHistory(sourceID string, limit int) ([]*DataSourceSyncHistory, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if limit <= 0 {
		limit = 20
	}

	rows, err := d.db.Query(`
		SELECT id, source_id, started_at, completed_at, items_processed,
		       memories_created, duplicates_skipped, status, error
		FROM data_source_sync_history
		WHERE source_id = ?
		ORDER BY started_at DESC
		LIMIT ?
	`, sourceID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get sync history: %w", err)
	}
	defer rows.Close()

	return scanSyncHistory(rows)
}

// GetLatestSyncHistory retrieves the most recent sync for a source
func (d *Database) GetLatestSyncHistory(sourceID string) (*DataSourceSyncHistory, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var sh DataSourceSyncHistory
	var completedAt sql.NullTime
	var errorMsg sql.NullString

	err := d.db.QueryRow(`
		SELECT id, source_id, started_at, completed_at, items_processed,
		       memories_created, duplicates_skipped, status, error
		FROM data_source_sync_history
		WHERE source_id = ?
		ORDER BY started_at DESC
		LIMIT 1
	`, sourceID).Scan(
		&sh.ID, &sh.SourceID, &sh.StartedAt, &completedAt,
		&sh.ItemsProcessed, &sh.MemoriesCreated, &sh.DuplicatesSkipped,
		&sh.Status, &errorMsg,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get latest sync: %w", err)
	}

	if completedAt.Valid {
		sh.CompletedAt = &completedAt.Time
	}
	sh.Error = errorMsg.String

	return &sh, nil
}

// =============================================================================
// DATA SOURCE STATISTICS
// =============================================================================

// GetDataSourceStats retrieves statistics for a data source
func (d *Database) GetDataSourceStats(sourceID string) (*DataSourceStats, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	stats := &DataSourceStats{}

	// Get memory count
	err := d.db.QueryRow(`
		SELECT COUNT(*) FROM memories WHERE source_id = ?
	`, sourceID).Scan(&stats.TotalMemories)
	if err != nil {
		return nil, fmt.Errorf("failed to count memories: %w", err)
	}

	// Get sync counts
	err = d.db.QueryRow(`
		SELECT
			COUNT(*),
			COALESCE(SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END), 0)
		FROM data_source_sync_history
		WHERE source_id = ?
	`, sourceID).Scan(&stats.TotalSyncs, &stats.SuccessfulSyncs, &stats.FailedSyncs)
	if err != nil {
		return nil, fmt.Errorf("failed to count syncs: %w", err)
	}

	// Get last sync time
	var lastSyncAt sql.NullTime
	err = d.db.QueryRow(`
		SELECT last_sync_at FROM data_sources WHERE id = ?
	`, sourceID).Scan(&lastSyncAt)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get last sync: %w", err)
	}
	if lastSyncAt.Valid {
		stats.LastSyncAt = &lastSyncAt.Time
	}

	// Get last error
	var lastError sql.NullString
	err = d.db.QueryRow(`
		SELECT error FROM data_source_sync_history
		WHERE source_id = ? AND status = 'failed'
		ORDER BY started_at DESC
		LIMIT 1
	`, sourceID).Scan(&lastError)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get last error: %w", err)
	}
	stats.LastError = lastError.String

	return stats, nil
}

// =============================================================================
// INGESTION OPERATIONS
// =============================================================================

// IngestMemory creates a memory from an ingestion item, handling deduplication
// Returns (memory_id, was_created, error) - was_created is false if duplicate was skipped
func (d *Database) IngestMemory(sourceID string, item *IngestItem) (string, bool, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check for duplicate using source_id + external_id
	var existingID string
	err := d.db.QueryRow(`
		SELECT id FROM memories WHERE source_id = ? AND external_id = ?
	`, sourceID, item.ExternalID).Scan(&existingID)

	if err == nil {
		// Duplicate found
		return existingID, false, nil
	}
	if err != sql.ErrNoRows {
		return "", false, fmt.Errorf("failed to check for duplicate: %w", err)
	}

	// Create new memory
	memoryID := uuid.New().String()
	now := time.Now()

	importance := item.Metadata.Importance
	if importance == 0 {
		importance = 5 // Default
	}

	var tagsJSON string
	if len(item.Metadata.Tags) > 0 {
		tagsJSON = tagsToJSON(item.Metadata.Tags)
	} else {
		tagsJSON = "[]"
	}

	// Use item timestamp if provided, otherwise use now
	createdAt := item.Timestamp
	if createdAt.IsZero() {
		createdAt = now
	}

	_, err = d.db.Exec(`
		INSERT INTO memories (
			id, content, source, importance, tags, session_id, domain,
			created_at, updated_at, agent_type, access_scope,
			source_id, external_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		memoryID, item.Content, item.Metadata.Author, importance, tagsJSON,
		nullString(item.Metadata.ThreadID), nullString(item.Metadata.Domain),
		createdAt, now, item.Metadata.SourceType, "session",
		sourceID, item.ExternalID,
	)

	if err != nil {
		return "", false, fmt.Errorf("failed to create memory: %w", err)
	}

	return memoryID, true, nil
}

// GetMemoriesBySource retrieves memories for a specific source
func (d *Database) GetMemoriesBySource(sourceID string, limit, offset int) ([]*Memory, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if limit <= 0 {
		limit = 50
	}

	query := `
		SELECT id, content, source, importance, tags, session_id, domain,
		       embedding, created_at, updated_at, agent_type, agent_context,
		       access_scope, slug, parent_memory_id, chunk_level, chunk_index,
		       source_id, external_id, cc_session_id
		FROM memories
		WHERE source_id = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := d.db.Query(query, sourceID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get memories by source: %w", err)
	}
	defer rows.Close()

	return scanMemoriesWithSource(rows)
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

func scanDataSources(rows *sql.Rows) ([]*DataSource, error) {
	var sources []*DataSource
	for rows.Next() {
		var ds DataSource
		var lastSyncAt sql.NullTime
		var lastSyncPosition, errorMessage sql.NullString

		err := rows.Scan(
			&ds.ID, &ds.SourceType, &ds.Name, &ds.Config, &ds.Status,
			&lastSyncAt, &lastSyncPosition, &errorMessage,
			&ds.CreatedAt, &ds.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan data source: %w", err)
		}

		if lastSyncAt.Valid {
			ds.LastSyncAt = &lastSyncAt.Time
		}
		ds.LastSyncPosition = lastSyncPosition.String
		ds.ErrorMessage = errorMessage.String

		sources = append(sources, &ds)
	}
	return sources, nil
}

func scanSyncHistory(rows *sql.Rows) ([]*DataSourceSyncHistory, error) {
	var history []*DataSourceSyncHistory
	for rows.Next() {
		var sh DataSourceSyncHistory
		var completedAt sql.NullTime
		var errorMsg sql.NullString

		err := rows.Scan(
			&sh.ID, &sh.SourceID, &sh.StartedAt, &completedAt,
			&sh.ItemsProcessed, &sh.MemoriesCreated, &sh.DuplicatesSkipped,
			&sh.Status, &errorMsg,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan sync history: %w", err)
		}

		if completedAt.Valid {
			sh.CompletedAt = &completedAt.Time
		}
		sh.Error = errorMsg.String

		history = append(history, &sh)
	}
	return history, nil
}

func scanMemoriesWithSource(rows *sql.Rows) ([]*Memory, error) {
	var memories []*Memory
	for rows.Next() {
		var m Memory
		var tagsJSON string
		var source, sessionID, domain, agentContext, slug, parentMemoryID sql.NullString
		var sourceID, externalID, ccSessionID sql.NullString
		var embedding []byte

		err := rows.Scan(
			&m.ID, &m.Content, &source, &m.Importance, &tagsJSON, &sessionID, &domain,
			&embedding, &m.CreatedAt, &m.UpdatedAt, &m.AgentType, &agentContext,
			&m.AccessScope, &slug, &parentMemoryID, &m.ChunkLevel, &m.ChunkIndex,
			&sourceID, &externalID, &ccSessionID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan memory: %w", err)
		}

		m.Source = source.String
		m.SessionID = sessionID.String
		m.Domain = domain.String
		m.AgentContext = agentContext.String
		m.Slug = slug.String
		m.ParentMemoryID = parentMemoryID.String
		m.SourceID = sourceID.String
		m.ExternalID = externalID.String
		m.CCSessionID = ccSessionID.String
		m.Embedding = embedding
		m.Tags = ParseTags(tagsJSON)

		memories = append(memories, &m)
	}
	return memories, nil
}

func nullTimePtr(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return *t
}

func tagsToJSON(tags []string) string {
	if len(tags) == 0 {
		return "[]"
	}
	// Simple JSON array construction
	result := `["`
	for i, tag := range tags {
		if i > 0 {
			result += `","`
		}
		result += tag
	}
	result += `"]`
	return result
}
