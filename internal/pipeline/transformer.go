package pipeline

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/MycelicMemory/mycelicmemory/internal/database"
	"github.com/MycelicMemory/mycelicmemory/internal/relationships"
	"github.com/google/uuid"
)

// Transformer converts ConversationItems into database records.
// It handles session grouping, message creation, tool call extraction,
// and optional summary memory creation.
type Transformer struct {
	db     *database.Database
	relSvc *relationships.Service
}

// NewTransformer creates a new Transformer.
func NewTransformer(db *database.Database, relSvc *relationships.Service) *Transformer {
	return &Transformer{
		db:     db,
		relSvc: relSvc,
	}
}

// TransformBatch processes a batch of ConversationItems grouped by conversation.
// Items should ideally be pre-grouped by ConversationID, but the transformer
// handles mixed batches by grouping internally.
func (t *Transformer) TransformBatch(ctx context.Context, items []ConversationItem, sourceID string) (*IngestResult, error) {
	result := &IngestResult{SourceID: sourceID}

	// Group items by conversation
	convGroups := make(map[string][]ConversationItem)
	var convOrder []string
	for _, item := range items {
		key := item.ConversationID
		if key == "" {
			key = item.ExternalID // Fallback for non-conversation items
		}
		if _, exists := convGroups[key]; !exists {
			convOrder = append(convOrder, key)
		}
		convGroups[key] = append(convGroups[key], item)
	}

	// Process each conversation group
	for _, convID := range convOrder {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		convItems := convGroups[convID]
		convResult, err := t.transformConversation(ctx, convItems, sourceID)
		if err != nil {
			log.Warn("failed to transform conversation", "conversation_id", convID, "error", err)
			result.Errors++
			continue
		}
		mergeResults(result, convResult)
	}

	return result, nil
}

// transformConversation processes all items belonging to a single conversation.
func (t *Transformer) transformConversation(ctx context.Context, items []ConversationItem, sourceID string) (*IngestResult, error) {
	if len(items) == 0 {
		return &IngestResult{SourceID: sourceID}, nil
	}

	result := &IngestResult{SourceID: sourceID}
	first := items[0]

	// Extract conversation-level metadata
	convID := first.ConversationID
	projectOrSpace := first.ProjectOrSpace
	sourceType := first.SourceType

	// Compute stats from items
	var userMessages, assistantMessages, totalToolCalls int
	var firstPrompt, model string
	var firstTimestamp, lastTimestamp *time.Time

	for _, item := range items {
		ts := item.Timestamp
		if !ts.IsZero() {
			if firstTimestamp == nil {
				firstTimestamp = &ts
			}
			lastTimestamp = &ts
		}

		switch item.Role {
		case "user":
			userMessages++
			if firstPrompt == "" && item.Content != "" && !strings.HasPrefix(item.Content, "[Request interrupted") {
				firstPrompt = truncateStr(item.Content, 500)
			}
		case "assistant":
			assistantMessages++
			if model == "" {
				if m, ok := item.Metadata["model"]; ok {
					if ms, ok := m.(string); ok {
						model = ms
					}
				}
			}
		}

		totalToolCalls += len(item.Actions)
	}

	totalMessages := userMessages + assistantMessages
	title := generateConvTitle(firstPrompt)

	result.SessionsProcessed++

	// Compute project hash from project path for dedup
	projectHash := ""
	if projectOrSpace != "" {
		projectHash = encodeProjectHash(projectOrSpace)
	}

	// Check if session already exists (dedup)
	existing, err := t.db.GetCCSessionBySessionID(projectHash, convID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing session: %w", err)
	}

	if existing != nil {
		// Session exists — update counts
		result.SessionsUpdated++
		result.DuplicatesSkipped++

		syncPos := fmt.Sprintf("%d", len(items))
		msgCount := totalMessages
		userCount := userMessages
		assistantCount := assistantMessages
		toolCount := totalToolCalls

		err := t.db.UpdateCCSession(existing.ID, &database.CCSessionUpdate{
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
	result.SessionsCreated++

	ccSession := &database.CCSession{
		ID:                    sessionDBID,
		SessionID:             convID,
		ProjectPath:           projectOrSpace,
		ProjectHash:           projectHash,
		Model:                 model,
		Title:                 title,
		FirstPrompt:           firstPrompt,
		CreatedAt:             timeOrNowVal(firstTimestamp),
		UpdatedAt:             time.Now(),
		LastActivity:          lastTimestamp,
		MessageCount:          totalMessages,
		UserMessageCount:      userMessages,
		AssistantMessageCount: assistantMessages,
		ToolCallCount:         totalToolCalls,
		SourceID:              sourceID,
		LastSyncPosition:      fmt.Sprintf("%d", len(items)),
	}

	// Set file path from metadata if available
	if fp, ok := first.Metadata["file_path"]; ok {
		if fps, ok := fp.(string); ok {
			ccSession.FilePath = fps
		}
	}

	if err := t.db.CreateCCSession(ccSession); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Create messages and extract actions
	for i, item := range items {
		msgID := uuid.New().String()

		msg := &database.CCMessage{
			ID:            msgID,
			SessionID:     sessionDBID,
			Role:          item.Role,
			Content:       truncateStr(item.Content, 50000),
			Timestamp:     timePtr(item.Timestamp),
			SequenceIndex: i,
			HasToolUse:    len(item.Actions) > 0,
			TokenCount:    len(item.Content) / 4, // rough estimate
		}

		if err := t.db.CreateCCMessage(msg); err != nil {
			log.Debug("failed to create message", "error", err)
			continue
		}
		result.MessagesCreated++

		// Create actions (tool calls)
		for _, action := range item.Actions {
			tcID := uuid.New().String()
			tc := &database.CCToolCall{
				ID:        tcID,
				SessionID: sessionDBID,
				MessageID: msgID,
				ToolName:  action.Name,
				InputJSON: truncateStr(action.Input, 10000),
				Success:   action.Success,
				FilePath:  action.FilePath,
				Operation: action.Operation,
				Timestamp: timePtr(action.Timestamp),
			}

			if err := t.db.CreateCCToolCall(tc); err != nil {
				log.Debug("failed to create tool call", "error", err)
				continue
			}
			result.ActionsCreated++
		}
	}

	// Create summary memory if we have meaningful content
	if firstPrompt != "" && sourceType != "" {
		summaryID, err := t.createSummaryMemory(ctx, ccSession, sourceType)
		if err != nil {
			log.Warn("failed to create summary memory", "session", convID, "error", err)
		} else if summaryID != "" {
			if err := t.db.LinkSessionToSummaryMemory(sessionDBID, summaryID); err != nil {
				log.Warn("failed to link session to summary", "error", err)
			}
			result.MemoriesCreated++
		}
	}

	return result, nil
}

// createSummaryMemory creates a memory node representing a conversation session.
func (t *Transformer) createSummaryMemory(_ context.Context, session *database.CCSession, sourceType string) (string, error) {
	projectName := filepath.Base(session.ProjectPath)

	// Build human-readable label based on source type
	sourceLabel := sourceType
	switch sourceType {
	case "claude-code-local":
		sourceLabel = "Claude Code Session"
	case "slack":
		sourceLabel = "Slack Conversation"
	case "discord":
		sourceLabel = "Discord Conversation"
	case "telegram":
		sourceLabel = "Telegram Chat"
	case "imessage":
		sourceLabel = "iMessage Chat"
	}

	content := fmt.Sprintf("[%s] %s\n\nProject: %s\nMessages: %d (%d user, %d assistant)\nTool calls: %d\nDate: %s\n\nFirst prompt: %s",
		sourceLabel,
		session.Title,
		session.ProjectPath,
		session.MessageCount,
		session.UserMessageCount,
		session.AssistantMessageCount,
		session.ToolCallCount,
		session.CreatedAt.Format("2006-01-02 15:04"),
		session.FirstPrompt,
	)

	tags := []string{"conversation", sourceType, strings.ToLower(projectName)}

	mem := &database.Memory{
		ID:          uuid.New().String(),
		Content:     content,
		Source:      sourceType + "-session",
		Importance:  6,
		Tags:        tags,
		Domain:      "conversations",
		CreatedAt:   session.CreatedAt,
		UpdatedAt:   time.Now(),
		AgentType:   sourceType,
		AccessScope: "session",
		CCSessionID: session.ID,
	}

	if err := t.db.CreateMemory(mem); err != nil {
		return "", fmt.Errorf("failed to create summary memory: %w", err)
	}

	return mem.ID, nil
}

// encodeProjectHash converts a project path to the hash format used as directory name.
// This is the inverse of claude.DecodeProjectPath.
func encodeProjectHash(projectPath string) string {
	// Normalize separators
	path := strings.ReplaceAll(projectPath, "\\", "/")

	// Windows: "C:/dev/active" -> "C--dev-active"
	if len(path) >= 2 && path[1] == ':' {
		driveLetter := string(path[0])
		rest := path[2:]
		rest = strings.TrimPrefix(rest, "/")
		parts := strings.Split(rest, "/")
		return driveLetter + "--" + strings.Join(parts, "-")
	}

	// Unix: "/home/user/project" -> "-home-user-project"
	path = strings.TrimPrefix(path, "/")
	return "-" + strings.ReplaceAll(path, "/", "-")
}

func generateConvTitle(firstPrompt string) string {
	if firstPrompt == "" {
		return "Untitled session"
	}
	lines := strings.SplitN(firstPrompt, "\n", 2)
	title := lines[0]
	if len(title) > 100 {
		title = title[:97] + "..."
	}
	return title
}

func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

func timeOrNowVal(t *time.Time) time.Time {
	if t == nil {
		return time.Now()
	}
	return *t
}

func timePtr(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}
