package memory

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/MycelicMemory/ultrathink/internal/database"
	"github.com/MycelicMemory/ultrathink/pkg/config"
)

// TestServiceStore tests memory storage functionality
func TestServiceStore(t *testing.T) {
	svc := newTestService(t)

	t.Run("BasicStore", func(t *testing.T) {
		result, err := svc.Store(&StoreOptions{
			Content:    "Test memory content",
			Importance: 7,
			Tags:       []string{"test", "golang"},
		})
		if err != nil {
			t.Fatalf("Failed to store memory: %v", err)
		}

		if result.Memory.ID == "" {
			t.Error("Memory ID should be generated")
		}
		if result.Memory.Content != "Test memory content" {
			t.Errorf("Content mismatch: got %q", result.Memory.Content)
		}
		if result.Memory.Importance != 7 {
			t.Errorf("Importance mismatch: expected 7, got %d", result.Memory.Importance)
		}
		if !result.IsNew {
			t.Error("Expected IsNew to be true")
		}
		if result.SessionID == "" {
			t.Error("SessionID should be detected")
		}
	})

	t.Run("StoreWithDefaults", func(t *testing.T) {
		result, err := svc.Store(&StoreOptions{
			Content: "Minimal memory",
		})
		if err != nil {
			t.Fatalf("Failed to store memory: %v", err)
		}

		if result.Memory.Importance != 5 {
			t.Errorf("Expected default importance 5, got %d", result.Memory.Importance)
		}
		if result.Memory.AccessScope != "session" {
			t.Errorf("Expected default access_scope 'session', got %s", result.Memory.AccessScope)
		}
	})

	t.Run("StoreWithDomain", func(t *testing.T) {
		result, err := svc.Store(&StoreOptions{
			Content: "Domain memory",
			Domain:  "testing",
		})
		if err != nil {
			t.Fatalf("Failed to store memory: %v", err)
		}

		if result.Memory.Domain != "testing" {
			t.Errorf("Expected domain 'testing', got %s", result.Memory.Domain)
		}
	})

	t.Run("StoreEmptyContent", func(t *testing.T) {
		_, err := svc.Store(&StoreOptions{
			Content: "",
		})
		if err == nil {
			t.Error("Expected error for empty content")
		}
	})

	t.Run("StoreWhitespaceContent", func(t *testing.T) {
		_, err := svc.Store(&StoreOptions{
			Content: "   ",
		})
		if err == nil {
			t.Error("Expected error for whitespace-only content")
		}
	})

	t.Run("ImportanceValidation", func(t *testing.T) {
		// Importance < 1 should default to 5
		result, _ := svc.Store(&StoreOptions{
			Content:    "Test",
			Importance: 0,
		})
		if result.Memory.Importance != 5 {
			t.Errorf("Expected importance 5 for 0 input, got %d", result.Memory.Importance)
		}

		// Importance > 10 should cap at 10
		result, _ = svc.Store(&StoreOptions{
			Content:    "Test",
			Importance: 15,
		})
		if result.Memory.Importance != 10 {
			t.Errorf("Expected importance 10 for 15 input, got %d", result.Memory.Importance)
		}
	})

	t.Run("TagNormalization", func(t *testing.T) {
		result, err := svc.Store(&StoreOptions{
			Content: "Test",
			Tags:    []string{"  TEST  ", "Golang", "test", " "}, // duplicates and whitespace
		})
		if err != nil {
			t.Fatalf("Failed to store memory: %v", err)
		}

		// Should be normalized to ["test", "golang"]
		if len(result.Memory.Tags) != 2 {
			t.Errorf("Expected 2 unique tags, got %d: %v", len(result.Memory.Tags), result.Memory.Tags)
		}
	})
}

// TestServiceGet tests memory retrieval functionality
func TestServiceGet(t *testing.T) {
	svc := newTestService(t)

	// Store a memory first
	stored, _ := svc.Store(&StoreOptions{
		Content:    "Test memory",
		Importance: 8,
	})

	t.Run("GetByID", func(t *testing.T) {
		mem, err := svc.Get(&GetOptions{ID: stored.Memory.ID})
		if err != nil {
			t.Fatalf("Failed to get memory: %v", err)
		}
		if mem == nil {
			t.Fatal("Expected memory, got nil")
		}
		if mem.Content != "Test memory" {
			t.Errorf("Content mismatch: got %q", mem.Content)
		}
	})

	t.Run("GetNotFound", func(t *testing.T) {
		mem, err := svc.Get(&GetOptions{ID: "nonexistent-id"})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if mem != nil {
			t.Error("Expected nil for nonexistent memory")
		}
	})

	t.Run("GetNoIDOrSlug", func(t *testing.T) {
		_, err := svc.Get(&GetOptions{})
		if err == nil {
			t.Error("Expected error when no ID or slug provided")
		}
	})
}

// TestServiceUpdate tests memory update functionality
func TestServiceUpdate(t *testing.T) {
	svc := newTestService(t)

	// Store a memory first
	stored, _ := svc.Store(&StoreOptions{
		Content:    "Original content",
		Importance: 5,
	})

	t.Run("UpdateContent", func(t *testing.T) {
		newContent := "Updated content"
		mem, err := svc.Update(&UpdateOptions{
			ID:      stored.Memory.ID,
			Content: &newContent,
		})
		if err != nil {
			t.Fatalf("Failed to update memory: %v", err)
		}
		if mem.Content != newContent {
			t.Errorf("Content not updated: got %q", mem.Content)
		}
	})

	t.Run("UpdateImportance", func(t *testing.T) {
		newImportance := 9
		mem, err := svc.Update(&UpdateOptions{
			ID:         stored.Memory.ID,
			Importance: &newImportance,
		})
		if err != nil {
			t.Fatalf("Failed to update memory: %v", err)
		}
		if mem.Importance != 9 {
			t.Errorf("Importance not updated: got %d", mem.Importance)
		}
	})

	t.Run("UpdateInvalidImportance", func(t *testing.T) {
		invalidImportance := 15
		_, err := svc.Update(&UpdateOptions{
			ID:         stored.Memory.ID,
			Importance: &invalidImportance,
		})
		if err == nil {
			t.Error("Expected error for invalid importance")
		}
	})

	t.Run("UpdateNotFound", func(t *testing.T) {
		newContent := "test"
		_, err := svc.Update(&UpdateOptions{
			ID:      "nonexistent-id",
			Content: &newContent,
		})
		if err == nil {
			t.Error("Expected error for nonexistent memory")
		}
	})

	t.Run("UpdateNoID", func(t *testing.T) {
		newContent := "test"
		_, err := svc.Update(&UpdateOptions{
			Content: &newContent,
		})
		if err == nil {
			t.Error("Expected error when no ID provided")
		}
	})
}

// TestServiceDelete tests memory deletion functionality
func TestServiceDelete(t *testing.T) {
	svc := newTestService(t)

	// Store a memory first
	stored, _ := svc.Store(&StoreOptions{
		Content: "To be deleted",
	})

	t.Run("Delete", func(t *testing.T) {
		err := svc.Delete(stored.Memory.ID)
		if err != nil {
			t.Fatalf("Failed to delete memory: %v", err)
		}

		// Verify deleted
		mem, _ := svc.Get(&GetOptions{ID: stored.Memory.ID})
		if mem != nil {
			t.Error("Memory should be deleted")
		}
	})

	t.Run("DeleteNotFound", func(t *testing.T) {
		err := svc.Delete("nonexistent-id")
		if err == nil {
			t.Error("Expected error for nonexistent memory")
		}
	})

	t.Run("DeleteNoID", func(t *testing.T) {
		err := svc.Delete("")
		if err == nil {
			t.Error("Expected error when no ID provided")
		}
	})
}

// TestServiceList tests memory listing functionality
func TestServiceList(t *testing.T) {
	svc := newTestService(t)

	// Store test memories
	for i := 0; i < 10; i++ {
		svc.Store(&StoreOptions{
			Content:    "Test memory",
			Importance: i + 1,
			Domain:     "testing",
		})
	}

	t.Run("ListAll", func(t *testing.T) {
		memories, err := svc.List(&ListOptions{})
		if err != nil {
			t.Fatalf("Failed to list memories: %v", err)
		}
		if len(memories) != 10 {
			t.Errorf("Expected 10 memories, got %d", len(memories))
		}
	})

	t.Run("ListWithLimit", func(t *testing.T) {
		memories, err := svc.List(&ListOptions{Limit: 5})
		if err != nil {
			t.Fatalf("Failed to list memories: %v", err)
		}
		if len(memories) != 5 {
			t.Errorf("Expected 5 memories, got %d", len(memories))
		}
	})

	t.Run("ListByDomain", func(t *testing.T) {
		memories, err := svc.List(&ListOptions{Domain: "testing"})
		if err != nil {
			t.Fatalf("Failed to list memories: %v", err)
		}
		if len(memories) != 10 {
			t.Errorf("Expected 10 memories in 'testing' domain, got %d", len(memories))
		}
	})

	t.Run("ListByImportance", func(t *testing.T) {
		memories, err := svc.List(&ListOptions{MinImportance: 8})
		if err != nil {
			t.Fatalf("Failed to list memories: %v", err)
		}
		if len(memories) != 3 { // 8, 9, 10
			t.Errorf("Expected 3 memories with importance >= 8, got %d", len(memories))
		}
	})
}

// TestServiceStats tests statistics functionality
func TestServiceStats(t *testing.T) {
	svc := newTestService(t)

	// Store some memories
	svc.Store(&StoreOptions{Content: "Test 1", Domain: "domain1"})
	svc.Store(&StoreOptions{Content: "Test 2", Domain: "domain2"})

	stats, err := svc.GetStats()
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	if stats.TotalMemories != 2 {
		t.Errorf("Expected 2 memories, got %d", stats.TotalMemories)
	}
	if stats.SessionID == "" {
		t.Error("SessionID should be set")
	}
}

// TestNormalizeTags tests tag normalization
func TestNormalizeTags(t *testing.T) {
	tests := []struct {
		input    []string
		expected int
	}{
		{[]string{"test", "TEST", "Test"}, 1},         // Deduplicate
		{[]string{"  tag  ", "tag"}, 1},               // Trim and deduplicate
		{[]string{"a", "b", "c"}, 3},                  // Keep unique
		{[]string{"", "  ", "valid"}, 1},              // Filter empty
		{nil, 0},                                      // Handle nil
		{[]string{}, 0},                               // Handle empty
	}

	for _, tt := range tests {
		result := normalizeTags(tt.input)
		if len(result) != tt.expected {
			t.Errorf("normalizeTags(%v) = %v, expected %d tags", tt.input, result, tt.expected)
		}
	}
}

// TestDateRangeFilter tests date range filtering
func TestDateRangeFilter(t *testing.T) {
	svc := newTestService(t)

	// Store memories - they'll have current timestamps
	svc.Store(&StoreOptions{Content: "Today's memory"})

	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	tomorrow := now.Add(24 * time.Hour)

	t.Run("DateRangeIncludesNow", func(t *testing.T) {
		memories, err := svc.List(&ListOptions{
			StartDate: &yesterday,
			EndDate:   &tomorrow,
		})
		if err != nil {
			t.Fatalf("Failed to list memories: %v", err)
		}
		if len(memories) < 1 {
			t.Error("Expected at least 1 memory in date range")
		}
	})

	t.Run("DateRangeExcludesNow", func(t *testing.T) {
		oldStart := now.Add(-48 * time.Hour)
		oldEnd := now.Add(-24 * time.Hour)
		memories, err := svc.List(&ListOptions{
			StartDate: &oldStart,
			EndDate:   &oldEnd,
		})
		if err != nil {
			t.Fatalf("Failed to list memories: %v", err)
		}
		if len(memories) != 0 {
			t.Errorf("Expected 0 memories in old date range, got %d", len(memories))
		}
	})
}

// Helper function to create a test service
func newTestService(t *testing.T) *Service {
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
	return NewService(db, cfg)
}
