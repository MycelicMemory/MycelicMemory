// Package dependencies provides centralized checking and messaging for optional dependencies.
package dependencies

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/MycelicMemory/mycelicmemory/pkg/config"
)

// Status represents the status of an optional dependency
type Status string

const (
	StatusAvailable   Status = "available"
	StatusUnavailable Status = "unavailable"
	StatusDisabled    Status = "disabled"
	StatusMissing     Status = "missing"
)

// DependencyInfo contains information about a dependency
type DependencyInfo struct {
	Name         string
	Status       Status
	Version      string
	URL          string
	Message      string
	Models       []string // For Ollama, list of available models
	MissingItems []string // Models that are required but missing
}

// CheckResult contains the results of checking all dependencies
type CheckResult struct {
	Ollama DependencyInfo
	Qdrant DependencyInfo
}

// Check checks all optional dependencies and returns their status
func Check(cfg *config.Config) *CheckResult {
	result := &CheckResult{}

	result.Ollama = checkOllama(cfg)
	result.Qdrant = checkQdrant(cfg)

	return result
}

// checkOllama checks Ollama availability and model status
func checkOllama(cfg *config.Config) DependencyInfo {
	info := DependencyInfo{
		Name: "Ollama",
		URL:  cfg.Ollama.BaseURL,
	}

	if !cfg.Ollama.Enabled {
		info.Status = StatusDisabled
		info.Message = "Ollama is disabled in configuration"
		return info
	}

	// Check if Ollama is running
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", cfg.Ollama.BaseURL+"/api/tags", nil)
	if err != nil {
		info.Status = StatusUnavailable
		info.Message = "Failed to create request"
		return info
	}

	resp, err := client.Do(req)
	if err != nil {
		info.Status = StatusMissing
		info.Message = "Ollama is not running or not installed"
		return info
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		info.Status = StatusUnavailable
		info.Message = fmt.Sprintf("Ollama returned status %d", resp.StatusCode)
		return info
	}

	// Parse available models
	var modelsResp struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		info.Status = StatusAvailable
		info.Message = "Ollama is running but could not list models"
		return info
	}

	// Collect available models
	modelSet := make(map[string]bool)
	for _, m := range modelsResp.Models {
		info.Models = append(info.Models, m.Name)
		// Also track base model names (without tags)
		baseName := strings.Split(m.Name, ":")[0]
		modelSet[m.Name] = true
		modelSet[baseName] = true
	}

	// Check for required models
	requiredModels := []string{cfg.Ollama.ChatModel, cfg.Ollama.EmbeddingModel}
	for _, model := range requiredModels {
		baseName := strings.Split(model, ":")[0]
		if !modelSet[model] && !modelSet[baseName] {
			info.MissingItems = append(info.MissingItems, model)
		}
	}

	if len(info.MissingItems) > 0 {
		info.Status = StatusAvailable
		info.Message = fmt.Sprintf("Ollama is running but missing required models: %s", strings.Join(info.MissingItems, ", "))
	} else {
		info.Status = StatusAvailable
		info.Message = "Ollama is running with all required models"
	}

	// Try to get version
	info.Version = getOllamaVersion(cfg.Ollama.BaseURL, client)

	return info
}

func getOllamaVersion(baseURL string, client *http.Client) string {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/api/version", nil)
	if err != nil {
		return ""
	}

	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	var versionResp struct {
		Version string `json:"version"`
	}

	if json.NewDecoder(resp.Body).Decode(&versionResp) == nil {
		return versionResp.Version
	}
	return ""
}

// checkQdrant checks Qdrant availability
func checkQdrant(cfg *config.Config) DependencyInfo {
	info := DependencyInfo{
		Name: "Qdrant",
		URL:  cfg.Qdrant.URL,
	}

	if !cfg.Qdrant.Enabled {
		info.Status = StatusDisabled
		info.Message = "Qdrant is disabled in configuration"
		return info
	}

	// Check if Qdrant is running
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", cfg.Qdrant.URL+"/collections", nil)
	if err != nil {
		info.Status = StatusUnavailable
		info.Message = "Failed to create request"
		return info
	}

	resp, err := client.Do(req)
	if err != nil {
		info.Status = StatusMissing
		info.Message = "Qdrant is not running or not installed"
		return info
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		info.Status = StatusUnavailable
		info.Message = fmt.Sprintf("Qdrant returned status %d", resp.StatusCode)
		return info
	}

	info.Status = StatusAvailable
	info.Message = "Qdrant is running"

	// Try to get version
	info.Version = getQdrantVersion(cfg.Qdrant.URL, client)

	return info
}

func getQdrantVersion(baseURL string, client *http.Client) string {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", baseURL, nil)
	if err != nil {
		return ""
	}

	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	var versionResp struct {
		Version string `json:"version"`
	}

	if json.NewDecoder(resp.Body).Decode(&versionResp) == nil {
		return versionResp.Version
	}
	return ""
}

// HasAnyMissing returns true if any dependencies are missing
func (r *CheckResult) HasAnyMissing() bool {
	return r.Ollama.Status == StatusMissing || r.Qdrant.Status == StatusMissing
}

// HasMissingModels returns true if Ollama is missing required models
func (r *CheckResult) HasMissingModels() bool {
	return len(r.Ollama.MissingItems) > 0
}

// AIFeaturesAvailable returns true if all AI features are available
func (r *CheckResult) AIFeaturesAvailable() bool {
	return r.Ollama.Status == StatusAvailable && len(r.Ollama.MissingItems) == 0
}

// SemanticSearchAvailable returns true if semantic search is available
func (r *CheckResult) SemanticSearchAvailable() bool {
	return r.Ollama.Status == StatusAvailable && r.Qdrant.Status == StatusAvailable && len(r.Ollama.MissingItems) == 0
}

// FormatWarning formats a warning message for display
func FormatWarning(result *CheckResult) string {
	var buf bytes.Buffer

	if result.Ollama.Status == StatusMissing || result.Ollama.Status == StatusUnavailable {
		buf.WriteString("⚠️  WARNING: Ollama is not available - AI features disabled\n")
	} else if len(result.Ollama.MissingItems) > 0 {
		buf.WriteString(fmt.Sprintf("⚠️  WARNING: Missing Ollama models: %s\n", strings.Join(result.Ollama.MissingItems, ", ")))
	}

	if result.Qdrant.Status == StatusMissing || result.Qdrant.Status == StatusUnavailable {
		buf.WriteString("⚠️  WARNING: Qdrant is not available - semantic search disabled\n")
	}

	if buf.Len() > 0 {
		buf.WriteString("   Run 'mycelicmemory doctor' for details and installation instructions.\n")
	}

	return buf.String()
}

// FormatShortWarning formats a brief inline warning
func FormatShortWarning(result *CheckResult) string {
	var warnings []string

	if result.Ollama.Status == StatusMissing || result.Ollama.Status == StatusUnavailable {
		warnings = append(warnings, "Ollama unavailable")
	} else if len(result.Ollama.MissingItems) > 0 {
		warnings = append(warnings, "missing Ollama models")
	}

	if result.Qdrant.Status == StatusMissing || result.Qdrant.Status == StatusUnavailable {
		if result.Qdrant.Status != StatusDisabled {
			warnings = append(warnings, "Qdrant unavailable")
		}
	}

	if len(warnings) > 0 {
		return fmt.Sprintf("[AI: %s]", strings.Join(warnings, ", "))
	}
	return ""
}

// InstallInstructions returns installation instructions for missing dependencies
type InstallInstructions struct {
	Ollama *OllamaInstallInstructions
	Qdrant *QdrantInstallInstructions
}

// OllamaInstallInstructions contains Ollama-specific install steps
type OllamaInstallInstructions struct {
	InstallSteps []string
	ModelSteps   []string
}

// QdrantInstallInstructions contains Qdrant-specific install steps
type QdrantInstallInstructions struct {
	InstallSteps []string
}

// GetInstallInstructions returns installation instructions for missing dependencies
func GetInstallInstructions(result *CheckResult, cfg *config.Config) *InstallInstructions {
	instructions := &InstallInstructions{}

	if result.Ollama.Status == StatusMissing || result.Ollama.Status == StatusUnavailable || len(result.Ollama.MissingItems) > 0 {
		instructions.Ollama = getOllamaInstructions(result, cfg)
	}

	if result.Qdrant.Status == StatusMissing || result.Qdrant.Status == StatusUnavailable {
		instructions.Qdrant = getQdrantInstructions()
	}

	return instructions
}

func getOllamaInstructions(result *CheckResult, cfg *config.Config) *OllamaInstallInstructions {
	instr := &OllamaInstallInstructions{}

	if result.Ollama.Status == StatusMissing || result.Ollama.Status == StatusUnavailable {
		switch runtime.GOOS {
		case "darwin":
			instr.InstallSteps = []string{
				"1. Install Ollama:",
				"   brew install ollama",
				"   OR download from: https://ollama.ai/download",
				"",
				"2. Start Ollama:",
				"   ollama serve",
			}
		case "linux":
			instr.InstallSteps = []string{
				"1. Install Ollama:",
				"   curl -fsSL https://ollama.ai/install.sh | sh",
				"",
				"2. Start Ollama:",
				"   ollama serve",
				"   OR: systemctl start ollama",
			}
		case "windows":
			instr.InstallSteps = []string{
				"1. Install Ollama:",
				"   Download from: https://ollama.ai/download/windows",
				"   OR: winget install Ollama.Ollama",
				"",
				"2. Start Ollama:",
				"   Ollama runs automatically after installation",
				"   OR open 'Ollama' from Start Menu",
			}
		default:
			instr.InstallSteps = []string{
				"1. Install Ollama from: https://ollama.ai",
				"2. Start Ollama: ollama serve",
			}
		}
	}

	// Model installation steps
	if len(result.Ollama.MissingItems) > 0 || result.Ollama.Status == StatusMissing || result.Ollama.Status == StatusUnavailable {
		instr.ModelSteps = []string{
			"3. Pull required models:",
		}
		for _, model := range []string{cfg.Ollama.ChatModel, cfg.Ollama.EmbeddingModel} {
			instr.ModelSteps = append(instr.ModelSteps, fmt.Sprintf("   ollama pull %s", model))
		}
	}

	return instr
}

func getQdrantInstructions() *QdrantInstallInstructions {
	instr := &QdrantInstallInstructions{}

	switch runtime.GOOS {
	case "darwin", "linux":
		instr.InstallSteps = []string{
			"Option 1 - Docker (recommended):",
			"   docker run -p 6333:6333 -v qdrant_storage:/qdrant/storage qdrant/qdrant",
			"",
			"Option 2 - Binary:",
			"   Download from: https://github.com/qdrant/qdrant/releases",
			"   ./qdrant",
		}
	case "windows":
		instr.InstallSteps = []string{
			"Option 1 - Docker Desktop (recommended):",
			"   docker run -p 6333:6333 -v qdrant_storage:/qdrant/storage qdrant/qdrant",
			"",
			"Option 2 - Binary:",
			"   Download from: https://github.com/qdrant/qdrant/releases",
			"   qdrant.exe",
		}
	default:
		instr.InstallSteps = []string{
			"Install via Docker: docker run -p 6333:6333 qdrant/qdrant",
			"OR download from: https://github.com/qdrant/qdrant/releases",
		}
	}

	return instr
}

// FormatDoctorReport formats a detailed doctor report
func FormatDoctorReport(result *CheckResult, cfg *config.Config) string {
	var buf bytes.Buffer

	// Ollama section
	buf.WriteString("Ollama... ")
	switch result.Ollama.Status {
	case StatusAvailable:
		if len(result.Ollama.MissingItems) > 0 {
			buf.WriteString("PARTIAL\n")
		} else {
			buf.WriteString("OK\n")
		}
		buf.WriteString(fmt.Sprintf("  URL: %s\n", result.Ollama.URL))
		if result.Ollama.Version != "" {
			buf.WriteString(fmt.Sprintf("  Version: %s\n", result.Ollama.Version))
		}
		buf.WriteString(fmt.Sprintf("  Chat Model: %s\n", cfg.Ollama.ChatModel))
		buf.WriteString(fmt.Sprintf("  Embedding Model: %s\n", cfg.Ollama.EmbeddingModel))
		if len(result.Ollama.MissingItems) > 0 {
			buf.WriteString(fmt.Sprintf("  ⚠️  Missing Models: %s\n", strings.Join(result.Ollama.MissingItems, ", ")))
		}
		if len(result.Ollama.Models) > 0 {
			buf.WriteString(fmt.Sprintf("  Available Models: %s\n", strings.Join(result.Ollama.Models, ", ")))
		}
	case StatusDisabled:
		buf.WriteString("DISABLED\n")
		buf.WriteString("  AI features are disabled in configuration.\n")
	case StatusMissing, StatusUnavailable:
		buf.WriteString("NOT AVAILABLE\n")
		buf.WriteString(fmt.Sprintf("  %s\n", result.Ollama.Message))
		buf.WriteString("  AI features will be disabled.\n")
	}

	buf.WriteString("\n")

	// Qdrant section
	buf.WriteString("Qdrant... ")
	switch result.Qdrant.Status {
	case StatusAvailable:
		buf.WriteString("OK\n")
		buf.WriteString(fmt.Sprintf("  URL: %s\n", result.Qdrant.URL))
		if result.Qdrant.Version != "" {
			buf.WriteString(fmt.Sprintf("  Version: %s\n", result.Qdrant.Version))
		}
	case StatusDisabled:
		buf.WriteString("DISABLED\n")
		buf.WriteString("  Vector search is disabled in configuration.\n")
	case StatusMissing, StatusUnavailable:
		buf.WriteString("NOT AVAILABLE\n")
		buf.WriteString(fmt.Sprintf("  %s\n", result.Qdrant.Message))
		buf.WriteString("  Semantic search will be disabled.\n")
	}

	// Installation instructions if needed
	instructions := GetInstallInstructions(result, cfg)
	hasInstructions := false

	if instructions.Ollama != nil && (len(instructions.Ollama.InstallSteps) > 0 || len(instructions.Ollama.ModelSteps) > 0) {
		if !hasInstructions {
			buf.WriteString("\n")
			buf.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
			buf.WriteString("INSTALLATION INSTRUCTIONS\n")
			buf.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
			hasInstructions = true
		}
		buf.WriteString("\n")
		buf.WriteString("📦 Ollama Setup:\n")
		for _, step := range instructions.Ollama.InstallSteps {
			buf.WriteString(fmt.Sprintf("%s\n", step))
		}
		for _, step := range instructions.Ollama.ModelSteps {
			buf.WriteString(fmt.Sprintf("%s\n", step))
		}
	}

	if instructions.Qdrant != nil && len(instructions.Qdrant.InstallSteps) > 0 {
		if !hasInstructions {
			buf.WriteString("\n")
			buf.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
			buf.WriteString("INSTALLATION INSTRUCTIONS\n")
			buf.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
			hasInstructions = true
		}
		buf.WriteString("\n")
		buf.WriteString("📦 Qdrant Setup (optional, for semantic search):\n")
		for _, step := range instructions.Qdrant.InstallSteps {
			buf.WriteString(fmt.Sprintf("%s\n", step))
		}
	}

	if hasInstructions {
		buf.WriteString("\n")
		buf.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	}

	return buf.String()
}

// ShouldShowWarning returns true if a warning should be shown for the given command
func ShouldShowWarning(result *CheckResult, command string) bool {
	// Always show warnings for commands that use AI features
	aiCommands := map[string]bool{
		"remember":   true, // indexing
		"search":     true, // semantic search
		"analyze":    true,
		"categorize": true,
		"relate":     true,
	}

	if aiCommands[command] {
		return !result.AIFeaturesAvailable()
	}

	return false
}
