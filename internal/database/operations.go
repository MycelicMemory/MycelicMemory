package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// CreateMemory inserts a new memory into the database
// VERIFIED: Matches local-memory store_memory behavior
func (d *Database) CreateMemory(m *Memory) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Generate UUID if not provided
	if m.ID == "" {
		m.ID = uuid.New().String()
	}

	// Set timestamps
	now := time.Now()
	if m.CreatedAt.IsZero() {
		m.CreatedAt = now
	}
	m.UpdatedAt = now

	// Default values
	if m.Importance == 0 {
		m.Importance = 5
	}
	if m.AgentType == "" {
		m.AgentType = "unknown"
	}
	if m.AccessScope == "" {
		m.AccessScope = "session"
	}

	// Serialize tags to JSON
	tagsJSON := m.TagsJSON()

	// Use NULL for empty strings in nullable fields to avoid unique constraint issues
	_, err := d.db.Exec(`
		INSERT INTO memories (
			id, content, source, importance, tags, session_id, domain,
			embedding, created_at, updated_at, agent_type, agent_context,
			access_scope, slug, parent_memory_id, chunk_level, chunk_index
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		m.ID, m.Content, nullString(m.Source), m.Importance, tagsJSON,
		nullString(m.SessionID), nullString(m.Domain),
		m.Embedding, m.CreatedAt, m.UpdatedAt, m.AgentType, nullString(m.AgentContext),
		m.AccessScope, nullString(m.Slug),
		nullString(m.ParentMemoryID), m.ChunkLevel, m.ChunkIndex,
	)

	if err != nil {
		return fmt.Errorf("failed to create memory: %w", err)
	}

	return nil
}

// GetMemory retrieves a memory by ID
// VERIFIED: Matches local-memory get_memory_by_id behavior
func (d *Database) GetMemory(id string) (*Memory, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var m Memory
	var tagsJSON string
	var source, sessionID, domain, agentContext, slug, parentMemoryID sql.NullString
	var embedding []byte

	err := d.db.QueryRow(`
		SELECT id, content, source, importance, tags, session_id, domain,
		       embedding, created_at, updated_at, agent_type, agent_context,
		       access_scope, slug, parent_memory_id, chunk_level, chunk_index
		FROM memories WHERE id = ?
	`, id).Scan(
		&m.ID, &m.Content, &source, &m.Importance, &tagsJSON, &sessionID, &domain,
		&embedding, &m.CreatedAt, &m.UpdatedAt, &m.AgentType, &agentContext,
		&m.AccessScope, &slug, &parentMemoryID, &m.ChunkLevel, &m.ChunkIndex,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get memory: %w", err)
	}

	// Handle nullable fields
	m.Source = source.String
	m.SessionID = sessionID.String
	m.Domain = domain.String
	m.AgentContext = agentContext.String
	m.Slug = slug.String
	m.ParentMemoryID = parentMemoryID.String
	m.Embedding = embedding
	m.Tags = ParseTags(tagsJSON)

	return &m, nil
}

// UpdateMemory updates an existing memory
// VERIFIED: Matches local-memory update_memory behavior
func (d *Database) UpdateMemory(id string, updates *MemoryUpdate) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Build dynamic update query
	var setClauses []string
	var args []interface{}

	if updates.Content != nil {
		setClauses = append(setClauses, "content = ?")
		args = append(args, *updates.Content)
	}
	if updates.Importance != nil {
		setClauses = append(setClauses, "importance = ?")
		args = append(args, *updates.Importance)
	}
	if updates.Tags != nil {
		tagsJSON, _ := json.Marshal(updates.Tags)
		setClauses = append(setClauses, "tags = ?")
		args = append(args, string(tagsJSON))
	}
	if updates.Source != nil {
		setClauses = append(setClauses, "source = ?")
		args = append(args, *updates.Source)
	}
	if updates.Domain != nil {
		setClauses = append(setClauses, "domain = ?")
		args = append(args, *updates.Domain)
	}

	if len(setClauses) == 0 {
		return nil // No updates to apply
	}

	// Always update updated_at
	setClauses = append(setClauses, "updated_at = ?")
	args = append(args, time.Now())

	// Add WHERE clause
	args = append(args, id)

	query := fmt.Sprintf("UPDATE memories SET %s WHERE id = ?", strings.Join(setClauses, ", "))

	result, err := d.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update memory: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("memory not found: %s", id)
	}

	return nil
}

// MemoryUpdate represents optional updates to a memory
type MemoryUpdate struct {
	Content    *string
	Importance *int
	Tags       []string
	Source     *string
	Domain     *string
}

// DeleteMemory removes a memory by ID
// VERIFIED: Matches local-memory delete_memory behavior (CASCADE deletes relationships)
func (d *Database) DeleteMemory(id string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	result, err := d.db.Exec("DELETE FROM memories WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete memory: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("memory not found: %s", id)
	}

	return nil
}

// ListMemories retrieves memories with optional filters
// VERIFIED: Matches local-memory list behavior with pagination
func (d *Database) ListMemories(filters *MemoryFilters) ([]*Memory, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var whereClauses []string
	var args []interface{}

	if filters.SessionID != "" {
		whereClauses = append(whereClauses, "session_id = ?")
		args = append(args, filters.SessionID)
	}
	if filters.Domain != "" {
		whereClauses = append(whereClauses, "domain = ?")
		args = append(args, filters.Domain)
	}
	if filters.MinImportance > 0 {
		whereClauses = append(whereClauses, "importance >= ?")
		args = append(args, filters.MinImportance)
	}
	if filters.MaxImportance > 0 {
		whereClauses = append(whereClauses, "importance <= ?")
		args = append(args, filters.MaxImportance)
	}
	if filters.StartDate != nil {
		whereClauses = append(whereClauses, "created_at >= ?")
		args = append(args, *filters.StartDate)
	}
	if filters.EndDate != nil {
		whereClauses = append(whereClauses, "created_at <= ?")
		args = append(args, *filters.EndDate)
	}
	if len(filters.Tags) > 0 {
		// Match any of the tags using JSON
		for _, tag := range filters.Tags {
			whereClauses = append(whereClauses, "tags LIKE ?")
			args = append(args, "%\""+tag+"\"%")
		}
	}

	query := `
		SELECT id, content, source, importance, tags, session_id, domain,
		       embedding, created_at, updated_at, agent_type, agent_context,
		       access_scope, slug, parent_memory_id, chunk_level, chunk_index
		FROM memories
	`

	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	query += " ORDER BY created_at DESC"

	// Apply pagination
	limit := filters.Limit
	if limit <= 0 {
		limit = 50 // Default limit
	}
	query += fmt.Sprintf(" LIMIT %d", limit)

	if filters.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", filters.Offset)
	}

	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list memories: %w", err)
	}
	defer rows.Close()

	return scanMemories(rows)
}

// SearchFTS performs full-text search using FTS5
// VERIFIED: Matches local-memory keyword search behavior
func (d *Database) SearchFTS(query string, filters *SearchFilters) ([]*SearchResult, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if query == "" {
		return nil, fmt.Errorf("search query is required")
	}

	// Escape FTS5 special characters and build query
	ftsQuery := escapeFTS5Query(query)

	var whereClauses []string
	var args []interface{}

	args = append(args, ftsQuery)

	if filters.SessionID != "" {
		whereClauses = append(whereClauses, "m.session_id = ?")
		args = append(args, filters.SessionID)
	}
	if filters.Domain != "" {
		whereClauses = append(whereClauses, "m.domain = ?")
		args = append(args, filters.Domain)
	}

	// Query FTS5 table and join with memories for full data
	// The FTS5 table stores its own content for reliable sync
	sqlQuery := `
		SELECT m.id, m.content, m.source, m.importance, m.tags, m.session_id, m.domain,
		       m.embedding, m.created_at, m.updated_at, m.agent_type, m.agent_context,
		       m.access_scope, m.slug,
		       bm25(memories_fts) as relevance
		FROM memories_fts fts
		JOIN memories m ON m.id = fts.id
		WHERE memories_fts MATCH ?
	`

	if len(whereClauses) > 0 {
		sqlQuery += " AND " + strings.Join(whereClauses, " AND ")
	}

	sqlQuery += " ORDER BY relevance"

	// Apply limit
	limit := filters.Limit
	if limit <= 0 {
		limit = 10
	}
	sqlQuery += fmt.Sprintf(" LIMIT %d", limit)

	rows, err := d.db.Query(sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}
	defer rows.Close()

	var results []*SearchResult
	for rows.Next() {
		var m Memory
		var tagsJSON string
		var source, sessionID, domain, agentContext, slug sql.NullString
		var embedding []byte
		var relevance float64

		err := rows.Scan(
			&m.ID, &m.Content, &source, &m.Importance, &tagsJSON, &sessionID, &domain,
			&embedding, &m.CreatedAt, &m.UpdatedAt, &m.AgentType, &agentContext,
			&m.AccessScope, &slug, &relevance,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan search result: %w", err)
		}

		m.Source = source.String
		m.SessionID = sessionID.String
		m.Domain = domain.String
		m.AgentContext = agentContext.String
		m.Slug = slug.String
		m.Embedding = embedding
		m.Tags = ParseTags(tagsJSON)

		// Convert BM25 score to 0-1 relevance (BM25 returns negative values, lower is better)
		// Normalize: -10 (best) to 0 (worst) -> 1.0 to 0.0
		normalizedRelevance := 1.0 + (relevance / 10.0)
		if normalizedRelevance > 1.0 {
			normalizedRelevance = 1.0
		}
		if normalizedRelevance < 0.0 {
			normalizedRelevance = 0.0
		}

		results = append(results, &SearchResult{
			Memory:    &m,
			Relevance: normalizedRelevance,
		})
	}

	return results, nil
}

// escapeFTS5Query escapes special FTS5 characters and converts multi-word
// queries to use OR operator for better recall in semantic retrieval.
// FTS5's default behavior treats space-separated words as implicit AND,
// which returns 0 results when no single document contains all words.
// Using OR allows matching any keyword with BM25 ranking preferring
// documents that match more keywords.
func escapeFTS5Query(query string) string {
	// Escape double quotes first
	query = strings.ReplaceAll(query, "\"", "\"\"")

	// Check if query already contains FTS5 operators (don't modify)
	upperQuery := strings.ToUpper(query)
	if strings.Contains(upperQuery, " OR ") ||
		strings.Contains(upperQuery, " AND ") ||
		strings.Contains(upperQuery, " NOT ") ||
		strings.Contains(upperQuery, " NEAR") {
		return query
	}

	// Split into words and join with OR for better recall
	words := strings.Fields(query)
	if len(words) > 1 {
		// Filter out very short words that may cause noise
		var validWords []string
		for _, w := range words {
			if len(w) >= 2 { // Keep words with 2+ characters
				validWords = append(validWords, w)
			}
		}
		if len(validWords) > 1 {
			return strings.Join(validWords, " OR ")
		}
		if len(validWords) == 1 {
			return validWords[0]
		}
	}
	return query
}

// CreateRelationship creates a relationship between two memories
// VERIFIED: Matches local-memory create relationship behavior
func (d *Database) CreateRelationship(r *Relationship) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Validate relationship type
	if !IsValidRelationshipType(r.RelationshipType) {
		return fmt.Errorf("invalid relationship type: %s", r.RelationshipType)
	}

	// Validate strength
	if r.Strength < 0.0 || r.Strength > 1.0 {
		return fmt.Errorf("strength must be between 0.0 and 1.0")
	}

	// Generate UUID if not provided
	if r.ID == "" {
		r.ID = uuid.New().String()
	}

	if r.CreatedAt.IsZero() {
		r.CreatedAt = time.Now()
	}

	_, err := d.db.Exec(`
		INSERT INTO memory_relationships (
			id, source_memory_id, target_memory_id, relationship_type,
			strength, context, auto_generated, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`,
		r.ID, r.SourceMemoryID, r.TargetMemoryID, r.RelationshipType,
		r.Strength, r.Context, r.AutoGenerated, r.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create relationship: %w", err)
	}

	return nil
}

// GetRelationshipsBetween gets all relationships between two memories
func (d *Database) GetRelationshipsBetween(sourceID, targetID string) ([]*Relationship, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	rows, err := d.db.Query(`
		SELECT id, source_memory_id, target_memory_id, relationship_type,
		       strength, context, auto_generated, created_at
		FROM memory_relationships
		WHERE (source_memory_id = ? AND target_memory_id = ?)
		   OR (source_memory_id = ? AND target_memory_id = ?)
	`, sourceID, targetID, targetID, sourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get relationships: %w", err)
	}
	defer rows.Close()

	var relationships []*Relationship
	for rows.Next() {
		var r Relationship
		var context sql.NullString
		if err := rows.Scan(&r.ID, &r.SourceMemoryID, &r.TargetMemoryID,
			&r.RelationshipType, &r.Strength, &context, &r.AutoGenerated, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan relationship: %w", err)
		}
		if context.Valid {
			r.Context = context.String
		}
		relationships = append(relationships, &r)
	}

	return relationships, nil
}

// GetRelationshipsForMemory gets all relationships for a memory ID
func (d *Database) GetRelationshipsForMemory(memoryID string) ([]*Relationship, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	rows, err := d.db.Query(`
		SELECT id, source_memory_id, target_memory_id, relationship_type,
		       strength, context, auto_generated, created_at
		FROM memory_relationships
		WHERE source_memory_id = ? OR target_memory_id = ?
	`, memoryID, memoryID)
	if err != nil {
		return nil, fmt.Errorf("failed to get relationships: %w", err)
	}
	defer rows.Close()

	var relationships []*Relationship
	for rows.Next() {
		var r Relationship
		var context sql.NullString
		if err := rows.Scan(&r.ID, &r.SourceMemoryID, &r.TargetMemoryID,
			&r.RelationshipType, &r.Strength, &context, &r.AutoGenerated, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan relationship: %w", err)
		}
		if context.Valid {
			r.Context = context.String
		}
		relationships = append(relationships, &r)
	}

	return relationships, nil
}

// FindRelated finds memories related to a given memory
// VERIFIED: Matches local-memory find_related behavior
func (d *Database) FindRelated(memoryID string, filters *RelationshipFilters) ([]*Memory, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var whereClauses []string
	var args []interface{}

	args = append(args, memoryID, memoryID)

	if filters.Type != "" {
		whereClauses = append(whereClauses, "r.relationship_type = ?")
		args = append(args, filters.Type)
	}
	if filters.MinStrength > 0 {
		whereClauses = append(whereClauses, "r.strength >= ?")
		args = append(args, filters.MinStrength)
	}

	query := `
		SELECT DISTINCT m.id, m.content, m.source, m.importance, m.tags, m.session_id, m.domain,
		       m.embedding, m.created_at, m.updated_at, m.agent_type, m.agent_context,
		       m.access_scope, m.slug, m.parent_memory_id, m.chunk_level, m.chunk_index
		FROM memories m
		JOIN memory_relationships r ON (
			(r.source_memory_id = ? AND r.target_memory_id = m.id) OR
			(r.target_memory_id = ? AND r.source_memory_id = m.id)
		)
	`

	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	query += " ORDER BY m.importance DESC"

	limit := filters.Limit
	if limit <= 0 {
		limit = 10
	}
	query += fmt.Sprintf(" LIMIT %d", limit)

	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to find related: %w", err)
	}
	defer rows.Close()

	return scanMemories(rows)
}

// GetGraph retrieves the relationship graph starting from a memory
// VERIFIED: Matches local-memory map_graph behavior with BFS traversal
func (d *Database) GetGraph(rootID string, depth int) (*Graph, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if depth <= 0 {
		depth = 2 // Default depth
	}
	if depth > 5 {
		depth = 5 // Max depth
	}

	// BFS traversal
	visited := make(map[string]int) // memoryID -> distance
	queue := []string{rootID}
	visited[rootID] = 0

	var edges []GraphEdge
	edgeSet := make(map[string]bool) // Track unique edges

	for len(queue) > 0 {
		currentID := queue[0]
		queue = queue[1:]
		currentDist := visited[currentID]

		if currentDist >= depth {
			continue
		}

		// Get relationships for current node
		rows, err := d.db.Query(`
			SELECT id, source_memory_id, target_memory_id, relationship_type, strength
			FROM memory_relationships
			WHERE source_memory_id = ? OR target_memory_id = ?
		`, currentID, currentID)
		if err != nil {
			return nil, fmt.Errorf("failed to get relationships: %w", err)
		}

		for rows.Next() {
			var relID, sourceID, targetID, relType string
			var strength float64
			if err := rows.Scan(&relID, &sourceID, &targetID, &relType, &strength); err != nil {
				rows.Close()
				return nil, fmt.Errorf("failed to scan relationship: %w", err)
			}

			// Add edge if not already added
			edgeKey := sourceID + "-" + targetID
			if !edgeSet[edgeKey] {
				edges = append(edges, GraphEdge{
					SourceID: sourceID,
					TargetID: targetID,
					Type:     relType,
					Strength: strength,
				})
				edgeSet[edgeKey] = true
			}

			// Add connected node to queue
			otherID := targetID
			if targetID == currentID {
				otherID = sourceID
			}

			if _, seen := visited[otherID]; !seen {
				visited[otherID] = currentDist + 1
				queue = append(queue, otherID)
			}
		}
		rows.Close()
	}

	// Build nodes list
	var nodes []GraphNode
	for memID, dist := range visited {
		var content string
		var importance int
		err := d.db.QueryRow(`
			SELECT content, importance FROM memories WHERE id = ?
		`, memID).Scan(&content, &importance)
		if err != nil {
			continue // Skip if memory not found
		}

		nodes = append(nodes, GraphNode{
			ID:         memID,
			Content:    content,
			Importance: importance,
			Distance:   dist,
		})
	}

	return &Graph{
		Nodes: nodes,
		Edges: edges,
	}, nil
}

// CreateCategory creates a new category
// VERIFIED: Matches local-memory categories create behavior
func (d *Database) CreateCategory(c *Category) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if c.ID == "" {
		c.ID = uuid.New().String()
	}

	if c.CreatedAt.IsZero() {
		c.CreatedAt = time.Now()
	}

	if c.ConfidenceThreshold == 0 {
		c.ConfidenceThreshold = 0.7 // Default
	}

	_, err := d.db.Exec(`
		INSERT INTO categories (
			id, name, description, parent_category_id,
			confidence_threshold, auto_generated, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`,
		c.ID, c.Name, c.Description, nullString(c.ParentCategoryID),
		c.ConfidenceThreshold, c.AutoGenerated, c.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create category: %w", err)
	}

	return nil
}

// ListCategories retrieves all categories
// VERIFIED: Matches local-memory categories list behavior
func (d *Database) ListCategories() ([]*Category, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	rows, err := d.db.Query(`
		SELECT id, name, description, parent_category_id,
		       confidence_threshold, auto_generated, created_at
		FROM categories
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list categories: %w", err)
	}
	defer rows.Close()

	var categories []*Category
	for rows.Next() {
		var c Category
		var parentID sql.NullString
		err := rows.Scan(
			&c.ID, &c.Name, &c.Description, &parentID,
			&c.ConfidenceThreshold, &c.AutoGenerated, &c.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan category: %w", err)
		}
		c.ParentCategoryID = parentID.String
		categories = append(categories, &c)
	}

	return categories, nil
}

// CategorizeMemory assigns a memory to a category
// VERIFIED: Matches local-memory categorize behavior
func (d *Database) CategorizeMemory(memoryID, categoryID string, confidence float64, reasoning string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if confidence < 0.0 || confidence > 1.0 {
		return fmt.Errorf("confidence must be between 0.0 and 1.0")
	}

	_, err := d.db.Exec(`
		INSERT OR REPLACE INTO memory_categorizations (
			memory_id, category_id, confidence, reasoning, created_at
		) VALUES (?, ?, ?, ?, ?)
	`, memoryID, categoryID, confidence, reasoning, time.Now())

	if err != nil {
		return fmt.Errorf("failed to categorize memory: %w", err)
	}

	return nil
}

// CreateDomain creates a new domain
// VERIFIED: Matches local-memory domains create behavior
func (d *Database) CreateDomain(dom *Domain) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if dom.ID == "" {
		dom.ID = uuid.New().String()
	}

	now := time.Now()
	if dom.CreatedAt.IsZero() {
		dom.CreatedAt = now
	}
	dom.UpdatedAt = now

	_, err := d.db.Exec(`
		INSERT INTO domains (id, name, description, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`, dom.ID, dom.Name, dom.Description, dom.CreatedAt, dom.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create domain: %w", err)
	}

	return nil
}

// ListDomains retrieves all domains
// VERIFIED: Matches local-memory domains list behavior
func (d *Database) ListDomains() ([]*Domain, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	rows, err := d.db.Query(`
		SELECT id, name, description, created_at, updated_at
		FROM domains
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list domains: %w", err)
	}
	defer rows.Close()

	var domains []*Domain
	for rows.Next() {
		var dom Domain
		var description sql.NullString
		err := rows.Scan(&dom.ID, &dom.Name, &description, &dom.CreatedAt, &dom.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan domain: %w", err)
		}
		dom.Description = description.String
		domains = append(domains, &dom)
	}

	return domains, nil
}

// ListSessions retrieves all sessions
// VERIFIED: Matches local-memory sessions list behavior
func (d *Database) ListSessions() ([]*AgentSession, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	rows, err := d.db.Query(`
		SELECT session_id, agent_type, agent_context, created_at,
		       last_accessed, is_active, metadata
		FROM agent_sessions
		ORDER BY last_accessed DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*AgentSession
	for rows.Next() {
		var s AgentSession
		var agentContext sql.NullString
		err := rows.Scan(
			&s.SessionID, &s.AgentType, &agentContext, &s.CreatedAt,
			&s.LastAccessed, &s.IsActive, &s.Metadata,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}
		s.AgentContext = agentContext.String
		sessions = append(sessions, &s)
	}

	return sessions, nil
}

// GetMemoryCountBySession returns the count of memories for a session
func (d *Database) GetMemoryCountBySession(sessionID string) (int, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var count int
	err := d.db.QueryRow(`SELECT COUNT(*) FROM memories WHERE session_id = ?`, sessionID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count memories: %w", err)
	}
	return count, nil
}

// EnsureSession creates or updates a session to track it
// VERIFIED: Matches local-memory session auto-tracking behavior
func (d *Database) EnsureSession(sessionID string, agentType string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if agentType == "" {
		agentType = "unknown"
	}

	now := time.Now()

	// Try to update existing session
	result, err := d.db.Exec(`
		UPDATE agent_sessions
		SET last_accessed = ?, is_active = 1
		WHERE session_id = ?
	`, now, sessionID)
	if err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		// Session doesn't exist, create it
		_, err = d.db.Exec(`
			INSERT INTO agent_sessions (session_id, agent_type, created_at, last_accessed, is_active, metadata)
			VALUES (?, ?, ?, ?, 1, '{}')
		`, sessionID, agentType, now, now)
		if err != nil {
			return fmt.Errorf("failed to create session: %w", err)
		}
	}

	return nil
}

// DomainStats contains statistics for a domain
type DomainStats struct {
	MemoryCount       int
	AverageImportance float64
}

// GetDomainStats retrieves statistics for a specific domain
func (d *Database) GetDomainStats(domainName string) (*DomainStats, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var count int
	var avgImportance float64

	err := d.db.QueryRow(`
		SELECT COUNT(*), COALESCE(AVG(importance), 0)
		FROM memories
		WHERE domain = ?
	`, domainName).Scan(&count, &avgImportance)
	if err != nil {
		return nil, fmt.Errorf("failed to get domain stats: %w", err)
	}

	return &DomainStats{
		MemoryCount:       count,
		AverageImportance: avgImportance,
	}, nil
}

// RecordMetric records a performance metric
// VERIFIED: Matches local-memory performance tracking
func (d *Database) RecordMetric(operationType string, executionTimeMs int, memoryCount int) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, err := d.db.Exec(`
		INSERT INTO performance_metrics (operation_type, execution_time_ms, memory_count, timestamp)
		VALUES (?, ?, ?, ?)
	`, operationType, executionTimeMs, memoryCount, time.Now())

	return err
}

// GetChildChunks retrieves all chunks belonging to a parent memory
func (d *Database) GetChildChunks(parentID string) ([]*Memory, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	rows, err := d.db.Query(`
		SELECT id, content, source, importance, tags, session_id, domain,
		       embedding, created_at, updated_at, agent_type, agent_context,
		       access_scope, slug, parent_memory_id, chunk_level, chunk_index
		FROM memories
		WHERE parent_memory_id = ?
		ORDER BY chunk_index ASC
	`, parentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get child chunks: %w", err)
	}
	defer rows.Close()

	return scanMemories(rows)
}

// GetRootMemories retrieves only root memories (not chunks)
func (d *Database) GetRootMemories(filters *MemoryFilters) ([]*Memory, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var whereClauses []string
	var args []interface{}

	// Always filter to root memories only (chunk_level = 0 or parent_memory_id IS NULL)
	whereClauses = append(whereClauses, "(chunk_level = 0 OR parent_memory_id IS NULL)")

	if filters.SessionID != "" {
		whereClauses = append(whereClauses, "session_id = ?")
		args = append(args, filters.SessionID)
	}
	if filters.Domain != "" {
		whereClauses = append(whereClauses, "domain = ?")
		args = append(args, filters.Domain)
	}

	query := `
		SELECT id, content, source, importance, tags, session_id, domain,
		       embedding, created_at, updated_at, agent_type, agent_context,
		       access_scope, slug, parent_memory_id, chunk_level, chunk_index
		FROM memories
		WHERE ` + strings.Join(whereClauses, " AND ") + `
		ORDER BY created_at DESC
	`

	limit := filters.Limit
	if limit <= 0 {
		limit = 50
	}
	query += fmt.Sprintf(" LIMIT %d", limit)

	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get root memories: %w", err)
	}
	defer rows.Close()

	return scanMemories(rows)
}

// Helper functions

func scanMemories(rows *sql.Rows) ([]*Memory, error) {
	var memories []*Memory
	for rows.Next() {
		var m Memory
		var tagsJSON string
		var source, sessionID, domain, agentContext, slug, parentMemoryID sql.NullString
		var embedding []byte

		err := rows.Scan(
			&m.ID, &m.Content, &source, &m.Importance, &tagsJSON, &sessionID, &domain,
			&embedding, &m.CreatedAt, &m.UpdatedAt, &m.AgentType, &agentContext,
			&m.AccessScope, &slug, &parentMemoryID, &m.ChunkLevel, &m.ChunkIndex,
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
		m.Embedding = embedding
		m.Tags = ParseTags(tagsJSON)

		memories = append(memories, &m)
	}
	return memories, nil
}

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

// =============================================================================
// BENCHMARK OPERATIONS
// =============================================================================

// CreateBenchmarkRun creates a new benchmark run record
func (d *Database) CreateBenchmarkRun(run *BenchmarkRun) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if run.ID == "" {
		run.ID = uuid.New().String()
	}

	if run.StartedAt.IsZero() {
		run.StartedAt = time.Now()
	}

	if run.Status == "" {
		run.Status = "pending"
	}

	if run.CreatedBy == "" {
		run.CreatedBy = "manual"
	}

	_, err := d.db.Exec(`
		INSERT INTO benchmark_runs (
			id, started_at, completed_at, status,
			git_commit_hash, git_branch, git_dirty,
			config_snapshot, benchmark_type,
			overall_score, overall_f1, overall_bleu1,
			total_questions, total_correct, duration_seconds,
			error_message, baseline_run_id, improvement_from_baseline,
			is_best_run, autonomous_loop_id, iteration_number,
			change_description, created_by, notes
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		run.ID, run.StartedAt, run.CompletedAt, run.Status,
		run.GitCommitHash, nullString(run.GitBranch), run.GitDirty,
		run.ConfigSnapshot, run.BenchmarkType,
		run.OverallScore, run.OverallF1, run.OverallBleu1,
		run.TotalQuestions, run.TotalCorrect, run.DurationSeconds,
		nullString(run.ErrorMessage), nullString(run.BaselineRunID), run.ImprovementFromBaseline,
		run.IsBestRun, nullString(run.AutonomousLoopID), run.IterationNumber,
		nullString(run.ChangeDescription), run.CreatedBy, nullString(run.Notes),
	)

	if err != nil {
		return fmt.Errorf("failed to create benchmark run: %w", err)
	}

	return nil
}

// UpdateBenchmarkRun updates an existing benchmark run
func (d *Database) UpdateBenchmarkRun(run *BenchmarkRun) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, err := d.db.Exec(`
		UPDATE benchmark_runs SET
			completed_at = ?, status = ?,
			overall_score = ?, overall_f1 = ?, overall_bleu1 = ?,
			total_questions = ?, total_correct = ?, duration_seconds = ?,
			error_message = ?, improvement_from_baseline = ?, is_best_run = ?,
			notes = ?
		WHERE id = ?
	`,
		run.CompletedAt, run.Status,
		run.OverallScore, run.OverallF1, run.OverallBleu1,
		run.TotalQuestions, run.TotalCorrect, run.DurationSeconds,
		nullString(run.ErrorMessage), run.ImprovementFromBaseline, run.IsBestRun,
		nullString(run.Notes), run.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update benchmark run: %w", err)
	}

	return nil
}

// GetBenchmarkRun retrieves a benchmark run by ID
func (d *Database) GetBenchmarkRun(id string) (*BenchmarkRun, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var run BenchmarkRun
	var completedAt sql.NullTime
	var gitBranch, errorMessage, baselineRunID, loopID, changeDesc, notes sql.NullString
	var overallScore, overallF1, overallBleu1, durationSeconds, improvement sql.NullFloat64
	var totalQuestions, totalCorrect sql.NullInt64

	err := d.db.QueryRow(`
		SELECT id, started_at, completed_at, status,
		       git_commit_hash, git_branch, git_dirty,
		       config_snapshot, benchmark_type,
		       overall_score, overall_f1, overall_bleu1,
		       total_questions, total_correct, duration_seconds,
		       error_message, baseline_run_id, improvement_from_baseline,
		       is_best_run, autonomous_loop_id, iteration_number,
		       change_description, created_by, notes
		FROM benchmark_runs WHERE id = ?
	`, id).Scan(
		&run.ID, &run.StartedAt, &completedAt, &run.Status,
		&run.GitCommitHash, &gitBranch, &run.GitDirty,
		&run.ConfigSnapshot, &run.BenchmarkType,
		&overallScore, &overallF1, &overallBleu1,
		&totalQuestions, &totalCorrect, &durationSeconds,
		&errorMessage, &baselineRunID, &improvement,
		&run.IsBestRun, &loopID, &run.IterationNumber,
		&changeDesc, &run.CreatedBy, &notes,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get benchmark run: %w", err)
	}

	// Handle nullable fields
	if completedAt.Valid {
		run.CompletedAt = &completedAt.Time
	}
	run.GitBranch = gitBranch.String
	run.ErrorMessage = errorMessage.String
	run.BaselineRunID = baselineRunID.String
	run.AutonomousLoopID = loopID.String
	run.ChangeDescription = changeDesc.String
	run.Notes = notes.String

	if overallScore.Valid {
		run.OverallScore = &overallScore.Float64
	}
	if overallF1.Valid {
		run.OverallF1 = &overallF1.Float64
	}
	if overallBleu1.Valid {
		run.OverallBleu1 = &overallBleu1.Float64
	}
	if durationSeconds.Valid {
		run.DurationSeconds = &durationSeconds.Float64
	}
	if improvement.Valid {
		run.ImprovementFromBaseline = &improvement.Float64
	}
	if totalQuestions.Valid {
		q := int(totalQuestions.Int64)
		run.TotalQuestions = &q
	}
	if totalCorrect.Valid {
		c := int(totalCorrect.Int64)
		run.TotalCorrect = &c
	}

	return &run, nil
}

// ListBenchmarkRuns retrieves benchmark runs with optional filters
func (d *Database) ListBenchmarkRuns(filters *BenchmarkRunFilters) ([]*BenchmarkRun, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var whereClauses []string
	var args []interface{}

	if filters.Status != "" && filters.Status != "all" {
		whereClauses = append(whereClauses, "status = ?")
		args = append(args, filters.Status)
	}
	if filters.BenchmarkType != "" {
		whereClauses = append(whereClauses, "benchmark_type = ?")
		args = append(args, filters.BenchmarkType)
	}
	if filters.GitCommit != "" {
		whereClauses = append(whereClauses, "git_commit_hash LIKE ?")
		args = append(args, filters.GitCommit+"%")
	}
	if filters.LoopID != "" {
		whereClauses = append(whereClauses, "autonomous_loop_id = ?")
		args = append(args, filters.LoopID)
	}
	if filters.Since != nil {
		whereClauses = append(whereClauses, "started_at >= ?")
		args = append(args, *filters.Since)
	}
	if filters.Until != nil {
		whereClauses = append(whereClauses, "started_at <= ?")
		args = append(args, *filters.Until)
	}

	query := `
		SELECT id, started_at, completed_at, status,
		       git_commit_hash, git_branch, git_dirty,
		       config_snapshot, benchmark_type,
		       overall_score, overall_f1, overall_bleu1,
		       total_questions, total_correct, duration_seconds,
		       error_message, baseline_run_id, improvement_from_baseline,
		       is_best_run, autonomous_loop_id, iteration_number,
		       change_description, created_by, notes
		FROM benchmark_runs
	`

	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	query += " ORDER BY started_at DESC"

	limit := filters.Limit
	if limit <= 0 {
		limit = 20
	}
	query += fmt.Sprintf(" LIMIT %d", limit)

	if filters.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", filters.Offset)
	}

	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list benchmark runs: %w", err)
	}
	defer rows.Close()

	var runs []*BenchmarkRun
	for rows.Next() {
		var run BenchmarkRun
		var completedAt sql.NullTime
		var gitBranch, errorMessage, baselineRunID, loopID, changeDesc, notes sql.NullString
		var overallScore, overallF1, overallBleu1, durationSeconds, improvement sql.NullFloat64
		var totalQuestions, totalCorrect sql.NullInt64

		err := rows.Scan(
			&run.ID, &run.StartedAt, &completedAt, &run.Status,
			&run.GitCommitHash, &gitBranch, &run.GitDirty,
			&run.ConfigSnapshot, &run.BenchmarkType,
			&overallScore, &overallF1, &overallBleu1,
			&totalQuestions, &totalCorrect, &durationSeconds,
			&errorMessage, &baselineRunID, &improvement,
			&run.IsBestRun, &loopID, &run.IterationNumber,
			&changeDesc, &run.CreatedBy, &notes,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan benchmark run: %w", err)
		}

		if completedAt.Valid {
			run.CompletedAt = &completedAt.Time
		}
		run.GitBranch = gitBranch.String
		run.ErrorMessage = errorMessage.String
		run.BaselineRunID = baselineRunID.String
		run.AutonomousLoopID = loopID.String
		run.ChangeDescription = changeDesc.String
		run.Notes = notes.String

		if overallScore.Valid {
			run.OverallScore = &overallScore.Float64
		}
		if overallF1.Valid {
			run.OverallF1 = &overallF1.Float64
		}
		if overallBleu1.Valid {
			run.OverallBleu1 = &overallBleu1.Float64
		}
		if durationSeconds.Valid {
			run.DurationSeconds = &durationSeconds.Float64
		}
		if improvement.Valid {
			run.ImprovementFromBaseline = &improvement.Float64
		}
		if totalQuestions.Valid {
			q := int(totalQuestions.Int64)
			run.TotalQuestions = &q
		}
		if totalCorrect.Valid {
			c := int(totalCorrect.Int64)
			run.TotalCorrect = &c
		}

		runs = append(runs, &run)
	}

	return runs, nil
}

// GetBestBenchmarkRun returns the best performing run
func (d *Database) GetBestBenchmarkRun(benchmarkType string) (*BenchmarkRun, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var id string
	err := d.db.QueryRow(`
		SELECT id FROM benchmark_runs
		WHERE benchmark_type = ? AND status = 'completed' AND is_best_run = 1
		ORDER BY overall_score DESC
		LIMIT 1
	`, benchmarkType).Scan(&id)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get best run: %w", err)
	}

	return d.GetBenchmarkRun(id)
}

// CreateBenchmarkCategoryResult creates a category result record
func (d *Database) CreateBenchmarkCategoryResult(result *BenchmarkCategoryResult) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if result.ID == "" {
		result.ID = uuid.New().String()
	}

	_, err := d.db.Exec(`
		INSERT INTO benchmark_results_by_category (
			id, run_id, category,
			llm_judge_accuracy, f1_score, bleu1_score,
			total_questions, correct_count,
			previous_best_accuracy, improvement
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		result.ID, result.RunID, result.Category,
		result.LLMJudgeAccuracy, result.F1Score, result.Bleu1Score,
		result.TotalQuestions, result.CorrectCount,
		result.PreviousBestAccuracy, result.Improvement,
	)

	if err != nil {
		return fmt.Errorf("failed to create category result: %w", err)
	}

	return nil
}

// GetBenchmarkCategoryResults gets all category results for a run
func (d *Database) GetBenchmarkCategoryResults(runID string) ([]*BenchmarkCategoryResult, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	rows, err := d.db.Query(`
		SELECT id, run_id, category,
		       llm_judge_accuracy, f1_score, bleu1_score,
		       total_questions, correct_count,
		       previous_best_accuracy, improvement
		FROM benchmark_results_by_category
		WHERE run_id = ?
		ORDER BY category
	`, runID)
	if err != nil {
		return nil, fmt.Errorf("failed to get category results: %w", err)
	}
	defer rows.Close()

	var results []*BenchmarkCategoryResult
	for rows.Next() {
		var r BenchmarkCategoryResult
		var accuracy, f1, bleu1, prevBest, improvement sql.NullFloat64
		var total, correct sql.NullInt64

		err := rows.Scan(
			&r.ID, &r.RunID, &r.Category,
			&accuracy, &f1, &bleu1,
			&total, &correct,
			&prevBest, &improvement,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan category result: %w", err)
		}

		if accuracy.Valid {
			r.LLMJudgeAccuracy = &accuracy.Float64
		}
		if f1.Valid {
			r.F1Score = &f1.Float64
		}
		if bleu1.Valid {
			r.Bleu1Score = &bleu1.Float64
		}
		if total.Valid {
			t := int(total.Int64)
			r.TotalQuestions = &t
		}
		if correct.Valid {
			c := int(correct.Int64)
			r.CorrectCount = &c
		}
		if prevBest.Valid {
			r.PreviousBestAccuracy = &prevBest.Float64
		}
		if improvement.Valid {
			r.Improvement = &improvement.Float64
		}

		results = append(results, &r)
	}

	return results, nil
}

// CreateBenchmarkQuestionResult creates a question result record
func (d *Database) CreateBenchmarkQuestionResult(result *BenchmarkQuestionResult) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if result.ID == "" {
		result.ID = uuid.New().String()
	}

	_, err := d.db.Exec(`
		INSERT INTO benchmark_question_results (
			id, run_id, question_id, category, question_text,
			gold_answer, generated_answer,
			llm_judge_label, f1_score, bleu1_score,
			context_length, memories_used, retrieval_time_ms, generation_time_ms,
			changed_from_previous, previous_was_correct
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		result.ID, result.RunID, result.QuestionID, result.Category, result.QuestionText,
		result.GoldAnswer, nullString(result.GeneratedAnswer),
		result.LLMJudgeLabel, result.F1Score, result.Bleu1Score,
		result.ContextLength, result.MemoriesUsed, result.RetrievalTimeMs, result.GenerationTimeMs,
		result.ChangedFromPrevious, result.PreviousWasCorrect,
	)

	if err != nil {
		return fmt.Errorf("failed to create question result: %w", err)
	}

	return nil
}

// GetBenchmarkQuestionResults retrieves all question results for a run
func (d *Database) GetBenchmarkQuestionResults(runID string) ([]*BenchmarkQuestionResult, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	rows, err := d.db.Query(`
		SELECT id, run_id, question_id, category, question_text,
		       gold_answer, generated_answer,
		       llm_judge_label, f1_score, bleu1_score,
		       context_length, memories_used, retrieval_time_ms, generation_time_ms,
		       changed_from_previous, previous_was_correct
		FROM benchmark_question_results
		WHERE run_id = ?
		ORDER BY category, question_id
	`, runID)

	if err != nil {
		return nil, fmt.Errorf("failed to query question results: %w", err)
	}
	defer rows.Close()

	var results []*BenchmarkQuestionResult
	for rows.Next() {
		var r BenchmarkQuestionResult
		var generatedAnswer sql.NullString
		var llmJudgeLabel, contextLength, memoriesUsed, retrievalTimeMs, generationTimeMs sql.NullInt64
		var f1Score, bleu1Score sql.NullFloat64
		var changedFromPrevious, previousWasCorrect sql.NullBool

		err := rows.Scan(
			&r.ID, &r.RunID, &r.QuestionID, &r.Category, &r.QuestionText,
			&r.GoldAnswer, &generatedAnswer,
			&llmJudgeLabel, &f1Score, &bleu1Score,
			&contextLength, &memoriesUsed, &retrievalTimeMs, &generationTimeMs,
			&changedFromPrevious, &previousWasCorrect,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan question result: %w", err)
		}

		r.GeneratedAnswer = generatedAnswer.String
		if llmJudgeLabel.Valid {
			v := int(llmJudgeLabel.Int64)
			r.LLMJudgeLabel = &v
		}
		if f1Score.Valid {
			r.F1Score = &f1Score.Float64
		}
		if bleu1Score.Valid {
			r.Bleu1Score = &bleu1Score.Float64
		}
		if contextLength.Valid {
			v := int(contextLength.Int64)
			r.ContextLength = &v
		}
		if memoriesUsed.Valid {
			v := int(memoriesUsed.Int64)
			r.MemoriesUsed = &v
		}
		if retrievalTimeMs.Valid {
			v := int(retrievalTimeMs.Int64)
			r.RetrievalTimeMs = &v
		}
		if generationTimeMs.Valid {
			v := int(generationTimeMs.Int64)
			r.GenerationTimeMs = &v
		}
		if changedFromPrevious.Valid {
			r.ChangedFromPrevious = &changedFromPrevious.Bool
		}
		if previousWasCorrect.Valid {
			r.PreviousWasCorrect = &previousWasCorrect.Bool
		}

		results = append(results, &r)
	}

	return results, nil
}

// CreateAutonomousLoop creates a new autonomous loop record
func (d *Database) CreateAutonomousLoop(loop *AutonomousLoop) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if loop.ID == "" {
		loop.ID = uuid.New().String()
	}

	if loop.StartedAt.IsZero() {
		loop.StartedAt = time.Now()
	}

	if loop.Status == "" {
		loop.Status = "running"
	}

	_, err := d.db.Exec(`
		INSERT INTO autonomous_loops (
			id, started_at, completed_at, status,
			max_iterations, min_improvement_threshold, convergence_threshold,
			total_iterations, baseline_score, final_score, best_score, best_run_id,
			stop_reason, changes_attempted, changes_accepted, changes_rejected
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		loop.ID, loop.StartedAt, loop.CompletedAt, loop.Status,
		loop.MaxIterations, loop.MinImprovementThreshold, loop.ConvergenceThreshold,
		loop.TotalIterations, loop.BaselineScore, loop.FinalScore, loop.BestScore, nullString(loop.BestRunID),
		nullString(loop.StopReason), nullString(loop.ChangesAttempted), nullString(loop.ChangesAccepted), nullString(loop.ChangesRejected),
	)

	if err != nil {
		return fmt.Errorf("failed to create autonomous loop: %w", err)
	}

	return nil
}

// UpdateAutonomousLoop updates an existing autonomous loop
func (d *Database) UpdateAutonomousLoop(loop *AutonomousLoop) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	_, err := d.db.Exec(`
		UPDATE autonomous_loops SET
			completed_at = ?, status = ?,
			total_iterations = ?, final_score = ?, best_score = ?, best_run_id = ?,
			stop_reason = ?, changes_attempted = ?, changes_accepted = ?, changes_rejected = ?
		WHERE id = ?
	`,
		loop.CompletedAt, loop.Status,
		loop.TotalIterations, loop.FinalScore, loop.BestScore, nullString(loop.BestRunID),
		nullString(loop.StopReason), nullString(loop.ChangesAttempted), nullString(loop.ChangesAccepted), nullString(loop.ChangesRejected),
		loop.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update autonomous loop: %w", err)
	}

	return nil
}

// GetAutonomousLoop retrieves an autonomous loop by ID
func (d *Database) GetAutonomousLoop(id string) (*AutonomousLoop, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var loop AutonomousLoop
	var completedAt sql.NullTime
	var baseline, final, best sql.NullFloat64
	var bestRunID, stopReason, attempted, accepted, rejected sql.NullString

	err := d.db.QueryRow(`
		SELECT id, started_at, completed_at, status,
		       max_iterations, min_improvement_threshold, convergence_threshold,
		       total_iterations, baseline_score, final_score, best_score, best_run_id,
		       stop_reason, changes_attempted, changes_accepted, changes_rejected
		FROM autonomous_loops WHERE id = ?
	`, id).Scan(
		&loop.ID, &loop.StartedAt, &completedAt, &loop.Status,
		&loop.MaxIterations, &loop.MinImprovementThreshold, &loop.ConvergenceThreshold,
		&loop.TotalIterations, &baseline, &final, &best, &bestRunID,
		&stopReason, &attempted, &accepted, &rejected,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get autonomous loop: %w", err)
	}

	if completedAt.Valid {
		loop.CompletedAt = &completedAt.Time
	}
	if baseline.Valid {
		loop.BaselineScore = &baseline.Float64
	}
	if final.Valid {
		loop.FinalScore = &final.Float64
	}
	if best.Valid {
		loop.BestScore = &best.Float64
	}
	loop.BestRunID = bestRunID.String
	loop.StopReason = stopReason.String
	loop.ChangesAttempted = attempted.String
	loop.ChangesAccepted = accepted.String
	loop.ChangesRejected = rejected.String

	return &loop, nil
}
