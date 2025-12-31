package locomo

import (
	"regexp"
	"strings"
	"unicode"
)

// TokenizeAnswer tokenizes a string for F1 score calculation.
// This follows the standard QA evaluation tokenization:
// 1. Lowercase
// 2. Remove punctuation
// 3. Remove articles (a, an, the)
// 4. Split on whitespace
func TokenizeAnswer(s string) []string {
	// Lowercase
	s = strings.ToLower(s)

	// Remove punctuation
	var builder strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
			builder.WriteRune(r)
		} else {
			builder.WriteRune(' ')
		}
	}
	s = builder.String()

	// Split on whitespace
	words := strings.Fields(s)

	// Remove articles
	articles := map[string]bool{"a": true, "an": true, "the": true}
	var filtered []string
	for _, w := range words {
		if !articles[w] && w != "" {
			filtered = append(filtered, w)
		}
	}

	return filtered
}

// NormalizeAnswer normalizes an answer string for comparison.
// Returns the normalized string and its tokens.
func NormalizeAnswer(s string) (string, []string) {
	tokens := TokenizeAnswer(s)
	return strings.Join(tokens, " "), tokens
}

// CalculateF1 calculates F1, precision, and recall scores between
// a generated answer and ground truth answer.
// This is token-level F1 as used in SQuAD-style evaluation.
func CalculateF1(generated, groundTruth string) (f1, precision, recall float64) {
	genTokens := TokenizeAnswer(generated)
	gtTokens := TokenizeAnswer(groundTruth)

	if len(genTokens) == 0 && len(gtTokens) == 0 {
		return 1.0, 1.0, 1.0
	}
	if len(genTokens) == 0 {
		return 0.0, 0.0, 0.0
	}
	if len(gtTokens) == 0 {
		return 0.0, 0.0, 0.0
	}

	// Count common tokens
	gtSet := make(map[string]int)
	for _, t := range gtTokens {
		gtSet[t]++
	}

	genSet := make(map[string]int)
	for _, t := range genTokens {
		genSet[t]++
	}

	// Calculate overlap (intersection with multiplicity)
	common := 0
	for token, genCount := range genSet {
		if gtCount, ok := gtSet[token]; ok {
			if genCount < gtCount {
				common += genCount
			} else {
				common += gtCount
			}
		}
	}

	// Calculate precision and recall
	precision = float64(common) / float64(len(genTokens))
	recall = float64(common) / float64(len(gtTokens))

	// Calculate F1
	if precision+recall == 0 {
		f1 = 0.0
	} else {
		f1 = 2 * precision * recall / (precision + recall)
	}

	return f1, precision, recall
}

// CalculateExactMatch checks if the generated answer exactly matches
// the ground truth after normalization.
func CalculateExactMatch(generated, groundTruth string) bool {
	genNorm, _ := NormalizeAnswer(generated)
	gtNorm, _ := NormalizeAnswer(groundTruth)
	return genNorm == gtNorm
}

// CalculateBatchMetrics calculates aggregate metrics for a batch of results.
func CalculateBatchMetrics(results []QuestionResult) Metrics {
	if len(results) == 0 {
		return Metrics{}
	}

	var sumF1, sumPrecision, sumRecall float64
	for _, r := range results {
		sumF1 += r.F1
		sumPrecision += r.Precision
		sumRecall += r.Recall
	}

	n := float64(len(results))
	return Metrics{
		F1:        sumF1 / n * 100,        // Convert to percentage
		Precision: sumPrecision / n * 100, // Convert to percentage
		Recall:    sumRecall / n * 100,    // Convert to percentage
		Count:     len(results),
	}
}

// CalculateCategoryMetrics groups results by category and calculates
// per-category metrics.
func CalculateCategoryMetrics(results []QuestionResult) map[QuestionCategory]Metrics {
	byCategory := make(map[QuestionCategory][]QuestionResult)

	for _, r := range results {
		byCategory[r.Category] = append(byCategory[r.Category], r)
	}

	metrics := make(map[QuestionCategory]Metrics)
	for cat, catResults := range byCategory {
		metrics[cat] = CalculateBatchMetrics(catResults)
	}

	return metrics
}

// ExtractAnswer extracts an answer from an AI response.
// The AI may wrap the answer in various formats; this attempts
// to extract just the answer portion.
func ExtractAnswer(response string) string {
	response = strings.TrimSpace(response)

	// Pattern for "The answer is X" or "Answer: X"
	answerPattern := regexp.MustCompile(`(?i)^(?:the\s+)?answer(?:\s+is)?[:\s]+(.+)$`)
	if matches := answerPattern.FindStringSubmatch(response); len(matches) > 1 {
		extracted := strings.TrimSpace(matches[1])
		if extracted != "" {
			return extracted
		}
	}

	// Pattern for "Based on X, Y" - extract Y
	basedOnPattern := regexp.MustCompile(`(?i)^based on[^,]+,\s*(.+)$`)
	if matches := basedOnPattern.FindStringSubmatch(response); len(matches) > 1 {
		extracted := strings.TrimSpace(matches[1])
		if extracted != "" {
			return extracted
		}
	}

	return response
}

// CompareWithBaseline compares benchmark results against published baselines.
func CompareWithBaseline(results *BenchmarkResults) []BaselineComparison {
	comparisons := make([]BaselineComparison, len(PublishedBaselines))

	for i, baseline := range PublishedBaselines {
		comparisons[i] = BaselineComparison{
			Baseline:   baseline,
			OurF1:      results.Overall.F1,
			Difference: results.Overall.F1 - baseline.F1,
		}
	}

	return comparisons
}

// BaselineComparison represents a comparison with a baseline
type BaselineComparison struct {
	Baseline   Baseline
	OurF1      float64
	Difference float64 // Positive means we're better
}
