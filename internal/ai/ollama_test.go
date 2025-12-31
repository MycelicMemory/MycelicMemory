package ai

import (
	"context"
	"testing"
	"time"

	"github.com/MycelicMemory/ultrathink/pkg/config"
)

// TestOllamaClient tests the Ollama client functionality
func TestOllamaClient(t *testing.T) {
	cfg := &config.OllamaConfig{
		Enabled:        true,
		BaseURL:        "http://localhost:11434",
		EmbeddingModel: "nomic-embed-text",
		ChatModel:      "qwen2.5:3b",
	}

	client := NewOllamaClient(cfg)

	t.Run("NewOllamaClient", func(t *testing.T) {
		if client == nil {
			t.Fatal("NewOllamaClient should not return nil")
		}
		if !client.IsEnabled() {
			t.Error("Client should be enabled")
		}
		if client.EmbeddingModel() != "nomic-embed-text" {
			t.Errorf("Expected embedding model 'nomic-embed-text', got %s", client.EmbeddingModel())
		}
		if client.ChatModel() != "qwen2.5:3b" {
			t.Errorf("Expected chat model 'qwen2.5:3b', got %s", client.ChatModel())
		}
	})

	t.Run("DefaultValues", func(t *testing.T) {
		emptyClient := NewOllamaClient(&config.OllamaConfig{Enabled: true})
		if emptyClient.EmbeddingModel() != "nomic-embed-text" {
			t.Errorf("Default embedding model should be 'nomic-embed-text', got %s", emptyClient.EmbeddingModel())
		}
		if emptyClient.ChatModel() != "qwen2.5:3b" {
			t.Errorf("Default chat model should be 'qwen2.5:3b', got %s", emptyClient.ChatModel())
		}
	})

	t.Run("DisabledClient", func(t *testing.T) {
		disabledClient := NewOllamaClient(&config.OllamaConfig{Enabled: false})
		if disabledClient.IsEnabled() {
			t.Error("Disabled client should not be enabled")
		}
		if disabledClient.IsAvailable() {
			t.Error("Disabled client should not be available")
		}
	})
}

// TestOllamaClientIntegration tests with actual Ollama server
// Skip if Ollama is not available
func TestOllamaClientIntegration(t *testing.T) {
	cfg := &config.OllamaConfig{
		Enabled:        true,
		BaseURL:        "http://localhost:11434",
		EmbeddingModel: "nomic-embed-text",
		ChatModel:      "qwen2.5:3b",
	}

	client := NewOllamaClient(cfg)

	// Skip if Ollama is not available
	if !client.IsAvailable() {
		t.Skip("Ollama is not available, skipping integration tests")
	}

	t.Run("GenerateEmbedding", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		embedding, err := client.GenerateEmbedding(ctx, "Test embedding text")
		if err != nil {
			t.Fatalf("GenerateEmbedding failed: %v", err)
		}

		// nomic-embed-text produces 768-dimensional embeddings
		if len(embedding) != 768 {
			t.Errorf("Expected 768-dimensional embedding, got %d", len(embedding))
		}

		// Check that values are reasonable
		for i, v := range embedding {
			if v < -10 || v > 10 {
				t.Errorf("Embedding value at %d seems unreasonable: %f", i, v)
				break
			}
		}
	})

	t.Run("GenerateEmbeddingEmpty", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		_, err := client.GenerateEmbedding(ctx, "")
		// Empty text should still work (Ollama handles it)
		if err != nil {
			t.Logf("Empty embedding: %v", err)
		}
	})

	t.Run("Generate", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		response, err := client.Generate(ctx, "Say hello in one word.")
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}

		if response == "" {
			t.Error("Expected non-empty response")
		}
	})

	t.Run("Chat", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		messages := []ChatMessage{
			{Role: "user", Content: "What is 2 + 2?"},
		}

		response, err := client.Chat(ctx, messages)
		if err != nil {
			t.Fatalf("Chat failed: %v", err)
		}

		if response == "" {
			t.Error("Expected non-empty response")
		}
	})

	t.Run("AnswerQuestion", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		context := []string{
			"Go is a programming language created at Google.",
			"Go was designed by Robert Griesemer, Rob Pike, and Ken Thompson.",
		}

		result, err := client.AnswerQuestion(ctx, "Who created Go?", context)
		if err != nil {
			t.Fatalf("AnswerQuestion failed: %v", err)
		}

		if result == nil {
			t.Fatal("Expected non-nil result")
		}
		if result.Answer == "" {
			t.Error("Expected non-empty answer")
		}
		if result.Confidence <= 0 {
			t.Error("Expected positive confidence")
		}
	})

	t.Run("Summarize", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		texts := []string{
			"Learned about Go programming today.",
			"Studied Go concurrency patterns.",
			"Read about Go channels and goroutines.",
		}

		result, err := client.Summarize(ctx, texts, "week")
		if err != nil {
			t.Fatalf("Summarize failed: %v", err)
		}

		if result == nil {
			t.Fatal("Expected non-nil result")
		}
		if result.Summary == "" {
			t.Error("Expected non-empty summary")
		}
		if result.MemoryCount != 3 {
			t.Errorf("Expected memory count 3, got %d", result.MemoryCount)
		}
	})

	t.Run("SummarizeEmpty", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		result, err := client.Summarize(ctx, []string{}, "week")
		if err != nil {
			t.Fatalf("Summarize empty failed: %v", err)
		}

		if result.MemoryCount != 0 {
			t.Errorf("Expected memory count 0, got %d", result.MemoryCount)
		}
	})

	t.Run("AnalyzePatterns", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		texts := []string{
			"Used Go for a REST API project.",
			"Built another Go service for data processing.",
			"Created a Go CLI tool.",
		}

		result, err := client.AnalyzePatterns(ctx, texts, "Go programming")
		if err != nil {
			t.Fatalf("AnalyzePatterns failed: %v", err)
		}

		if result == nil {
			t.Fatal("Expected non-nil result")
		}
	})

	t.Run("SuggestRelationships", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		source := "Go uses goroutines for concurrency"
		target := "Goroutines are lightweight threads managed by the Go runtime"

		suggestion, err := client.SuggestRelationships(ctx, source, target, "id1", "id2")
		if err != nil {
			t.Fatalf("SuggestRelationships failed: %v", err)
		}

		// May or may not find a relationship
		if suggestion != nil {
			t.Logf("Found relationship: %s with confidence %.2f", suggestion.Type, suggestion.Confidence)
		}
	})

	t.Run("GetModels", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		models, err := client.GetModels(ctx)
		if err != nil {
			t.Fatalf("GetModels failed: %v", err)
		}

		if len(models) == 0 {
			t.Error("Expected at least one model")
		}

		t.Logf("Available models: %v", models)
	})
}

// TestParseSummaryResponse tests the summary response parser
func TestParseSummaryResponse(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectSummary string
		expectThemes  int
	}{
		{
			name:          "Standard format",
			input:         "SUMMARY: This is the summary.\nKEY THEMES: [theme1], [theme2], [theme3]",
			expectSummary: "This is the summary.",
			expectThemes:  3,
		},
		{
			name:          "Mixed case",
			input:         "summary: Lower case summary.\nkey themes: one, two",
			expectSummary: "Lower case summary.",
			expectThemes:  2,
		},
		{
			name:          "No structured format",
			input:         "Just a plain response without structure.",
			expectSummary: "Just a plain response without structure.",
			expectThemes:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary, themes := parseSummaryResponse(tt.input)

			if summary != tt.expectSummary {
				t.Errorf("Expected summary %q, got %q", tt.expectSummary, summary)
			}
			if len(themes) != tt.expectThemes {
				t.Errorf("Expected %d themes, got %d", tt.expectThemes, len(themes))
			}
		})
	}
}

// TestParseRelationshipResponse tests the relationship response parser
func TestParseRelationshipResponse(t *testing.T) {
	tests := []struct {
		name             string
		input            string
		expectType       string
		expectConfidence float64
		expectReasoning  string
	}{
		{
			name:             "Standard format",
			input:            "TYPE: references\nCONFIDENCE: 0.8\nREASONING: Both discuss similar topics",
			expectType:       "references",
			expectConfidence: 0.8,
			expectReasoning:  "Both discuss similar topics",
		},
		{
			name:             "No relationship",
			input:            "TYPE: none",
			expectType:       "none",
			expectConfidence: 0.5,
			expectReasoning:  "",
		},
		{
			name:             "Mixed case",
			input:            "type: similar\nconfidence: 0.6",
			expectType:       "similar",
			expectConfidence: 0.6,
			expectReasoning:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			relType, confidence, reasoning := parseRelationshipResponse(tt.input)

			if relType != tt.expectType {
				t.Errorf("Expected type %q, got %q", tt.expectType, relType)
			}
			if confidence != tt.expectConfidence {
				t.Errorf("Expected confidence %f, got %f", tt.expectConfidence, confidence)
			}
			if reasoning != tt.expectReasoning {
				t.Errorf("Expected reasoning %q, got %q", tt.expectReasoning, reasoning)
			}
		})
	}
}
