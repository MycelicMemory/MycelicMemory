package vector

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/MycelicMemory/ultrathink/pkg/config"
)

// QdrantClient provides vector storage capabilities via Qdrant
// VERIFIED: Matches local-memory Qdrant integration
type QdrantClient struct {
	baseURL        string
	collectionName string
	httpClient     *http.Client
	enabled        bool
	dimension      int // 768 for nomic-embed-text
}

// NewQdrantClient creates a new Qdrant client
func NewQdrantClient(cfg *config.QdrantConfig) *QdrantClient {
	client := &QdrantClient{
		baseURL:        cfg.URL,
		collectionName: "ultrathink-memories",
		enabled:        cfg.Enabled,
		dimension:      768, // nomic-embed-text dimension
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// Set defaults
	if client.baseURL == "" {
		client.baseURL = "http://localhost:6333"
	}

	return client
}

// IsAvailable checks if Qdrant is available
func (c *QdrantClient) IsAvailable() bool {
	if !c.enabled {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/collections", nil)
	if err != nil {
		return false
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// InitCollection creates the collection if it doesn't exist
// VERIFIED: HNSW configuration (m=16, ef_construct=100)
func (c *QdrantClient) InitCollection(ctx context.Context) error {
	if !c.enabled {
		return fmt.Errorf("qdrant is not enabled")
	}

	// Check if collection exists
	exists, err := c.collectionExists(ctx)
	if err != nil {
		return fmt.Errorf("failed to check collection: %w", err)
	}

	if exists {
		return nil
	}

	// Create collection with HNSW configuration
	createReq := map[string]interface{}{
		"vectors": map[string]interface{}{
			"size":     c.dimension,
			"distance": "Cosine",
		},
		"hnsw_config": map[string]interface{}{
			"m":            16,  // VERIFIED from local-memory
			"ef_construct": 100, // VERIFIED from local-memory
		},
	}

	jsonBody, err := json.Marshal(createReq)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/collections/%s", c.baseURL, c.collectionName)
	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("create collection request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("create collection failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (c *QdrantClient) collectionExists(ctx context.Context) (bool, error) {
	url := fmt.Sprintf("%s/collections/%s", c.baseURL, c.collectionName)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}

// Point represents a vector point in Qdrant
type Point struct {
	ID      string                 `json:"id"`
	Vector  []float64              `json:"vector"`
	Payload map[string]interface{} `json:"payload,omitempty"`
}

// UpsertPoints inserts or updates vector points
func (c *QdrantClient) UpsertPoints(ctx context.Context, points []Point) error {
	if !c.enabled {
		return fmt.Errorf("qdrant is not enabled")
	}

	// Convert to Qdrant format
	qdrantPoints := make([]map[string]interface{}, len(points))
	for i, p := range points {
		qdrantPoints[i] = map[string]interface{}{
			"id":      p.ID,
			"vector":  p.Vector,
			"payload": p.Payload,
		}
	}

	upsertReq := map[string]interface{}{
		"points": qdrantPoints,
	}

	jsonBody, err := json.Marshal(upsertReq)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/collections/%s/points", c.baseURL, c.collectionName)
	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("upsert request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upsert failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// Upsert inserts or updates a single vector
func (c *QdrantClient) Upsert(ctx context.Context, id string, vector []float64, payload map[string]interface{}) error {
	return c.UpsertPoints(ctx, []Point{{ID: id, Vector: vector, Payload: payload}})
}

// SearchResult represents a search result from Qdrant
type SearchResult struct {
	ID      string                 `json:"id"`
	Score   float64                `json:"score"`
	Payload map[string]interface{} `json:"payload,omitempty"`
}

// SearchOptions contains options for vector search
type SearchOptions struct {
	Vector    []float64
	Limit     int
	MinScore  float64
	Filter    map[string]interface{}
	WithPayload bool
}

// Search performs vector similarity search
func (c *QdrantClient) Search(ctx context.Context, opts *SearchOptions) ([]SearchResult, error) {
	if !c.enabled {
		return nil, fmt.Errorf("qdrant is not enabled")
	}

	if len(opts.Vector) != c.dimension {
		return nil, fmt.Errorf("vector dimension mismatch: expected %d, got %d", c.dimension, len(opts.Vector))
	}

	limit := opts.Limit
	if limit <= 0 {
		limit = 10
	}

	searchReq := map[string]interface{}{
		"vector":       opts.Vector,
		"limit":        limit,
		"with_payload": opts.WithPayload,
	}

	if opts.MinScore > 0 {
		searchReq["score_threshold"] = opts.MinScore
	}

	if opts.Filter != nil {
		searchReq["filter"] = opts.Filter
	}

	jsonBody, err := json.Marshal(searchReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/collections/%s/points/search", c.baseURL, c.collectionName)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("search request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("search failed with status %d: %s", resp.StatusCode, string(body))
	}

	var searchResp struct {
		Result []struct {
			ID      interface{}            `json:"id"`
			Score   float64                `json:"score"`
			Payload map[string]interface{} `json:"payload"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	results := make([]SearchResult, len(searchResp.Result))
	for i, r := range searchResp.Result {
		// Handle ID as string or number
		var id string
		switch v := r.ID.(type) {
		case string:
			id = v
		case float64:
			id = fmt.Sprintf("%.0f", v)
		default:
			id = fmt.Sprintf("%v", v)
		}

		results[i] = SearchResult{
			ID:      id,
			Score:   r.Score,
			Payload: r.Payload,
		}
	}

	return results, nil
}

// Delete removes a vector by ID
func (c *QdrantClient) Delete(ctx context.Context, ids []string) error {
	if !c.enabled {
		return fmt.Errorf("qdrant is not enabled")
	}

	deleteReq := map[string]interface{}{
		"points": ids,
	}

	jsonBody, err := json.Marshal(deleteReq)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/collections/%s/points/delete", c.baseURL, c.collectionName)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("delete request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetPoint retrieves a point by ID
func (c *QdrantClient) GetPoint(ctx context.Context, id string) (*Point, error) {
	if !c.enabled {
		return nil, fmt.Errorf("qdrant is not enabled")
	}

	url := fmt.Sprintf("%s/collections/%s/points/%s", c.baseURL, c.collectionName, id)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get failed with status %d: %s", resp.StatusCode, string(body))
	}

	var getResp struct {
		Result struct {
			ID      string                 `json:"id"`
			Vector  []float64              `json:"vector"`
			Payload map[string]interface{} `json:"payload"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&getResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &Point{
		ID:      getResp.Result.ID,
		Vector:  getResp.Result.Vector,
		Payload: getResp.Result.Payload,
	}, nil
}

// CollectionInfo represents collection statistics
type CollectionInfo struct {
	VectorCount int64 `json:"vectors_count"`
	PointsCount int64 `json:"points_count"`
	Status      string `json:"status"`
}

// GetCollectionInfo returns collection statistics
func (c *QdrantClient) GetCollectionInfo(ctx context.Context) (*CollectionInfo, error) {
	if !c.enabled {
		return nil, fmt.Errorf("qdrant is not enabled")
	}

	url := fmt.Sprintf("%s/collections/%s", c.baseURL, c.collectionName)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get collection request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get collection failed with status %d: %s", resp.StatusCode, string(body))
	}

	var infoResp struct {
		Result struct {
			VectorsCount int64  `json:"vectors_count"`
			PointsCount  int64  `json:"points_count"`
			Status       string `json:"status"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&infoResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &CollectionInfo{
		VectorCount: infoResp.Result.VectorsCount,
		PointsCount: infoResp.Result.PointsCount,
		Status:      infoResp.Result.Status,
	}, nil
}

// IsEnabled returns whether Qdrant is enabled
func (c *QdrantClient) IsEnabled() bool {
	return c.enabled
}

// CollectionName returns the collection name
func (c *QdrantClient) CollectionName() string {
	return c.collectionName
}

// Dimension returns the vector dimension
func (c *QdrantClient) Dimension() int {
	return c.dimension
}
