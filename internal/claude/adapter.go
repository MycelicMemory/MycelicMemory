package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/MycelicMemory/mycelicmemory/internal/pipeline"
)

// Adapter implements pipeline.SourceAdapter for Claude Code local JSONL files.
// It wraps the existing Reader to read conversations from ~/.claude/projects/
// and converts them into universal ConversationItems.
type Adapter struct {
	reader      *Reader
	config      AdapterConfig
	checkpoint  string
	minMessages int
}

// AdapterConfig holds configuration for the Claude adapter.
type AdapterConfig struct {
	ClaudeDir   string `json:"claude_dir"`             // Path to ~/.claude (auto-detected if empty)
	ProjectPath string `json:"project_path,omitempty"` // Filter to specific project
	MinMessages int    `json:"min_messages,omitempty"`  // Skip sessions with fewer messages (default 3)
}

// NewAdapter creates a new Claude source adapter.
func NewAdapter(reader *Reader) *Adapter {
	return &Adapter{
		reader:      reader,
		minMessages: 3,
	}
}

// Type returns the source type identifier.
func (a *Adapter) Type() string {
	return "claude-code-local"
}

// Configure initializes the adapter with source-specific config JSON.
func (a *Adapter) Configure(config json.RawMessage) error {
	if len(config) == 0 || string(config) == "{}" {
		return nil
	}

	var cfg AdapterConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return fmt.Errorf("invalid claude adapter config: %w", err)
	}
	a.config = cfg

	if cfg.MinMessages > 0 {
		a.minMessages = cfg.MinMessages
	}

	// Re-initialize reader if a different claude dir is configured
	if cfg.ClaudeDir != "" && cfg.ClaudeDir != a.reader.ClaudeDir() {
		a.reader = NewReader(cfg.ClaudeDir)
	}

	return nil
}

// ReadItems streams ConversationItems from Claude Code JSONL files.
// Each conversation file becomes a batch of items sharing a ConversationID.
func (a *Adapter) ReadItems(ctx context.Context, checkpoint string) (<-chan pipeline.ConversationItem, <-chan error) {
	itemsCh := make(chan pipeline.ConversationItem, 100)
	errCh := make(chan error, 1)

	go func() {
		defer close(itemsCh)
		defer close(errCh)

		projects, err := a.reader.ListProjects()
		if err != nil {
			errCh <- fmt.Errorf("failed to list projects: %w", err)
			return
		}

		for _, project := range projects {
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			default:
			}

			// Filter by project path if configured
			if a.config.ProjectPath != "" && project.Path != a.config.ProjectPath {
				continue
			}

			files, err := a.reader.ListConversationFiles(project.Hash)
			if err != nil {
				log.Warn("failed to list conversations for project", "project", project.Hash, "error", err)
				continue
			}

			for _, filePath := range files {
				select {
				case <-ctx.Done():
					errCh <- ctx.Err()
					return
				default:
				}

				conv, err := a.reader.ReadConversation(filePath)
				if err != nil {
					log.Warn("failed to read conversation", "file", filePath, "error", err)
					continue
				}

				// Skip sessions with too few messages
				if len(conv.Messages) < a.minMessages {
					continue
				}

				// Convert each message to a ConversationItem
				for i, raw := range conv.Messages {
					var parsed ParsedMessage
					if raw.Message != nil {
						if err := json.Unmarshal(raw.Message, &parsed); err != nil {
							continue
						}
					}

					item := a.convertMessage(raw, parsed, i, conv, &project)

					select {
					case itemsCh <- item:
					case <-ctx.Done():
						errCh <- ctx.Err()
						return
					}
				}

				// Update checkpoint to last processed file
				a.checkpoint = filePath
			}
		}

		errCh <- nil
	}()

	return itemsCh, errCh
}

// Checkpoint returns the current position for resume.
func (a *Adapter) Checkpoint() string {
	return a.checkpoint
}

// Validate checks that the claude directory is accessible.
func (a *Adapter) Validate() error {
	if a.reader.ClaudeDir() == "" {
		return fmt.Errorf("claude directory not configured or detected")
	}

	projects, err := a.reader.ListProjects()
	if err != nil {
		return fmt.Errorf("cannot access claude projects: %w", err)
	}

	if len(projects) == 0 {
		return fmt.Errorf("no projects found in %s", a.reader.ClaudeDir())
	}

	return nil
}

// convertMessage converts a single Claude JSONL message into a ConversationItem.
func (a *Adapter) convertMessage(raw RawMessage, parsed ParsedMessage, index int, conv *ConversationFile, project *ProjectInfo) pipeline.ConversationItem {
	textContent := ExtractTextContent(parsed.Content)
	ts := ParseTimestamp(raw.Timestamp)

	item := pipeline.ConversationItem{
		ExternalID:     fmt.Sprintf("%s:%s:%d", project.Hash, conv.SessionID, index),
		SourceType:     "claude-code-local",
		ConversationID: conv.SessionID,
		ProjectOrSpace: project.Path,
		Role:           raw.Type, // "user" or "assistant"
		Author:         raw.Type, // Claude doesn't have display names
		Content:        textContent,
		ContentType:    "text",
		SequenceIndex:  index,
		Metadata: map[string]any{
			"file_path": conv.FilePath,
		},
	}

	if ts != nil {
		item.Timestamp = *ts
	}

	// Extract model from assistant messages
	if parsed.Model != "" {
		item.Metadata["model"] = parsed.Model
	}

	// Extract tool calls from assistant messages
	if raw.Type == "assistant" {
		toolUses := ExtractToolUses(parsed.Content)
		for _, tu := range toolUses {
			inputStr := ""
			if tu.Input != nil {
				inputStr = string(tu.Input)
			}

			action := pipeline.Action{
				Type:      "tool_call",
				Name:      tu.Name,
				Input:     truncate(inputStr, 10000),
				Success:   true,
				FilePath:  ExtractFilePath(tu.Name, tu.Input),
				Operation: classifyOperation(tu.Name),
			}
			if ts != nil {
				action.Timestamp = *ts
			}

			item.Actions = append(item.Actions, action)
		}
	}

	// Extract thread info
	if raw.CWD != "" {
		item.Metadata["cwd"] = raw.CWD
	}

	return item
}

// ConvertIngestOptions converts legacy IngestOptions to adapter configuration.
func ConvertIngestOptions(opts *IngestOptions) json.RawMessage {
	cfg := AdapterConfig{
		ProjectPath: opts.ProjectPath,
		MinMessages: opts.MinMessages,
	}
	data, _ := json.Marshal(cfg)
	return data
}

// RunLegacyIngestion runs the existing ingestion flow through the new pipeline.
// This maintains full backward compatibility with the existing MCP tool.
func RunLegacyIngestion(ctx context.Context, reader *Reader, q *pipeline.Queue, sourceID string, opts *IngestOptions) (*pipeline.IngestResult, error) {
	adapter := NewAdapter(reader)

	// Apply options
	if opts.ProjectPath != "" || opts.MinMessages > 0 {
		config := ConvertIngestOptions(opts)
		if err := adapter.Configure(config); err != nil {
			return nil, fmt.Errorf("failed to configure adapter: %w", err)
		}
	}

	// If not creating summaries, we need to override the transformer behavior.
	// For now, summaries are always created through the pipeline.
	// The createSummaries option can be handled at the transformer level later.
	_ = opts.CreateSummaries

	result, err := q.EnqueueDirect(ctx, adapter, sourceID, "")
	if err != nil {
		return nil, err
	}

	// Map pipeline result to legacy format
	return result, nil
}

// LegacyResultToClaudeResult converts pipeline.IngestResult to claude.IngestResult
// for backward compatibility with existing callers.
func LegacyResultToClaudeResult(pr *pipeline.IngestResult) *IngestResult {
	if pr == nil {
		return &IngestResult{}
	}
	return &IngestResult{
		SessionsProcessed: pr.SessionsProcessed,
		SessionsCreated:   pr.SessionsCreated,
		SessionsUpdated:   pr.SessionsUpdated,
		MessagesCreated:   pr.MessagesCreated,
		ToolCallsCreated:  pr.ActionsCreated,
		MemoriesLinked:    pr.MemoriesCreated,
	}
}

// encodeProjectHash creates a hash from project path (for dedup matching).
// Package-internal version — the canonical one is in pipeline/transformer.go.
func encodeProjectHashLocal(projectPath string) string {
	path := strings.ReplaceAll(projectPath, "\\", "/")
	if len(path) >= 2 && path[1] == ':' {
		driveLetter := string(path[0])
		rest := path[2:]
		rest = strings.TrimPrefix(rest, "/")
		parts := strings.Split(rest, "/")
		return driveLetter + "--" + strings.Join(parts, "-")
	}
	path = strings.TrimPrefix(path, "/")
	return "-" + strings.ReplaceAll(path, "/", "-")
}
