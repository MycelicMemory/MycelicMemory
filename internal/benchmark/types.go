// Package benchmark provides benchmark execution and tracking services.
package benchmark

import (
	"encoding/json"
	"time"
)

// RunConfig holds configuration for a benchmark run
type RunConfig struct {
	BenchmarkType  string `json:"benchmark_type"`  // "locomo"
	MaxQuestions   int    `json:"max_questions"`   // 0 = all questions
	QuestionTypes  []string `json:"question_types"` // Filter by category
	TopK           int    `json:"top_k"`           // Number of memories to retrieve
	Verbose        bool   `json:"verbose"`
	UseSummaries   bool   `json:"use_summaries"`
	Async          bool   `json:"async"`           // Run asynchronously
	ChangeDesc     string `json:"change_description"` // Description of code change being tested
}

// DefaultRunConfig returns default configuration
func DefaultRunConfig() *RunConfig {
	return &RunConfig{
		BenchmarkType: "locomo",
		MaxQuestions:  0,
		TopK:          10,
		Verbose:       false,
		Async:         false,
	}
}

// ToJSON serializes config to JSON
func (c *RunConfig) ToJSON() string {
	data, _ := json.Marshal(c)
	return string(data)
}

// GitState captures the state of the git repository
type GitState struct {
	CommitHash string `json:"commit_hash"`
	Branch     string `json:"branch"`
	Dirty      bool   `json:"dirty"`
	ShortHash  string `json:"short_hash"`
}

// RunStatus represents the status of a benchmark run
type RunStatus string

const (
	StatusPending   RunStatus = "pending"
	StatusRunning   RunStatus = "running"
	StatusCompleted RunStatus = "completed"
	StatusFailed    RunStatus = "failed"
	StatusCancelled RunStatus = "cancelled"
)

// CategoryScores holds scores for a single category
type CategoryScores struct {
	Category        string  `json:"category"`
	LLMJudgeAccuracy float64 `json:"llm_judge_accuracy"`
	F1Score         float64 `json:"f1_score"`
	BLEU1Score      float64 `json:"bleu1_score"`
	TotalQuestions  int     `json:"total_questions"`
	CorrectCount    int     `json:"correct_count"`
}

// OverallScores holds aggregate scores
type OverallScores struct {
	LLMJudgeAccuracy float64 `json:"llm_judge_accuracy"`
	F1Score          float64 `json:"f1_score"`
	BLEU1Score       float64 `json:"bleu1_score"`
	TotalQuestions   int     `json:"total_questions"`
	TotalCorrect     int     `json:"total_correct"`
}

// QuestionResult holds result for a single question
type QuestionResult struct {
	QuestionID      string  `json:"question_id"`
	Category        string  `json:"category"`
	QuestionText    string  `json:"question_text"`
	GoldAnswer      string  `json:"gold_answer"`
	GeneratedAnswer string  `json:"generated_answer"`
	LLMJudgeLabel   int     `json:"llm_judge_label"` // 0 or 1
	F1Score         float64 `json:"f1_score"`
	BLEU1Score      float64 `json:"bleu1_score"`
	ContextLength   int     `json:"context_length"`
	MemoriesUsed    int     `json:"memories_used"`
	RetrievalTimeMs int     `json:"retrieval_time_ms"`
	GenerationTimeMs int    `json:"generation_time_ms"`
}

// RunResults holds complete results from a benchmark run
type RunResults struct {
	RunID          string                     `json:"run_id"`
	Status         RunStatus                  `json:"status"`
	StartedAt      time.Time                  `json:"started_at"`
	CompletedAt    *time.Time                 `json:"completed_at,omitempty"`
	DurationSecs   float64                    `json:"duration_seconds"`
	Git            GitState                   `json:"git"`
	Config         RunConfig                  `json:"config"`
	Overall        OverallScores              `json:"overall"`
	ByCategory     map[string]CategoryScores  `json:"by_category"`
	Questions      []QuestionResult           `json:"questions,omitempty"`
	ErrorMessage   string                     `json:"error_message,omitempty"`
}

// Progress represents benchmark execution progress
type Progress struct {
	RunID           string    `json:"run_id"`
	Status          RunStatus `json:"status"`
	TotalQuestions  int       `json:"total_questions"`
	CompletedCount  int       `json:"completed_count"`
	CurrentQuestion string    `json:"current_question,omitempty"`
	PercentComplete float64   `json:"percent_complete"`
	ElapsedSecs     float64   `json:"elapsed_seconds"`
	EstimatedRemaining float64 `json:"estimated_remaining_seconds,omitempty"`
}

// Comparison holds comparison between two runs
type Comparison struct {
	RunA           string                       `json:"run_a"`
	RunB           string                       `json:"run_b"`
	OverallDiff    ScoreDiff                    `json:"overall_diff"`
	CategoryDiffs  map[string]ScoreDiff         `json:"category_diffs"`
	Improvements   []string                     `json:"improvements"`
	Regressions    []string                     `json:"regressions"`
	ChangedQuestions []QuestionComparison       `json:"changed_questions,omitempty"`
}

// ScoreDiff holds the difference between two scores
type ScoreDiff struct {
	Before    float64 `json:"before"`
	After     float64 `json:"after"`
	Diff      float64 `json:"diff"`
	PercentChange float64 `json:"percent_change"`
	Improved  bool    `json:"improved"`
}

// QuestionComparison compares a single question across runs
type QuestionComparison struct {
	QuestionID     string `json:"question_id"`
	Category       string `json:"category"`
	WasCorrect     bool   `json:"was_correct"`
	NowCorrect     bool   `json:"now_correct"`
	Improved       bool   `json:"improved"`
	Regressed      bool   `json:"regressed"`
}

// LoopConfig holds configuration for autonomous improvement loop
type LoopConfig struct {
	MaxIterations           int     `json:"max_iterations"`
	MinImprovementThreshold float64 `json:"min_improvement_threshold"`
	ConvergenceThreshold    float64 `json:"convergence_threshold"`
	TimeoutMinutes          int     `json:"timeout_minutes"`
}

// DefaultLoopConfig returns default loop configuration
func DefaultLoopConfig() *LoopConfig {
	return &LoopConfig{
		MaxIterations:           10,
		MinImprovementThreshold: 0.01,  // 1% improvement required
		ConvergenceThreshold:    0.005, // 0.5% change is considered convergence
		TimeoutMinutes:          120,   // 2 hours
	}
}

// LoopStatus represents the status of an autonomous loop
type LoopStatus string

const (
	LoopRunning   LoopStatus = "running"
	LoopCompleted LoopStatus = "completed"
	LoopStopped   LoopStatus = "stopped"
	LoopFailed    LoopStatus = "failed"
)

// LoopState holds current state of an improvement loop
type LoopState struct {
	ID              string     `json:"id"`
	Status          LoopStatus `json:"status"`
	CurrentIteration int       `json:"current_iteration"`
	MaxIterations   int        `json:"max_iterations"`
	BaselineScore   float64    `json:"baseline_score"`
	CurrentScore    float64    `json:"current_score"`
	BestScore       float64    `json:"best_score"`
	BestRunID       string     `json:"best_run_id"`
	LastChange      string     `json:"last_change"`
	StopReason      string     `json:"stop_reason,omitempty"`
	StartedAt       time.Time  `json:"started_at"`
	ElapsedMinutes  float64    `json:"elapsed_minutes"`
}
