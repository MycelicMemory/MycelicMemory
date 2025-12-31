package locomo

import (
	"fmt"
	"strings"
	"time"
)

// ReportGenerator generates human-readable reports from benchmark results
type ReportGenerator struct{}

// NewReportGenerator creates a new report generator
func NewReportGenerator() *ReportGenerator {
	return &ReportGenerator{}
}

// GenerateMarkdown generates a Markdown report from benchmark results
func (g *ReportGenerator) GenerateMarkdown(results *BenchmarkResults) string {
	var sb strings.Builder

	// Header
	sb.WriteString("# LoCoMo Benchmark Results\n\n")
	sb.WriteString(fmt.Sprintf("**Date:** %s\n\n", results.Timestamp.Format("January 2, 2006 15:04:05")))
	sb.WriteString(fmt.Sprintf("**Model:** %s\n\n", results.Model))
	sb.WriteString(fmt.Sprintf("**Retrieval Strategy:** %s\n\n", results.Strategy))
	sb.WriteString(fmt.Sprintf("**Duration:** %s\n\n", results.Duration.Round(time.Second)))

	// Overall Results
	sb.WriteString("## Overall Results\n\n")
	sb.WriteString("| Metric | Score |\n")
	sb.WriteString("|--------|-------|\n")
	sb.WriteString(fmt.Sprintf("| **F1** | %.2f |\n", results.Overall.F1))
	sb.WriteString(fmt.Sprintf("| Precision | %.2f |\n", results.Overall.Precision))
	sb.WriteString(fmt.Sprintf("| Recall | %.2f |\n", results.Overall.Recall))
	sb.WriteString(fmt.Sprintf("| Questions | %d |\n\n", results.Overall.Count))

	// Category Breakdown
	sb.WriteString("## Results by Category\n\n")
	sb.WriteString("| Category | F1 | Precision | Recall | Count |\n")
	sb.WriteString("|----------|-----|-----------|--------|-------|\n")

	categories := []QuestionCategory{
		CategorySingleHop,
		CategoryMultiHop,
		CategoryTemporal,
		CategoryCommonsense,
		CategoryAdversarial,
	}

	for _, cat := range categories {
		if metrics, ok := results.Categories[cat]; ok {
			sb.WriteString(fmt.Sprintf("| %s | %.2f | %.2f | %.2f | %d |\n",
				cat, metrics.F1, metrics.Precision, metrics.Recall, metrics.Count))
		}
	}
	sb.WriteString("\n")

	// Baseline Comparison
	sb.WriteString("## Comparison with Baselines\n\n")
	sb.WriteString("| Model | F1 | Difference |\n")
	sb.WriteString("|-------|-----|------------|\n")

	for _, baseline := range PublishedBaselines {
		diff := results.Overall.F1 - baseline.F1
		diffStr := fmt.Sprintf("%+.1f", diff)
		if diff > 0 {
			diffStr = "✅ " + diffStr
		} else if diff < 0 {
			diffStr = "❌ " + diffStr
		}
		sb.WriteString(fmt.Sprintf("| %s | %.1f | %s |\n", baseline.Model, baseline.F1, diffStr))
	}
	sb.WriteString("\n")

	// Configuration
	sb.WriteString("## Configuration\n\n")
	sb.WriteString("```json\n")
	sb.WriteString(fmt.Sprintf("{\n"))
	sb.WriteString(fmt.Sprintf("  \"retrieval_strategy\": \"%s\",\n", results.Config.RetrievalStrategy))
	sb.WriteString(fmt.Sprintf("  \"top_k\": %d,\n", results.Config.TopK))
	if results.Config.Category != "" {
		sb.WriteString(fmt.Sprintf("  \"category_filter\": \"%s\",\n", results.Config.Category))
	}
	sb.WriteString(fmt.Sprintf("  \"verbose\": %t\n", results.Config.Verbose))
	sb.WriteString("}\n")
	sb.WriteString("```\n\n")

	return sb.String()
}

// GenerateSummary generates a brief one-line summary
func (g *ReportGenerator) GenerateSummary(results *BenchmarkResults) string {
	return fmt.Sprintf("LoCoMo %s: F1=%.2f (P=%.2f, R=%.2f) on %d questions in %s",
		results.Strategy,
		results.Overall.F1,
		results.Overall.Precision,
		results.Overall.Recall,
		results.Overall.Count,
		results.Duration.Round(time.Second))
}

// GenerateComparisonReport generates a comparison report between two runs
func (g *ReportGenerator) GenerateComparisonReport(comparison *ComparisonResult) string {
	var sb strings.Builder

	sb.WriteString("# Benchmark Comparison\n\n")

	sb.WriteString("## Runs Compared\n\n")
	sb.WriteString(fmt.Sprintf("**Run A:** %s (%s) - F1: %.2f\n\n",
		comparison.ResultA.Strategy,
		comparison.ResultA.Timestamp.Format("2006-01-02 15:04"),
		comparison.ResultA.Overall.F1))
	sb.WriteString(fmt.Sprintf("**Run B:** %s (%s) - F1: %.2f\n\n",
		comparison.ResultB.Strategy,
		comparison.ResultB.Timestamp.Format("2006-01-02 15:04"),
		comparison.ResultB.Overall.F1))

	// Overall Change
	sb.WriteString("## Overall Change\n\n")
	changeIcon := "📊"
	if comparison.Overall.Improved {
		changeIcon = "📈"
	} else if comparison.Overall.F1Diff < 0 {
		changeIcon = "📉"
	}

	sb.WriteString(fmt.Sprintf("%s **F1:** %+.2f (%.2f → %.2f)\n\n",
		changeIcon,
		comparison.Overall.F1Diff,
		comparison.ResultA.Overall.F1,
		comparison.ResultB.Overall.F1))

	// Category Changes
	sb.WriteString("## Category Changes\n\n")
	sb.WriteString("| Category | Change | A → B |\n")
	sb.WriteString("|----------|--------|-------|\n")

	for cat, diff := range comparison.Categories {
		icon := "➡️"
		if diff.F1Diff > 1 {
			icon = "⬆️"
		} else if diff.F1Diff < -1 {
			icon = "⬇️"
		}

		metricsA := comparison.ResultA.Categories[cat]
		metricsB := comparison.ResultB.Categories[cat]

		sb.WriteString(fmt.Sprintf("| %s | %s %+.2f | %.2f → %.2f |\n",
			cat, icon, diff.F1Diff, metricsA.F1, metricsB.F1))
	}
	sb.WriteString("\n")

	return sb.String()
}

// GenerateProgressReport generates a progress report from multiple runs
func (g *ReportGenerator) GenerateProgressReport(results []*BenchmarkResults) string {
	if len(results) == 0 {
		return "No results to report.\n"
	}

	var sb strings.Builder

	sb.WriteString("# LoCoMo Benchmark Progress\n\n")
	sb.WriteString(fmt.Sprintf("**Total Runs:** %d\n\n", len(results)))

	// History table
	sb.WriteString("## Run History\n\n")
	sb.WriteString("| Date | Strategy | Model | F1 | Duration |\n")
	sb.WriteString("|------|----------|-------|-----|----------|\n")

	for _, r := range results {
		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %.2f | %s |\n",
			r.Timestamp.Format("2006-01-02"),
			r.Strategy,
			truncateModel(r.Model),
			r.Overall.F1,
			r.Duration.Round(time.Second)))
	}
	sb.WriteString("\n")

	// Best results
	var bestF1 float64
	var bestRun *BenchmarkResults
	for _, r := range results {
		if r.Overall.F1 > bestF1 {
			bestF1 = r.Overall.F1
			bestRun = r
		}
	}

	if bestRun != nil {
		sb.WriteString("## Best Result\n\n")
		sb.WriteString(fmt.Sprintf("**F1:** %.2f\n", bestRun.Overall.F1))
		sb.WriteString(fmt.Sprintf("**Strategy:** %s\n", bestRun.Strategy))
		sb.WriteString(fmt.Sprintf("**Model:** %s\n", bestRun.Model))
		sb.WriteString(fmt.Sprintf("**Date:** %s\n\n", bestRun.Timestamp.Format("2006-01-02")))
	}

	// Aggregate stats
	agg := Aggregate(results)
	if agg != nil {
		sb.WriteString("## Aggregate Statistics\n\n")
		sb.WriteString(fmt.Sprintf("- **Mean F1:** %.2f\n", agg.Overall.MeanF1))
		sb.WriteString(fmt.Sprintf("- **Std Dev:** %.2f\n", agg.Overall.StdDevF1))
		sb.WriteString(fmt.Sprintf("- **Range:** %.2f - %.2f\n\n", agg.Overall.MinF1, agg.Overall.MaxF1))
	}

	return sb.String()
}

// GenerateErrorAnalysis generates an analysis of errors from results
func (g *ReportGenerator) GenerateErrorAnalysis(results *BenchmarkResults) string {
	var sb strings.Builder

	sb.WriteString("# Error Analysis\n\n")

	// Find worst performing questions
	type scoredQuestion struct {
		result QuestionResult
		index  int
	}

	var questions []scoredQuestion
	for i, q := range results.Questions {
		questions = append(questions, scoredQuestion{q, i})
	}

	// Sort by F1 (ascending - worst first)
	for i := 0; i < len(questions)-1; i++ {
		for j := i + 1; j < len(questions); j++ {
			if questions[j].result.F1 < questions[i].result.F1 {
				questions[i], questions[j] = questions[j], questions[i]
			}
		}
	}

	sb.WriteString("## Worst Performing Questions (Bottom 10)\n\n")

	limit := 10
	if len(questions) < limit {
		limit = len(questions)
	}

	for i := 0; i < limit; i++ {
		q := questions[i].result
		sb.WriteString(fmt.Sprintf("### %d. %s (F1: %.2f)\n\n", i+1, q.Category, q.F1))
		sb.WriteString(fmt.Sprintf("**Question:** %s\n\n", q.Question))
		sb.WriteString(fmt.Sprintf("**Expected:** %s\n\n", q.GroundTruth))
		sb.WriteString(fmt.Sprintf("**Generated:** %s\n\n", q.GeneratedAnswer))
		sb.WriteString(fmt.Sprintf("**Evidence Found:** %t | **Memories Retrieved:** %d\n\n", q.EvidenceFound, q.RetrievedMemories))
		sb.WriteString("---\n\n")
	}

	// Category analysis
	sb.WriteString("## Error Distribution by Category\n\n")

	zeroCounts := make(map[QuestionCategory]int)
	lowCounts := make(map[QuestionCategory]int)
	totalCounts := make(map[QuestionCategory]int)

	for _, q := range results.Questions {
		totalCounts[q.Category]++
		if q.F1 == 0 {
			zeroCounts[q.Category]++
		} else if q.F1 < 0.3 {
			lowCounts[q.Category]++
		}
	}

	sb.WriteString("| Category | Total | Zero F1 | Low F1 (<0.3) | Error Rate |\n")
	sb.WriteString("|----------|-------|---------|---------------|------------|\n")

	for _, cat := range []QuestionCategory{CategorySingleHop, CategoryMultiHop, CategoryTemporal, CategoryCommonsense, CategoryAdversarial} {
		if total := totalCounts[cat]; total > 0 {
			errorRate := float64(zeroCounts[cat]+lowCounts[cat]) / float64(total) * 100
			sb.WriteString(fmt.Sprintf("| %s | %d | %d | %d | %.1f%% |\n",
				cat, total, zeroCounts[cat], lowCounts[cat], errorRate))
		}
	}
	sb.WriteString("\n")

	return sb.String()
}

func truncateModel(model string) string {
	if len(model) > 20 {
		return model[:17] + "..."
	}
	return model
}

// PrintResults prints results to stdout in a formatted way
func PrintResults(results *BenchmarkResults) {
	gen := NewReportGenerator()
	fmt.Println(gen.GenerateMarkdown(results))
}

// PrintSummary prints a brief summary to stdout
func PrintSummary(results *BenchmarkResults) {
	gen := NewReportGenerator()
	fmt.Println(gen.GenerateSummary(results))
}
