package memory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSessionDetector tests session ID detection
func TestSessionDetector(t *testing.T) {
	t.Run("GitDirectoryStrategy", func(t *testing.T) {
		detector := NewSessionDetector(SessionStrategyGitDirectory)
		sessionID := detector.DetectSessionID()

		// Should have daemon- prefix
		if !strings.HasPrefix(sessionID, "daemon-") {
			t.Errorf("Expected daemon- prefix, got %s", sessionID)
		}

		// Should not be empty after prefix
		if len(sessionID) <= len("daemon-") {
			t.Errorf("Session ID too short: %s", sessionID)
		}
	})

	t.Run("ManualStrategy", func(t *testing.T) {
		detector := NewSessionDetector(SessionStrategyManual)
		detector.ManualID = "custom-session-123"

		sessionID := detector.DetectSessionID()
		if sessionID != "custom-session-123" {
			t.Errorf("Expected custom session ID, got %s", sessionID)
		}
	})

	t.Run("ManualStrategyFallback", func(t *testing.T) {
		detector := NewSessionDetector(SessionStrategyManual)
		// No manual ID set - should fallback to git-directory

		sessionID := detector.DetectSessionID()
		if !strings.HasPrefix(sessionID, "daemon-") {
			t.Errorf("Expected fallback to git-directory strategy, got %s", sessionID)
		}
	})

	t.Run("CachingBehavior", func(t *testing.T) {
		detector := NewSessionDetector(SessionStrategyGitDirectory)

		// First call
		sessionID1 := detector.DetectSessionID()
		// Second call (should be cached)
		sessionID2 := detector.DetectSessionID()

		if sessionID1 != sessionID2 {
			t.Errorf("Session ID should be cached: %s != %s", sessionID1, sessionID2)
		}
	})

	t.Run("CustomPrefix", func(t *testing.T) {
		detector := NewSessionDetector(SessionStrategyGitDirectory)
		detector.Prefix = "custom-"

		_ = detector.DetectSessionID()
		// Clear cache first by changing strategy back
		detector.cacheDir = ""

		sessionID := detector.DetectSessionID()
		if !strings.HasPrefix(sessionID, "custom-") {
			t.Errorf("Expected custom- prefix, got %s", sessionID)
		}
	})
}

// TestFindGitRoot tests git root detection
func TestFindGitRoot(t *testing.T) {
	t.Run("NoGitDirectory", func(t *testing.T) {
		tmpDir := t.TempDir()
		root := findGitRoot(tmpDir)
		if root != "" {
			t.Errorf("Expected empty string for non-git directory, got %s", root)
		}
	})

	t.Run("WithGitDirectory", func(t *testing.T) {
		tmpDir := t.TempDir()
		gitDir := filepath.Join(tmpDir, ".git")
		_ = os.Mkdir(gitDir, 0755)

		root := findGitRoot(tmpDir)
		if root != tmpDir {
			t.Errorf("Expected %s, got %s", tmpDir, root)
		}
	})

	t.Run("NestedDirectory", func(t *testing.T) {
		tmpDir := t.TempDir()
		gitDir := filepath.Join(tmpDir, ".git")
		_ = os.Mkdir(gitDir, 0755)

		nestedDir := filepath.Join(tmpDir, "src", "pkg")
		_ = os.MkdirAll(nestedDir, 0755)

		root := findGitRoot(nestedDir)
		if root != tmpDir {
			t.Errorf("Expected %s, got %s", tmpDir, root)
		}
	})
}

// TestSanitizeDirectoryName tests directory name sanitization
func TestSanitizeDirectoryName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"with-hyphen", "with-hyphen"},
		{"with_underscore", "with_underscore"},
		{"WithCaps", "withcaps"},
		{"with spaces", "with-spaces"},
		{"with.dots", "with-dots"},
		{"special!@#chars", "specialchars"},
		{"123-numbers", "123-numbers"},
		{"", ""},
	}

	for _, tt := range tests {
		result := sanitizeDirectoryName(tt.input)
		if result != tt.expected {
			t.Errorf("sanitizeDirectoryName(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

// TestGetAgentType tests agent type detection
func TestGetAgentType(t *testing.T) {
	// Save original env vars
	originalMCP := os.Getenv("MCP_SERVER")
	originalCode := os.Getenv("CLAUDE_CODE")
	originalAPI := os.Getenv("LOCAL_MEMORY_API")

	// Clean up after test
	defer func() {
		os.Setenv("MCP_SERVER", originalMCP)
		os.Setenv("CLAUDE_CODE", originalCode)
		os.Setenv("LOCAL_MEMORY_API", originalAPI)
	}()

	t.Run("DefaultUnknown", func(t *testing.T) {
		os.Unsetenv("MCP_SERVER")
		os.Unsetenv("CLAUDE_CODE")
		os.Unsetenv("LOCAL_MEMORY_API")

		agentType := GetAgentType()
		if agentType != "unknown" {
			t.Errorf("Expected 'unknown', got %s", agentType)
		}
	})

	t.Run("MCPServer", func(t *testing.T) {
		os.Setenv("MCP_SERVER", "true")
		os.Unsetenv("CLAUDE_CODE")
		os.Unsetenv("LOCAL_MEMORY_API")

		agentType := GetAgentType()
		if agentType != "claude-desktop" {
			t.Errorf("Expected 'claude-desktop', got %s", agentType)
		}
	})

	t.Run("ClaudeCode", func(t *testing.T) {
		os.Unsetenv("MCP_SERVER")
		os.Setenv("CLAUDE_CODE", "true")
		os.Unsetenv("LOCAL_MEMORY_API")

		agentType := GetAgentType()
		if agentType != "claude-code" {
			t.Errorf("Expected 'claude-code', got %s", agentType)
		}
	})

	t.Run("API", func(t *testing.T) {
		os.Unsetenv("MCP_SERVER")
		os.Unsetenv("CLAUDE_CODE")
		os.Setenv("LOCAL_MEMORY_API", "true")

		agentType := GetAgentType()
		if agentType != "api" {
			t.Errorf("Expected 'api', got %s", agentType)
		}
	})
}

// TestGetAgentContext tests agent context detection
func TestGetAgentContext(t *testing.T) {
	context := GetAgentContext()

	// Should have prefix
	if !strings.HasPrefix(context, "project:") && !strings.HasPrefix(context, "cwd:") {
		t.Errorf("Expected 'project:' or 'cwd:' prefix, got %s", context)
	}

	// Should not be empty after prefix
	parts := strings.SplitN(context, ":", 2)
	if len(parts) < 2 || parts[1] == "" {
		t.Errorf("Context value empty: %s", context)
	}
}
