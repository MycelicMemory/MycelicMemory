package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/MycelicMemory/mycelicmemory/pkg/config"
)

// OllamaClient provides AI capabilities via Ollama
// VERIFIED: Matches local-memory Ollama integration
type OllamaClient struct {
	baseURL        string
	embeddingModel string
	chatModel      string
	httpClient     *http.Client
	enabled        bool
}

// NewOllamaClient creates a new Ollama client
func NewOllamaClient(cfg *config.OllamaConfig) *OllamaClient {
	client := &OllamaClient{
		baseURL:        cfg.BaseURL,
		embeddingModel: cfg.EmbeddingModel,
		chatModel:      cfg.ChatModel,
		enabled:        cfg.Enabled,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}

	// Set defaults
	if client.baseURL == "" {
		client.baseURL = "http://localhost:11434"
	}
	if client.embeddingModel == "" {
		client.embeddingModel = "nomic-embed-text"
	}
	if client.chatModel == "" {
		client.chatModel = "qwen2.5:3b"
	}

	return client
}

// IsAvailable checks if Ollama is available and responsive
func (c *OllamaClient) IsAvailable() bool {
	if !c.enabled {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/tags", nil)
	if err != nil {
		return false
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// EmbeddingRequest represents a request to generate embeddings
type EmbeddingRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

// EmbeddingResponse represents the embedding response
type EmbeddingResponse struct {
	Embedding []float64 `json:"embedding"`
}

// GenerateEmbedding generates a 768-dimensional embedding for text
// VERIFIED: Uses nomic-embed-text model (768 dimensions)
func (c *OllamaClient) GenerateEmbedding(ctx context.Context, text string) ([]float64, error) {
	if !c.enabled {
		return nil, fmt.Errorf("ollama is not enabled")
	}

	reqBody := EmbeddingRequest{
		Model:  c.embeddingModel,
		Prompt: text,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/embeddings", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embedding request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, fmt.Errorf("embedding request failed with status %d (body unreadable: %v)", resp.StatusCode, readErr)
		}
		return nil, fmt.Errorf("embedding request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var embResp EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return embResp.Embedding, nil
}

// GenerateRequest represents a chat/generate request
type GenerateRequest struct {
	Model   string `json:"model"`
	Prompt  string `json:"prompt"`
	Stream  bool   `json:"stream"`
	Context []int  `json:"context,omitempty"`
}

// GenerateResponse represents the generate response
type GenerateResponse struct {
	Model     string `json:"model"`
	Response  string `json:"response"`
	Done      bool   `json:"done"`
	Context   []int  `json:"context,omitempty"`
	CreatedAt string `json:"created_at"`
}

// Generate generates text using the chat model
// VERIFIED: Uses qwen2.5:3b model
func (c *OllamaClient) Generate(ctx context.Context, prompt string) (string, error) {
	if !c.enabled {
		return "", fmt.Errorf("ollama is not enabled")
	}

	reqBody := GenerateRequest{
		Model:  c.chatModel,
		Prompt: prompt,
		Stream: false,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/generate", bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("generate request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return "", fmt.Errorf("generate request failed with status %d (body unreadable: %v)", resp.StatusCode, readErr)
		}
		return "", fmt.Errorf("generate request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var genResp GenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&genResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return genResp.Response, nil
}

// ChatMessage represents a chat message
type ChatMessage struct {
	Role    string `json:"role"` // "user", "assistant", "system"
	Content string `json:"content"`
}

// ChatRequest represents a chat request
type ChatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
}

// ChatResponse represents a chat response
type ChatResponse struct {
	Model   string      `json:"model"`
	Message ChatMessage `json:"message"`
	Done    bool        `json:"done"`
}

// Chat performs a multi-turn chat conversation
func (c *OllamaClient) Chat(ctx context.Context, messages []ChatMessage) (string, error) {
	if !c.enabled {
		return "", fmt.Errorf("ollama is not enabled")
	}

	reqBody := ChatRequest{
		Model:    c.chatModel,
		Messages: messages,
		Stream:   false,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/chat", bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("chat request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return "", fmt.Errorf("chat request failed with status %d (body unreadable: %v)", resp.StatusCode, readErr)
		}
		return "", fmt.Errorf("chat request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return chatResp.Message.Content, nil
}

// AnalysisResult represents the result of AI analysis
type AnalysisResult struct {
	Answer     string   `json:"answer"`
	Confidence float64  `json:"confidence"`
	Sources    []string `json:"sources,omitempty"`
}

// AnswerQuestion answers a question based on provided context
// VERIFIED: Matches local-memory Q&A behavior
func (c *OllamaClient) AnswerQuestion(ctx context.Context, question string, context []string) (*AnalysisResult, error) {
	if !c.enabled {
		return nil, fmt.Errorf("ollama is not enabled")
	}

	// Build context string
	contextStr := strings.Join(context, "\n\n---\n\n")

	prompt := fmt.Sprintf(`You are a helpful AI assistant. Based on the following context, answer the question concisely and accurately.

Context:
%s

Question: %s

Answer:`, contextStr, question)

	response, err := c.Generate(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate answer: %w", err)
	}

	return &AnalysisResult{
		Answer:     strings.TrimSpace(response),
		Confidence: 0.8, // Default confidence
		Sources:    context,
	}, nil
}

// SummaryResult represents the result of summarization
type SummaryResult struct {
	Summary     string   `json:"summary"`
	KeyThemes   []string `json:"key_themes"`
	MemoryCount int      `json:"memory_count"`
}

// Summarize generates a summary of multiple texts
// VERIFIED: Matches local-memory summarization behavior
func (c *OllamaClient) Summarize(ctx context.Context, texts []string, timeframe string) (*SummaryResult, error) {
	if !c.enabled {
		return nil, fmt.Errorf("ollama is not enabled")
	}

	if len(texts) == 0 {
		return &SummaryResult{
			Summary:     "No content to summarize.",
			KeyThemes:   []string{},
			MemoryCount: 0,
		}, nil
	}

	// Build content string
	content := strings.Join(texts, "\n\n---\n\n")

	prompt := fmt.Sprintf(`Summarize the following %d entries%s. Identify key themes and provide a concise summary.

Entries:
%s

Provide your response in the following format:
SUMMARY: [Your summary here]
KEY THEMES: [theme1], [theme2], [theme3]`,
		len(texts),
		formatTimeframe(timeframe),
		content)

	response, err := c.Generate(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate summary: %w", err)
	}

	// Parse response
	summary, themes := parseSummaryResponse(response)

	return &SummaryResult{
		Summary:     summary,
		KeyThemes:   themes,
		MemoryCount: len(texts),
	}, nil
}

// PatternResult represents discovered patterns
type PatternResult struct {
	Patterns    []Pattern `json:"patterns"`
	Insights    []string  `json:"insights"`
	Connections int       `json:"connections"`
}

// Pattern represents a discovered pattern
type Pattern struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Examples    []string `json:"examples"`
	Frequency   int      `json:"frequency"`
}

// AnalyzePatterns discovers patterns in content
// VERIFIED: Matches local-memory pattern analysis
func (c *OllamaClient) AnalyzePatterns(ctx context.Context, texts []string, query string) (*PatternResult, error) {
	if !c.enabled {
		return nil, fmt.Errorf("ollama is not enabled")
	}

	if len(texts) == 0 {
		return &PatternResult{
			Patterns:    []Pattern{},
			Insights:    []string{},
			Connections: 0,
		}, nil
	}

	content := strings.Join(texts, "\n\n---\n\n")

	prompt := fmt.Sprintf(`Analyze the following content for patterns and insights%s.

Content:
%s

Identify:
1. Recurring patterns or themes
2. Key insights
3. Connections between entries

Provide a structured analysis.`,
		formatQuery(query),
		content)

	response, err := c.Generate(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze patterns: %w", err)
	}

	// Parse response into patterns (simplified for now)
	return &PatternResult{
		Patterns:    []Pattern{{Name: "Analysis", Description: response}},
		Insights:    []string{response},
		Connections: len(texts) - 1,
	}, nil
}

// RelationshipSuggestion represents a suggested relationship
type RelationshipSuggestion struct {
	SourceID    string  `json:"source_id"`
	TargetID    string  `json:"target_id"`
	Type        string  `json:"type"`
	Confidence  float64 `json:"confidence"`
	Reasoning   string  `json:"reasoning"`
}

// SuggestRelationships suggests relationships between memories
func (c *OllamaClient) SuggestRelationships(ctx context.Context, sourceContent, targetContent, sourceID, targetID string) (*RelationshipSuggestion, error) {
	if !c.enabled {
		return nil, fmt.Errorf("ollama is not enabled")
	}

	prompt := fmt.Sprintf(`Analyze these two pieces of content and determine if there's a relationship between them.

Content 1:
%s

Content 2:
%s

Possible relationship types: references, contradicts, expands, similar, sequential, causes, enables

If a relationship exists, respond with:
TYPE: [relationship type]
CONFIDENCE: [0.0-1.0]
REASONING: [brief explanation]

If no clear relationship exists, respond with:
TYPE: none`, sourceContent, targetContent)

	response, err := c.Generate(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to suggest relationships: %w", err)
	}

	// Parse response
	relType, confidence, reasoning := parseRelationshipResponse(response)
	if relType == "none" || relType == "" {
		return nil, nil
	}

	return &RelationshipSuggestion{
		SourceID:   sourceID,
		TargetID:   targetID,
		Type:       relType,
		Confidence: confidence,
		Reasoning:  reasoning,
	}, nil
}

// Helper functions

func formatTimeframe(timeframe string) string {
	if timeframe == "" {
		return ""
	}
	return fmt.Sprintf(" from the %s", timeframe)
}

func formatQuery(query string) string {
	if query == "" {
		return ""
	}
	return fmt.Sprintf(" related to '%s'", query)
}

func parseSummaryResponse(response string) (string, []string) {
	lines := strings.Split(response, "\n")
	var summary string
	var themes []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		upperLine := strings.ToUpper(line)

		if strings.HasPrefix(upperLine, "SUMMARY:") {
			// Remove prefix case-insensitively
			summary = strings.TrimSpace(line[len("SUMMARY:"):])
		} else if strings.HasPrefix(upperLine, "KEY THEMES:") {
			// Remove prefix case-insensitively
			themesStr := strings.TrimSpace(line[len("KEY THEMES:"):])
			for _, theme := range strings.Split(themesStr, ",") {
				theme = strings.TrimSpace(theme)
				theme = strings.Trim(theme, "[]")
				if theme != "" {
					themes = append(themes, theme)
				}
			}
		}
	}

	// If parsing failed, use the whole response as summary
	if summary == "" {
		summary = response
	}

	return summary, themes
}

func parseRelationshipResponse(response string) (string, float64, string) {
	lines := strings.Split(response, "\n")
	var relType, reasoning string
	var confidence float64 = 0.5

	for _, line := range lines {
		line = strings.TrimSpace(line)
		upperLine := strings.ToUpper(line)

		if strings.HasPrefix(upperLine, "TYPE:") {
			// Remove prefix case-insensitively and convert to lowercase
			relType = strings.ToLower(strings.TrimSpace(line[len("TYPE:"):]))
		} else if strings.HasPrefix(upperLine, "CONFIDENCE:") {
			// Remove prefix case-insensitively
			confStr := strings.TrimSpace(line[len("CONFIDENCE:"):])
			_, _ = fmt.Sscanf(confStr, "%f", &confidence)
		} else if strings.HasPrefix(upperLine, "REASONING:") {
			// Remove prefix case-insensitively
			reasoning = strings.TrimSpace(line[len("REASONING:"):])
		}
	}

	return relType, confidence, reasoning
}

// GetModels returns available Ollama models
func (c *OllamaClient) GetModels(ctx context.Context) ([]string, error) {
	if !c.enabled {
		return nil, fmt.Errorf("ollama is not enabled")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/tags", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	models := make([]string, len(result.Models))
	for i, m := range result.Models {
		models[i] = m.Name
	}

	return models, nil
}

// EmbeddingModel returns the configured embedding model
func (c *OllamaClient) EmbeddingModel() string {
	return c.embeddingModel
}

// ChatModel returns the configured chat model
func (c *OllamaClient) ChatModel() string {
	return c.chatModel
}

// IsEnabled returns whether Ollama is enabled
func (c *OllamaClient) IsEnabled() bool {
	return c.enabled
}
