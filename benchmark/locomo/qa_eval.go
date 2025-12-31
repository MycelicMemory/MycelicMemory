package locomo

import (
	"context"
	"fmt"
	"time"

	"github.com/MycelicMemory/ultrathink/internal/ai"
	"github.com/MycelicMemory/ultrathink/internal/database"
	"github.com/MycelicMemory/ultrathink/internal/search"
)

// QAEvaluator evaluates QA questions against the LoCoMo benchmark
type QAEvaluator struct {
	db        *database.Database
	search    *search.Engine
	ai        *ai.Manager
	ingester  *Ingester
	retriever Retriever
	generator *AnswerGenerator
	config    *EvaluationConfig
}

// NewQAEvaluator creates a new QA evaluator
func NewQAEvaluator(
	db *database.Database,
	searchEngine *search.Engine,
	aiManager *ai.Manager,
	ingester *Ingester,
	config *EvaluationConfig,
) (*QAEvaluator, error) {
	// Create retriever based on config
	retriever, err := NewRetriever(config.RetrievalStrategy, &RetrieverConfig{
		DB:        db,
		Search:    searchEngine,
		AIManager: aiManager,
		Ingester:  ingester,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create retriever: %w", err)
	}

	// Create answer generator
	generator := NewAnswerGenerator(aiManager, 4000)

	return &QAEvaluator{
		db:        db,
		search:    searchEngine,
		ai:        aiManager,
		ingester:  ingester,
		retriever: retriever,
		generator: generator,
		config:    config,
	}, nil
}

// Evaluate runs the full QA evaluation
func (e *QAEvaluator) Evaluate(dataset *Dataset) (*BenchmarkResults, error) {
	start := time.Now()

	results := &BenchmarkResults{
		Benchmark:  "locomo",
		Version:    "0.1.0", // TODO: Get from version package
		Timestamp:  time.Now(),
		Strategy:   e.config.RetrievalStrategy,
		Config:     *e.config,
		Categories: make(map[QuestionCategory]Metrics),
		Questions:  []QuestionResult{},
	}

	// Get AI model info
	if e.ai != nil {
		status := e.ai.GetStatus()
		if status.OllamaAvailable {
			results.Model = status.OllamaModel
		}
	}

	// Filter conversations if specified
	conversations := dataset.Conversations
	if len(e.config.ConversationIDs) > 0 {
		conversations = filterConversations(dataset.Conversations, e.config.ConversationIDs)
	}

	// Evaluate each conversation
	for _, conv := range conversations {
		if e.config.Verbose {
			log.Info("evaluating conversation", "id", conv.ID, "qa_count", len(conv.QA))
		}

		convResults, err := e.evaluateConversation(&conv)
		if err != nil {
			log.Error("failed to evaluate conversation", "id", conv.ID, "error", err)
			continue
		}

		results.Questions = append(results.Questions, convResults...)
	}

	// Calculate aggregate metrics
	results.Overall = CalculateBatchMetrics(results.Questions)
	results.Categories = CalculateCategoryMetrics(results.Questions)
	results.Duration = time.Since(start)

	log.Info("evaluation complete",
		"questions", len(results.Questions),
		"f1", fmt.Sprintf("%.2f", results.Overall.F1),
		"duration", results.Duration)

	return results, nil
}

// evaluateConversation evaluates all QA pairs for a single conversation
func (e *QAEvaluator) evaluateConversation(conv *Conversation) ([]QuestionResult, error) {
	var results []QuestionResult

	for _, qa := range conv.QA {
		// Skip if filtering by category and this doesn't match
		if e.config.Category != "" && qa.Category != e.config.Category {
			continue
		}

		result, err := e.evaluateQuestion(conv.ID, &qa)
		if err != nil {
			if e.config.Verbose {
				log.Warn("failed to evaluate question",
					"conv_id", conv.ID,
					"question", truncate(qa.Question, 50),
					"error", err)
			}
			// Record failed evaluation
			result = &QuestionResult{
				ConversationID:  conv.ID,
				Question:        qa.Question,
				Category:        qa.Category,
				GroundTruth:     qa.Answer,
				GeneratedAnswer: "[ERROR: " + err.Error() + "]",
				F1:              0,
				Precision:       0,
				Recall:          0,
			}
		}

		results = append(results, *result)
	}

	return results, nil
}

// evaluateQuestion evaluates a single QA pair
func (e *QAEvaluator) evaluateQuestion(convID string, qa *QAAnnotation) (*QuestionResult, error) {
	if e.config.Verbose {
		log.Debug("evaluating question",
			"conv_id", convID,
			"category", qa.Category,
			"question", truncate(qa.Question, 80))
	}

	// Retrieve relevant memories
	topK := e.config.TopK
	if topK <= 0 {
		topK = 10
	}

	memories, err := e.retriever.Retrieve(qa.Question, convID, topK)
	if err != nil {
		return nil, fmt.Errorf("retrieval failed: %w", err)
	}

	// Check if evidence was found (for metrics)
	evidenceFound := e.checkEvidenceFound(convID, qa.Evidence, memories)

	// Generate answer
	var generatedAnswer string
	if e.generator != nil && e.ai != nil {
		generatedAnswer, err = e.generateAnswerWithContext(qa.Question, memories)
		if err != nil {
			// If generation fails, fall back to simple extraction
			generatedAnswer = e.extractSimpleAnswer(memories, qa.Question)
		}
	} else {
		// No AI available, use simple extraction
		generatedAnswer = e.extractSimpleAnswer(memories, qa.Question)
	}

	// Calculate F1 score
	f1, precision, recall := CalculateF1(generatedAnswer, qa.Answer)

	result := &QuestionResult{
		ConversationID:    convID,
		Question:          qa.Question,
		Category:          qa.Category,
		GroundTruth:       qa.Answer,
		GeneratedAnswer:   generatedAnswer,
		RetrievedMemories: len(memories),
		F1:                f1,
		Precision:         precision,
		Recall:            recall,
		EvidenceFound:     evidenceFound,
	}

	if e.config.Verbose {
		log.Debug("question evaluated",
			"f1", fmt.Sprintf("%.3f", f1),
			"evidence_found", evidenceFound,
			"memories", len(memories))
	}

	return result, nil
}

// generateAnswerWithContext generates an answer using the AI with memory context
func (e *QAEvaluator) generateAnswerWithContext(question string, memories []*database.Memory) (string, error) {
	// Extract content from memories
	contents := make([]string, len(memories))
	for i, mem := range memories {
		contents[i] = mem.Content
	}

	// Use the AI manager's analysis capability
	ctx := context.Background()
	result, err := e.ai.Analyze(ctx, &ai.AnalysisOptions{
		Type:     "question",
		Question: question,
	})
	if err != nil {
		return "", err
	}

	return result.Answer, nil
}

// extractSimpleAnswer does basic answer extraction when AI is unavailable
func (e *QAEvaluator) extractSimpleAnswer(memories []*database.Memory, question string) string {
	if len(memories) == 0 {
		return "No relevant information found."
	}

	// Return the most relevant memory's content (simplified)
	return memories[0].Content
}

// checkEvidenceFound checks if the retrieved memories contain the evidence
func (e *QAEvaluator) checkEvidenceFound(convID string, evidence []string, memories []*database.Memory) bool {
	if len(evidence) == 0 {
		return true // No evidence required
	}

	// Get memory IDs from the evidence dialogue IDs
	evidenceMemIDs := make(map[string]bool)
	for _, diaID := range evidence {
		if memID, ok := e.ingester.GetMemoryForDialogue(convID, diaID); ok {
			evidenceMemIDs[memID] = true
		}
	}

	// Check if any retrieved memory is in the evidence set
	for _, mem := range memories {
		if evidenceMemIDs[mem.ID] {
			return true
		}
	}

	return false
}

// EvaluateSingleQuestion evaluates a single question (useful for debugging)
func (e *QAEvaluator) EvaluateSingleQuestion(convID, question, groundTruth string, category QuestionCategory) (*QuestionResult, error) {
	qa := &QAAnnotation{
		Question: question,
		Answer:   groundTruth,
		Category: category,
	}

	return e.evaluateQuestion(convID, qa)
}

// Helper functions

func filterConversations(conversations []Conversation, ids []string) []Conversation {
	idSet := make(map[string]bool)
	for _, id := range ids {
		idSet[id] = true
	}

	var filtered []Conversation
	for _, conv := range conversations {
		if idSet[conv.ID] {
			filtered = append(filtered, conv)
		}
	}
	return filtered
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// QAEvaluatorStats provides statistics about the evaluation
type QAEvaluatorStats struct {
	TotalQuestions     int
	EvaluatedQuestions int
	SkippedQuestions   int
	ByCategory         map[QuestionCategory]int
}

// GetStats returns statistics about the QA questions in a dataset
func GetQAStats(dataset *Dataset, category QuestionCategory) *QAEvaluatorStats {
	stats := &QAEvaluatorStats{
		ByCategory: make(map[QuestionCategory]int),
	}

	for _, conv := range dataset.Conversations {
		for _, qa := range conv.QA {
			stats.TotalQuestions++
			stats.ByCategory[qa.Category]++

			if category == "" || qa.Category == category {
				stats.EvaluatedQuestions++
			} else {
				stats.SkippedQuestions++
			}
		}
	}

	return stats
}

// QuickEval performs a quick evaluation on a subset of questions
func (e *QAEvaluator) QuickEval(dataset *Dataset, maxQuestions int) (*BenchmarkResults, error) {
	// Create a limited dataset
	limited := &Dataset{
		Conversations: make([]Conversation, 0),
	}

	questionCount := 0
	for _, conv := range dataset.Conversations {
		if questionCount >= maxQuestions {
			break
		}

		limitedConv := conv
		limitedConv.QA = make([]QAAnnotation, 0)

		for _, qa := range conv.QA {
			if questionCount >= maxQuestions {
				break
			}
			limitedConv.QA = append(limitedConv.QA, qa)
			questionCount++
		}

		if len(limitedConv.QA) > 0 {
			limited.Conversations = append(limited.Conversations, limitedConv)
		}
	}

	return e.Evaluate(limited)
}
