package memory

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// SessionStrategy defines how session IDs are detected
type SessionStrategy string

const (
	// SessionStrategyGitDirectory uses the git repository root directory name
	// VERIFIED: Local Memory uses "daemon-{directory_name}" format
	SessionStrategyGitDirectory SessionStrategy = "git-directory"

	// SessionStrategyManual requires explicit session ID
	SessionStrategyManual SessionStrategy = "manual"

	// SessionStrategyHash uses a hash of the git remote URL
	SessionStrategyHash SessionStrategy = "hash"
)

// SessionDetector handles session ID detection based on strategy
type SessionDetector struct {
	Strategy  SessionStrategy
	ManualID  string
	Prefix    string // Default: "daemon-"
	cacheDir  string
	cacheID   string
}

// NewSessionDetector creates a new session detector
func NewSessionDetector(strategy SessionStrategy) *SessionDetector {
	return &SessionDetector{
		Strategy: strategy,
		Prefix:   "daemon-",
	}
}

// DetectSessionID returns the session ID based on the configured strategy
// VERIFIED: Matches local-memory session detection behavior
func (d *SessionDetector) DetectSessionID() string {
	switch d.Strategy {
	case SessionStrategyManual:
		if d.ManualID != "" {
			return d.ManualID
		}
		return d.detectGitDirectory()
	case SessionStrategyHash:
		return d.detectGitHash()
	case SessionStrategyGitDirectory:
		fallthrough
	default:
		return d.detectGitDirectory()
	}
}

// detectGitDirectory returns session ID based on git repository directory name
// Format: "daemon-{directory_name}"
// VERIFIED: This is the exact format used by local-memory
func (d *SessionDetector) detectGitDirectory() string {
	// Check cache first
	cwd, _ := os.Getwd()
	if d.cacheDir == cwd && d.cacheID != "" {
		return d.cacheID
	}

	// Try to find git root
	gitRoot := findGitRoot(cwd)
	if gitRoot != "" {
		dirName := filepath.Base(gitRoot)
		d.cacheDir = cwd
		d.cacheID = d.Prefix + sanitizeDirectoryName(dirName)
		return d.cacheID
	}

	// Fallback to current directory name
	dirName := filepath.Base(cwd)
	d.cacheDir = cwd
	d.cacheID = d.Prefix + sanitizeDirectoryName(dirName)
	return d.cacheID
}

// detectGitHash returns session ID based on hash of git remote URL
func (d *SessionDetector) detectGitHash() string {
	cwd, _ := os.Getwd()
	gitRoot := findGitRoot(cwd)
	if gitRoot == "" {
		return d.detectGitDirectory()
	}

	// Get remote URL
	cmd := exec.Command("git", "-C", gitRoot, "config", "--get", "remote.origin.url")
	output, err := cmd.Output()
	if err != nil {
		return d.detectGitDirectory()
	}

	remoteURL := strings.TrimSpace(string(output))
	if remoteURL == "" {
		return d.detectGitDirectory()
	}

	// Create hash
	hash := sha256.Sum256([]byte(remoteURL))
	shortHash := hex.EncodeToString(hash[:8])
	return d.Prefix + shortHash
}

// findGitRoot finds the root of the git repository
func findGitRoot(startDir string) string {
	dir := startDir
	for {
		gitDir := filepath.Join(dir, ".git")
		if info, err := os.Stat(gitDir); err == nil && info.IsDir() {
			return dir
		}

		// Check for .git file (submodule or worktree)
		if info, err := os.Stat(gitDir); err == nil && !info.IsDir() {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root
			return ""
		}
		dir = parent
	}
}

// sanitizeDirectoryName removes special characters from directory name
func sanitizeDirectoryName(name string) string {
	// Replace special characters with hyphens
	var result strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			result.WriteRune(r)
		} else if r == ' ' || r == '.' {
			result.WriteRune('-')
		}
	}
	return strings.ToLower(result.String())
}

// GetAgentType returns the agent type based on invocation context
// VERIFIED: 4 valid agent types from local-memory
func GetAgentType() string {
	// Check for MCP invocation
	if os.Getenv("MCP_SERVER") != "" {
		return "claude-desktop"
	}

	// Check for Claude Code context
	if os.Getenv("CLAUDE_CODE") != "" {
		return "claude-code"
	}

	// Check for API context
	if os.Getenv("LOCAL_MEMORY_API") != "" {
		return "api"
	}

	// Default to CLI/unknown
	return "unknown"
}

// GetAgentContext returns context information about the current agent
func GetAgentContext() string {
	// Try to get project name from git
	cwd, _ := os.Getwd()
	gitRoot := findGitRoot(cwd)
	if gitRoot != "" {
		return "project:" + filepath.Base(gitRoot)
	}
	return "cwd:" + filepath.Base(cwd)
}
