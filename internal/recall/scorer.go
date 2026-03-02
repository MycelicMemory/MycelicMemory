package recall

import (
	"math"
	"sort"
	"time"
)

const (
	WeightSemantic   = 0.40
	WeightKeyword    = 0.20
	WeightImportance = 0.15
	WeightRecency    = 0.15
	WeightGraph      = 0.10
	RecencyHalfLife  = 30.0 // days
)

// scoredCandidate tracks scoring signals for a memory candidate
type scoredCandidate struct {
	memoryID          string
	semanticScore     float64
	keywordScore      float64
	importance        int
	createdAt         time.Time
	relationshipCount int
	matchType         string // "semantic", "keyword", "graph"
	relationChain     []RelationLink
}

// scoreAndRank computes final scores for all candidates and returns top-K
func (e *Engine) scoreAndRank(candidates map[string]*scoredCandidate, limit int, mode string) []scoredCandidate {
	ranked := make([]scoredCandidate, 0, len(candidates))

	for _, c := range candidates {
		var finalScore float64

		switch mode {
		case "full":
			finalScore = WeightSemantic*c.semanticScore +
				WeightKeyword*normalizeKeywordScore(c.keywordScore) +
				WeightImportance*normalizeImportance(c.importance) +
				WeightRecency*recencyDecay(c.createdAt, RecencyHalfLife) +
				WeightGraph*graphConnectivity(c.relationshipCount)
		case "keyword_graph":
			// Redistribute semantic weight to keyword and graph
			finalScore = (WeightKeyword+WeightSemantic*0.5)*normalizeKeywordScore(c.keywordScore) +
				WeightImportance*normalizeImportance(c.importance) +
				WeightRecency*recencyDecay(c.createdAt, RecencyHalfLife) +
				(WeightGraph+WeightSemantic*0.5)*graphConnectivity(c.relationshipCount)
		default: // keyword_only
			// Redistribute semantic+graph to keyword, importance, recency
			kwWeight := WeightKeyword + WeightSemantic + WeightGraph
			finalScore = kwWeight*normalizeKeywordScore(c.keywordScore) +
				WeightImportance*normalizeImportance(c.importance) +
				WeightRecency*recencyDecay(c.createdAt, RecencyHalfLife)
		}

		c.semanticScore = finalScore // reuse field for final score
		ranked = append(ranked, *c)
	}

	// Sort by final score descending
	sort.Slice(ranked, func(i, j int) bool {
		return ranked[i].semanticScore > ranked[j].semanticScore
	})

	if len(ranked) > limit {
		ranked = ranked[:limit]
	}

	return ranked
}

// normalizeKeywordScore converts BM25 score to 0-1 range
func normalizeKeywordScore(bm25Score float64) float64 {
	if bm25Score == 0 {
		return 0
	}
	return 1.0 / (1.0 + math.Abs(bm25Score))
}

// normalizeImportance converts importance (1-10) to 0-1 range
func normalizeImportance(importance int) float64 {
	if importance < 1 {
		importance = 1
	}
	if importance > 10 {
		importance = 10
	}
	return float64(importance) / 10.0
}

// recencyDecay computes exponential decay based on age
func recencyDecay(createdAt time.Time, halfLifeDays float64) float64 {
	if createdAt.IsZero() {
		return 0.5 // default for unknown dates
	}
	ageDays := time.Since(createdAt).Hours() / 24.0
	if ageDays < 0 {
		ageDays = 0
	}
	// exp(-0.693 * age_days / half_life) gives 0.5 at half_life
	return math.Exp(-0.693 * ageDays / halfLifeDays)
}

// graphConnectivity normalizes relationship count to 0-1
func graphConnectivity(relationshipCount int) float64 {
	if relationshipCount <= 0 {
		return 0
	}
	return math.Min(1.0, float64(relationshipCount)/5.0)
}
