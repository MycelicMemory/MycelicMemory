package locomo

import (
	"math"
	"testing"
)

func TestTokenizeAnswer(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"Hello World", []string{"hello", "world"}},
		{"The quick brown fox", []string{"quick", "brown", "fox"}},
		{"a cat and an apple", []string{"cat", "and", "apple"}},
		{"Hello, World!", []string{"hello", "world"}},
		{"It's a test", []string{"it", "s", "test"}},
		{"123 Main Street", []string{"123", "main", "street"}},
		{"", []string{}},
		{"   ", []string{}},
		{"the", []string{}},
		{"a an the", []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := TokenizeAnswer(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("TokenizeAnswer(%q) = %v, want %v", tt.input, result, tt.expected)
				return
			}
			for i, token := range result {
				if token != tt.expected[i] {
					t.Errorf("TokenizeAnswer(%q)[%d] = %q, want %q", tt.input, i, token, tt.expected[i])
				}
			}
		})
	}
}

func almostEqual(a, b, epsilon float64) bool {
	return math.Abs(a-b) < epsilon
}

func TestCalculateF1(t *testing.T) {
	epsilon := 0.001

	tests := []struct {
		name      string
		generated string
		truth     string
		f1        float64
		precision float64
		recall    float64
	}{
		{
			name:      "exact match",
			generated: "Paris",
			truth:     "Paris",
			f1:        1.0,
			precision: 1.0,
			recall:    1.0,
		},
		{
			name:      "exact match with articles",
			generated: "The Eiffel Tower",
			truth:     "Eiffel Tower",
			f1:        1.0,
			precision: 1.0,
			recall:    1.0,
		},
		{
			name:      "partial match",
			generated: "Eiffel Tower in Paris",
			truth:     "Eiffel Tower",
			f1:        2.0 / 3.0, // 2*P*R/(P+R) = 2*0.5*1/(0.5+1) = 0.667
			precision: 0.5,       // 2 common out of 4 generated (eiffel, tower, in, paris)
			recall:    1.0,       // 2 common out of 2 truth
		},
		{
			name:      "no match",
			generated: "London",
			truth:     "Paris",
			f1:        0.0,
			precision: 0.0,
			recall:    0.0,
		},
		{
			name:      "empty generated",
			generated: "",
			truth:     "Paris",
			f1:        0.0,
			precision: 0.0,
			recall:    0.0,
		},
		{
			name:      "empty truth",
			generated: "Paris",
			truth:     "",
			f1:        0.0,
			precision: 0.0,
			recall:    0.0,
		},
		{
			name:      "both empty",
			generated: "",
			truth:     "",
			f1:        1.0,
			precision: 1.0,
			recall:    1.0,
		},
		{
			name:      "case insensitive",
			generated: "PARIS",
			truth:     "paris",
			f1:        1.0,
			precision: 1.0,
			recall:    1.0,
		},
		{
			name:      "punctuation ignored",
			generated: "Paris, France!",
			truth:     "Paris France",
			f1:        1.0,
			precision: 1.0,
			recall:    1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f1, precision, recall := CalculateF1(tt.generated, tt.truth)

			if !almostEqual(f1, tt.f1, epsilon) {
				t.Errorf("F1: got %v, want %v", f1, tt.f1)
			}
			if !almostEqual(precision, tt.precision, epsilon) {
				t.Errorf("Precision: got %v, want %v", precision, tt.precision)
			}
			if !almostEqual(recall, tt.recall, epsilon) {
				t.Errorf("Recall: got %v, want %v", recall, tt.recall)
			}
		})
	}
}

func TestCalculateExactMatch(t *testing.T) {
	tests := []struct {
		generated string
		truth     string
		expected  bool
	}{
		{"Paris", "Paris", true},
		{"paris", "PARIS", true},
		{"The Eiffel Tower", "Eiffel Tower", true},
		{"Paris, France", "Paris France", true},
		{"London", "Paris", false},
		{"Paris France", "Paris", false},
	}

	for _, tt := range tests {
		t.Run(tt.generated+"_"+tt.truth, func(t *testing.T) {
			result := CalculateExactMatch(tt.generated, tt.truth)
			if result != tt.expected {
				t.Errorf("CalculateExactMatch(%q, %q) = %v, want %v",
					tt.generated, tt.truth, result, tt.expected)
			}
		})
	}
}

func TestCalculateBatchMetrics(t *testing.T) {
	results := []QuestionResult{
		{F1: 0.5, Precision: 0.6, Recall: 0.4},
		{F1: 0.8, Precision: 0.9, Recall: 0.7},
		{F1: 1.0, Precision: 1.0, Recall: 1.0},
	}

	metrics := CalculateBatchMetrics(results)

	expectedF1 := (0.5 + 0.8 + 1.0) / 3 * 100
	if !almostEqual(metrics.F1, expectedF1, 0.01) {
		t.Errorf("F1: got %v, want %v", metrics.F1, expectedF1)
	}

	if metrics.Count != 3 {
		t.Errorf("Count: got %v, want 3", metrics.Count)
	}
}

func TestCalculateCategoryMetrics(t *testing.T) {
	results := []QuestionResult{
		{Category: CategorySingleHop, F1: 0.5, Precision: 0.5, Recall: 0.5},
		{Category: CategorySingleHop, F1: 0.7, Precision: 0.7, Recall: 0.7},
		{Category: CategoryMultiHop, F1: 0.3, Precision: 0.3, Recall: 0.3},
		{Category: CategoryTemporal, F1: 0.9, Precision: 0.9, Recall: 0.9},
	}

	metrics := CalculateCategoryMetrics(results)

	if len(metrics) != 3 {
		t.Errorf("Expected 3 categories, got %d", len(metrics))
	}

	singleHop := metrics[CategorySingleHop]
	if singleHop.Count != 2 {
		t.Errorf("SingleHop count: got %v, want 2", singleHop.Count)
	}

	multiHop := metrics[CategoryMultiHop]
	if multiHop.Count != 1 {
		t.Errorf("MultiHop count: got %v, want 1", multiHop.Count)
	}
}

func TestExtractAnswer(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Paris", "Paris"},
		{"The answer is Paris", "Paris"},
		{"Answer: Paris", "Paris"},
		{"Based on the context, Paris", "Paris"},
		{"  Paris  ", "Paris"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ExtractAnswer(tt.input)
			if result != tt.expected {
				t.Errorf("ExtractAnswer(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
