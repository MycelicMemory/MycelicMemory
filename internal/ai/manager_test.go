package ai

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/MycelicMemory/mycelicmemory/internal/database"
	"github.com/MycelicMemory/mycelicmemory/pkg/config"
)

// newTestManager creates a manager for testing
func newTestManager(t *testing.T) (*Manager, *database.Database) {
	t.Helper()

	// Create test database
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	if err := db.InitSchema(); err != nil {
		t.Fatalf("Failed to initialize schema: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	cfg := config.DefaultConfig()
	manager := NewManager(db, cfg)

	return manager, db
}

// TestNewManager tests manager creation
func TestNewManager(t *testing.T) {
	manager, _ := newTestManager(t)

	if manager == nil {
		t.Fatal("NewManager should not return nil")
	}

	if manager.Ollama() == nil {
		t.Error("Ollama client should not be nil")
	}

	if manager.Qdrant() == nil {
		t.Error("Qdrant client should not be nil")
	}
}

// TestManagerInitialize tests initialization
func TestManagerInitialize(t *testing.T) {
	manager, _ := newTestManager(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Should not error even if services unavailable
	err := manager.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Should be idempotent
	err = manager.Initialize(ctx)
	if err != nil {
		t.Fatalf("Second Initialize failed: %v", err)
	}
}

// TestManagerGetStatus tests status reporting
func TestManagerGetStatus(t *testing.T) {
	manager, _ := newTestManager(t)

	status := manager.GetStatus()

	if status == nil {
		t.Fatal("GetStatus should not return nil")
	}

	// With default config, both should be enabled
	if !status.OllamaEnabled {
		t.Error("Ollama should be enabled by default")
	}
	if !status.QdrantEnabled {
		t.Error("Qdrant should be enabled by default")
	}

	t.Logf("Status: Ollama(enabled=%v, available=%v), Qdrant(enabled=%v, available=%v)",
		status.OllamaEnabled, status.OllamaAvailable,
		status.QdrantEnabled, status.QdrantAvailable)
}

// TestManagerIntegration tests with actual services
func TestManagerIntegration(t *testing.T) {
	manager, db := newTestManager(t)

	status := manager.GetStatus()

	// Skip if AI services not available
	if !status.OllamaAvailable || !status.QdrantAvailable {
		t.Skip("AI services not available, skipping integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Initialize manager
	if err := manager.Initialize(ctx); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Create test memory
	mem := &database.Memory{
		Content:    "Test content about Go programming and concurrency",
		Importance: 7,
		SessionID:  "test-session",
		Domain:     "programming",
	}
	if err := db.CreateMemory(mem); err != nil {
		t.Fatalf("Failed to create memory: %v", err)
	}

	t.Run("IndexMemory", func(t *testing.T) {
		err := manager.IndexMemory(ctx, mem)
		if err != nil {
			t.Fatalf("IndexMemory failed: %v", err)
		}

		// Verify embedding was stored
		if len(mem.Embedding) == 0 {
			t.Error("Expected embedding to be stored in memory")
		}
	})

	t.Run("SemanticSearch", func(t *testing.T) {
		// Wait a bit for indexing
		time.Sleep(100 * time.Millisecond)

		results, err := manager.SemanticSearch(ctx, &SemanticSearchOptions{
			Query:        "Go concurrency",
			Limit:        5,
			WithMetadata: true,
		})
		if err != nil {
			t.Fatalf("SemanticSearch failed: %v", err)
		}

		if len(results) == 0 {
			t.Error("Expected at least one result")
		}

		for _, r := range results {
			t.Logf("Result: ID=%s, Score=%.4f", r.MemoryID, r.Score)
		}
	})

	t.Run("SemanticSearchWithFilters", func(t *testing.T) {
		results, err := manager.SemanticSearch(ctx, &SemanticSearchOptions{
			Query:     "programming",
			Limit:     5,
			SessionID: "test-session",
			Domain:    "programming",
		})
		if err != nil {
			t.Fatalf("SemanticSearch with filters failed: %v", err)
		}

		for _, r := range results {
			if r.SessionID != "" && r.SessionID != "test-session" {
				t.Errorf("Expected session_id 'test-session', got %s", r.SessionID)
			}
			if r.Domain != "" && r.Domain != "programming" {
				t.Errorf("Expected domain 'programming', got %s", r.Domain)
			}
		}
	})

	t.Run("DeleteMemoryIndex", func(t *testing.T) {
		err := manager.DeleteMemoryIndex(ctx, mem.ID)
		if err != nil {
			t.Fatalf("DeleteMemoryIndex failed: %v", err)
		}
	})
}

// TestManagerAnalyze tests analysis functions
func TestManagerAnalyze(t *testing.T) {
	manager, db := newTestManager(t)

	status := manager.GetStatus()

	// Skip if Ollama not available
	if !status.OllamaAvailable {
		t.Skip("Ollama not available, skipping analysis tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Create test memories
	memories := []string{
		"Learned about Go goroutines today",
		"Studied Go channels for concurrent programming",
		"Read about Go sync package and mutexes",
		"Practiced building concurrent Go applications",
	}

	for _, content := range memories {
		mem := &database.Memory{
			Content:    content,
			Importance: 5,
			SessionID:  "test-session",
			Domain:     "programming",
		}
		if err := db.CreateMemory(mem); err != nil {
			t.Fatalf("Failed to create memory: %v", err)
		}
	}

	t.Run("AnalyzeQuestion", func(t *testing.T) {
		response, err := manager.Analyze(ctx, &AnalysisOptions{
			Type:     "question",
			Question: "What have I learned about Go concurrency?",
			Limit:    10,
		})
		if err != nil {
			t.Fatalf("Analyze question failed: %v", err)
		}

		if response == nil {
			t.Fatal("Expected non-nil response")
		}
		if response.Type != "question" {
			t.Errorf("Expected type 'question', got %s", response.Type)
		}
		if response.Answer == "" {
			t.Error("Expected non-empty answer")
		}

		t.Logf("Answer: %s", response.Answer)
	})

	t.Run("AnalyzeSummarize", func(t *testing.T) {
		response, err := manager.Analyze(ctx, &AnalysisOptions{
			Type:      "summarize",
			Timeframe: "all",
			Limit:     10,
		})
		if err != nil {
			t.Fatalf("Analyze summarize failed: %v", err)
		}

		if response == nil {
			t.Fatal("Expected non-nil response")
		}
		if response.Type != "summarize" {
			t.Errorf("Expected type 'summarize', got %s", response.Type)
		}
		if response.Summary == "" {
			t.Error("Expected non-empty summary")
		}

		t.Logf("Summary: %s", response.Summary)
		t.Logf("Key themes: %v", response.KeyThemes)
	})

	t.Run("AnalyzePatterns", func(t *testing.T) {
		response, err := manager.Analyze(ctx, &AnalysisOptions{
			Type:  "analyze",
			Query: "Go programming",
			Limit: 10,
		})
		if err != nil {
			t.Fatalf("Analyze patterns failed: %v", err)
		}

		if response == nil {
			t.Fatal("Expected non-nil response")
		}
		if response.Type != "analyze" {
			t.Errorf("Expected type 'analyze', got %s", response.Type)
		}

		t.Logf("Insights: %v", response.Insights)
	})

	t.Run("AnalyzeTemporalPatterns", func(t *testing.T) {
		response, err := manager.Analyze(ctx, &AnalysisOptions{
			Type:  "temporal_patterns",
			Query: "Go concurrency",
			Limit: 10,
		})
		if err != nil {
			t.Fatalf("Analyze temporal patterns failed: %v", err)
		}

		if response == nil {
			t.Fatal("Expected non-nil response")
		}
		if response.Type != "temporal_patterns" {
			t.Errorf("Expected type 'temporal_patterns', got %s", response.Type)
		}
	})

	t.Run("AnalyzeNoMemories", func(t *testing.T) {
		response, err := manager.Analyze(ctx, &AnalysisOptions{
			Type:     "question",
			Question: "What is this?",
			Domain:   "nonexistent-domain-xyz",
			Limit:    10,
		})
		if err != nil {
			t.Fatalf("Analyze with no memories failed: %v", err)
		}

		if response.MemoryCount != 0 {
			t.Errorf("Expected 0 memories, got %d", response.MemoryCount)
		}
	})

	t.Run("AnalyzeInvalidType", func(t *testing.T) {
		_, err := manager.Analyze(ctx, &AnalysisOptions{
			Type: "invalid-type",
		})
		if err == nil {
			t.Error("Expected error for invalid analysis type")
		}
	})
}

// TestManagerDiscoverRelationships tests relationship discovery
func TestManagerDiscoverRelationships(t *testing.T) {
	manager, db := newTestManager(t)

	status := manager.GetStatus()

	// Skip if Ollama not available
	if !status.OllamaAvailable {
		t.Skip("Ollama not available, skipping relationship discovery tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Create related memories
	memories := []string{
		"Go uses goroutines for lightweight concurrency",
		"Goroutines are managed by the Go runtime scheduler",
		"Channels in Go allow safe communication between goroutines",
	}

	for _, content := range memories {
		mem := &database.Memory{
			Content:    content,
			Importance: 5,
		}
		if err := db.CreateMemory(mem); err != nil {
			t.Fatalf("Failed to create memory: %v", err)
		}
	}

	suggestions, err := manager.DiscoverRelationships(ctx, 5)
	if err != nil {
		t.Fatalf("DiscoverRelationships failed: %v", err)
	}

	t.Logf("Found %d relationship suggestions", len(suggestions))
	for _, s := range suggestions {
		t.Logf("Suggestion: %s -> %s (%s, %.2f confidence)",
			s.SourceID[:8], s.TargetID[:8], s.Type, s.Confidence)
	}
}

// TestManagerTimeframes tests timeframe filtering
func TestManagerTimeframes(t *testing.T) {
	manager, db := newTestManager(t)

	// Create memories with different dates
	now := time.Now()

	testCases := []struct {
		content   string
		createdAt time.Time
	}{
		{"Today's memory", now},
		{"Yesterday's memory", now.AddDate(0, 0, -1)},
		{"Last week's memory", now.AddDate(0, 0, -5)},
		{"Last month's memory", now.AddDate(0, -1, 0)},
	}

	for _, tc := range testCases {
		mem := &database.Memory{
			Content:    tc.content,
			Importance: 5,
			CreatedAt:  tc.createdAt,
		}
		if err := db.CreateMemory(mem); err != nil {
			t.Fatalf("Failed to create memory: %v", err)
		}
	}

	status := manager.GetStatus()
	if !status.OllamaAvailable {
		t.Skip("Ollama not available, skipping timeframe tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	t.Run("TimeframeToday", func(t *testing.T) {
		response, err := manager.Analyze(ctx, &AnalysisOptions{
			Type:      "summarize",
			Timeframe: "today",
			Limit:     50,
		})
		if err != nil {
			t.Fatalf("Analyze with today timeframe failed: %v", err)
		}
		t.Logf("Today: %d memories", response.MemoryCount)
	})

	t.Run("TimeframeWeek", func(t *testing.T) {
		response, err := manager.Analyze(ctx, &AnalysisOptions{
			Type:      "summarize",
			Timeframe: "week",
			Limit:     50,
		})
		if err != nil {
			t.Fatalf("Analyze with week timeframe failed: %v", err)
		}
		t.Logf("Week: %d memories", response.MemoryCount)
	})

	t.Run("TimeframeMonth", func(t *testing.T) {
		response, err := manager.Analyze(ctx, &AnalysisOptions{
			Type:      "summarize",
			Timeframe: "month",
			Limit:     50,
		})
		if err != nil {
			t.Fatalf("Analyze with month timeframe failed: %v", err)
		}
		t.Logf("Month: %d memories", response.MemoryCount)
	})

	t.Run("TimeframeAll", func(t *testing.T) {
		response, err := manager.Analyze(ctx, &AnalysisOptions{
			Type:      "summarize",
			Timeframe: "all",
			Limit:     50,
		})
		if err != nil {
			t.Fatalf("Analyze with all timeframe failed: %v", err)
		}
		t.Logf("All: %d memories", response.MemoryCount)
	})
}

// TestHelperFunctions tests helper functions
func TestHelperFunctions(t *testing.T) {
	t.Run("float64SliceToBytes", func(t *testing.T) {
		input := []float64{1.0, 2.0, 3.0}
		bytes := float64SliceToBytes(input)
		if len(bytes) == 0 {
			t.Error("Expected non-empty bytes")
		}
	})

	t.Run("bytesToFloat64Slice", func(t *testing.T) {
		input := []float64{1.0, 2.0, 3.0}
		bytes := float64SliceToBytes(input)
		output := bytesToFloat64Slice(bytes)

		if len(output) != len(input) {
			t.Errorf("Expected length %d, got %d", len(input), len(output))
		}

		for i := range input {
			if output[i] != input[i] {
				t.Errorf("Value mismatch at %d: expected %f, got %f", i, input[i], output[i])
			}
		}
	})
}
