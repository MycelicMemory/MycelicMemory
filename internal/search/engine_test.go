package search

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/MycelicMemory/mycelicmemory/internal/database"
	"github.com/MycelicMemory/mycelicmemory/pkg/config"
)

// TestSearchEngine tests the search engine functionality
func TestSearchEngine(t *testing.T) {
	engine := newTestEngine(t)

	// Create test data
	createTestMemories(t, engine.db)

	t.Run("KeywordSearch", func(t *testing.T) {
		results, err := engine.Search(&SearchOptions{
			Query:      "Go programming",
			SearchType: SearchTypeKeyword,
		})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		if len(results) == 0 {
			t.Error("Expected at least 1 result")
		}
		for _, r := range results {
			if r.MatchType != "keyword" {
				t.Errorf("Expected match_type 'keyword', got %s", r.MatchType)
			}
		}
	})

	t.Run("TagSearch", func(t *testing.T) {
		results, err := engine.Search(&SearchOptions{
			Tags:       []string{"golang"},
			SearchType: SearchTypeTags,
		})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		if len(results) == 0 {
			t.Error("Expected at least 1 result for 'golang' tag")
		}
		for _, r := range results {
			if r.MatchType != "tag" {
				t.Errorf("Expected match_type 'tag', got %s", r.MatchType)
			}
		}
	})

	t.Run("DateRangeSearch", func(t *testing.T) {
		now := time.Now()
		yesterday := now.Add(-24 * time.Hour)
		tomorrow := now.Add(24 * time.Hour)

		results, err := engine.Search(&SearchOptions{
			StartDate:  &yesterday,
			EndDate:    &tomorrow,
			SearchType: SearchTypeDateRange,
		})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		if len(results) == 0 {
			t.Error("Expected at least 1 result in date range")
		}
		for _, r := range results {
			if r.MatchType != "date" {
				t.Errorf("Expected match_type 'date', got %s", r.MatchType)
			}
		}
	})

	t.Run("HybridSearch", func(t *testing.T) {
		results, err := engine.Search(&SearchOptions{
			Query:      "Python",
			SearchType: SearchTypeHybrid,
		})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		if len(results) == 0 {
			t.Error("Expected at least 1 result")
		}
	})

	t.Run("SemanticSearchFallback", func(t *testing.T) {
		// Semantic search should fall back to keyword when Qdrant is disabled
		results, err := engine.Search(&SearchOptions{
			Query:      "machine learning",
			SearchType: SearchTypeSemantic,
		})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		// Should not error, just fall back
		_ = results
	})

	t.Run("SearchWithLimit", func(t *testing.T) {
		results, err := engine.Search(&SearchOptions{
			Query:      "programming",
			SearchType: SearchTypeKeyword,
			Limit:      2,
		})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		if len(results) > 2 {
			t.Errorf("Expected at most 2 results, got %d", len(results))
		}
	})

	t.Run("SearchWithMinRelevance", func(t *testing.T) {
		results, err := engine.Search(&SearchOptions{
			Query:        "Go",
			SearchType:   SearchTypeKeyword,
			MinRelevance: 0.5,
		})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		for _, r := range results {
			if r.Relevance < 0.5 {
				t.Errorf("Result below min relevance: %f", r.Relevance)
			}
		}
	})

	t.Run("SearchWithDomainFilter", func(t *testing.T) {
		results, err := engine.Search(&SearchOptions{
			Query:      "programming",
			SearchType: SearchTypeKeyword,
			Domain:     "programming",
		})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		for _, r := range results {
			if r.Memory.Domain != "programming" {
				t.Errorf("Expected domain 'programming', got %s", r.Memory.Domain)
			}
		}
	})
}

// TestSearchValidation tests search option validation
func TestSearchValidation(t *testing.T) {
	engine := newTestEngine(t)

	t.Run("KeywordSearchRequiresQuery", func(t *testing.T) {
		_, err := engine.Search(&SearchOptions{
			Query:      "",
			SearchType: SearchTypeKeyword,
		})
		if err == nil {
			t.Error("Expected error for keyword search without query")
		}
	})

	t.Run("TagSearchRequiresTags", func(t *testing.T) {
		_, err := engine.Search(&SearchOptions{
			Tags:       nil,
			SearchType: SearchTypeTags,
		})
		if err == nil {
			t.Error("Expected error for tag search without tags")
		}
	})

	t.Run("DateRangeSearchRequiresDates", func(t *testing.T) {
		_, err := engine.Search(&SearchOptions{
			SearchType: SearchTypeDateRange,
		})
		if err == nil {
			t.Error("Expected error for date range search without dates")
		}
	})

	t.Run("InvalidMinRelevance", func(t *testing.T) {
		_, err := engine.Search(&SearchOptions{
			Query:        "test",
			SearchType:   SearchTypeKeyword,
			MinRelevance: 1.5,
		})
		if err == nil {
			t.Error("Expected error for min_relevance > 1")
		}
	})

	t.Run("NegativeLimit", func(t *testing.T) {
		_, err := engine.Search(&SearchOptions{
			Query:      "test",
			SearchType: SearchTypeKeyword,
			Limit:      -1,
		})
		if err == nil {
			t.Error("Expected error for negative limit")
		}
	})
}

// TestTagMatching tests tag matching logic
func TestTagMatching(t *testing.T) {
	t.Run("OROperator", func(t *testing.T) {
		memoryTags := []string{"go", "python", "rust"}
		searchTags := []string{"go", "javascript"}

		count := countTagMatches(memoryTags, searchTags, "OR")
		if count != 1 { // "go" matches
			t.Errorf("Expected 1 match, got %d", count)
		}
	})

	t.Run("ANDOperator", func(t *testing.T) {
		memoryTags := []string{"go", "python", "rust"}
		searchTags := []string{"go", "python"}

		count := countTagMatches(memoryTags, searchTags, "AND")
		if count != 2 { // Both match
			t.Errorf("Expected 2 matches, got %d", count)
		}

		searchTags = []string{"go", "javascript"}
		count = countTagMatches(memoryTags, searchTags, "AND")
		if count != 0 { // Not all match
			t.Errorf("Expected 0 matches for AND with partial match, got %d", count)
		}
	})

	t.Run("CaseInsensitive", func(t *testing.T) {
		memoryTags := []string{"Go", "PYTHON"}
		searchTags := []string{"go", "python"}

		count := countTagMatches(memoryTags, searchTags, "OR")
		if count != 2 {
			t.Errorf("Expected 2 case-insensitive matches, got %d", count)
		}
	})
}

// TestMergeResults tests result merging for hybrid search
func TestMergeResults(t *testing.T) {
	mem1 := &database.Memory{ID: "1", Content: "Test 1"}
	mem2 := &database.Memory{ID: "2", Content: "Test 2"}
	mem3 := &database.Memory{ID: "3", Content: "Test 3"}

	a := []*SearchResult{
		{Memory: mem1, Relevance: 0.8, MatchType: "keyword"},
		{Memory: mem2, Relevance: 0.5, MatchType: "keyword"},
	}

	b := []*SearchResult{
		{Memory: mem2, Relevance: 0.9, MatchType: "semantic"}, // Higher relevance for mem2
		{Memory: mem3, Relevance: 0.7, MatchType: "semantic"},
	}

	merged := mergeResults(a, b)

	if len(merged) != 3 {
		t.Errorf("Expected 3 merged results, got %d", len(merged))
	}

	// Check that mem2 has weighted combined score with boost
	// Formula: (0.5 * 0.4 + 0.9 * 0.6) * 1.2 = 0.888
	for _, r := range merged {
		if r.Memory.ID == "2" {
			expectedScore := 0.888
			if r.Relevance < expectedScore-0.01 || r.Relevance > expectedScore+0.01 {
				t.Errorf("Expected mem2 to have relevance ~%.3f (weighted combination with boost), got %f", expectedScore, r.Relevance)
			}
			break
		}
	}
}

// TestIntelligentSearch tests intelligent search functionality
func TestIntelligentSearch(t *testing.T) {
	engine := newTestEngine(t)
	createTestMemories(t, engine.db)

	results, err := engine.IntelligentSearch("Go programming", &SearchOptions{})
	if err != nil {
		t.Fatalf("Intelligent search failed: %v", err)
	}

	// Should return results (falls back to hybrid search for now)
	if len(results) == 0 {
		t.Error("Expected at least 1 result from intelligent search")
	}
}

// TestListSearch tests list search (no query)
func TestListSearch(t *testing.T) {
	engine := newTestEngine(t)
	createTestMemories(t, engine.db)

	results, err := engine.Search(&SearchOptions{
		Domain: "programming",
	})
	if err != nil {
		t.Fatalf("List search failed: %v", err)
	}

	if len(results) == 0 {
		t.Error("Expected results from list search")
	}

	for _, r := range results {
		if r.MatchType != "list" {
			t.Errorf("Expected match_type 'list', got %s", r.MatchType)
		}
	}
}

// Helper function to create a test engine
func newTestEngine(t *testing.T) *Engine {
	t.Helper()

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
	return NewEngine(db, cfg)
}

// Helper function to create test memories
func createTestMemories(t *testing.T, db *database.Database) {
	t.Helper()

	testData := []struct {
		content string
		tags    []string
		domain  string
	}{
		{"Go programming language basics", []string{"golang", "programming"}, "programming"},
		{"Python for data science", []string{"python", "data"}, "programming"},
		{"JavaScript frontend development", []string{"javascript", "frontend"}, "programming"},
		{"Go advanced concurrency patterns", []string{"golang", "concurrency"}, "programming"},
		{"Machine learning with Python", []string{"python", "ml"}, "data-science"},
	}

	for _, td := range testData {
		mem := &database.Memory{
			Content: td.content,
			Tags:    td.tags,
			Domain:  td.domain,
		}
		if err := db.CreateMemory(mem); err != nil {
			t.Fatalf("Failed to create test memory: %v", err)
		}
	}
}
