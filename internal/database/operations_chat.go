package database

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// CONVERSATION OPERATIONS
// =============================================================================

// CreateCCSession creates a new conversation record
func (d *Database) CreateCCSession(s *Conversation) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if s.ID == "" {
		s.ID = uuid.New().String()
	}

	now := time.Now()
	if s.CreatedAt.IsZero() {
		s.CreatedAt = now
	}
	s.UpdatedAt = now

	_, err := d.db.Exec(`
		INSERT INTO conversations (
			id, session_id, project_path, project_hash, model, title, first_prompt,
			summary, created_at, updated_at, last_activity,
			message_count, user_message_count, assistant_message_count, tool_call_count,
			source_id, file_path, last_sync_position, summary_memory_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		s.ID, s.SessionID, s.ProjectPath, s.ProjectHash,
		nullString(s.Model), nullString(s.Title), nullString(s.FirstPrompt),
		nullString(s.Summary), s.CreatedAt, s.UpdatedAt, nullTimePtr(s.LastActivity),
		s.MessageCount, s.UserMessageCount, s.AssistantMessageCount, s.ToolCallCount,
		nullString(s.SourceID), nullString(s.FilePath), nullString(s.LastSyncPosition),
		nullString(s.SummaryMemoryID),
	)

	if err != nil {
		return fmt.Errorf("failed to create conversation: %w", err)
	}

	return nil
}

// GetCCSession retrieves a conversation by internal ID
func (d *Database) GetCCSession(id string) (*Conversation, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return d.getConversationByQuery("SELECT id, session_id, project_path, project_hash, model, title, first_prompt, summary, created_at, updated_at, last_activity, message_count, user_message_count, assistant_message_count, tool_call_count, source_id, file_path, last_sync_position, summary_memory_id FROM conversations WHERE id = ?", id)
}

// GetCCSessionBySessionID retrieves a conversation by project hash + session ID (dedup lookup)
func (d *Database) GetCCSessionBySessionID(projectHash, sessionID string) (*Conversation, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return d.getConversationByQuery("SELECT id, session_id, project_path, project_hash, model, title, first_prompt, summary, created_at, updated_at, last_activity, message_count, user_message_count, assistant_message_count, tool_call_count, source_id, file_path, last_sync_position, summary_memory_id FROM conversations WHERE project_hash = ? AND session_id = ?", projectHash, sessionID)
}

func (d *Database) getConversationByQuery(query string, args ...interface{}) (*Conversation, error) {
	var s Conversation
	var model, title, firstPrompt, summary, sourceID, filePath, lastSyncPos, summaryMemID sql.NullString
	var lastActivity sql.NullTime

	err := d.db.QueryRow(query, args...).Scan(
		&s.ID, &s.SessionID, &s.ProjectPath, &s.ProjectHash,
		&model, &title, &firstPrompt, &summary,
		&s.CreatedAt, &s.UpdatedAt, &lastActivity,
		&s.MessageCount, &s.UserMessageCount, &s.AssistantMessageCount, &s.ToolCallCount,
		&sourceID, &filePath, &lastSyncPos, &summaryMemID,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}

	s.Model = model.String
	s.Title = title.String
	s.FirstPrompt = firstPrompt.String
	s.Summary = summary.String
	s.SourceID = sourceID.String
	s.FilePath = filePath.String
	s.LastSyncPosition = lastSyncPos.String
	s.SummaryMemoryID = summaryMemID.String
	if lastActivity.Valid {
		s.LastActivity = &lastActivity.Time
	}

	return &s, nil
}

// ListCCSessions retrieves conversations with optional filters
func (d *Database) ListCCSessions(filters *ConversationFilters) ([]*Conversation, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var whereClauses []string
	var args []interface{}

	if filters.ProjectPath != "" {
		whereClauses = append(whereClauses, "project_path = ?")
		args = append(args, filters.ProjectPath)
	}
	if filters.MinMessages > 0 {
		whereClauses = append(whereClauses, "message_count >= ?")
		args = append(args, filters.MinMessages)
	}

	query := `
		SELECT id, session_id, project_path, project_hash, model, title, first_prompt,
		       summary, created_at, updated_at, last_activity,
		       message_count, user_message_count, assistant_message_count, tool_call_count,
		       source_id, file_path, last_sync_position, summary_memory_id
		FROM conversations
	`

	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	query += " ORDER BY created_at DESC"

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
		return nil, fmt.Errorf("failed to list conversations: %w", err)
	}
	defer rows.Close()

	return scanConversations(rows)
}

// UpdateCCSession updates an existing conversation
func (d *Database) UpdateCCSession(id string, updates *ConversationUpdate) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	var setClauses []string
	var args []interface{}

	if updates.Title != nil {
		setClauses = append(setClauses, "title = ?")
		args = append(args, *updates.Title)
	}
	if updates.Summary != nil {
		setClauses = append(setClauses, "summary = ?")
		args = append(args, *updates.Summary)
	}
	if updates.MessageCount != nil {
		setClauses = append(setClauses, "message_count = ?")
		args = append(args, *updates.MessageCount)
	}
	if updates.UserMsgCount != nil {
		setClauses = append(setClauses, "user_message_count = ?")
		args = append(args, *updates.UserMsgCount)
	}
	if updates.AssistantMsgCount != nil {
		setClauses = append(setClauses, "assistant_message_count = ?")
		args = append(args, *updates.AssistantMsgCount)
	}
	if updates.ToolCallCount != nil {
		setClauses = append(setClauses, "tool_call_count = ?")
		args = append(args, *updates.ToolCallCount)
	}
	if updates.LastSyncPosition != nil {
		setClauses = append(setClauses, "last_sync_position = ?")
		args = append(args, *updates.LastSyncPosition)
	}
	if updates.SummaryMemoryID != nil {
		setClauses = append(setClauses, "summary_memory_id = ?")
		args = append(args, *updates.SummaryMemoryID)
	}

	if len(setClauses) == 0 {
		return nil
	}

	setClauses = append(setClauses, "updated_at = ?")
	args = append(args, time.Now())
	args = append(args, id)

	query := fmt.Sprintf("UPDATE conversations SET %s WHERE id = ?", strings.Join(setClauses, ", "))

	result, err := d.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update conversation: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("conversation not found: %s", id)
	}

	return nil
}

// DeleteCCSession removes a conversation by ID (CASCADE deletes messages and actions)
func (d *Database) DeleteCCSession(id string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	result, err := d.db.Exec("DELETE FROM conversations WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete conversation: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("conversation not found: %s", id)
	}

	return nil
}

// =============================================================================
// MESSAGE OPERATIONS
// =============================================================================

// CreateCCMessage creates a new message in a conversation
func (d *Database) CreateCCMessage(m *ConversationMessage) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if m.ID == "" {
		m.ID = uuid.New().String()
	}

	_, err := d.db.Exec(`
		INSERT INTO messages (
			id, session_id, role, content, timestamp, sequence_index, has_tool_use, token_count
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`,
		m.ID, m.SessionID, m.Role, m.Content,
		nullTimePtr(m.Timestamp), m.SequenceIndex, m.HasToolUse, m.TokenCount,
	)

	if err != nil {
		return fmt.Errorf("failed to create message: %w", err)
	}

	return nil
}

// GetCCMessages retrieves messages for a conversation ordered by sequence index
func (d *Database) GetCCMessages(sessionID string, limit, offset int) ([]*ConversationMessage, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if limit <= 0 {
		limit = 100
	}

	rows, err := d.db.Query(`
		SELECT id, session_id, role, content, timestamp, sequence_index, has_tool_use, token_count
		FROM messages
		WHERE session_id = ?
		ORDER BY sequence_index ASC
		LIMIT ? OFFSET ?
	`, sessionID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}
	defer rows.Close()

	return scanConversationMessages(rows)
}

// SearchCCMessages searches messages by content across conversations
func (d *Database) SearchCCMessages(query string, projectPath string, limit int) ([]*ConversationMessage, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if limit <= 0 {
		limit = 20
	}

	var args []interface{}
	searchPattern := "%" + query + "%"

	sqlQuery := `
		SELECT m.id, m.session_id, m.role, m.content, m.timestamp, m.sequence_index, m.has_tool_use, m.token_count
		FROM messages m
		JOIN conversations s ON s.id = m.session_id
		WHERE m.content LIKE ?
	`
	args = append(args, searchPattern)

	if projectPath != "" {
		sqlQuery += " AND s.project_path = ?"
		args = append(args, projectPath)
	}

	sqlQuery += " ORDER BY m.timestamp DESC LIMIT ?"
	args = append(args, limit)

	rows, err := d.db.Query(sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search messages: %w", err)
	}
	defer rows.Close()

	return scanConversationMessages(rows)
}

// =============================================================================
// ACTION OPERATIONS
// =============================================================================

// CreateCCToolCall creates a new action record
func (d *Database) CreateCCToolCall(tc *ConversationAction) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if tc.ID == "" {
		tc.ID = uuid.New().String()
	}

	_, err := d.db.Exec(`
		INSERT INTO actions (
			id, session_id, message_id, tool_name, input_json, result_text,
			success, filepath, operation, timestamp
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		tc.ID, tc.SessionID, nullString(tc.MessageID), tc.ToolName,
		nullString(tc.InputJSON), nullString(tc.ResultText),
		tc.Success, nullString(tc.FilePath), nullString(tc.Operation),
		nullTimePtr(tc.Timestamp),
	)

	if err != nil {
		return fmt.Errorf("failed to create action: %w", err)
	}

	return nil
}

// GetCCToolCalls retrieves actions for a conversation
func (d *Database) GetCCToolCalls(sessionID string) ([]*ConversationAction, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	rows, err := d.db.Query(`
		SELECT id, session_id, message_id, tool_name, input_json, result_text,
		       success, filepath, operation, timestamp
		FROM actions
		WHERE session_id = ?
		ORDER BY timestamp ASC
	`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get actions: %w", err)
	}
	defer rows.Close()

	return scanConversationActions(rows)
}

// GetFileOperations retrieves file-touching actions for a conversation
func (d *Database) GetFileOperations(sessionID string) ([]*ConversationAction, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	rows, err := d.db.Query(`
		SELECT id, session_id, message_id, tool_name, input_json, result_text,
		       success, filepath, operation, timestamp
		FROM actions
		WHERE session_id = ? AND filepath IS NOT NULL AND filepath != ''
		ORDER BY timestamp ASC
	`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get file operations: %w", err)
	}
	defer rows.Close()

	return scanConversationActions(rows)
}

// =============================================================================
// CROSS-LAYER OPERATIONS
// =============================================================================

// GetSessionMemories retrieves memories linked to a conversation
func (d *Database) GetSessionMemories(conversationID string, limit, offset int) ([]*Memory, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if limit <= 0 {
		limit = 50
	}

	rows, err := d.db.Query(`
		SELECT id, content, source, importance, tags, session_id, domain,
		       embedding, created_at, updated_at, agent_type, agent_context,
		       access_scope, slug, parent_memory_id, chunk_level, chunk_index,
		       conversation_id
		FROM memories
		WHERE conversation_id = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, conversationID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get session memories: %w", err)
	}
	defer rows.Close()

	return scanMemories(rows)
}

// UpdateMemoryCCSession links a memory to a conversation
func (d *Database) UpdateMemoryCCSession(memoryID, conversationID string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	result, err := d.db.Exec(`
		UPDATE memories SET conversation_id = ?, updated_at = ? WHERE id = ?
	`, nullString(conversationID), time.Now(), memoryID)
	if err != nil {
		return fmt.Errorf("failed to update memory conversation: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("memory not found: %s", memoryID)
	}

	return nil
}

// LinkSessionToSummaryMemory sets the summary_memory_id on a conversation
func (d *Database) LinkSessionToSummaryMemory(conversationID, summaryMemoryID string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	result, err := d.db.Exec(`
		UPDATE conversations SET summary_memory_id = ?, updated_at = ? WHERE id = ?
	`, summaryMemoryID, time.Now(), conversationID)
	if err != nil {
		return fmt.Errorf("failed to link conversation to summary memory: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("conversation not found: %s", conversationID)
	}

	return nil
}

// =============================================================================
// SCAN HELPERS
// =============================================================================

func scanConversations(rows *sql.Rows) ([]*Conversation, error) {
	var sessions []*Conversation
	for rows.Next() {
		var s Conversation
		var model, title, firstPrompt, summary, sourceID, filePath, lastSyncPos, summaryMemID sql.NullString
		var lastActivity sql.NullTime

		err := rows.Scan(
			&s.ID, &s.SessionID, &s.ProjectPath, &s.ProjectHash,
			&model, &title, &firstPrompt, &summary,
			&s.CreatedAt, &s.UpdatedAt, &lastActivity,
			&s.MessageCount, &s.UserMessageCount, &s.AssistantMessageCount, &s.ToolCallCount,
			&sourceID, &filePath, &lastSyncPos, &summaryMemID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan conversation: %w", err)
		}

		s.Model = model.String
		s.Title = title.String
		s.FirstPrompt = firstPrompt.String
		s.Summary = summary.String
		s.SourceID = sourceID.String
		s.FilePath = filePath.String
		s.LastSyncPosition = lastSyncPos.String
		s.SummaryMemoryID = summaryMemID.String
		if lastActivity.Valid {
			s.LastActivity = &lastActivity.Time
		}

		sessions = append(sessions, &s)
	}
	return sessions, nil
}

// scanCCSessions is an alias for backward compatibility
var scanCCSessions = scanConversations

func scanConversationMessages(rows *sql.Rows) ([]*ConversationMessage, error) {
	var messages []*ConversationMessage
	for rows.Next() {
		var m ConversationMessage
		var timestamp sql.NullTime

		err := rows.Scan(
			&m.ID, &m.SessionID, &m.Role, &m.Content,
			&timestamp, &m.SequenceIndex, &m.HasToolUse, &m.TokenCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}

		if timestamp.Valid {
			m.Timestamp = &timestamp.Time
		}

		messages = append(messages, &m)
	}
	return messages, nil
}

// scanCCMessages is an alias for backward compatibility
var scanCCMessages = scanConversationMessages

func scanConversationActions(rows *sql.Rows) ([]*ConversationAction, error) {
	var actions []*ConversationAction
	for rows.Next() {
		var tc ConversationAction
		var messageID, inputJSON, resultText, filePath, operation sql.NullString
		var timestamp sql.NullTime

		err := rows.Scan(
			&tc.ID, &tc.SessionID, &messageID, &tc.ToolName,
			&inputJSON, &resultText, &tc.Success, &filePath, &operation,
			&timestamp,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan action: %w", err)
		}

		tc.MessageID = messageID.String
		tc.InputJSON = inputJSON.String
		tc.ResultText = resultText.String
		tc.FilePath = filePath.String
		tc.Operation = operation.String
		if timestamp.Valid {
			tc.Timestamp = &timestamp.Time
		}

		actions = append(actions, &tc)
	}
	return actions, nil
}

// scanCCToolCalls is an alias for backward compatibility
var scanCCToolCalls = scanConversationActions
