package locomo

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/MycelicMemory/ultrathink/internal/ai"
	"github.com/MycelicMemory/ultrathink/internal/database"
	"github.com/MycelicMemory/ultrathink/internal/search"
)

// Retriever defines the interface for memory retrieval strategies
type Retriever interface {
	// Retrieve fetches relevant memories for a question
	Retrieve(question string, convID string, topK int) ([]*database.Memory, error)

	// Strategy returns the strategy name
	Strategy() RetrievalStrategy
}

// RetrieverConfig contains configuration for building retrievers
type RetrieverConfig struct {
	DB        *database.Database
	Search    *search.Engine
	AIManager *ai.Manager
	Ingester  *Ingester
}

// NewRetriever creates a retriever for the specified strategy
func NewRetriever(strategy RetrievalStrategy, cfg *RetrieverConfig) (Retriever, error) {
	switch strategy {
	case StrategyDirect:
		return &DirectRetriever{
			db:       cfg.DB,
			ingester: cfg.Ingester,
		}, nil
	case StrategyDialogRAG:
		return &DialogRAGRetriever{
			search:   cfg.Search,
			ingester: cfg.Ingester,
		}, nil
	case StrategyObservationRAG:
		return &ObservationRAGRetriever{
			search:   cfg.Search,
			ai:       cfg.AIManager,
			ingester: cfg.Ingester,
		}, nil
	case StrategySummaryRAG:
		return &SummaryRAGRetriever{
			search:   cfg.Search,
			ai:       cfg.AIManager,
			ingester: cfg.Ingester,
		}, nil
	default:
		return nil, fmt.Errorf("unknown retrieval strategy: %s", strategy)
	}
}

// DirectRetriever returns all memories for a conversation
// This simulates putting the entire conversation in context
type DirectRetriever struct {
	db       *database.Database
	ingester *Ingester
}

func (r *DirectRetriever) Strategy() RetrievalStrategy {
	return StrategyDirect
}

func (r *DirectRetriever) Retrieve(question string, convID string, topK int) ([]*database.Memory, error) {
	// Get all memories for this conversation
	memories, err := r.ingester.GetConversationMemories(convID)
	if err != nil {
		return nil, err
	}

	// Sort by creation time for chronological order
	sort.Slice(memories, func(i, j int) bool {
		return memories[i].CreatedAt.Before(memories[j].CreatedAt)
	})

	// If topK is specified and less than total, truncate from the beginning
	// (keeping most recent context)
	if topK > 0 && len(memories) > topK {
		memories = memories[len(memories)-topK:]
	}

	return memories, nil
}

// DialogRAGRetriever uses semantic search over dialogue turns
type DialogRAGRetriever struct {
	search   *search.Engine
	ingester *Ingester
}

func (r *DialogRAGRetriever) Strategy() RetrievalStrategy {
	return StrategyDialogRAG
}

func (r *DialogRAGRetriever) Retrieve(question string, convID string, topK int) ([]*database.Memory, error) {
	if topK <= 0 {
		topK = 10
	}

	// Search for relevant memories using the question as query
	results, err := r.search.Search(&search.SearchOptions{
		Query:      question,
		SearchType: search.SearchTypeHybrid, // Use hybrid for best results
		Limit:      topK,
		Domain:     LoCoMoDomain,
		Tags:       []string{"conv_" + convID},
	})
	if err != nil {
		return nil, err
	}

	// Extract memories from results
	memories := make([]*database.Memory, len(results))
	for i, result := range results {
		memories[i] = result.Memory
	}

	return memories, nil
}

// ObservationRAGRetriever searches over pre-generated observations
// Observations are higher-level facts derived from conversations
type ObservationRAGRetriever struct {
	search   *search.Engine
	ai       *ai.Manager
	ingester *Ingester
}

func (r *ObservationRAGRetriever) Strategy() RetrievalStrategy {
	return StrategyObservationRAG
}

func (r *ObservationRAGRetriever) Retrieve(question string, convID string, topK int) ([]*database.Memory, error) {
	if topK <= 0 {
		topK = 10
	}

	// First try to find existing observation memories
	results, err := r.search.Search(&search.SearchOptions{
		Query:      question,
		SearchType: search.SearchTypeHybrid,
		Limit:      topK,
		Domain:     LoCoMoDomain,
		Tags:       []string{"conv_" + convID, "observation"},
	})
	if err != nil {
		return nil, err
	}

	// If we found observations, return them
	if len(results) > 0 {
		memories := make([]*database.Memory, len(results))
		for i, result := range results {
			memories[i] = result.Memory
		}
		return memories, nil
	}

	// Fallback to dialog RAG if no observations available
	dialogRetriever := &DialogRAGRetriever{
		search:   r.search,
		ingester: r.ingester,
	}
	return dialogRetriever.Retrieve(question, convID, topK)
}

// SummaryRAGRetriever searches over session summaries
type SummaryRAGRetriever struct {
	search   *search.Engine
	ai       *ai.Manager
	ingester *Ingester
}

func (r *SummaryRAGRetriever) Strategy() RetrievalStrategy {
	return StrategySummaryRAG
}

func (r *SummaryRAGRetriever) Retrieve(question string, convID string, topK int) ([]*database.Memory, error) {
	if topK <= 0 {
		topK = 10
	}

	// First try to find existing summary memories
	results, err := r.search.Search(&search.SearchOptions{
		Query:      question,
		SearchType: search.SearchTypeHybrid,
		Limit:      topK,
		Domain:     LoCoMoDomain,
		Tags:       []string{"conv_" + convID, "summary"},
	})
	if err != nil {
		return nil, err
	}

	// If we found summaries, return them
	if len(results) > 0 {
		memories := make([]*database.Memory, len(results))
		for i, result := range results {
			memories[i] = result.Memory
		}
		return memories, nil
	}

	// Fallback to dialog RAG if no summaries available
	dialogRetriever := &DialogRAGRetriever{
		search:   r.search,
		ingester: r.ingester,
	}
	return dialogRetriever.Retrieve(question, convID, topK)
}

// ContextBuilder formats retrieved memories for LLM context
type ContextBuilder struct {
	MaxTokens int // Approximate token limit
}

// NewContextBuilder creates a new context builder
func NewContextBuilder(maxTokens int) *ContextBuilder {
	if maxTokens <= 0 {
		maxTokens = 4000 // Default context size
	}
	return &ContextBuilder{MaxTokens: maxTokens}
}

// BuildContext formats memories into a context string for the LLM
func (c *ContextBuilder) BuildContext(memories []*database.Memory, question string) string {
	var sb strings.Builder

	sb.WriteString("You are answering questions based on a conversation history.\n")
	sb.WriteString("Use ONLY the information provided below to answer. If the answer is not in the context, say so.\n\n")

	sb.WriteString("=== Conversation Context ===\n\n")

	// Estimate tokens (rough: 4 chars per token)
	currentTokens := sb.Len() / 4

	for i, mem := range memories {
		content := mem.Content

		// Truncate if we're running low on tokens
		memTokens := len(content) / 4
		if currentTokens+memTokens > c.MaxTokens-200 { // Reserve for question
			break
		}

		sb.WriteString(fmt.Sprintf("[%d] %s\n\n", i+1, content))
		currentTokens += memTokens + 5
	}

	sb.WriteString("=== Question ===\n\n")
	sb.WriteString(question)
	sb.WriteString("\n\n")
	sb.WriteString("Answer concisely based only on the context above:\n")

	return sb.String()
}

// AnswerGenerator generates answers using the AI manager
type AnswerGenerator struct {
	ai             *ai.Manager
	contextBuilder *ContextBuilder
}

// NewAnswerGenerator creates a new answer generator
func NewAnswerGenerator(aiManager *ai.Manager, maxContextTokens int) *AnswerGenerator {
	return &AnswerGenerator{
		ai:             aiManager,
		contextBuilder: NewContextBuilder(maxContextTokens),
	}
}

// GenerateAnswer generates an answer to a question given retrieved memories
func (g *AnswerGenerator) GenerateAnswer(question string, memories []*database.Memory) (string, error) {
	if g.ai == nil {
		return "", fmt.Errorf("AI manager not available")
	}

	// Check if chat is available
	status := g.ai.GetStatus()
	if !status.OllamaEnabled || !status.OllamaAvailable {
		return "", fmt.Errorf("Ollama is not available")
	}

	// Extract content from memories
	contents := make([]string, len(memories))
	for i, mem := range memories {
		contents[i] = mem.Content
	}

	// Use AI manager's analysis capability
	ctx := context.Background()
	result, err := g.ai.Analyze(ctx, &ai.AnalysisOptions{
		Type:     "question",
		Question: question,
		Limit:    len(memories),
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate answer: %w", err)
	}

	// Clean up the response
	answer := strings.TrimSpace(result.Answer)
	answer = ExtractAnswer(answer)

	return answer, nil
}

// MultiQueryRetriever performs multiple retrieval passes for complex questions
type MultiQueryRetriever struct {
	baseRetriever Retriever
	ai            *ai.Manager
}

// NewMultiQueryRetriever creates a retriever that handles multi-hop questions
func NewMultiQueryRetriever(base Retriever, aiManager *ai.Manager) *MultiQueryRetriever {
	return &MultiQueryRetriever{
		baseRetriever: base,
		ai:            aiManager,
	}
}

// Retrieve performs multi-pass retrieval for complex questions
func (r *MultiQueryRetriever) Retrieve(question string, convID string, topK int) ([]*database.Memory, error) {
	// For multi-hop questions, we might want to:
	// 1. Break down the question into sub-questions
	// 2. Retrieve for each sub-question
	// 3. Merge and deduplicate results

	// For now, just do enhanced retrieval with more results
	perQueryK := topK / 2
	if perQueryK < 5 {
		perQueryK = 5
	}

	// First retrieval pass
	memories1, err := r.baseRetriever.Retrieve(question, convID, perQueryK)
	if err != nil {
		return nil, err
	}

	// Extract entities/topics from the question for a second pass
	// This is a simplified approach - could use AI for better extraction
	keywords := extractKeywords(question)
	if len(keywords) == 0 {
		return memories1, nil
	}

	// Second retrieval pass with keywords
	keywordQuery := strings.Join(keywords, " ")
	memories2, err := r.baseRetriever.Retrieve(keywordQuery, convID, perQueryK)
	if err != nil {
		return memories1, nil // Return first pass if second fails
	}

	// Merge and deduplicate
	return mergeMemories(memories1, memories2), nil
}

func (r *MultiQueryRetriever) Strategy() RetrievalStrategy {
	return r.baseRetriever.Strategy()
}

// extractKeywords extracts important keywords from a question
func extractKeywords(question string) []string {
	// Simple keyword extraction - remove stopwords and short words
	stopwords := map[string]bool{
		"what": true, "when": true, "where": true, "who": true, "how": true,
		"why": true, "which": true, "is": true, "are": true, "was": true,
		"were": true, "the": true, "a": true, "an": true, "and": true,
		"or": true, "but": true, "in": true, "on": true, "at": true,
		"to": true, "for": true, "of": true, "with": true, "by": true,
		"from": true, "did": true, "does": true, "do": true, "has": true,
		"have": true, "had": true, "be": true, "been": true, "being": true,
		"that": true, "this": true, "it": true, "they": true, "them": true,
		"their": true, "about": true,
	}

	words := strings.Fields(strings.ToLower(question))
	var keywords []string

	for _, word := range words {
		// Clean word
		word = strings.Trim(word, "?.,!\"'")
		if len(word) <= 2 {
			continue
		}
		if stopwords[word] {
			continue
		}
		keywords = append(keywords, word)
	}

	return keywords
}

// mergeMemories merges two memory slices, removing duplicates
func mergeMemories(a, b []*database.Memory) []*database.Memory {
	seen := make(map[string]bool)
	var result []*database.Memory

	for _, mem := range a {
		if !seen[mem.ID] {
			seen[mem.ID] = true
			result = append(result, mem)
		}
	}

	for _, mem := range b {
		if !seen[mem.ID] {
			seen[mem.ID] = true
			result = append(result, mem)
		}
	}

	return result
}
