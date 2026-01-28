package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/MycelicMemory/mycelicmemory/internal/database"
	"github.com/MycelicMemory/mycelicmemory/internal/logging"
	"github.com/MycelicMemory/mycelicmemory/internal/vector"
	"github.com/MycelicMemory/mycelicmemory/pkg/config"
)

var log = logging.GetLogger("ai")

// Manager coordinates all AI operations
// This is the central hub for AI capabilities used by REST API in Phase 5
type Manager struct {
	ollama      *OllamaClient
	qdrant      *vector.QdrantClient
	db          *database.Database
	config      *config.Config
	mu          sync.RWMutex
	initialized bool
}

// NewManager creates a new AI manager
func NewManager(db *database.Database, cfg *config.Config) *Manager {
	return &Manager{
		ollama: NewOllamaClient(&cfg.Ollama),
		qdrant: vector.NewQdrantClient(&cfg.Qdrant),
		db:     db,
		config: cfg,
	}
}

// Initialize initializes AI services
func (m *Manager) Initialize(ctx context.Context) error {
	log.Info("initializing AI services")

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.initialized {
		log.Debug("AI services already initialized")
		return nil
	}

	// Initialize Qdrant collection if enabled
	if m.qdrant.IsEnabled() && m.qdrant.IsAvailable() {
		log.Debug("initializing Qdrant collection")
		if err := m.qdrant.InitCollection(ctx); err != nil {
			log.Warn("failed to initialize Qdrant collection", "error", err)
		}
	}

	// Check Ollama availability
	if m.ollama.IsEnabled() {
		if m.ollama.IsAvailable() {
			log.Info("Ollama available", "model", m.ollama.ChatModel())
		} else {
			log.Warn("Ollama enabled but not available")
		}
	}

	m.initialized = true
	log.Info("AI services initialized")
	return nil
}

// Status returns the status of AI services
type Status struct {
	OllamaEnabled   bool   `json:"ollama_enabled"`
	OllamaAvailable bool   `json:"ollama_available"`
	OllamaModel     string `json:"ollama_model,omitempty"`
	QdrantEnabled   bool   `json:"qdrant_enabled"`
	QdrantAvailable bool   `json:"qdrant_available"`
	VectorCount     int64  `json:"vector_count,omitempty"`
}

// GetStatus returns the current status of AI services
func (m *Manager) GetStatus() *Status {
	status := &Status{
		OllamaEnabled:   m.ollama.IsEnabled(),
		OllamaAvailable: m.ollama.IsAvailable(),
		QdrantEnabled:   m.qdrant.IsEnabled(),
		QdrantAvailable: m.qdrant.IsAvailable(),
	}

	if status.OllamaAvailable {
		status.OllamaModel = m.ollama.ChatModel()
	}

	if status.QdrantAvailable {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if info, err := m.qdrant.GetCollectionInfo(ctx); err == nil {
			status.VectorCount = info.VectorCount
		}
	}

	return status
}

// SemanticSearchOptions contains options for semantic search
type SemanticSearchOptions struct {
	Query        string
	Limit        int
	MinScore     float64
	SessionID    string
	Domain       string
	WithMetadata bool
}

// SemanticSearchResult represents a semantic search result
type SemanticSearchResult struct {
	MemoryID  string  `json:"memory_id"`
	Score     float64 `json:"score"`
	Content   string  `json:"content,omitempty"`
	Domain    string  `json:"domain,omitempty"`
	SessionID string  `json:"session_id,omitempty"`
}

// SemanticSearch performs vector similarity search
// VERIFIED: Matches local-memory semantic search behavior
func (m *Manager) SemanticSearch(ctx context.Context, opts *SemanticSearchOptions) ([]SemanticSearchResult, error) {
	if !m.ollama.IsEnabled() || !m.qdrant.IsEnabled() {
		return nil, fmt.Errorf("semantic search requires Ollama and Qdrant to be enabled")
	}

	if !m.ollama.IsAvailable() || !m.qdrant.IsAvailable() {
		return nil, fmt.Errorf("semantic search services not available")
	}

	// Generate query embedding
	embedding, err := m.ollama.GenerateEmbedding(ctx, opts.Query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Search Qdrant
	limit := opts.Limit
	if limit <= 0 {
		limit = 10
	}

	// Build filter if needed
	var filter map[string]interface{}
	if opts.SessionID != "" || opts.Domain != "" {
		must := []map[string]interface{}{}
		if opts.SessionID != "" {
			must = append(must, map[string]interface{}{
				"key":   "session_id",
				"match": map[string]interface{}{"value": opts.SessionID},
			})
		}
		if opts.Domain != "" {
			must = append(must, map[string]interface{}{
				"key":   "domain",
				"match": map[string]interface{}{"value": opts.Domain},
			})
		}
		filter = map[string]interface{}{"must": must}
	}

	searchResults, err := m.qdrant.Search(ctx, &vector.SearchOptions{
		Vector:      embedding,
		Limit:       limit,
		MinScore:    opts.MinScore,
		Filter:      filter,
		WithPayload: true,
	})
	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}

	// Convert results
	results := make([]SemanticSearchResult, len(searchResults))
	for i, r := range searchResults {
		result := SemanticSearchResult{
			MemoryID: r.ID,
			Score:    r.Score,
		}

		// Extract metadata from payload
		if r.Payload != nil {
			if content, ok := r.Payload["content"].(string); ok {
				result.Content = content
			}
			if domain, ok := r.Payload["domain"].(string); ok {
				result.Domain = domain
			}
			if sessionID, ok := r.Payload["session_id"].(string); ok {
				result.SessionID = sessionID
			}
		}

		results[i] = result
	}

	return results, nil
}

// IndexMemory indexes a memory for semantic search
func (m *Manager) IndexMemory(ctx context.Context, memory *database.Memory) error {
	if !m.ollama.IsEnabled() || !m.qdrant.IsEnabled() {
		return nil // Silently skip if AI not enabled
	}

	if !m.ollama.IsAvailable() || !m.qdrant.IsAvailable() {
		return nil // Silently skip if services unavailable
	}

	// Generate embedding
	embedding, err := m.ollama.GenerateEmbedding(ctx, memory.Content)
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Store in Qdrant
	payload := map[string]interface{}{
		"content":    memory.Content,
		"session_id": memory.SessionID,
		"domain":     memory.Domain,
		"importance": memory.Importance,
		"created_at": memory.CreatedAt.Format(time.RFC3339),
	}

	if err := m.qdrant.Upsert(ctx, memory.ID, embedding, payload); err != nil {
		return fmt.Errorf("failed to store vector: %w", err)
	}

	// Store embedding in database for future use
	memory.Embedding = float64SliceToBytes(embedding)

	return nil
}

// DeleteMemoryIndex removes a memory from the vector index
func (m *Manager) DeleteMemoryIndex(ctx context.Context, memoryID string) error {
	if !m.qdrant.IsEnabled() || !m.qdrant.IsAvailable() {
		return nil
	}

	return m.qdrant.Delete(ctx, []string{memoryID})
}

// AnalysisOptions contains options for analysis operations
type AnalysisOptions struct {
	Type      string // "question", "summarize", "analyze", "temporal_patterns"
	Question  string
	Query     string
	Timeframe string // "today", "week", "month", "all"
	Limit     int
	SessionID string
	Domain    string
}

// AnalysisResponse represents an analysis response
type AnalysisResponse struct {
	Type           string             `json:"type"`
	Answer         string             `json:"answer,omitempty"`
	Summary        string             `json:"summary,omitempty"`
	KeyThemes      []string           `json:"key_themes,omitempty"`
	Patterns       []Pattern          `json:"patterns,omitempty"`
	Insights       []string           `json:"insights,omitempty"`
	MemoryCount    int                `json:"memory_count"`
	Sources        []string           `json:"sources,omitempty"`
	SourceMemories []*database.Memory `json:"source_memories,omitempty"`
	Confidence     float64            `json:"confidence,omitempty"`
}

// Pattern is defined in ollama.go

// Analyze performs AI analysis on memories
// VERIFIED: Matches local-memory analysis behavior
func (m *Manager) Analyze(ctx context.Context, opts *AnalysisOptions) (*AnalysisResponse, error) {
	if !m.ollama.IsEnabled() {
		return nil, fmt.Errorf("analysis requires Ollama to be enabled")
	}

	if !m.ollama.IsAvailable() {
		return nil, fmt.Errorf("Ollama is not available")
	}

	// Get relevant memories
	memories, err := m.getMemoriesForAnalysis(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get memories: %w", err)
	}

	if len(memories) == 0 {
		return &AnalysisResponse{
			Type:        opts.Type,
			Answer:      "No memories found matching the criteria.",
			MemoryCount: 0,
		}, nil
	}

	// Extract content
	contents := make([]string, len(memories))
	for i, mem := range memories {
		contents[i] = mem.Content
	}

	var resp *AnalysisResponse

	switch opts.Type {
	case "question":
		resp, err = m.answerQuestion(ctx, opts.Question, contents)
	case "summarize":
		resp, err = m.summarize(ctx, contents, opts.Timeframe)
	case "analyze":
		resp, err = m.analyzePatterns(ctx, contents, opts.Query)
	case "temporal_patterns":
		resp, err = m.analyzeTemporalPatterns(ctx, memories, opts.Query)
	default:
		return nil, fmt.Errorf("unknown analysis type: %s", opts.Type)
	}

	if err != nil {
		return nil, err
	}

	// Add source memories to response
	resp.SourceMemories = memories
	return resp, nil
}

func (m *Manager) getMemoriesForAnalysis(ctx context.Context, opts *AnalysisOptions) ([]*database.Memory, error) {
	filters := &database.MemoryFilters{
		SessionID: opts.SessionID,
		Domain:    opts.Domain,
		Limit:     opts.Limit,
	}

	if filters.Limit <= 0 {
		filters.Limit = 50
	}

	// Apply timeframe filter
	if opts.Timeframe != "" && opts.Timeframe != "all" {
		now := time.Now()
		var startDate time.Time

		switch opts.Timeframe {
		case "today":
			startDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		case "week":
			startDate = now.AddDate(0, 0, -7)
		case "month":
			startDate = now.AddDate(0, -1, 0)
		}

		filters.StartDate = &startDate
	}

	return m.db.ListMemories(filters)
}

func (m *Manager) answerQuestion(ctx context.Context, question string, contents []string) (*AnalysisResponse, error) {
	result, err := m.ollama.AnswerQuestion(ctx, question, contents)
	if err != nil {
		return nil, err
	}

	return &AnalysisResponse{
		Type:        "question",
		Answer:      result.Answer,
		MemoryCount: len(contents),
		Sources:     result.Sources,
		Confidence:  result.Confidence,
	}, nil
}

func (m *Manager) summarize(ctx context.Context, contents []string, timeframe string) (*AnalysisResponse, error) {
	result, err := m.ollama.Summarize(ctx, contents, timeframe)
	if err != nil {
		return nil, err
	}

	return &AnalysisResponse{
		Type:        "summarize",
		Summary:     result.Summary,
		KeyThemes:   result.KeyThemes,
		MemoryCount: result.MemoryCount,
	}, nil
}

func (m *Manager) analyzePatterns(ctx context.Context, contents []string, query string) (*AnalysisResponse, error) {
	result, err := m.ollama.AnalyzePatterns(ctx, contents, query)
	if err != nil {
		return nil, err
	}

	patterns := make([]Pattern, len(result.Patterns))
	for i, p := range result.Patterns {
		patterns[i] = Pattern{
			Name:        p.Name,
			Description: p.Description,
			Examples:    p.Examples,
		}
	}

	return &AnalysisResponse{
		Type:        "analyze",
		Patterns:    patterns,
		Insights:    result.Insights,
		MemoryCount: len(contents),
	}, nil
}

func (m *Manager) analyzeTemporalPatterns(ctx context.Context, memories []*database.Memory, concept string) (*AnalysisResponse, error) {
	// Group memories by time period
	contents := make([]string, len(memories))
	for i, mem := range memories {
		contents[i] = fmt.Sprintf("[%s] %s", mem.CreatedAt.Format("2006-01-02"), mem.Content)
	}

	prompt := "Analyze the temporal patterns and learning progression in these entries"
	if concept != "" {
		prompt += fmt.Sprintf(" related to '%s'", concept)
	}

	result, err := m.ollama.AnalyzePatterns(ctx, contents, prompt)
	if err != nil {
		return nil, err
	}

	return &AnalysisResponse{
		Type:        "temporal_patterns",
		Insights:    result.Insights,
		MemoryCount: len(memories),
	}, nil
}

// DiscoverRelationships discovers potential relationships between memories
func (m *Manager) DiscoverRelationships(ctx context.Context, limit int) ([]RelationshipSuggestion, error) {
	if !m.ollama.IsEnabled() || !m.ollama.IsAvailable() {
		return nil, fmt.Errorf("relationship discovery requires Ollama")
	}

	// Get recent memories
	memories, err := m.db.ListMemories(&database.MemoryFilters{Limit: limit * 2})
	if err != nil {
		return nil, fmt.Errorf("failed to get memories: %w", err)
	}

	if len(memories) < 2 {
		return []RelationshipSuggestion{}, nil
	}

	var suggestions []RelationshipSuggestion

	// Compare pairs (limited to avoid too many API calls)
	maxPairs := limit
	if maxPairs <= 0 {
		maxPairs = 10
	}

	pairCount := 0
	for i := 0; i < len(memories) && pairCount < maxPairs; i++ {
		for j := i + 1; j < len(memories) && pairCount < maxPairs; j++ {
			suggestion, err := m.ollama.SuggestRelationships(
				ctx,
				memories[i].Content,
				memories[j].Content,
				memories[i].ID,
				memories[j].ID,
			)
			if err != nil {
				continue // Skip on error
			}
			if suggestion != nil {
				suggestions = append(suggestions, *suggestion)
			}
			pairCount++
		}
	}

	return suggestions, nil
}

// Ollama returns the Ollama client
func (m *Manager) Ollama() *OllamaClient {
	return m.ollama
}

// Qdrant returns the Qdrant client
func (m *Manager) Qdrant() *vector.QdrantClient {
	return m.qdrant
}

// Helper function to convert float64 slice to bytes
func float64SliceToBytes(floats []float64) []byte {
	// Simple encoding: store as JSON
	data, _ := json.Marshal(floats)
	return data
}

// Helper function to convert bytes to float64 slice
func bytesToFloat64Slice(data []byte) []float64 {
	var floats []float64
	_ = json.Unmarshal(data, &floats)
	return floats
}
