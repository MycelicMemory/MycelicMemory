package benchmark

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
)

const (
	// DefaultBridgeURL is the default Python bridge server URL
	DefaultBridgeURL = "http://localhost:9876"

	// BridgeTimeout is the maximum time to wait for bridge responses
	BridgeTimeout = 30 * time.Minute
)

// Runner handles benchmark execution
type Runner struct {
	bridgeURL  string
	httpClient *http.Client
	repoPath   string
}

// NewRunner creates a new benchmark runner
func NewRunner(repoPath string) *Runner {
	return &Runner{
		bridgeURL: DefaultBridgeURL,
		httpClient: &http.Client{
			Timeout: BridgeTimeout,
		},
		repoPath: repoPath,
	}
}

// SetBridgeURL sets a custom bridge URL
func (r *Runner) SetBridgeURL(url string) {
	r.bridgeURL = url
}

// CheckBridge checks if the Python bridge is available
func (r *Runner) CheckBridge() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", r.bridgeURL+"/health", nil)
	if err != nil {
		return ErrPythonBridgeNotAvailable
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return ErrPythonBridgeNotAvailable
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ErrPythonBridgeNotAvailable
	}

	return nil
}

// BridgeRunRequest is the request to the Python bridge
type BridgeRunRequest struct {
	RunID         string   `json:"run_id"`
	BenchmarkType string   `json:"benchmark_type"`
	MaxQuestions  int      `json:"max_questions"`
	Categories    []string `json:"categories,omitempty"`
	RandomSample  bool     `json:"random_sample"`
	Seed          *int     `json:"seed,omitempty"`
	Verbose       bool     `json:"verbose"`
}

// BridgeRunResponse is the response from the Python bridge
type BridgeRunResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
	Results struct {
		Overall struct {
			LLMJudgeAccuracy float64 `json:"llm_judge_accuracy"`
			F1Score          float64 `json:"f1_score"`
			BLEU1Score       float64 `json:"bleu1_score"`
			TotalQuestions   int     `json:"total_questions"`
			TotalCorrect     int     `json:"total_correct"`
		} `json:"overall"`
		ByCategory map[string]struct {
			LLMJudgeAccuracy float64 `json:"llm_judge_accuracy"`
			F1Score          float64 `json:"f1_score"`
			BLEU1Score       float64 `json:"bleu1_score"`
			Count            int     `json:"count"`
			Correct          int     `json:"correct"`
		} `json:"by_category"`
		Questions []struct {
			ID              string  `json:"id"`
			Category        string  `json:"category"`
			Question        string  `json:"question"`
			GoldAnswer      string  `json:"gold_answer"`
			GeneratedAnswer string  `json:"generated_answer"`
			LLMJudgeLabel   int     `json:"llm_judge_label"`
			F1Score         float64 `json:"f1_score"`
			BLEU1Score      float64 `json:"bleu1_score"`
		} `json:"questions"`
		// Enhanced metrics
		Latency *struct {
			MeanSeconds   float64 `json:"mean_latency_seconds"`
			MedianSeconds float64 `json:"median_latency_seconds"`
			P95Seconds    float64 `json:"p95_latency_seconds"`
			P99Seconds    float64 `json:"p99_latency_seconds"`
			MinSeconds    float64 `json:"min_latency_seconds"`
			MaxSeconds    float64 `json:"max_latency_seconds"`
			StdDevSeconds float64 `json:"stdev_latency_seconds"`
		} `json:"latency,omitempty"`
		Tokens *struct {
			TotalInput  int     `json:"total_input_tokens"`
			TotalOutput int     `json:"total_output_tokens"`
			Total       int     `json:"total_tokens"`
			MeanInput   float64 `json:"mean_input_tokens"`
			MeanOutput  float64 `json:"mean_output_tokens"`
		} `json:"tokens,omitempty"`
		Cost *struct {
			InputCostUSD       float64 `json:"input_cost_usd"`
			OutputCostUSD      float64 `json:"output_cost_usd"`
			TotalCostUSD       float64 `json:"total_cost_usd"`
			CostPerQuestionUSD float64 `json:"cost_per_question_usd"`
		} `json:"cost_estimation,omitempty"`
		DurationSecs float64 `json:"duration_seconds,omitempty"`
	} `json:"results"`
}

// Run executes a benchmark run
func (r *Runner) Run(ctx context.Context, config *RunConfig) (*RunResults, error) {
	// Capture git state first
	gitState, err := CaptureGitState(r.repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to capture git state: %w", err)
	}

	// Generate run ID
	runID := uuid.New().String()
	startedAt := time.Now()

	results := &RunResults{
		RunID:     runID,
		Status:    StatusRunning,
		StartedAt: startedAt,
		Git:       *gitState,
		Config:    *config,
	}

	// Check bridge availability
	if err := r.CheckBridge(); err != nil {
		results.Status = StatusFailed
		results.ErrorMessage = err.Error()
		now := time.Now()
		results.CompletedAt = &now
		return results, err
	}

	// Prepare bridge request
	bridgeReq := &BridgeRunRequest{
		RunID:         runID,
		BenchmarkType: config.BenchmarkType,
		MaxQuestions:  config.MaxQuestions,
		Categories:    config.QuestionTypes,
		RandomSample:  config.RandomSample,
		Seed:          config.Seed,
		Verbose:       config.Verbose,
	}

	reqBody, err := json.Marshal(bridgeReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Call Python bridge
	req, err := http.NewRequestWithContext(ctx, "POST", r.bridgeURL+"/run", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		results.Status = StatusFailed
		results.ErrorMessage = fmt.Sprintf("bridge request failed: %v", err)
		now := time.Now()
		results.CompletedAt = &now
		return results, ErrBenchmarkFailed
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		results.Status = StatusFailed
		results.ErrorMessage = fmt.Sprintf("failed to read response: %v", err)
		now := time.Now()
		results.CompletedAt = &now
		return results, ErrBenchmarkFailed
	}

	var bridgeResp BridgeRunResponse
	if err := json.Unmarshal(body, &bridgeResp); err != nil {
		results.Status = StatusFailed
		results.ErrorMessage = fmt.Sprintf("failed to parse response: %v", err)
		now := time.Now()
		results.CompletedAt = &now
		return results, ErrBenchmarkFailed
	}

	if !bridgeResp.Success {
		results.Status = StatusFailed
		results.ErrorMessage = bridgeResp.Error
		now := time.Now()
		results.CompletedAt = &now
		return results, ErrBenchmarkFailed
	}

	// Convert bridge response to results
	now := time.Now()
	results.CompletedAt = &now
	results.DurationSecs = now.Sub(startedAt).Seconds()
	results.Status = StatusCompleted

	results.Overall = OverallScores{
		LLMJudgeAccuracy: bridgeResp.Results.Overall.LLMJudgeAccuracy,
		F1Score:          bridgeResp.Results.Overall.F1Score,
		BLEU1Score:       bridgeResp.Results.Overall.BLEU1Score,
		TotalQuestions:   bridgeResp.Results.Overall.TotalQuestions,
		TotalCorrect:     int(bridgeResp.Results.Overall.LLMJudgeAccuracy * float64(bridgeResp.Results.Overall.TotalQuestions) / 100),
	}

	results.ByCategory = make(map[string]CategoryScores)
	for cat, scores := range bridgeResp.Results.ByCategory {
		results.ByCategory[cat] = CategoryScores{
			Category:         cat,
			LLMJudgeAccuracy: scores.LLMJudgeAccuracy,
			F1Score:          scores.F1Score,
			BLEU1Score:       scores.BLEU1Score,
			TotalQuestions:   scores.Count,
			CorrectCount:     int(scores.LLMJudgeAccuracy * float64(scores.Count) / 100),
		}
	}

	results.Questions = make([]QuestionResult, len(bridgeResp.Results.Questions))
	for i, q := range bridgeResp.Results.Questions {
		results.Questions[i] = QuestionResult{
			QuestionID:      q.ID,
			Category:        q.Category,
			QuestionText:    q.Question,
			GoldAnswer:      q.GoldAnswer,
			GeneratedAnswer: q.GeneratedAnswer,
			LLMJudgeLabel:   q.LLMJudgeLabel,
			F1Score:         q.F1Score,
			BLEU1Score:      q.BLEU1Score,
		}
	}

	// Parse enhanced metrics if available
	if bridgeResp.Results.Latency != nil {
		results.Latency = &LatencyStats{
			MeanSeconds:   bridgeResp.Results.Latency.MeanSeconds,
			MedianSeconds: bridgeResp.Results.Latency.MedianSeconds,
			P95Seconds:    bridgeResp.Results.Latency.P95Seconds,
			P99Seconds:    bridgeResp.Results.Latency.P99Seconds,
			MinSeconds:    bridgeResp.Results.Latency.MinSeconds,
			MaxSeconds:    bridgeResp.Results.Latency.MaxSeconds,
			StdDevSeconds: bridgeResp.Results.Latency.StdDevSeconds,
		}
	}

	if bridgeResp.Results.Tokens != nil {
		results.Tokens = &TokenStats{
			TotalInput:  bridgeResp.Results.Tokens.TotalInput,
			TotalOutput: bridgeResp.Results.Tokens.TotalOutput,
			Total:       bridgeResp.Results.Tokens.Total,
			MeanInput:   bridgeResp.Results.Tokens.MeanInput,
			MeanOutput:  bridgeResp.Results.Tokens.MeanOutput,
		}
	}

	if bridgeResp.Results.Cost != nil {
		results.Cost = &CostEstimation{
			InputCostUSD:       bridgeResp.Results.Cost.InputCostUSD,
			OutputCostUSD:      bridgeResp.Results.Cost.OutputCostUSD,
			TotalCostUSD:       bridgeResp.Results.Cost.TotalCostUSD,
			CostPerQuestionUSD: bridgeResp.Results.Cost.CostPerQuestionUSD,
		}
	}

	return results, nil
}

// GetProgress gets the progress of a running benchmark
func (r *Runner) GetProgress(ctx context.Context, runID string) (*Progress, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", r.bridgeURL+"/status?run_id="+runID, nil)
	if err != nil {
		return nil, err
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var progress Progress
	if err := json.NewDecoder(resp.Body).Decode(&progress); err != nil {
		return nil, err
	}

	return &progress, nil
}

// Cancel cancels a running benchmark
func (r *Runner) Cancel(ctx context.Context, runID string) error {
	reqBody := map[string]string{"run_id": runID}
	body, _ := json.Marshal(reqBody)

	req, err := http.NewRequestWithContext(ctx, "POST", r.bridgeURL+"/cancel", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("cancel failed with status %d", resp.StatusCode)
	}

	return nil
}
