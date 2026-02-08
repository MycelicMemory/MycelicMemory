package claude

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/MycelicMemory/mycelicmemory/internal/logging"
)

var log = logging.GetLogger("claude")

// Reader reads Claude Code conversation data from ~/.claude/
type Reader struct {
	claudeDir string
}

// ProjectInfo represents a Claude Code project directory
type ProjectInfo struct {
	Hash string // directory name in projects/ (e.g. "C--dev-active-ai-MycelicMemory")
	Path string // decoded project path (e.g. "C:\dev\active\ai\MycelicMemory")
}

// ConversationFile represents a parsed JSONL conversation file
type ConversationFile struct {
	FilePath  string
	SessionID string // from filename (UUID before .jsonl)
	Messages  []RawMessage
}

// RawMessage represents a single line from the JSONL file
type RawMessage struct {
	Type      string          `json:"type"`       // "user", "assistant", "file-history-snapshot"
	SessionID string          `json:"sessionId"`
	Timestamp string          `json:"timestamp"`
	UUID      string          `json:"uuid"`
	CWD       string          `json:"cwd"`
	Message   json.RawMessage `json:"message"`
}

// ParsedMessage is the message field within a RawMessage
type ParsedMessage struct {
	Role    string          `json:"role"`    // "user", "assistant"
	Content json.RawMessage `json:"content"` // string or []ContentBlock
	Model   string          `json:"model,omitempty"`
}

// ContentBlock represents a content block in a message
type ContentBlock struct {
	Type  string `json:"type"`            // "text", "tool_use", "tool_result"
	Text  string `json:"text,omitempty"`
	ID    string `json:"id,omitempty"`    // tool_use_id
	Name  string `json:"name,omitempty"`  // tool name
	Input json.RawMessage `json:"input,omitempty"` // tool input
}

// NewReader creates a reader for Claude Code conversation files
func NewReader(claudeDir string) *Reader {
	if claudeDir == "" {
		claudeDir = DetectClaudeDir()
	}
	return &Reader{claudeDir: claudeDir}
}

// DetectClaudeDir finds the ~/.claude directory on the current OS
func DetectClaudeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Warn("could not detect home directory", "error", err)
		return ""
	}

	claudeDir := filepath.Join(home, ".claude")
	if _, err := os.Stat(claudeDir); err != nil {
		log.Debug("claude directory not found", "path", claudeDir)
		return ""
	}

	return claudeDir
}

// ClaudeDir returns the configured claude directory
func (r *Reader) ClaudeDir() string {
	return r.claudeDir
}

// ListProjects scans the projects/ subdirectory and returns all project entries
func (r *Reader) ListProjects() ([]ProjectInfo, error) {
	projectsDir := filepath.Join(r.claudeDir, "projects")
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read projects directory: %w", err)
	}

	var projects []ProjectInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		projects = append(projects, ProjectInfo{
			Hash: entry.Name(),
			Path: DecodeProjectPath(entry.Name()),
		})
	}

	return projects, nil
}

// ListConversationFiles lists all .jsonl files in a project directory (top-level only, skips subagents)
func (r *Reader) ListConversationFiles(projectHash string) ([]string, error) {
	projectDir := filepath.Join(r.claudeDir, "projects", projectHash)
	entries, err := os.ReadDir(projectDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read project directory: %w", err)
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".jsonl") {
			files = append(files, filepath.Join(projectDir, entry.Name()))
		}
	}

	return files, nil
}

// ReadConversation parses a JSONL file into a ConversationFile
func (r *Reader) ReadConversation(filePath string) (*ConversationFile, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open conversation file: %w", err)
	}
	defer f.Close()

	// Extract session ID from filename
	base := filepath.Base(filePath)
	sessionID := strings.TrimSuffix(base, ".jsonl")

	conv := &ConversationFile{
		FilePath:  filePath,
		SessionID: sessionID,
	}

	scanner := bufio.NewScanner(f)
	// Increase buffer for large JSONL lines
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var msg RawMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			// Skip unparseable lines
			continue
		}

		// Only keep user and assistant messages
		if msg.Type == "user" || msg.Type == "assistant" {
			conv.Messages = append(conv.Messages, msg)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanner error reading %s: %w", filePath, err)
	}

	return conv, nil
}

// DecodeProjectPath converts a project hash back to a filesystem path
// e.g. "C--dev-active-ai-MycelicMemory" -> "C:\dev\active\ai\MycelicMemory" (Windows)
// e.g. "-home-user-project" -> "/home/user/project" (Linux)
func DecodeProjectPath(hash string) string {
	sep := string(filepath.Separator)
	if runtime.GOOS == "windows" {
		// Windows: "C--dev-active-ai-MycelicMemory" -> "C:\dev\active\ai\MycelicMemory"
		// First two chars before -- are drive letter
		if len(hash) >= 3 && hash[1] == '-' && hash[2] == '-' {
			driveLetter := string(hash[0])
			rest := hash[3:]
			parts := strings.Split(rest, "-")
			return driveLetter + ":" + sep + strings.Join(parts, sep)
		}
		// Fallback: just replace - with separator
		return strings.ReplaceAll(hash, "-", sep)
	}

	// Unix: "-home-user-project" -> "/home/user/project"
	return sep + strings.ReplaceAll(strings.TrimPrefix(hash, "-"), "-", sep)
}

// ExtractTextContent extracts text content from a message's content field
func ExtractTextContent(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	// Try as string first
	var str string
	if err := json.Unmarshal(raw, &str); err == nil {
		return str
	}

	// Try as array of content blocks
	var blocks []ContentBlock
	if err := json.Unmarshal(raw, &blocks); err == nil {
		var parts []string
		for _, b := range blocks {
			if b.Type == "text" && b.Text != "" {
				parts = append(parts, b.Text)
			}
		}
		return strings.Join(parts, "\n")
	}

	return ""
}

// ExtractToolUses extracts tool_use blocks from a message's content field
func ExtractToolUses(raw json.RawMessage) []ContentBlock {
	if len(raw) == 0 {
		return nil
	}

	var blocks []ContentBlock
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return nil
	}

	var toolUses []ContentBlock
	for _, b := range blocks {
		if b.Type == "tool_use" {
			toolUses = append(toolUses, b)
		}
	}
	return toolUses
}

// HasToolUse checks if a message's content contains any tool_use blocks
func HasToolUse(raw json.RawMessage) bool {
	return len(ExtractToolUses(raw)) > 0
}

// ParseTimestamp parses an ISO 8601 timestamp string
func ParseTimestamp(ts string) *time.Time {
	if ts == "" {
		return nil
	}

	// Try common formats
	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.999Z",
		"2006-01-02T15:04:05Z",
	}

	for _, fmt := range formats {
		if t, err := time.Parse(fmt, ts); err == nil {
			return &t
		}
	}

	return nil
}
