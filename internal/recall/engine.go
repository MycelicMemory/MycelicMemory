package recall

import (
	"context"
	"time"

	"github.com/MycelicMemory/mycelicmemory/internal/ai"
	"github.com/MycelicMemory/mycelicmemory/internal/database"
	"github.com/MycelicMemory/mycelicmemory/internal/logging"
	"github.com/MycelicMemory/mycelicmemory/internal/relationships"
	"github.com/MycelicMemory/mycelicmemory/internal/search"
	"github.com/MycelicMemory/mycelicmemory/pkg/config"
)

// Engine provides proactive memory recall using multi-signal scoring
type Engine struct {
	db        *database.Database
	cfg       *config.Config
	aiManager *ai.Manager
	searchEng *search.Engine
	relSvc    *relationships.Service
	log       *logging.Logger
}

// RecallRequest contains parameters for a context recall query
type RecallRequest struct {
	Context string   `json:"context"`
	Files   []string `json:"files,omitempty"`
	Project string   `json:"project,omitempty"`
	Limit   int      `json:"limit,omitempty"`
	Depth   int      `json:"depth,omitempty"`
}

// RecallResult contains the results of a context recall
type RecallResult struct {
	Memories      []RecallMemory `json:"memories"`
	TotalFound    int            `json:"total_found"`
	SearchMode    string         `json:"search_mode"`
	Timing        RecallTiming   `json:"timing"`
	GraphExpanded int            `json:"graph_expanded"`
}

// RecallMemory represents a recalled memory with scoring metadata
type RecallMemory struct {
	Memory        *database.Memory `json:"memory"`
	Score         float64          `json:"score"`
	MatchType     string           `json:"match_type"`
	RelationChain []RelationLink   `json:"relation_chain,omitempty"`
}

// RelationLink represents one hop in a relationship chain
type RelationLink struct {
	FromID   string  `json:"from_id"`
	ToID     string  `json:"to_id"`
	Type     string  `json:"type"`
	Strength float64 `json:"strength"`
}

// RecallTiming contains timing breakdown for the recall operation
type RecallTiming struct {
	EmbeddingMs int64 `json:"embedding_ms"`
	SemanticMs  int64 `json:"semantic_ms"`
	KeywordMs   int64 `json:"keyword_ms"`
	GraphMs     int64 `json:"graph_ms"`
	RerankMs    int64 `json:"rerank_ms"`
	TotalMs     int64 `json:"total_ms"`
}

// NewEngine creates a new recall engine
func NewEngine(db *database.Database, cfg *config.Config, aiManager *ai.Manager, searchEng *search.Engine, relSvc *relationships.Service) *Engine {
	return &Engine{
		db:        db,
		cfg:       cfg,
		aiManager: aiManager,
		searchEng: searchEng,
		relSvc:    relSvc,
		log:       logging.GetLogger("recall"),
	}
}

// Recall performs proactive memory recall for a given context
func (e *Engine) Recall(ctx context.Context, req *RecallRequest) (*RecallResult, error) {
	totalStart := time.Now()
	timing := RecallTiming{}

	// Set defaults
	if req.Limit <= 0 {
		req.Limit = 10
	}
	if req.Depth <= 0 {
		req.Depth = 1
	}

	// Determine search mode
	mode := e.determineMode()
	e.log.Debug("recall mode", "mode", mode)

	// Build context for search
	keywords := extractKeywords(req.Context, 20)
	if len(req.Files) > 0 {
		fileTags := extractTagsFromFiles(req.Files)
		keywords = append(keywords, fileTags...)
	}
	if req.Project != "" {
		keywords = append(keywords, extractKeywords(req.Project, 5)...)
	}

	candidates := make(map[string]*scoredCandidate)

	// 1. Semantic search (if available)
	if mode == "full" {
		start := time.Now()
		embText := truncateForEmbedding(req.Context, 2000)

		semanticResults, err := e.aiManager.SemanticSearch(ctx, &ai.SemanticSearchOptions{
			Query:        embText,
			Limit:        req.Limit * 3,
			MinScore:     0.3,
			WithMetadata: true,
		})
		timing.EmbeddingMs = time.Since(start).Milliseconds()
		timing.SemanticMs = timing.EmbeddingMs // combined for simplicity

		if err != nil {
			e.log.Warn("semantic search failed, falling back", "error", err)
			mode = "keyword_graph"
		} else {
			for _, r := range semanticResults {
				candidates[r.MemoryID] = &scoredCandidate{
					memoryID:      r.MemoryID,
					semanticScore: r.Score,
					matchType:     "semantic",
				}
			}
		}
	}

	// 2. Keyword search (always)
	kwStart := time.Now()
	query := buildSearchQuery(keywords)
	if query != "" {
		kwResults, err := e.searchEng.Search(&search.SearchOptions{
			Query:      query,
			SearchType: search.SearchTypeKeyword,
			Limit:      req.Limit * 2,
		})
		if err != nil {
			e.log.Warn("keyword search failed", "error", err)
		} else {
			for _, r := range kwResults {
				if existing, ok := candidates[r.Memory.ID]; ok {
					// Memory already found via semantic — add keyword score
					existing.keywordScore = r.Relevance
				} else {
					candidates[r.Memory.ID] = &scoredCandidate{
						memoryID:     r.Memory.ID,
						keywordScore: r.Relevance,
						matchType:    "keyword",
						importance:   r.Memory.Importance,
						createdAt:    r.Memory.CreatedAt,
					}
				}
			}
		}
	}
	timing.KeywordMs = time.Since(kwStart).Milliseconds()

	// 3. Graph expansion (top 5 semantic hits)
	graphExpanded := 0
	if mode == "full" || mode == "keyword_graph" {
		graphStart := time.Now()

		// Get top candidates by semantic score for graph expansion
		var topIDs []string
		for id, c := range candidates {
			if c.matchType == "semantic" && len(topIDs) < 5 {
				topIDs = append(topIDs, id)
			}
		}
		// If no semantic, use top keyword hits
		if len(topIDs) == 0 {
			for id := range candidates {
				topIDs = append(topIDs, id)
				if len(topIDs) >= 5 {
					break
				}
			}
		}

		for _, memID := range topIDs {
			rels, err := e.db.GetRelationshipsForMemory(memID)
			if err != nil {
				continue
			}

			// Count relationships for the source node
			if c, ok := candidates[memID]; ok {
				c.relationshipCount = len(rels)
			}

			for _, rel := range rels {
				neighborID := rel.TargetMemoryID
				if neighborID == memID {
					neighborID = rel.SourceMemoryID
				}

				// Only expand relevant relationship types
				switch rel.RelationshipType {
				case "expands", "references", "enables", "similar":
					// good types for recall
				default:
					continue
				}

				if _, exists := candidates[neighborID]; !exists {
					candidates[neighborID] = &scoredCandidate{
						memoryID:  neighborID,
						matchType: "graph",
						relationChain: []RelationLink{{
							FromID:   memID,
							ToID:     neighborID,
							Type:     rel.RelationshipType,
							Strength: rel.Strength,
						}},
					}
					graphExpanded++
				} else {
					// Update relationship count for existing candidate
					candidates[neighborID].relationshipCount++
				}
			}
		}
		timing.GraphMs = time.Since(graphStart).Milliseconds()
	}

	// 4. Fetch full memory data for all candidates
	memoryMap := make(map[string]*database.Memory)
	for memID := range candidates {
		mem, err := e.db.GetMemory(memID)
		if err != nil || mem == nil {
			delete(candidates, memID)
			continue
		}
		memoryMap[memID] = mem

		// Fill in metadata for scoring
		c := candidates[memID]
		c.importance = mem.Importance
		c.createdAt = mem.CreatedAt
	}

	// 5. Score and rank
	rerankStart := time.Now()
	ranked := e.scoreAndRank(candidates, req.Limit, mode)
	timing.RerankMs = time.Since(rerankStart).Milliseconds()

	// 6. Build result
	memories := make([]RecallMemory, 0, len(ranked))
	for _, r := range ranked {
		mem := memoryMap[r.memoryID]
		if mem == nil {
			continue
		}
		memories = append(memories, RecallMemory{
			Memory:        mem,
			Score:         r.semanticScore, // final score stored here
			MatchType:     r.matchType,
			RelationChain: r.relationChain,
		})
	}

	timing.TotalMs = time.Since(totalStart).Milliseconds()

	return &RecallResult{
		Memories:      memories,
		TotalFound:    len(candidates),
		SearchMode:    mode,
		Timing:        timing,
		GraphExpanded: graphExpanded,
	}, nil
}

// determineMode checks AI availability to decide search strategy
func (e *Engine) determineMode() string {
	if e.aiManager == nil {
		return "keyword_only"
	}

	status := e.aiManager.GetStatus()
	if status.OllamaAvailable && status.QdrantAvailable {
		return "full"
	}
	if status.QdrantAvailable {
		return "keyword_graph"
	}
	return "keyword_only"
}
