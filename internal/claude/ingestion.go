package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/MycelicMemory/mycelicmemory/internal/database"
	"github.com/MycelicMemory/mycelicmemory/internal/relationships"
	"github.com/google/uuid"
)

// Ingester ingests Claude Code conversations into the database
type Ingester struct {
	reader *Reader
	db     *database.Database
	relSvc *relationships.Service
}

// IngestOptions controls what gets ingested
type IngestOptions struct {
	ProjectPath     string // filter to specific project (empty = all)
	MinMessages     int    // skip sessions with fewer messages (default 3)
	CreateSummaries bool   // create graph node memories for sessions
}

// IngestResult contains the results of an ingestion run
type IngestResult struct {
	SessionsProcessed int `json:"sessions_processed"`
	SessionsCreated   int `json:"sessions_created"`
	SessionsUpdated   int `json:"sessions_updated"`
	MessagesCreated   int `json:"messages_created"`
	ToolCallsCreated  int `json:"tool_calls_created"`
	MemoriesLinked    int `json:"memories_linked"`
}

// NewIngester creates a new conversation ingester
func NewIngester(reader *Reader, db *database.Database, relSvc *relationships.Service) *Ingester {
	return &Ingester{
		reader: reader,
		db:     db,
		relSvc: relSvc,
	}
}

// IngestAll iterates all projects and sessions, ingesting into the database
func (ing *Ingester) IngestAll(ctx context.Context, opts *IngestOptions) (*IngestResult, error) {
	if opts.MinMessages <= 0 {
		opts.MinMessages = 3
	}

	result := &IngestResult{}

	projects, err := ing.reader.ListProjects()
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	// Ensure data source exists
	sourceID, err := ing.ensureDataSource()
	if err != nil {
		return nil, fmt.Errorf("failed to ensure data source: %w", err)
	}

	for _, project := range projects {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		// Filter by project path if specified
		if opts.ProjectPath != "" && project.Path != opts.ProjectPath {
			continue
		}

		files, err := ing.reader.ListConversationFiles(project.Hash)
		if err != nil {
			log.Warn("failed to list conversations for project", "project", project.Hash, "error", err)
			continue
		}

		for _, filePath := range files {
			select {
			case <-ctx.Done():
				return result, ctx.Err()
			default:
			}

			conv, err := ing.reader.ReadConversation(filePath)
			if err != nil {
				log.Warn("failed to read conversation", "file", filePath, "error", err)
				continue
			}

			// Skip sessions with too few messages
			if len(conv.Messages) < opts.MinMessages {
				continue
			}

			result.SessionsProcessed++

			sessionResult, err := ing.IngestSession(ctx, conv, &project, sourceID, opts.CreateSummaries)
			if err != nil {
				log.Warn("failed to ingest session", "session", conv.SessionID, "error", err)
				continue
			}

			if sessionResult.wasCreated {
				result.SessionsCreated++
			} else {
				result.SessionsUpdated++
			}
			result.MessagesCreated += sessionResult.messagesCreated
			result.ToolCallsCreated += sessionResult.toolCallsCreated
			result.MemoriesLinked += sessionResult.memoriesLinked
		}
	}

	return result, nil
}

type sessionIngestResult struct {
	sessionID       string
	wasCreated      bool
	messagesCreated int
	toolCallsCreated int
	memoriesLinked  int
}

// IngestSession ingests a single conversation into the database
func (ing *Ingester) IngestSession(ctx context.Context, conv *ConversationFile, project *ProjectInfo, sourceID string, createSummaries bool) (*sessionIngestResult, error) {
	result := &sessionIngestResult{}

	// Check if session already exists
	existing, err := ing.db.GetCCSessionBySessionID(project.Hash, conv.SessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing session: %w", err)
	}

	// Parse messages
	var userMessages, assistantMessages int
	var firstPrompt string
	var firstTimestamp, lastTimestamp *time.Time
	var model string
	var totalToolCalls int

	for i, raw := range conv.Messages {
		var parsed ParsedMessage
		if raw.Message != nil {
			if err := json.Unmarshal(raw.Message, &parsed); err != nil {
				continue
			}
		}

		ts := ParseTimestamp(raw.Timestamp)
		if ts != nil {
			if firstTimestamp == nil {
				firstTimestamp = ts
			}
			lastTimestamp = ts
		}

		switch raw.Type {
		case "user":
			userMessages++
			if firstPrompt == "" {
				text := ExtractTextContent(parsed.Content)
				if text != "" && !strings.HasPrefix(text, "[Request interrupted") {
					firstPrompt = truncate(text, 500)
				}
			}
		case "assistant":
			assistantMessages++
			if model == "" && parsed.Model != "" {
				model = parsed.Model
			}
			if HasToolUse(parsed.Content) {
				totalToolCalls += len(ExtractToolUses(parsed.Content))
			}
		}

		_ = i // used in loop
	}

	totalMessages := userMessages + assistantMessages
	title := generateTitle(firstPrompt)

	if existing != nil {
		// Session exists — update counts
		result.sessionID = existing.ID
		result.wasCreated = false

		msgCount := totalMessages
		userCount := userMessages
		assistantCount := assistantMessages
		toolCount := totalToolCalls
		syncPos := fmt.Sprintf("%d", len(conv.Messages))

		err := ing.db.UpdateCCSession(existing.ID, &database.CCSessionUpdate{
			Title:             &title,
			MessageCount:      &msgCount,
			UserMsgCount:      &userCount,
			AssistantMsgCount: &assistantCount,
			ToolCallCount:     &toolCount,
			LastSyncPosition:  &syncPos,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to update session: %w", err)
		}

		return result, nil
	}

	// Create new session
	sessionDBID := uuid.New().String()
	result.sessionID = sessionDBID
	result.wasCreated = true

	ccSession := &database.CCSession{
		ID:                    sessionDBID,
		SessionID:             conv.SessionID,
		ProjectPath:           project.Path,
		ProjectHash:           project.Hash,
		Model:                 model,
		Title:                 title,
		FirstPrompt:           firstPrompt,
		CreatedAt:             timeOrNow(firstTimestamp),
		UpdatedAt:             time.Now(),
		LastActivity:          lastTimestamp,
		MessageCount:          totalMessages,
		UserMessageCount:      userMessages,
		AssistantMessageCount: assistantMessages,
		ToolCallCount:         totalToolCalls,
		SourceID:              sourceID,
		FilePath:              conv.FilePath,
		LastSyncPosition:      fmt.Sprintf("%d", len(conv.Messages)),
	}

	if err := ing.db.CreateCCSession(ccSession); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Create messages
	for i, raw := range conv.Messages {
		var parsed ParsedMessage
		if raw.Message != nil {
			if err := json.Unmarshal(raw.Message, &parsed); err != nil {
				continue
			}
		}

		role := raw.Type
		if role == "user" {
			role = "user"
		} else if role == "assistant" {
			role = "assistant"
		}

		textContent := ExtractTextContent(parsed.Content)
		hasToolUse := HasToolUse(parsed.Content)
		ts := ParseTimestamp(raw.Timestamp)

		msg := &database.CCMessage{
			ID:            uuid.New().String(),
			SessionID:     sessionDBID,
			Role:          role,
			Content:       truncate(textContent, 50000),
			Timestamp:     ts,
			SequenceIndex: i,
			HasToolUse:    hasToolUse,
			TokenCount:    len(textContent) / 4, // rough estimate
		}

		if err := ing.db.CreateCCMessage(msg); err != nil {
			log.Debug("failed to create message", "error", err)
			continue
		}
		result.messagesCreated++

		// Extract tool calls from assistant messages
		if raw.Type == "assistant" {
			toolUses := ExtractToolUses(parsed.Content)
			for _, tu := range toolUses {
				inputStr := ""
				if tu.Input != nil {
					inputStr = string(tu.Input)
				}

				tc := &database.CCToolCall{
					ID:        tu.ID,
					SessionID: sessionDBID,
					MessageID: msg.ID,
					ToolName:  tu.Name,
					InputJSON: truncate(inputStr, 10000),
					Success:   true,
					FilePath:  ExtractFilePath(tu.Name, tu.Input),
					Operation: classifyOperation(tu.Name),
					Timestamp: ts,
				}

				if tc.ID == "" {
					tc.ID = uuid.New().String()
				}

				if err := ing.db.CreateCCToolCall(tc); err != nil {
					log.Debug("failed to create tool call", "error", err)
					continue
				}
				result.toolCallsCreated++
			}
		}
	}

	// Create summary memory as graph node if requested
	if createSummaries && firstPrompt != "" {
		summaryID, err := ing.createSummaryMemory(ctx, ccSession)
		if err != nil {
			log.Warn("failed to create summary memory", "session", conv.SessionID, "error", err)
		} else if summaryID != "" {
			if err := ing.db.LinkSessionToSummaryMemory(sessionDBID, summaryID); err != nil {
				log.Warn("failed to link session to summary", "error", err)
			}
			result.memoriesLinked++
		}
	}

	return result, nil
}

// createSummaryMemory creates a memory node representing a chat session
func (ing *Ingester) createSummaryMemory(ctx context.Context, session *database.CCSession) (string, error) {
	projectName := filepath.Base(session.ProjectPath)

	content := fmt.Sprintf("[Claude Code Session] %s\n\nProject: %s\nMessages: %d (%d user, %d assistant)\nTool calls: %d\nDate: %s\n\nFirst prompt: %s",
		session.Title,
		session.ProjectPath,
		session.MessageCount,
		session.UserMessageCount,
		session.AssistantMessageCount,
		session.ToolCallCount,
		session.CreatedAt.Format("2006-01-02 15:04"),
		session.FirstPrompt,
	)

	tags := []string{"conversation", "claude-code", strings.ToLower(projectName)}
	tagsJSON := tagsToJSON(tags)

	mem := &database.Memory{
		ID:          uuid.New().String(),
		Content:     content,
		Source:      "claude-code-session",
		Importance:  6,
		Tags:        tags,
		Domain:      "conversations",
		CreatedAt:   session.CreatedAt,
		UpdatedAt:   time.Now(),
		AgentType:   "claude-code",
		AccessScope: "session",
		CCSessionID: session.ID,
	}

	// We need to set tags JSON for the DB insert
	_ = tagsJSON

	if err := ing.db.CreateMemory(mem); err != nil {
		return "", fmt.Errorf("failed to create summary memory: %w", err)
	}

	return mem.ID, nil
}

// ensureDataSource creates the "claude-code-local" data source if it doesn't exist
func (ing *Ingester) ensureDataSource() (string, error) {
	// Check if already exists
	sources, err := ing.db.ListDataSources(&database.DataSourceFilters{
		SourceType: "claude-code-local",
		Limit:      1,
	})
	if err != nil {
		return "", err
	}

	if len(sources) > 0 {
		return sources[0].ID, nil
	}

	// Create new data source
	ds := &database.DataSource{
		ID:         uuid.New().String(),
		SourceType: "claude-code-local",
		Name:       "Claude Code Local Conversations",
		Config:     fmt.Sprintf(`{"claude_dir":"%s"}`, strings.ReplaceAll(ing.reader.ClaudeDir(), `\`, `\\`)),
		Status:     "active",
	}

	if err := ing.db.CreateDataSource(ds); err != nil {
		return "", fmt.Errorf("failed to create data source: %w", err)
	}

	return ds.ID, nil
}

// ExtractFilePath extracts a filepath from a tool call's input
func ExtractFilePath(toolName string, input json.RawMessage) string {
	if len(input) == 0 {
		return ""
	}

	var inputMap map[string]interface{}
	if err := json.Unmarshal(input, &inputMap); err != nil {
		return ""
	}

	// Different tools use different field names for file paths
	pathFields := []string{"file_path", "path", "filepath", "command"}
	for _, field := range pathFields {
		if v, ok := inputMap[field]; ok {
			if s, ok := v.(string); ok {
				// For 'command' field, try to extract path from bash commands
				if field == "command" {
					return ""
				}
				return s
			}
		}
	}

	return ""
}

func classifyOperation(toolName string) string {
	switch toolName {
	case "Read", "read":
		return "read"
	case "Write", "write":
		return "write"
	case "Edit", "edit", "MultiEdit":
		return "edit"
	case "Bash", "bash":
		return "execute"
	case "Glob", "glob":
		return "read"
	case "Grep", "grep":
		return "read"
	default:
		return ""
	}
}

func generateTitle(firstPrompt string) string {
	if firstPrompt == "" {
		return "Untitled session"
	}

	// Take first line, truncated
	lines := strings.SplitN(firstPrompt, "\n", 2)
	title := lines[0]

	if len(title) > 100 {
		title = title[:97] + "..."
	}

	return title
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

func timeOrNow(t *time.Time) time.Time {
	if t == nil {
		return time.Now()
	}
	return *t
}

func tagsToJSON(tags []string) string {
	if len(tags) == 0 {
		return "[]"
	}
	data, _ := json.Marshal(tags)
	return string(data)
}
