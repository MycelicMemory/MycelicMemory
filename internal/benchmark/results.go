package benchmark

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// CompareRuns compares two benchmark runs
func CompareRuns(runA, runB *RunResults) *Comparison {
	comp := &Comparison{
		RunA:          runA.RunID,
		RunB:          runB.RunID,
		CategoryDiffs: make(map[string]ScoreDiff),
	}

	// Overall comparison
	comp.OverallDiff = calculateDiff(runA.Overall.LLMJudgeAccuracy, runB.Overall.LLMJudgeAccuracy)

	// Category comparisons
	allCategories := make(map[string]bool)
	for cat := range runA.ByCategory {
		allCategories[cat] = true
	}
	for cat := range runB.ByCategory {
		allCategories[cat] = true
	}

	for cat := range allCategories {
		beforeScore := 0.0
		afterScore := 0.0

		if scores, ok := runA.ByCategory[cat]; ok {
			beforeScore = scores.LLMJudgeAccuracy
		}
		if scores, ok := runB.ByCategory[cat]; ok {
			afterScore = scores.LLMJudgeAccuracy
		}

		diff := calculateDiff(beforeScore, afterScore)
		comp.CategoryDiffs[cat] = diff

		if diff.Improved && diff.Diff > 0.1 {
			comp.Improvements = append(comp.Improvements, fmt.Sprintf("%s: %.1f%% → %.1f%% (+%.1f%%)", cat, beforeScore, afterScore, diff.Diff))
		} else if !diff.Improved && diff.Diff < -0.1 {
			comp.Regressions = append(comp.Regressions, fmt.Sprintf("%s: %.1f%% → %.1f%% (%.1f%%)", cat, beforeScore, afterScore, diff.Diff))
		}
	}

	// Question-level comparison
	if len(runA.Questions) > 0 && len(runB.Questions) > 0 {
		questionMapA := make(map[string]QuestionResult)
		for _, q := range runA.Questions {
			questionMapA[q.QuestionID] = q
		}

		for _, qB := range runB.Questions {
			qA, exists := questionMapA[qB.QuestionID]
			if !exists {
				continue
			}

			wasCorrect := qA.LLMJudgeLabel == 1
			nowCorrect := qB.LLMJudgeLabel == 1

			if wasCorrect != nowCorrect {
				comp.ChangedQuestions = append(comp.ChangedQuestions, QuestionComparison{
					QuestionID: qB.QuestionID,
					Category:   qB.Category,
					WasCorrect: wasCorrect,
					NowCorrect: nowCorrect,
					Improved:   nowCorrect && !wasCorrect,
					Regressed:  wasCorrect && !nowCorrect,
				})
			}
		}
	}

	return comp
}

func calculateDiff(before, after float64) ScoreDiff {
	diff := after - before
	percentChange := 0.0
	if before > 0 {
		percentChange = (diff / before) * 100
	}
	return ScoreDiff{
		Before:        before,
		After:         after,
		Diff:          diff,
		PercentChange: percentChange,
		Improved:      diff > 0,
	}
}

// FormatResults formats results for display
func FormatResults(results *RunResults) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Benchmark Run: %s\n", results.RunID))
	sb.WriteString(fmt.Sprintf("Status: %s\n", results.Status))
	sb.WriteString(fmt.Sprintf("Git: %s (%s)%s\n", results.Git.ShortHash, results.Git.Branch, dirtyStr(results.Git.Dirty)))
	sb.WriteString(fmt.Sprintf("Duration: %.1fs\n", results.DurationSecs))
	sb.WriteString("\n")

	sb.WriteString("Overall Scores:\n")
	sb.WriteString(fmt.Sprintf("  LLM Judge Accuracy: %.1f%%\n", results.Overall.LLMJudgeAccuracy))
	sb.WriteString(fmt.Sprintf("  F1 Score: %.4f\n", results.Overall.F1Score))
	sb.WriteString(fmt.Sprintf("  BLEU-1 Score: %.4f\n", results.Overall.BLEU1Score))
	sb.WriteString(fmt.Sprintf("  Total Questions: %d\n", results.Overall.TotalQuestions))
	sb.WriteString("\n")

	if len(results.ByCategory) > 0 {
		sb.WriteString("By Category:\n")

		// Sort categories for consistent output
		categories := make([]string, 0, len(results.ByCategory))
		for cat := range results.ByCategory {
			categories = append(categories, cat)
		}
		sort.Strings(categories)

		for _, cat := range categories {
			scores := results.ByCategory[cat]
			sb.WriteString(fmt.Sprintf("  %s: %.1f%% (%d questions)\n", cat, scores.LLMJudgeAccuracy, scores.TotalQuestions))
		}
	}

	if results.ErrorMessage != "" {
		sb.WriteString("\nError: " + results.ErrorMessage + "\n")
	}

	return sb.String()
}

func dirtyStr(dirty bool) string {
	if dirty {
		return " [dirty]"
	}
	return ""
}

// FormatComparison formats a comparison for display
func FormatComparison(comp *Comparison) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Comparison: %s vs %s\n", comp.RunA[:8], comp.RunB[:8]))
	sb.WriteString("\n")

	sb.WriteString("Overall:\n")
	sb.WriteString(fmt.Sprintf("  Accuracy: %.1f%% → %.1f%% (%+.1f%%)\n",
		comp.OverallDiff.Before, comp.OverallDiff.After, comp.OverallDiff.Diff))
	sb.WriteString("\n")

	if len(comp.Improvements) > 0 {
		sb.WriteString("Improvements:\n")
		for _, imp := range comp.Improvements {
			sb.WriteString(fmt.Sprintf("  ✓ %s\n", imp))
		}
		sb.WriteString("\n")
	}

	if len(comp.Regressions) > 0 {
		sb.WriteString("Regressions:\n")
		for _, reg := range comp.Regressions {
			sb.WriteString(fmt.Sprintf("  ✗ %s\n", reg))
		}
		sb.WriteString("\n")
	}

	if len(comp.ChangedQuestions) > 0 {
		improved := 0
		regressed := 0
		for _, q := range comp.ChangedQuestions {
			if q.Improved {
				improved++
			}
			if q.Regressed {
				regressed++
			}
		}
		sb.WriteString(fmt.Sprintf("Changed Questions: %d improved, %d regressed\n", improved, regressed))
	}

	return sb.String()
}

// ResultsToJSON converts results to JSON
func ResultsToJSON(results *RunResults) (string, error) {
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ComparisonToJSON converts comparison to JSON
func ComparisonToJSON(comp *Comparison) (string, error) {
	data, err := json.MarshalIndent(comp, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// AnalyzeWeakCategories returns categories with lowest accuracy, sorted worst-first
func AnalyzeWeakCategories(results *RunResults) []CategoryScores {
	categories := make([]CategoryScores, 0, len(results.ByCategory))
	for _, cat := range results.ByCategory {
		categories = append(categories, cat)
	}

	sort.Slice(categories, func(i, j int) bool {
		return categories[i].LLMJudgeAccuracy < categories[j].LLMJudgeAccuracy
	})

	return categories
}

// IdentifyFailedQuestions returns questions that were answered incorrectly
func IdentifyFailedQuestions(results *RunResults) []QuestionResult {
	failed := make([]QuestionResult, 0)
	for _, q := range results.Questions {
		if q.LLMJudgeLabel == 0 {
			failed = append(failed, q)
		}
	}
	return failed
}

// GenerateImprovementSuggestions generates suggestions based on weak areas
func GenerateImprovementSuggestions(results *RunResults) []string {
	suggestions := make([]string, 0)
	weakCategories := AnalyzeWeakCategories(results)

	for _, cat := range weakCategories {
		if cat.LLMJudgeAccuracy < 50 {
			// Match category names from LoCoMo benchmark (case-insensitive check)
			switch cat.Category {
			case "multi_hop", "Multi-Hop":
				suggestions = append(suggestions,
					"Multi-hop questions performing poorly - consider improving memory relationship discovery",
					"Try increasing top-k retrieval for multi-hop questions",
					"Consider adding chain-of-thought prompting for complex reasoning",
					"Multi-hop requires aggregating info across multiple evidence pieces")
			case "temporal", "Temporal":
				suggestions = append(suggestions,
					"Temporal reasoning needs improvement - ensure timestamps are properly indexed",
					"Consider adding temporal context to retrieval queries",
					"Check that date/time information is preserved in memory ingestion")
			case "single_hop", "Single-Hop":
				suggestions = append(suggestions,
					"Single-hop retrieval is weak - review embedding quality",
					"Check if relevant memories are being retrieved in top results",
					"Consider tuning similarity thresholds for retrieval")
			case "open_domain", "Open-Domain":
				suggestions = append(suggestions,
					"Open-domain questions struggling - may need broader context retrieval",
					"Consider increasing memory diversity in retrieval")
			case "adversarial", "Adversarial":
				suggestions = append(suggestions,
					"Adversarial questions failing - model should respond 'no information available' when context lacks answer",
					"Review prompt to better handle unanswerable questions")
			}
		}
	}

	if len(suggestions) == 0 && results.Overall.LLMJudgeAccuracy < 70 {
		suggestions = append(suggestions,
			"Consider increasing the number of retrieved memories (top-k)",
			"Review prompt templates for answer generation",
			"Analyze specific failed questions to identify patterns")
	}

	return suggestions
}
