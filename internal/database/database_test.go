package database

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestDatabaseOpenClose tests database connection lifecycle
func TestDatabaseOpenClose(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Verify database file was created
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("Database file was not created")
	}

	// Close and verify
	if err := db.Close(); err != nil {
		t.Fatalf("Failed to close database: %v", err)
	}
}

// TestDatabaseInitSchema tests schema initialization
func TestDatabaseInitSchema(t *testing.T) {
	db := newTestDB(t)

	// Verify schema version
	version, err := db.GetSchemaVersion()
	if err != nil {
		t.Fatalf("Failed to get schema version: %v", err)
	}
	if version != SchemaVersion {
		t.Errorf("Expected schema version %d, got %d", SchemaVersion, version)
	}

	// Verify tables exist
	tables := []string{
		"memories", "memory_relationships", "categories",
		"memory_categorizations", "domains", "vector_metadata",
		"agent_sessions", "performance_metrics", "migration_log",
		"schema_version", "memories_fts",
	}

	for _, table := range tables {
		exists, err := db.TableExists(table)
		if err != nil {
			t.Fatalf("Failed to check table %s: %v", table, err)
		}
		if !exists {
			t.Errorf("Table %s should exist", table)
		}
	}
}

// TestMemoryCRUD tests memory create, read, update, delete operations
func TestMemoryCRUD(t *testing.T) {
	db := newTestDB(t)

	t.Run("Create", func(t *testing.T) {
		mem := &Memory{
			Content:    "Test memory content",
			Importance: 7,
			Tags:       []string{"test", "golang"},
			Domain:     "testing",
			AgentType:  "api",
		}

		err := db.CreateMemory(mem)
		if err != nil {
			t.Fatalf("Failed to create memory: %v", err)
		}

		if mem.ID == "" {
			t.Error("Memory ID should be generated")
		}
		if mem.CreatedAt.IsZero() {
			t.Error("CreatedAt should be set")
		}
	})

	t.Run("CreateWithDefaults", func(t *testing.T) {
		mem := &Memory{
			Content: "Minimal memory",
		}

		err := db.CreateMemory(mem)
		if err != nil {
			t.Fatalf("Failed to create memory: %v", err)
		}

		// Verify defaults
		retrieved, err := db.GetMemory(mem.ID)
		if err != nil {
			t.Fatalf("Failed to get memory: %v", err)
		}
		if retrieved.Importance != 5 {
			t.Errorf("Expected default importance 5, got %d", retrieved.Importance)
		}
		if retrieved.AgentType != "unknown" {
			t.Errorf("Expected default agent_type 'unknown', got %s", retrieved.AgentType)
		}
		if retrieved.AccessScope != "session" {
			t.Errorf("Expected default access_scope 'session', got %s", retrieved.AccessScope)
		}
	})

	t.Run("Read", func(t *testing.T) {
		mem := &Memory{
			Content:    "Read test memory",
			Importance: 8,
			Tags:       []string{"read", "test"},
			Source:     "test-source",
			Domain:     "testing",
		}
		err := db.CreateMemory(mem)
		if err != nil {
			t.Fatalf("Failed to create memory: %v", err)
		}

		retrieved, err := db.GetMemory(mem.ID)
		if err != nil {
			t.Fatalf("Failed to get memory: %v", err)
		}
		if retrieved == nil {
			t.Fatal("Expected memory, got nil")
		}

		if retrieved.Content != mem.Content {
			t.Errorf("Content mismatch: expected %q, got %q", mem.Content, retrieved.Content)
		}
		if retrieved.Importance != mem.Importance {
			t.Errorf("Importance mismatch: expected %d, got %d", mem.Importance, retrieved.Importance)
		}
		if len(retrieved.Tags) != 2 {
			t.Errorf("Expected 2 tags, got %d", len(retrieved.Tags))
		}
		if retrieved.Source != mem.Source {
			t.Errorf("Source mismatch: expected %q, got %q", mem.Source, retrieved.Source)
		}
	})

	t.Run("ReadNotFound", func(t *testing.T) {
		retrieved, err := db.GetMemory("nonexistent-id")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if retrieved != nil {
			t.Error("Expected nil for nonexistent memory")
		}
	})

	t.Run("Update", func(t *testing.T) {
		mem := &Memory{
			Content:    "Original content",
			Importance: 5,
		}
		_ = db.CreateMemory(mem)

		newContent := "Updated content"
		newImportance := 9
		err := db.UpdateMemory(mem.ID, &MemoryUpdate{
			Content:    &newContent,
			Importance: &newImportance,
		})
		if err != nil {
			t.Fatalf("Failed to update memory: %v", err)
		}

		retrieved, _ := db.GetMemory(mem.ID)
		if retrieved.Content != newContent {
			t.Errorf("Content not updated: expected %q, got %q", newContent, retrieved.Content)
		}
		if retrieved.Importance != newImportance {
			t.Errorf("Importance not updated: expected %d, got %d", newImportance, retrieved.Importance)
		}
	})

	t.Run("UpdateNotFound", func(t *testing.T) {
		newContent := "test"
		err := db.UpdateMemory("nonexistent-id", &MemoryUpdate{
			Content: &newContent,
		})
		if err == nil {
			t.Error("Expected error for nonexistent memory")
		}
	})

	t.Run("Delete", func(t *testing.T) {
		mem := &Memory{
			Content: "To be deleted",
		}
		_ = db.CreateMemory(mem)

		err := db.DeleteMemory(mem.ID)
		if err != nil {
			t.Fatalf("Failed to delete memory: %v", err)
		}

		retrieved, _ := db.GetMemory(mem.ID)
		if retrieved != nil {
			t.Error("Memory should be deleted")
		}
	})

	t.Run("DeleteNotFound", func(t *testing.T) {
		err := db.DeleteMemory("nonexistent-id")
		if err == nil {
			t.Error("Expected error for nonexistent memory")
		}
	})
}

// TestListMemories tests memory listing with filters
func TestListMemories(t *testing.T) {
	db := newTestDB(t)

	// Create test memories
	for i := 0; i < 10; i++ {
		mem := &Memory{
			Content:    "Test memory " + string(rune('A'+i)),
			Importance: i + 1,
			Domain:     "test",
		}
		if i%2 == 0 {
			mem.Tags = []string{"even"}
		} else {
			mem.Tags = []string{"odd"}
		}
		_ = db.CreateMemory(mem)
	}

	t.Run("ListAll", func(t *testing.T) {
		memories, err := db.ListMemories(&MemoryFilters{})
		if err != nil {
			t.Fatalf("Failed to list memories: %v", err)
		}
		if len(memories) != 10 {
			t.Errorf("Expected 10 memories, got %d", len(memories))
		}
	})

	t.Run("ListWithLimit", func(t *testing.T) {
		memories, err := db.ListMemories(&MemoryFilters{Limit: 5})
		if err != nil {
			t.Fatalf("Failed to list memories: %v", err)
		}
		if len(memories) != 5 {
			t.Errorf("Expected 5 memories, got %d", len(memories))
		}
	})

	t.Run("ListWithOffset", func(t *testing.T) {
		memories, err := db.ListMemories(&MemoryFilters{Limit: 5, Offset: 5})
		if err != nil {
			t.Fatalf("Failed to list memories: %v", err)
		}
		if len(memories) != 5 {
			t.Errorf("Expected 5 memories, got %d", len(memories))
		}
	})

	t.Run("FilterByDomain", func(t *testing.T) {
		memories, err := db.ListMemories(&MemoryFilters{Domain: "test"})
		if err != nil {
			t.Fatalf("Failed to list memories: %v", err)
		}
		if len(memories) != 10 {
			t.Errorf("Expected 10 memories in 'test' domain, got %d", len(memories))
		}

		memories, err = db.ListMemories(&MemoryFilters{Domain: "nonexistent"})
		if err != nil {
			t.Fatalf("Failed to list memories: %v", err)
		}
		if len(memories) != 0 {
			t.Errorf("Expected 0 memories in nonexistent domain, got %d", len(memories))
		}
	})

	t.Run("FilterByImportance", func(t *testing.T) {
		memories, err := db.ListMemories(&MemoryFilters{MinImportance: 8})
		if err != nil {
			t.Fatalf("Failed to list memories: %v", err)
		}
		if len(memories) != 3 { // 8, 9, 10
			t.Errorf("Expected 3 memories with importance >= 8, got %d", len(memories))
		}

		memories, err = db.ListMemories(&MemoryFilters{MaxImportance: 3})
		if err != nil {
			t.Fatalf("Failed to list memories: %v", err)
		}
		if len(memories) != 3 { // 1, 2, 3
			t.Errorf("Expected 3 memories with importance <= 3, got %d", len(memories))
		}
	})

	t.Run("FilterByTags", func(t *testing.T) {
		memories, err := db.ListMemories(&MemoryFilters{Tags: []string{"even"}})
		if err != nil {
			t.Fatalf("Failed to list memories: %v", err)
		}
		if len(memories) != 5 {
			t.Errorf("Expected 5 memories with 'even' tag, got %d", len(memories))
		}
	})
}

// TestSearchFTS tests full-text search
func TestSearchFTS(t *testing.T) {
	db := newTestDB(t)

	// Create test memories
	testData := []struct {
		content string
		tags    []string
	}{
		{"Go programming language basics", []string{"golang", "programming"}},
		{"Python for data science", []string{"python", "data"}},
		{"JavaScript frontend development", []string{"javascript", "frontend"}},
		{"Go advanced concurrency patterns", []string{"golang", "concurrency"}},
		{"Machine learning with Python", []string{"python", "ml"}},
	}

	for _, td := range testData {
		mem := &Memory{
			Content: td.content,
			Tags:    td.tags,
			Domain:  "programming",
		}
		_ = db.CreateMemory(mem)
	}

	t.Run("SimpleSearch", func(t *testing.T) {
		results, err := db.SearchFTS("Go", &SearchFilters{})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		if len(results) != 2 {
			t.Errorf("Expected 2 results for 'Go', got %d", len(results))
		}
	})

	t.Run("PhraseSearch", func(t *testing.T) {
		results, err := db.SearchFTS("data science", &SearchFilters{})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		if len(results) < 1 {
			t.Error("Expected at least 1 result for 'data science'")
		}
	})

	t.Run("NoResults", func(t *testing.T) {
		results, err := db.SearchFTS("nonexistent content xyz", &SearchFilters{})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("Expected 0 results, got %d", len(results))
		}
	})

	t.Run("SearchWithLimit", func(t *testing.T) {
		results, err := db.SearchFTS("programming", &SearchFilters{Limit: 1})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		if len(results) > 1 {
			t.Errorf("Expected at most 1 result, got %d", len(results))
		}
	})

	t.Run("SearchWithDomainFilter", func(t *testing.T) {
		results, err := db.SearchFTS("Python", &SearchFilters{Domain: "programming"})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		if len(results) != 2 {
			t.Errorf("Expected 2 results for 'Python' in 'programming' domain, got %d", len(results))
		}

		results, err = db.SearchFTS("Python", &SearchFilters{Domain: "other"})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("Expected 0 results for 'Python' in 'other' domain, got %d", len(results))
		}
	})

	t.Run("RelevanceScores", func(t *testing.T) {
		results, err := db.SearchFTS("Go", &SearchFilters{})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		for _, r := range results {
			if r.Relevance < 0 || r.Relevance > 1 {
				t.Errorf("Relevance should be between 0 and 1, got %f", r.Relevance)
			}
		}
	})

	t.Run("EmptyQuery", func(t *testing.T) {
		_, err := db.SearchFTS("", &SearchFilters{})
		if err == nil {
			t.Error("Expected error for empty query")
		}
	})
}

// TestRelationships tests relationship CRUD and graph operations
func TestRelationships(t *testing.T) {
	db := newTestDB(t)

	// Create test memories
	mem1 := &Memory{Content: "Memory 1"}
	mem2 := &Memory{Content: "Memory 2"}
	mem3 := &Memory{Content: "Memory 3"}
	_ = db.CreateMemory(mem1)
	_ = db.CreateMemory(mem2)
	_ = db.CreateMemory(mem3)

	t.Run("CreateRelationship", func(t *testing.T) {
		rel := &Relationship{
			SourceMemoryID:   mem1.ID,
			TargetMemoryID:   mem2.ID,
			RelationshipType: "references",
			Strength:         0.8,
			Context:          "Test relationship",
		}

		err := db.CreateRelationship(rel)
		if err != nil {
			t.Fatalf("Failed to create relationship: %v", err)
		}

		if rel.ID == "" {
			t.Error("Relationship ID should be generated")
		}
	})

	t.Run("InvalidRelationshipType", func(t *testing.T) {
		rel := &Relationship{
			SourceMemoryID:   mem1.ID,
			TargetMemoryID:   mem2.ID,
			RelationshipType: "invalid-type",
			Strength:         0.5,
		}

		err := db.CreateRelationship(rel)
		if err == nil {
			t.Error("Expected error for invalid relationship type")
		}
	})

	t.Run("InvalidStrength", func(t *testing.T) {
		rel := &Relationship{
			SourceMemoryID:   mem1.ID,
			TargetMemoryID:   mem2.ID,
			RelationshipType: "similar",
			Strength:         1.5, // Invalid
		}

		err := db.CreateRelationship(rel)
		if err == nil {
			t.Error("Expected error for invalid strength")
		}
	})

	t.Run("FindRelated", func(t *testing.T) {
		// Create additional relationships
		_ = db.CreateRelationship(&Relationship{
			SourceMemoryID:   mem2.ID,
			TargetMemoryID:   mem3.ID,
			RelationshipType: "expands",
			Strength:         0.6,
		})

		related, err := db.FindRelated(mem1.ID, &RelationshipFilters{})
		if err != nil {
			t.Fatalf("Failed to find related: %v", err)
		}
		if len(related) < 1 {
			t.Error("Expected at least 1 related memory")
		}
	})

	t.Run("FindRelatedWithFilter", func(t *testing.T) {
		related, err := db.FindRelated(mem1.ID, &RelationshipFilters{
			Type: "references",
		})
		if err != nil {
			t.Fatalf("Failed to find related: %v", err)
		}
		if len(related) != 1 {
			t.Errorf("Expected 1 related memory with type 'references', got %d", len(related))
		}
	})

	t.Run("FindRelatedWithStrengthFilter", func(t *testing.T) {
		related, err := db.FindRelated(mem2.ID, &RelationshipFilters{
			MinStrength: 0.7,
		})
		if err != nil {
			t.Fatalf("Failed to find related: %v", err)
		}
		// Should find mem1 (strength 0.8) but not mem3 (strength 0.6)
		if len(related) != 1 {
			t.Errorf("Expected 1 related memory with strength >= 0.7, got %d", len(related))
		}
	})
}

// TestGetGraph tests graph traversal
func TestGetGraph(t *testing.T) {
	db := newTestDB(t)

	// Create a chain of memories: A -> B -> C -> D
	memA := &Memory{Content: "Memory A", Importance: 10}
	memB := &Memory{Content: "Memory B", Importance: 8}
	memC := &Memory{Content: "Memory C", Importance: 6}
	memD := &Memory{Content: "Memory D", Importance: 4}

	_ = db.CreateMemory(memA)
	_ = db.CreateMemory(memB)
	_ = db.CreateMemory(memC)
	_ = db.CreateMemory(memD)

	_ = db.CreateRelationship(&Relationship{
		SourceMemoryID:   memA.ID,
		TargetMemoryID:   memB.ID,
		RelationshipType: "sequential",
		Strength:         0.9,
	})
	_ = db.CreateRelationship(&Relationship{
		SourceMemoryID:   memB.ID,
		TargetMemoryID:   memC.ID,
		RelationshipType: "sequential",
		Strength:         0.9,
	})
	_ = db.CreateRelationship(&Relationship{
		SourceMemoryID:   memC.ID,
		TargetMemoryID:   memD.ID,
		RelationshipType: "sequential",
		Strength:         0.9,
	})

	t.Run("DefaultDepth", func(t *testing.T) {
		graph, err := db.GetGraph(memA.ID, 0) // Should use default depth of 2
		if err != nil {
			t.Fatalf("Failed to get graph: %v", err)
		}
		if len(graph.Nodes) < 3 {
			t.Errorf("Expected at least 3 nodes at depth 2, got %d", len(graph.Nodes))
		}
	})

	t.Run("Depth1", func(t *testing.T) {
		graph, err := db.GetGraph(memA.ID, 1)
		if err != nil {
			t.Fatalf("Failed to get graph: %v", err)
		}
		if len(graph.Nodes) != 2 {
			t.Errorf("Expected 2 nodes at depth 1 (A and B), got %d", len(graph.Nodes))
		}
	})

	t.Run("Depth3", func(t *testing.T) {
		graph, err := db.GetGraph(memA.ID, 3)
		if err != nil {
			t.Fatalf("Failed to get graph: %v", err)
		}
		if len(graph.Nodes) != 4 {
			t.Errorf("Expected 4 nodes at depth 3, got %d", len(graph.Nodes))
		}
	})

	t.Run("MaxDepth", func(t *testing.T) {
		graph, err := db.GetGraph(memA.ID, 10) // Should be capped at 5
		if err != nil {
			t.Fatalf("Failed to get graph: %v", err)
		}
		if len(graph.Nodes) != 4 {
			t.Errorf("Expected 4 nodes (all in chain), got %d", len(graph.Nodes))
		}
	})

	t.Run("EdgeCount", func(t *testing.T) {
		graph, err := db.GetGraph(memA.ID, 5)
		if err != nil {
			t.Fatalf("Failed to get graph: %v", err)
		}
		if len(graph.Edges) != 3 {
			t.Errorf("Expected 3 edges, got %d", len(graph.Edges))
		}
	})

	t.Run("NodeDistances", func(t *testing.T) {
		graph, err := db.GetGraph(memA.ID, 5)
		if err != nil {
			t.Fatalf("Failed to get graph: %v", err)
		}

		distances := make(map[string]int)
		for _, node := range graph.Nodes {
			distances[node.ID] = node.Distance
		}

		if distances[memA.ID] != 0 {
			t.Errorf("Root node should have distance 0, got %d", distances[memA.ID])
		}
		if distances[memB.ID] != 1 {
			t.Errorf("Node B should have distance 1, got %d", distances[memB.ID])
		}
		if distances[memC.ID] != 2 {
			t.Errorf("Node C should have distance 2, got %d", distances[memC.ID])
		}
	})
}

// TestCategories tests category operations
func TestCategories(t *testing.T) {
	db := newTestDB(t)

	t.Run("CreateCategory", func(t *testing.T) {
		cat := &Category{
			Name:        "Test Category",
			Description: "A test category",
		}

		err := db.CreateCategory(cat)
		if err != nil {
			t.Fatalf("Failed to create category: %v", err)
		}

		if cat.ID == "" {
			t.Error("Category ID should be generated")
		}
		if cat.ConfidenceThreshold != 0.7 {
			t.Errorf("Expected default confidence threshold 0.7, got %f", cat.ConfidenceThreshold)
		}
	})

	t.Run("ListCategories", func(t *testing.T) {
		_ = db.CreateCategory(&Category{Name: "Category A", Description: "A"})
		_ = db.CreateCategory(&Category{Name: "Category B", Description: "B"})

		categories, err := db.ListCategories()
		if err != nil {
			t.Fatalf("Failed to list categories: %v", err)
		}

		if len(categories) < 2 {
			t.Errorf("Expected at least 2 categories, got %d", len(categories))
		}
	})

	t.Run("CategorizeMemory", func(t *testing.T) {
		mem := &Memory{Content: "Test memory for categorization"}
		_ = db.CreateMemory(mem)

		cat := &Category{Name: "Test Cat", Description: "Test"}
		_ = db.CreateCategory(cat)

		err := db.CategorizeMemory(mem.ID, cat.ID, 0.9, "High confidence match")
		if err != nil {
			t.Fatalf("Failed to categorize memory: %v", err)
		}
	})

	t.Run("InvalidConfidence", func(t *testing.T) {
		mem := &Memory{Content: "Test"}
		_ = db.CreateMemory(mem)
		cat := &Category{Name: "Cat2", Description: "Test"}
		_ = db.CreateCategory(cat)

		err := db.CategorizeMemory(mem.ID, cat.ID, 1.5, "Invalid")
		if err == nil {
			t.Error("Expected error for invalid confidence")
		}
	})
}

// TestDomains tests domain operations
func TestDomains(t *testing.T) {
	db := newTestDB(t)

	t.Run("CreateDomain", func(t *testing.T) {
		dom := &Domain{
			Name:        "test-domain",
			Description: "A test domain",
		}

		err := db.CreateDomain(dom)
		if err != nil {
			t.Fatalf("Failed to create domain: %v", err)
		}

		if dom.ID == "" {
			t.Error("Domain ID should be generated")
		}
	})

	t.Run("ListDomains", func(t *testing.T) {
		_ = db.CreateDomain(&Domain{Name: "domain-a", Description: "A"})
		_ = db.CreateDomain(&Domain{Name: "domain-b", Description: "B"})

		domains, err := db.ListDomains()
		if err != nil {
			t.Fatalf("Failed to list domains: %v", err)
		}

		if len(domains) < 2 {
			t.Errorf("Expected at least 2 domains, got %d", len(domains))
		}
	})
}

// TestSessions tests session operations
func TestSessions(t *testing.T) {
	db := newTestDB(t)

	// Create a session manually for testing
	_, err := db.Exec(`
		INSERT INTO agent_sessions (session_id, agent_type, created_at, last_accessed, is_active, metadata)
		VALUES (?, ?, ?, ?, ?, ?)
	`, "test-session-1", "claude-code", time.Now(), time.Now(), true, "{}")
	if err != nil {
		t.Fatalf("Failed to create test session: %v", err)
	}

	sessions, err := db.ListSessions()
	if err != nil {
		t.Fatalf("Failed to list sessions: %v", err)
	}

	if len(sessions) != 1 {
		t.Errorf("Expected 1 session, got %d", len(sessions))
	}

	if sessions[0].AgentType != "claude-code" {
		t.Errorf("Expected agent_type 'claude-code', got %s", sessions[0].AgentType)
	}
}

// TestPerformanceMetrics tests metric recording
func TestPerformanceMetrics(t *testing.T) {
	db := newTestDB(t)

	err := db.RecordMetric("search", 45, 100)
	if err != nil {
		t.Fatalf("Failed to record metric: %v", err)
	}

	var count int
	_ = db.QueryRow("SELECT COUNT(*) FROM performance_metrics WHERE operation_type = ?", "search").Scan(&count)
	if count != 1 {
		t.Errorf("Expected 1 metric record, got %d", count)
	}
}

// TestRelationshipTypes tests validation of relationship types
func TestRelationshipTypes(t *testing.T) {
	validTypes := []string{"references", "contradicts", "expands", "similar", "sequential", "causes", "enables"}

	for _, rt := range validTypes {
		if !IsValidRelationshipType(rt) {
			t.Errorf("Type %q should be valid", rt)
		}
	}

	invalidTypes := []string{"invalid", "relates", "links", ""}
	for _, rt := range invalidTypes {
		if IsValidRelationshipType(rt) {
			t.Errorf("Type %q should be invalid", rt)
		}
	}
}

// TestAgentTypes tests validation of agent types
func TestAgentTypes(t *testing.T) {
	validTypes := []string{"claude-desktop", "claude-code", "api", "unknown"}

	for _, at := range validTypes {
		if !IsValidAgentType(at) {
			t.Errorf("Agent type %q should be valid", at)
		}
	}

	invalidTypes := []string{"invalid", "web", "mobile", ""}
	for _, at := range invalidTypes {
		if IsValidAgentType(at) {
			t.Errorf("Agent type %q should be invalid", at)
		}
	}
}

// TestDatabaseStats tests statistics retrieval
func TestDatabaseStats(t *testing.T) {
	db := newTestDB(t)

	// Create some test data
	for i := 0; i < 5; i++ {
		_ = db.CreateMemory(&Memory{Content: "Test memory"})
	}
	_ = db.CreateCategory(&Category{Name: "Cat", Description: "Test"})
	_ = db.CreateDomain(&Domain{Name: "domain"})

	stats, err := db.GetStats()
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	if stats.MemoryCount != 5 {
		t.Errorf("Expected 5 memories, got %d", stats.MemoryCount)
	}
	if stats.CategoryCount != 1 {
		t.Errorf("Expected 1 category, got %d", stats.CategoryCount)
	}
	if stats.DomainCount != 1 {
		t.Errorf("Expected 1 domain, got %d", stats.DomainCount)
	}
	if stats.SchemaVersion != SchemaVersion {
		t.Errorf("Expected schema version %d, got %d", SchemaVersion, stats.SchemaVersion)
	}
}

// TestCascadeDelete tests that relationships are deleted when memory is deleted
func TestCascadeDelete(t *testing.T) {
	db := newTestDB(t)

	mem1 := &Memory{Content: "Memory 1"}
	mem2 := &Memory{Content: "Memory 2"}
	_ = db.CreateMemory(mem1)
	_ = db.CreateMemory(mem2)

	_ = db.CreateRelationship(&Relationship{
		SourceMemoryID:   mem1.ID,
		TargetMemoryID:   mem2.ID,
		RelationshipType: "references",
		Strength:         0.5,
	})

	// Verify relationship exists
	var relCount int
	_ = db.QueryRow("SELECT COUNT(*) FROM memory_relationships").Scan(&relCount)
	if relCount != 1 {
		t.Fatalf("Expected 1 relationship, got %d", relCount)
	}

	// Delete mem1 - should cascade delete relationship
	_ = db.DeleteMemory(mem1.ID)

	_ = db.QueryRow("SELECT COUNT(*) FROM memory_relationships").Scan(&relCount)
	if relCount != 0 {
		t.Errorf("Expected 0 relationships after cascade delete, got %d", relCount)
	}
}

// TestFTS5Triggers tests that FTS5 triggers work correctly
func TestFTS5Triggers(t *testing.T) {
	db := newTestDB(t)

	// Create memory - should trigger FTS insert
	mem := &Memory{Content: "Unique searchable content xyz123"}
	_ = db.CreateMemory(mem)

	// Search should find it
	results, err := db.SearchFTS("xyz123", &SearchFilters{})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result after insert, got %d", len(results))
	}

	// Update memory content - should trigger FTS update
	newContent := "Updated unique content abc789"
	_ = db.UpdateMemory(mem.ID, &MemoryUpdate{Content: &newContent})

	// Old content should not be found
	results, _ = db.SearchFTS("xyz123", &SearchFilters{})
	if len(results) != 0 {
		t.Errorf("Expected 0 results for old content, got %d", len(results))
	}

	// New content should be found
	results, _ = db.SearchFTS("abc789", &SearchFilters{})
	if len(results) != 1 {
		t.Errorf("Expected 1 result for new content, got %d", len(results))
	}

	// Delete memory - should trigger FTS delete
	_ = db.DeleteMemory(mem.ID)

	results, _ = db.SearchFTS("abc789", &SearchFilters{})
	if len(results) != 0 {
		t.Errorf("Expected 0 results after delete, got %d", len(results))
	}
}

// Helper function to create a test database
func newTestDB(t *testing.T) *Database {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	if err := db.InitSchema(); err != nil {
		t.Fatalf("Failed to initialize schema: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	return db
}
