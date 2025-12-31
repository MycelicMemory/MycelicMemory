package benchmark

import (
	"os/exec"
	"strings"
)

// CaptureGitState captures the current git repository state
func CaptureGitState(repoPath string) (*GitState, error) {
	state := &GitState{}

	// Get commit hash
	hash, err := runGitCommand(repoPath, "rev-parse", "HEAD")
	if err != nil {
		return nil, err
	}
	state.CommitHash = hash
	if len(hash) >= 7 {
		state.ShortHash = hash[:7]
	}

	// Get branch name
	branch, err := runGitCommand(repoPath, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		state.Branch = "unknown"
	} else {
		state.Branch = branch
	}

	// Check if working tree is dirty
	status, err := runGitCommand(repoPath, "status", "--porcelain")
	if err == nil {
		state.Dirty = len(strings.TrimSpace(status)) > 0
	}

	return state, nil
}

// runGitCommand executes a git command and returns the output
func runGitCommand(repoPath string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	if repoPath != "" {
		cmd.Dir = repoPath
	}
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// RequireCleanWorkTree checks if the git working tree is clean
func RequireCleanWorkTree(repoPath string) error {
	state, err := CaptureGitState(repoPath)
	if err != nil {
		return err
	}
	if state.Dirty {
		return ErrDirtyWorkTree
	}
	return nil
}

// CreateBenchmarkBranch creates a new branch for benchmark iteration
func CreateBenchmarkBranch(repoPath, branchName string) error {
	_, err := runGitCommand(repoPath, "checkout", "-b", branchName)
	return err
}

// SwitchBranch switches to an existing branch
func SwitchBranch(repoPath, branchName string) error {
	_, err := runGitCommand(repoPath, "checkout", branchName)
	return err
}

// CommitChanges commits current changes with a message
func CommitChanges(repoPath, message string) (string, error) {
	// Add all changes
	if _, err := runGitCommand(repoPath, "add", "-A"); err != nil {
		return "", err
	}

	// Commit
	if _, err := runGitCommand(repoPath, "commit", "-m", message); err != nil {
		return "", err
	}

	// Get new commit hash
	hash, err := runGitCommand(repoPath, "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}

	return hash, nil
}

// RevertToCommit reverts to a specific commit (hard reset)
func RevertToCommit(repoPath, commitHash string) error {
	_, err := runGitCommand(repoPath, "reset", "--hard", commitHash)
	return err
}

// GetCommitMessage gets the commit message for a hash
func GetCommitMessage(repoPath, commitHash string) (string, error) {
	return runGitCommand(repoPath, "log", "-1", "--format=%s", commitHash)
}
