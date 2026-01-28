package relationships

import (
	"path/filepath"
	"testing"

	"github.com/MycelicMemory/mycelicmemory/internal/database"
	"github.com/MycelicMemory/mycelicmemory/pkg/config"
)

// TestRelationshipService tests the relationship service functionality
func TestRelationshipService(t *testing.T) {
	svc := newTestRelationshipService(t)

	// Create test memories
	mem1 := createTestMemory(t, svc.db, "Memory 1 about Go programming")
	mem2 := createTestMemory(t, svc.db, "Memory 2 about Go concurrency")
	mem3 := createTestMemory(t, svc.db, "Memory 3 about Python")

	t.Run("CreateRelationship", func(t *testing.T) {
		rel, err := svc.Create(&CreateOptions{
			SourceMemoryID:   mem1.ID,
			TargetMemoryID:   mem2.ID,
			RelationshipType: "references",
			Strength:         0.8,
			Context:          "Both about Go",
		})
		if err != nil {
			t.Fatalf("Failed to create relationship: %v", err)
		}

		if rel.ID == "" {
			t.Error("Relationship ID should be generated")
		}
		if rel.RelationshipType != "references" {
			t.Errorf("Expected type 'references', got %s", rel.RelationshipType)
		}
		if rel.Strength != 0.8 {
			t.Errorf("Expected strength 0.8, got %f", rel.Strength)
		}
	})

	t.Run("CreateWithAllTypes", func(t *testing.T) {
		types := []string{"references", "contradicts", "expands", "similar", "sequential", "causes", "enables"}

		for _, relType := range types {
			rel, err := svc.Create(&CreateOptions{
				SourceMemoryID:   mem1.ID,
				TargetMemoryID:   mem3.ID,
				RelationshipType: relType,
				Strength:         0.5,
			})
			if err != nil {
				t.Errorf("Failed to create %s relationship: %v", relType, err)
				continue
			}
			if rel.RelationshipType != relType {
				t.Errorf("Expected type %s, got %s", relType, rel.RelationshipType)
			}
		}
	})

	t.Run("CreateInvalidType", func(t *testing.T) {
		_, err := svc.Create(&CreateOptions{
			SourceMemoryID:   mem1.ID,
			TargetMemoryID:   mem2.ID,
			RelationshipType: "invalid-type",
			Strength:         0.5,
		})
		if err == nil {
			t.Error("Expected error for invalid relationship type")
		}
	})

	t.Run("CreateNonexistentSource", func(t *testing.T) {
		_, err := svc.Create(&CreateOptions{
			SourceMemoryID:   "nonexistent-id",
			TargetMemoryID:   mem2.ID,
			RelationshipType: "references",
			Strength:         0.5,
		})
		if err == nil {
			t.Error("Expected error for nonexistent source")
		}
	})

	t.Run("CreateNonexistentTarget", func(t *testing.T) {
		_, err := svc.Create(&CreateOptions{
			SourceMemoryID:   mem1.ID,
			TargetMemoryID:   "nonexistent-id",
			RelationshipType: "references",
			Strength:         0.5,
		})
		if err == nil {
			t.Error("Expected error for nonexistent target")
		}
	})

	t.Run("CreateDefaultStrength", func(t *testing.T) {
		rel, err := svc.Create(&CreateOptions{
			SourceMemoryID:   mem2.ID,
			TargetMemoryID:   mem3.ID,
			RelationshipType: "similar",
			Strength:         -1, // Invalid, should default to 0.5
		})
		if err != nil {
			t.Fatalf("Failed to create relationship: %v", err)
		}
		if rel.Strength != 0.5 {
			t.Errorf("Expected default strength 0.5, got %f", rel.Strength)
		}
	})

	t.Run("CreateCappedStrength", func(t *testing.T) {
		rel, err := svc.Create(&CreateOptions{
			SourceMemoryID:   mem2.ID,
			TargetMemoryID:   mem3.ID,
			RelationshipType: "expands",
			Strength:         1.5, // Too high, should cap at 1.0
		})
		if err != nil {
			t.Fatalf("Failed to create relationship: %v", err)
		}
		if rel.Strength != 1.0 {
			t.Errorf("Expected capped strength 1.0, got %f", rel.Strength)
		}
	})
}

// TestFindRelated tests finding related memories
func TestFindRelated(t *testing.T) {
	svc := newTestRelationshipService(t)

	// Create chain: A -> B -> C
	memA := createTestMemory(t, svc.db, "Memory A")
	memB := createTestMemory(t, svc.db, "Memory B")
	memC := createTestMemory(t, svc.db, "Memory C")

	_, _ = svc.Create(&CreateOptions{
		SourceMemoryID:   memA.ID,
		TargetMemoryID:   memB.ID,
		RelationshipType: "references",
		Strength:         0.8,
	})

	_, _ = svc.Create(&CreateOptions{
		SourceMemoryID:   memB.ID,
		TargetMemoryID:   memC.ID,
		RelationshipType: "expands",
		Strength:         0.6,
	})

	t.Run("FindRelatedBasic", func(t *testing.T) {
		results, err := svc.FindRelated(&FindRelatedOptions{
			MemoryID: memA.ID,
		})
		if err != nil {
			t.Fatalf("FindRelated failed: %v", err)
		}
		if len(results) == 0 {
			t.Error("Expected at least 1 related memory")
		}
	})

	t.Run("FindRelatedWithTypeFilter", func(t *testing.T) {
		results, err := svc.FindRelated(&FindRelatedOptions{
			MemoryID: memB.ID,
			Type:     "references",
		})
		if err != nil {
			t.Fatalf("FindRelated failed: %v", err)
		}
		// Should find A (references B)
		if len(results) != 1 {
			t.Errorf("Expected 1 related memory with 'references' type, got %d", len(results))
		}
	})

	t.Run("FindRelatedNoID", func(t *testing.T) {
		_, err := svc.FindRelated(&FindRelatedOptions{})
		if err == nil {
			t.Error("Expected error for empty memory_id")
		}
	})

	t.Run("FindRelatedNonexistent", func(t *testing.T) {
		_, err := svc.FindRelated(&FindRelatedOptions{
			MemoryID: "nonexistent-id",
		})
		if err == nil {
			t.Error("Expected error for nonexistent memory")
		}
	})
}

// TestMapGraph tests graph mapping functionality
func TestMapGraph(t *testing.T) {
	svc := newTestRelationshipService(t)

	// Create graph: A -> B -> C -> D
	memA := createTestMemory(t, svc.db, "Memory A")
	memB := createTestMemory(t, svc.db, "Memory B")
	memC := createTestMemory(t, svc.db, "Memory C")
	memD := createTestMemory(t, svc.db, "Memory D")

	_, _ = svc.Create(&CreateOptions{
		SourceMemoryID:   memA.ID,
		TargetMemoryID:   memB.ID,
		RelationshipType: "sequential",
		Strength:         0.9,
	})

	_, _ = svc.Create(&CreateOptions{
		SourceMemoryID:   memB.ID,
		TargetMemoryID:   memC.ID,
		RelationshipType: "sequential",
		Strength:         0.8,
	})

	_, _ = svc.Create(&CreateOptions{
		SourceMemoryID:   memC.ID,
		TargetMemoryID:   memD.ID,
		RelationshipType: "sequential",
		Strength:         0.7,
	})

	t.Run("MapGraphDepth1", func(t *testing.T) {
		result, err := svc.MapGraph(&MapGraphOptions{
			RootID: memA.ID,
			Depth:  1,
		})
		if err != nil {
			t.Fatalf("MapGraph failed: %v", err)
		}
		if result.TotalNodes != 2 { // A and B
			t.Errorf("Expected 2 nodes at depth 1, got %d", result.TotalNodes)
		}
	})

	t.Run("MapGraphDepth2", func(t *testing.T) {
		result, err := svc.MapGraph(&MapGraphOptions{
			RootID: memA.ID,
			Depth:  2,
		})
		if err != nil {
			t.Fatalf("MapGraph failed: %v", err)
		}
		if result.TotalNodes != 3 { // A, B, C
			t.Errorf("Expected 3 nodes at depth 2, got %d", result.TotalNodes)
		}
	})

	t.Run("MapGraphDefaultDepth", func(t *testing.T) {
		result, err := svc.MapGraph(&MapGraphOptions{
			RootID: memA.ID,
		})
		if err != nil {
			t.Fatalf("MapGraph failed: %v", err)
		}
		if result.MaxDepth != 2 { // Default depth
			t.Errorf("Expected default max depth 2, got %d", result.MaxDepth)
		}
	})

	t.Run("MapGraphMaxDepth", func(t *testing.T) {
		result, err := svc.MapGraph(&MapGraphOptions{
			RootID: memA.ID,
			Depth:  10, // Should cap at 5
		})
		if err != nil {
			t.Fatalf("MapGraph failed: %v", err)
		}
		if result.MaxDepth != 5 {
			t.Errorf("Expected capped max depth 5, got %d", result.MaxDepth)
		}
	})

	t.Run("MapGraphNoID", func(t *testing.T) {
		_, err := svc.MapGraph(&MapGraphOptions{})
		if err == nil {
			t.Error("Expected error for empty root_id")
		}
	})

	t.Run("MapGraphNonexistent", func(t *testing.T) {
		_, err := svc.MapGraph(&MapGraphOptions{
			RootID: "nonexistent-id",
		})
		if err == nil {
			t.Error("Expected error for nonexistent memory")
		}
	})

	t.Run("MapGraphWithTypeFilter", func(t *testing.T) {
		result, err := svc.MapGraph(&MapGraphOptions{
			RootID:       memA.ID,
			Depth:        3,
			IncludeTypes: []string{"sequential"},
		})
		if err != nil {
			t.Fatalf("MapGraph failed: %v", err)
		}
		// All edges should be sequential
		for _, edge := range result.Edges {
			if edge.Type != "sequential" {
				t.Errorf("Expected only 'sequential' edges, got %s", edge.Type)
			}
		}
	})

	t.Run("MapGraphWithStrengthFilter", func(t *testing.T) {
		result, err := svc.MapGraph(&MapGraphOptions{
			RootID:      memA.ID,
			Depth:       3,
			MinStrength: 0.85,
		})
		if err != nil {
			t.Fatalf("MapGraph failed: %v", err)
		}
		// Should only include edges with strength >= 0.85 (A->B at 0.9)
		for _, edge := range result.Edges {
			if edge.Strength < 0.85 {
				t.Errorf("Expected only edges with strength >= 0.85, got %f", edge.Strength)
			}
		}
	})
}

// TestDiscover tests relationship discovery
func TestDiscover(t *testing.T) {
	svc := newTestRelationshipService(t)

	// Create some memories
	createTestMemory(t, svc.db, "Go programming")
	createTestMemory(t, svc.db, "Go concurrency")

	t.Run("DiscoverBasic", func(t *testing.T) {
		results, err := svc.Discover(&DiscoverOptions{
			Limit: 10,
		})
		if err != nil {
			t.Fatalf("Discover failed: %v", err)
		}
		// Currently returns empty (Phase 4 implementation)
		_ = results
	})
}

// TestGetRelationshipTypes tests getting relationship types
func TestGetRelationshipTypes(t *testing.T) {
	types := GetRelationshipTypes()

	if len(types) != 7 {
		t.Errorf("Expected 7 relationship types, got %d", len(types))
	}

	expectedTypes := map[string]bool{
		"references":  true,
		"contradicts": true,
		"expands":     true,
		"similar":     true,
		"sequential":  true,
		"causes":      true,
		"enables":     true,
	}

	for _, rt := range types {
		if !expectedTypes[rt.Name] {
			t.Errorf("Unexpected relationship type: %s", rt.Name)
		}
		if rt.Description == "" {
			t.Errorf("Relationship type %s has empty description", rt.Name)
		}
	}
}

// TestValidateRelationshipType tests relationship type validation
func TestValidateRelationshipType(t *testing.T) {
	validTypes := []string{"references", "contradicts", "expands", "similar", "sequential", "causes", "enables"}
	for _, rt := range validTypes {
		if err := ValidateRelationshipType(rt); err != nil {
			t.Errorf("Expected %s to be valid, got error: %v", rt, err)
		}
	}

	// Test case insensitivity
	if err := ValidateRelationshipType("REFERENCES"); err != nil {
		t.Error("Expected case-insensitive validation")
	}

	invalidTypes := []string{"invalid", "relates", "links", ""}
	for _, rt := range invalidTypes {
		if err := ValidateRelationshipType(rt); err == nil {
			t.Errorf("Expected %q to be invalid", rt)
		}
	}
}

// Helper function to create a test relationship service
func newTestRelationshipService(t *testing.T) *Service {
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

// Helper function to create a test memory
func createTestMemory(t *testing.T, db *database.Database, content string) *database.Memory {
	t.Helper()

	mem := &database.Memory{
		Content:    content,
		Importance: 5,
	}
	if err := db.CreateMemory(mem); err != nil {
		t.Fatalf("Failed to create test memory: %v", err)
	}
	return mem
}
