package locomo

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// ResultsStore manages benchmark result persistence
type ResultsStore struct {
	baseDir string
}

// NewResultsStore creates a new results store
func NewResultsStore(baseDir string) *ResultsStore {
	return &ResultsStore{baseDir: baseDir}
}

// Save saves benchmark results to disk
func (s *ResultsStore) Save(results *BenchmarkResults) (string, error) {
	// Create results directory if needed
	if err := os.MkdirAll(s.baseDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create results directory: %w", err)
	}

	// Generate filename with timestamp
	filename := fmt.Sprintf("locomo_%s_%s.json",
		results.Strategy,
		results.Timestamp.Format("2006-01-02_15-04-05"))

	path := filepath.Join(s.baseDir, filename)

	// Marshal to JSON
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal results: %w", err)
	}

	// Write to file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write results: %w", err)
	}

	log.Info("saved benchmark results", "path", path)
	return path, nil
}

// Load loads benchmark results from a file
func (s *ResultsStore) Load(path string) (*BenchmarkResults, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read results: %w", err)
	}

	var results BenchmarkResults
	if err := json.Unmarshal(data, &results); err != nil {
		return nil, fmt.Errorf("failed to parse results: %w", err)
	}

	return &results, nil
}

// List returns all saved benchmark results
func (s *ResultsStore) List() ([]*ResultSummary, error) {
	files, err := filepath.Glob(filepath.Join(s.baseDir, "locomo_*.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to list results: %w", err)
	}

	var summaries []*ResultSummary
	for _, file := range files {
		results, err := s.Load(file)
		if err != nil {
			log.Warn("failed to load result file", "file", file, "error", err)
			continue
		}

		summaries = append(summaries, &ResultSummary{
			Path:      file,
			Timestamp: results.Timestamp,
			Strategy:  results.Strategy,
			Model:     results.Model,
			F1:        results.Overall.F1,
			Questions: results.Overall.Count,
			Duration:  results.Duration,
		})
	}

	// Sort by timestamp, newest first
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Timestamp.After(summaries[j].Timestamp)
	})

	return summaries, nil
}

// GetLatest returns the most recent benchmark result
func (s *ResultsStore) GetLatest() (*BenchmarkResults, error) {
	summaries, err := s.List()
	if err != nil {
		return nil, err
	}

	if len(summaries) == 0 {
		return nil, fmt.Errorf("no benchmark results found")
	}

	return s.Load(summaries[0].Path)
}

// GetByStrategy returns the most recent result for a specific strategy
func (s *ResultsStore) GetByStrategy(strategy RetrievalStrategy) (*BenchmarkResults, error) {
	summaries, err := s.List()
	if err != nil {
		return nil, err
	}

	for _, summary := range summaries {
		if summary.Strategy == strategy {
			return s.Load(summary.Path)
		}
	}

	return nil, fmt.Errorf("no results found for strategy: %s", strategy)
}

// ResultSummary provides a summary of benchmark results
type ResultSummary struct {
	Path      string            `json:"path"`
	Timestamp time.Time         `json:"timestamp"`
	Strategy  RetrievalStrategy `json:"strategy"`
	Model     string            `json:"model"`
	F1        float64           `json:"f1"`
	Questions int               `json:"questions"`
	Duration  time.Duration     `json:"duration"`
}

// Compare compares two benchmark results
func Compare(a, b *BenchmarkResults) *ComparisonResult {
	comparison := &ComparisonResult{
		ResultA:    a,
		ResultB:    b,
		Overall:    compareMetrics(a.Overall, b.Overall),
		Categories: make(map[QuestionCategory]MetricsDiff),
	}

	// Compare categories
	allCategories := make(map[QuestionCategory]bool)
	for cat := range a.Categories {
		allCategories[cat] = true
	}
	for cat := range b.Categories {
		allCategories[cat] = true
	}

	for cat := range allCategories {
		metricsA := a.Categories[cat]
		metricsB := b.Categories[cat]
		comparison.Categories[cat] = compareMetrics(metricsA, metricsB)
	}

	return comparison
}

// ComparisonResult contains the comparison between two benchmark runs
type ComparisonResult struct {
	ResultA    *BenchmarkResults
	ResultB    *BenchmarkResults
	Overall    MetricsDiff
	Categories map[QuestionCategory]MetricsDiff
}

// MetricsDiff represents the difference between two metrics
type MetricsDiff struct {
	F1Diff        float64 `json:"f1_diff"`
	PrecisionDiff float64 `json:"precision_diff"`
	RecallDiff    float64 `json:"recall_diff"`
	Improved      bool    `json:"improved"`
}

func compareMetrics(a, b Metrics) MetricsDiff {
	diff := MetricsDiff{
		F1Diff:        b.F1 - a.F1,
		PrecisionDiff: b.Precision - a.Precision,
		RecallDiff:    b.Recall - a.Recall,
	}
	diff.Improved = diff.F1Diff > 0
	return diff
}

// Aggregate aggregates multiple benchmark results
func Aggregate(results []*BenchmarkResults) *AggregatedResults {
	if len(results) == 0 {
		return nil
	}

	agg := &AggregatedResults{
		RunCount:   len(results),
		Categories: make(map[QuestionCategory]AggregatedMetrics),
	}

	// Aggregate overall metrics
	var f1Sum, precSum, recallSum float64
	var f1Min, f1Max float64 = 100, 0

	for _, r := range results {
		f1Sum += r.Overall.F1
		precSum += r.Overall.Precision
		recallSum += r.Overall.Recall

		if r.Overall.F1 < f1Min {
			f1Min = r.Overall.F1
		}
		if r.Overall.F1 > f1Max {
			f1Max = r.Overall.F1
		}
	}

	n := float64(len(results))
	agg.Overall = AggregatedMetrics{
		MeanF1:        f1Sum / n,
		MeanPrecision: precSum / n,
		MeanRecall:    recallSum / n,
		MinF1:         f1Min,
		MaxF1:         f1Max,
	}

	// Calculate standard deviation for F1
	var varianceSum float64
	for _, r := range results {
		diff := r.Overall.F1 - agg.Overall.MeanF1
		varianceSum += diff * diff
	}
	agg.Overall.StdDevF1 = sqrt(varianceSum / n)

	// Aggregate by category
	allCategories := make(map[QuestionCategory][]Metrics)
	for _, r := range results {
		for cat, metrics := range r.Categories {
			allCategories[cat] = append(allCategories[cat], metrics)
		}
	}

	for cat, metricsList := range allCategories {
		agg.Categories[cat] = aggregateMetricsList(metricsList)
	}

	return agg
}

// AggregatedResults contains aggregated metrics from multiple runs
type AggregatedResults struct {
	RunCount   int                                   `json:"run_count"`
	Overall    AggregatedMetrics                     `json:"overall"`
	Categories map[QuestionCategory]AggregatedMetrics `json:"categories"`
}

// AggregatedMetrics contains statistical aggregations
type AggregatedMetrics struct {
	MeanF1        float64 `json:"mean_f1"`
	MeanPrecision float64 `json:"mean_precision"`
	MeanRecall    float64 `json:"mean_recall"`
	StdDevF1      float64 `json:"std_dev_f1"`
	MinF1         float64 `json:"min_f1"`
	MaxF1         float64 `json:"max_f1"`
}

func aggregateMetricsList(list []Metrics) AggregatedMetrics {
	if len(list) == 0 {
		return AggregatedMetrics{}
	}

	var f1Sum, precSum, recallSum float64
	var f1Min, f1Max float64 = 100, 0

	for _, m := range list {
		f1Sum += m.F1
		precSum += m.Precision
		recallSum += m.Recall

		if m.F1 < f1Min {
			f1Min = m.F1
		}
		if m.F1 > f1Max {
			f1Max = m.F1
		}
	}

	n := float64(len(list))
	meanF1 := f1Sum / n

	var varianceSum float64
	for _, m := range list {
		diff := m.F1 - meanF1
		varianceSum += diff * diff
	}

	return AggregatedMetrics{
		MeanF1:        meanF1,
		MeanPrecision: precSum / n,
		MeanRecall:    recallSum / n,
		StdDevF1:      sqrt(varianceSum / n),
		MinF1:         f1Min,
		MaxF1:         f1Max,
	}
}

// Simple sqrt implementation to avoid math import for just one function
func sqrt(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x == 0 {
		return 0
	}

	// Newton's method
	z := x / 2
	for i := 0; i < 10; i++ {
		z = z - (z*z-x)/(2*z)
	}
	return z
}

// ExportCSV exports results to CSV format
func ExportCSV(results *BenchmarkResults) string {
	var csv string
	csv += "conversation_id,question,category,ground_truth,generated_answer,f1,precision,recall,evidence_found\n"

	for _, q := range results.Questions {
		csv += fmt.Sprintf("%s,%q,%s,%q,%q,%.4f,%.4f,%.4f,%t\n",
			q.ConversationID,
			q.Question,
			q.Category,
			q.GroundTruth,
			q.GeneratedAnswer,
			q.F1,
			q.Precision,
			q.Recall,
			q.EvidenceFound)
	}

	return csv
}
