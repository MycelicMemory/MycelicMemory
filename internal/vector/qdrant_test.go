package vector

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/MycelicMemory/ultrathink/pkg/config"
)

// TestQdrantClient tests the Qdrant client functionality
func TestQdrantClient(t *testing.T) {
	cfg := &config.QdrantConfig{
		Enabled: true,
		URL:     "http://localhost:6333",
	}

	client := NewQdrantClient(cfg)

	t.Run("NewQdrantClient", func(t *testing.T) {
		if client == nil {
			t.Fatal("NewQdrantClient should not return nil")
		}
		if !client.IsEnabled() {
			t.Error("Client should be enabled")
		}
		if client.CollectionName() != "ultrathink-memories" {
			t.Errorf("Expected collection name 'ultrathink-memories', got %s", client.CollectionName())
		}
		if client.Dimension() != 768 {
			t.Errorf("Expected dimension 768, got %d", client.Dimension())
		}
	})

	t.Run("DefaultValues", func(t *testing.T) {
		emptyClient := NewQdrantClient(&config.QdrantConfig{Enabled: true})
		if emptyClient.CollectionName() != "ultrathink-memories" {
			t.Errorf("Default collection should be 'ultrathink-memories', got %s", emptyClient.CollectionName())
		}
		if emptyClient.Dimension() != 768 {
			t.Errorf("Default dimension should be 768, got %d", emptyClient.Dimension())
		}
	})

	t.Run("DisabledClient", func(t *testing.T) {
		disabledClient := NewQdrantClient(&config.QdrantConfig{Enabled: false})
		if disabledClient.IsEnabled() {
			t.Error("Disabled client should not be enabled")
		}
		if disabledClient.IsAvailable() {
			t.Error("Disabled client should not be available")
		}
	})
}

// TestQdrantClientIntegration tests with actual Qdrant server
// Skip if Qdrant is not available
func TestQdrantClientIntegration(t *testing.T) {
	cfg := &config.QdrantConfig{
		Enabled: true,
		URL:     "http://localhost:6333",
	}

	client := NewQdrantClient(cfg)

	// Skip if Qdrant is not available
	if !client.IsAvailable() {
		t.Skip("Qdrant is not available, skipping integration tests")
	}

	t.Run("InitCollection", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := client.InitCollection(ctx)
		if err != nil {
			t.Fatalf("InitCollection failed: %v", err)
		}

		// Should be idempotent
		err = client.InitCollection(ctx)
		if err != nil {
			t.Fatalf("Second InitCollection failed: %v", err)
		}
	})

	t.Run("GetCollectionInfo", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		info, err := client.GetCollectionInfo(ctx)
		if err != nil {
			t.Fatalf("GetCollectionInfo failed: %v", err)
		}

		if info == nil {
			t.Fatal("Expected non-nil collection info")
		}

		t.Logf("Collection info: vectors=%d, points=%d, status=%s",
			info.VectorCount, info.PointsCount, info.Status)
	})

	// Test CRUD operations with a proper UUID
	testID := uuid.New().String()

	t.Run("Upsert", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Create a test vector (768 dimensions)
		vector := make([]float64, 768)
		for i := range vector {
			vector[i] = float64(i) / 768.0
		}

		payload := map[string]interface{}{
			"content":    "Test memory content",
			"session_id": "test-session",
			"domain":     "testing",
			"importance": 5,
		}

		err := client.Upsert(ctx, testID, vector, payload)
		if err != nil {
			t.Fatalf("Upsert failed: %v", err)
		}
	})

	t.Run("GetPoint", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		point, err := client.GetPoint(ctx, testID)
		if err != nil {
			t.Fatalf("GetPoint failed: %v", err)
		}

		if point == nil {
			t.Fatal("Expected non-nil point")
		}

		if point.ID != testID {
			t.Errorf("Expected ID %s, got %s", testID, point.ID)
		}

		if len(point.Vector) != 768 {
			t.Errorf("Expected vector dimension 768, got %d", len(point.Vector))
		}
	})

	t.Run("Search", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Search with similar vector
		queryVector := make([]float64, 768)
		for i := range queryVector {
			queryVector[i] = float64(i) / 768.0
		}

		results, err := client.Search(ctx, &SearchOptions{
			Vector:      queryVector,
			Limit:       5,
			MinScore:    0.0,
			WithPayload: true,
		})
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if len(results) == 0 {
			t.Error("Expected at least one result")
		}

		// Check first result
		if len(results) > 0 {
			result := results[0]
			t.Logf("Top result: ID=%s, Score=%.4f", result.ID, result.Score)

			if result.Score < 0.9 {
				t.Logf("Warning: Expected high similarity for same vector, got %.4f", result.Score)
			}
		}
	})

	t.Run("SearchWithFilter", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		queryVector := make([]float64, 768)
		for i := range queryVector {
			queryVector[i] = float64(i) / 768.0
		}

		filter := map[string]interface{}{
			"must": []map[string]interface{}{
				{
					"key":   "domain",
					"match": map[string]interface{}{"value": "testing"},
				},
			},
		}

		results, err := client.Search(ctx, &SearchOptions{
			Vector:      queryVector,
			Limit:       5,
			Filter:      filter,
			WithPayload: true,
		})
		if err != nil {
			t.Fatalf("Search with filter failed: %v", err)
		}

		// All results should have domain = testing
		for _, result := range results {
			if result.Payload != nil {
				if domain, ok := result.Payload["domain"].(string); ok {
					if domain != "testing" {
						t.Errorf("Expected domain 'testing', got %s", domain)
					}
				}
			}
		}
	})

	t.Run("Delete", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := client.Delete(ctx, []string{testID})
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		// Verify deletion
		point, err := client.GetPoint(ctx, testID)
		if err != nil {
			t.Fatalf("GetPoint after delete failed: %v", err)
		}

		if point != nil {
			t.Error("Expected nil point after deletion")
		}
	})

	t.Run("UpsertPoints", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		points := make([]Point, 3)
		for i := 0; i < 3; i++ {
			vector := make([]float64, 768)
			for j := range vector {
				vector[j] = float64(i+j) / 768.0
			}
			points[i] = Point{
				ID:     uuid.New().String(),
				Vector: vector,
				Payload: map[string]interface{}{
					"content": "Batch test " + string(rune('A'+i)),
				},
			}
		}

		err := client.UpsertPoints(ctx, points)
		if err != nil {
			t.Fatalf("UpsertPoints failed: %v", err)
		}

		// Cleanup
		ids := make([]string, len(points))
		for i, p := range points {
			ids[i] = p.ID
		}
		client.Delete(ctx, ids)
	})
}

// TestSearchOptions tests search options validation
func TestSearchOptions(t *testing.T) {
	cfg := &config.QdrantConfig{
		Enabled: true,
		URL:     "http://localhost:6333",
	}

	client := NewQdrantClient(cfg)

	// Skip if Qdrant is not available
	if !client.IsAvailable() {
		t.Skip("Qdrant is not available, skipping validation tests")
	}

	t.Run("VectorDimensionMismatch", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Wrong dimension vector
		wrongVector := make([]float64, 100)

		_, err := client.Search(ctx, &SearchOptions{
			Vector: wrongVector,
			Limit:  5,
		})

		if err == nil {
			t.Error("Expected error for wrong vector dimension")
		}
	})

	t.Run("DefaultLimit", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// First, ensure collection exists
		client.InitCollection(ctx)

		vector := make([]float64, 768)
		for i := range vector {
			vector[i] = 0.5
		}

		// With Limit 0, should use default of 10
		results, err := client.Search(ctx, &SearchOptions{
			Vector: vector,
			Limit:  0,
		})

		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// Results should be limited (or fewer if not enough points)
		if len(results) > 10 {
			t.Errorf("Expected at most 10 results, got %d", len(results))
		}
	})
}
