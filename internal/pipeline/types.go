package pipeline

import (
	"context"
	"encoding/json"
	"time"
)

// SourceAdapter is the interface every data source must implement.
// It abstracts reading from any external source (Claude JSONL, Slack, Discord, etc.)
// into a unified stream of ConversationItems.
type SourceAdapter interface {
	// Type returns the source type identifier (e.g. "claude-code-local", "slack")
	Type() string

	// Configure initializes the adapter with source-specific config JSON
	Configure(config json.RawMessage) error

	// ReadItems streams items from the source starting from the given checkpoint.
	// The checkpoint string is opaque to the pipeline — each adapter defines its own format.
	// Returns a channel of ConversationItem and a channel that receives a single error (or nil) on completion.
	ReadItems(ctx context.Context, checkpoint string) (<-chan ConversationItem, <-chan error)

	// Checkpoint returns the current position for resume on next run.
	Checkpoint() string

	// Validate checks that the adapter's configuration is valid and the source is reachable.
	Validate() error
}

// ConversationItem is the universal format for any ingested content.
// Every source adapter converts its native format into these items.
type ConversationItem struct {
	// Identity
	ExternalID string `json:"external_id"` // Unique ID in source system
	SourceType string `json:"source_type"` // "claude-code-local", "slack", "discord", etc.

	// Conversation-level grouping
	ConversationID string `json:"conversation_id"` // Thread/channel/session grouping key
	ProjectOrSpace string `json:"project_or_space"` // Workspace/project context

	// Message-level
	Role          string    `json:"role"`           // "user", "assistant", "bot", "system"
	Author        string    `json:"author"`         // Display name or ID
	Content       string    `json:"content"`        // Message text
	ContentType   string    `json:"content_type"`   // "text", "markdown", "code", "html"
	Timestamp     time.Time `json:"timestamp"`
	SequenceIndex int       `json:"sequence_index"`

	// Attachments & Actions
	Attachments []Attachment `json:"attachments,omitempty"`
	Actions     []Action     `json:"actions,omitempty"`

	// Metadata
	Metadata  map[string]any `json:"metadata,omitempty"`
	ThreadID  string         `json:"thread_id,omitempty"`  // For threaded conversations
	ReplyToID string         `json:"reply_to_id,omitempty"` // Parent message ID
}

// Attachment represents a file, image, link, or code block attached to a message.
type Attachment struct {
	Type     string `json:"type"`               // "file", "image", "link", "code_block"
	Name     string `json:"name,omitempty"`
	URL      string `json:"url,omitempty"`
	Content  string `json:"content,omitempty"`   // Inline content if available
	MimeType string `json:"mime_type,omitempty"`
}

// Action represents a tool call, reaction, edit, or other interactive event.
type Action struct {
	Type      string    `json:"type"`              // "tool_call", "reaction", "edit", "pin"
	Name      string    `json:"name,omitempty"`
	Input     string    `json:"input,omitempty"`   // JSON string
	Output    string    `json:"output,omitempty"`
	Success   bool      `json:"success"`
	Timestamp time.Time `json:"timestamp,omitempty"`
	FilePath  string    `json:"file_path,omitempty"`
	Operation string    `json:"operation,omitempty"` // "read", "write", "edit", "execute"
}

// ProgressUpdate reports pipeline progress during ingestion.
type ProgressUpdate struct {
	SourceID        string `json:"source_id"`
	SourceType      string `json:"source_type"`
	Phase           string `json:"phase"`            // "reading", "transforming", "storing"
	ItemsProcessed  int    `json:"items_processed"`
	ItemsTotal      int    `json:"items_total"`      // -1 if unknown
	SessionsCreated int    `json:"sessions_created"`
	MessagesCreated int    `json:"messages_created"`
	MemoriesCreated int    `json:"memories_created"`
	Errors          int    `json:"errors"`
	Checkpoint      string `json:"checkpoint"`
}

// IngestResult contains the final results of a pipeline run.
type IngestResult struct {
	SourceID          string    `json:"source_id"`
	SourceType        string    `json:"source_type"`
	SessionsProcessed int       `json:"sessions_processed"`
	SessionsCreated   int       `json:"sessions_created"`
	SessionsUpdated   int       `json:"sessions_updated"`
	MessagesCreated   int       `json:"messages_created"`
	ActionsCreated    int       `json:"actions_created"`
	MemoriesCreated   int       `json:"memories_created"`
	DuplicatesSkipped int       `json:"duplicates_skipped"`
	Errors            int       `json:"errors"`
	Duration          time.Duration `json:"duration"`
	Checkpoint        string    `json:"checkpoint"`
}
