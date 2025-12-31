// Package benchmark provides benchmark execution and tracking services.
package benchmark

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/MycelicMemory/ultrathink/internal/database"
)

// Service manages benchmark operations
type Service struct {
	db          *database.Database
	runner      *Runner
	loopManager *LoopManager
	catalog     *Catalog
	repoPath    string

	// Active run tracking
	mu          sync.RWMutex
	activeRunID string
}

// NewService creates a new benchmark service
func NewService(db *database.Database, repoPath string) *Service {
	svc := &Service{
		db:       db,
		runner:   NewRunner(repoPath),
		catalog:  NewCatalog(DefaultCatalogDir()),
		repoPath: repoPath,
	}
	svc.loopManager = NewLoopManager(svc, db)
	return svc
}

// SetBridgeURL configures the Python bridge URL
func (s *Service) SetBridgeURL(url string) {
	s.runner.SetBridgeURL(url)
}

// Run executes a benchmark and stores results
func (s *Service) Run(ctx context.Context, config *RunConfig) (*RunResults, error) {
	s.mu.Lock()
	if s.activeRunID != "" {
		s.mu.Unlock()
		return nil, ErrRunAlreadyRunning
	}

	// Validate config
	if config.BenchmarkType != "locomo" {
		s.mu.Unlock()
		return nil, ErrInvalidBenchmarkType
	}

	runID := uuid.New().String()
	s.activeRunID = runID
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.activeRunID = ""
		s.mu.Unlock()
	}()

	// Capture git state
	gitState, err := CaptureGitState(s.repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to capture git state: %w", err)
	}

	// Create database record
	dbRun := &database.BenchmarkRun{
		ID:             runID,
		StartedAt:      time.Now(),
		Status:         string(StatusRunning),
		GitCommitHash:  gitState.CommitHash,
		GitBranch:      gitState.Branch,
		GitDirty:       gitState.Dirty,
		ConfigSnapshot: config.ToJSON(),
		BenchmarkType:  config.BenchmarkType,
		CreatedBy:      "mcp",
	}

	if config.ChangeDesc != "" {
		dbRun.ChangeDescription = config.ChangeDesc
	}

	if err := s.db.CreateBenchmarkRun(dbRun); err != nil {
		return nil, fmt.Errorf("failed to create run record: %w", err)
	}

	// Execute benchmark
	results, err := s.runner.Run(ctx, config)
	if err != nil {
		// Update database with failure
		dbRun.Status = string(StatusFailed)
		dbRun.ErrorMessage = err.Error()
		now := time.Now()
		dbRun.CompletedAt = &now
		s.db.UpdateBenchmarkRun(dbRun)
		return results, err
	}

	// Update database with results
	now := time.Now()
	dbRun.CompletedAt = &now
	dbRun.Status = string(StatusCompleted)
	score := results.Overall.LLMJudgeAccuracy
	dbRun.OverallScore = &score
	f1 := results.Overall.F1Score
	dbRun.OverallF1 = &f1
	bleu := results.Overall.BLEU1Score
	dbRun.OverallBleu1 = &bleu
	total := results.Overall.TotalQuestions
	dbRun.TotalQuestions = &total
	correct := results.Overall.TotalCorrect
	dbRun.TotalCorrect = &correct
	duration := results.DurationSecs
	dbRun.DurationSeconds = &duration

	if err := s.db.UpdateBenchmarkRun(dbRun); err != nil {
		return nil, fmt.Errorf("failed to update run record: %w", err)
	}

	// Store category results
	for _, catScores := range results.ByCategory {
		catResult := &database.BenchmarkCategoryResult{
			ID:               uuid.New().String(),
			RunID:            runID,
			Category:         catScores.Category,
			LLMJudgeAccuracy: &catScores.LLMJudgeAccuracy,
			F1Score:          &catScores.F1Score,
			Bleu1Score:       &catScores.BLEU1Score,
			TotalQuestions:   &catScores.TotalQuestions,
			CorrectCount:     &catScores.CorrectCount,
		}
		s.db.CreateBenchmarkCategoryResult(catResult)
	}

	// Store question results
	for _, q := range results.Questions {
		qResult := &database.BenchmarkQuestionResult{
			ID:              uuid.New().String(),
			RunID:           runID,
			QuestionID:      q.QuestionID,
			Category:        q.Category,
			QuestionText:    q.QuestionText,
			GoldAnswer:      q.GoldAnswer,
			GeneratedAnswer: q.GeneratedAnswer,
			LLMJudgeLabel:   &q.LLMJudgeLabel,
			F1Score:         &q.F1Score,
			Bleu1Score:      &q.BLEU1Score,
		}
		if q.ContextLength > 0 {
			qResult.ContextLength = &q.ContextLength
		}
		if q.MemoriesUsed > 0 {
			qResult.MemoriesUsed = &q.MemoriesUsed
		}
		s.db.CreateBenchmarkQuestionResult(qResult)
	}

	// Check if this is the best run
	isBest := false
	best, err := s.db.GetBestBenchmarkRun("locomo")
	if err == nil && best != nil {
		if best.OverallScore == nil || (dbRun.OverallScore != nil && *dbRun.OverallScore > *best.OverallScore) {
			// This is the new best
			dbRun.IsBestRun = true
			isBest = true
			s.db.UpdateBenchmarkRun(dbRun)

			// Unmark previous best
			if best.ID != dbRun.ID {
				best.IsBestRun = false
				s.db.UpdateBenchmarkRun(best)
			}
		}
	} else if best == nil {
		// First run is the best
		dbRun.IsBestRun = true
		isBest = true
		s.db.UpdateBenchmarkRun(dbRun)
	}

	// Save to file catalog
	if s.catalog != nil {
		s.catalog.SaveRun(results)
		if isBest {
			s.catalog.UpdateBest(results)
		}
	}

	return results, nil
}

// GetRun retrieves a benchmark run by ID
func (s *Service) GetRun(runID string) (*database.BenchmarkRun, error) {
	return s.db.GetBenchmarkRun(runID)
}

// ListRuns lists benchmark runs with filtering
func (s *Service) ListRuns(filters *database.BenchmarkRunFilters) ([]*database.BenchmarkRun, error) {
	return s.db.ListBenchmarkRuns(filters)
}

// GetResults reconstructs full results from database
func (s *Service) GetResults(runID string) (*RunResults, error) {
	run, err := s.db.GetBenchmarkRun(runID)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, ErrRunNotFound
	}

	results := &RunResults{
		RunID:     run.ID,
		Status:    RunStatus(run.Status),
		StartedAt: run.StartedAt,
		Git: GitState{
			CommitHash: run.GitCommitHash,
			Branch:     run.GitBranch,
			Dirty:      run.GitDirty,
		},
	}

	if run.CompletedAt != nil {
		results.CompletedAt = run.CompletedAt
	}
	if run.DurationSeconds != nil {
		results.DurationSecs = *run.DurationSeconds
	}
	if run.OverallScore != nil {
		results.Overall.LLMJudgeAccuracy = *run.OverallScore
	}
	if run.OverallF1 != nil {
		results.Overall.F1Score = *run.OverallF1
	}
	if run.OverallBleu1 != nil {
		results.Overall.BLEU1Score = *run.OverallBleu1
	}
	if run.TotalQuestions != nil {
		results.Overall.TotalQuestions = *run.TotalQuestions
	}
	if run.TotalCorrect != nil {
		results.Overall.TotalCorrect = *run.TotalCorrect
	}
	results.ErrorMessage = run.ErrorMessage

	// Load category results
	catResults, err := s.db.GetBenchmarkCategoryResults(runID)
	if err == nil {
		results.ByCategory = make(map[string]CategoryScores)
		for _, cat := range catResults {
			cs := CategoryScores{Category: cat.Category}
			if cat.LLMJudgeAccuracy != nil {
				cs.LLMJudgeAccuracy = *cat.LLMJudgeAccuracy
			}
			if cat.F1Score != nil {
				cs.F1Score = *cat.F1Score
			}
			if cat.Bleu1Score != nil {
				cs.BLEU1Score = *cat.Bleu1Score
			}
			if cat.TotalQuestions != nil {
				cs.TotalQuestions = *cat.TotalQuestions
			}
			if cat.CorrectCount != nil {
				cs.CorrectCount = *cat.CorrectCount
			}
			results.ByCategory[cat.Category] = cs
		}
	}

	// Load question results
	qResults, err := s.db.GetBenchmarkQuestionResults(runID)
	if err == nil {
		results.Questions = make([]QuestionResult, len(qResults))
		for i, q := range qResults {
			qr := QuestionResult{
				QuestionID:      q.QuestionID,
				Category:        q.Category,
				QuestionText:    q.QuestionText,
				GoldAnswer:      q.GoldAnswer,
				GeneratedAnswer: q.GeneratedAnswer,
			}
			if q.LLMJudgeLabel != nil {
				qr.LLMJudgeLabel = *q.LLMJudgeLabel
			}
			if q.F1Score != nil {
				qr.F1Score = *q.F1Score
			}
			if q.Bleu1Score != nil {
				qr.BLEU1Score = *q.Bleu1Score
			}
			if q.ContextLength != nil {
				qr.ContextLength = *q.ContextLength
			}
			if q.MemoriesUsed != nil {
				qr.MemoriesUsed = *q.MemoriesUsed
			}
			results.Questions[i] = qr
		}
	}

	return results, nil
}

// Compare compares two benchmark runs
func (s *Service) Compare(runIDA, runIDB string) (*Comparison, error) {
	resultsA, err := s.GetResults(runIDA)
	if err != nil {
		return nil, fmt.Errorf("failed to get run A: %w", err)
	}

	resultsB, err := s.GetResults(runIDB)
	if err != nil {
		return nil, fmt.Errorf("failed to get run B: %w", err)
	}

	return CompareRuns(resultsA, resultsB), nil
}

// GetBestRun returns the best benchmark run
func (s *Service) GetBestRun(benchmarkType string) (*database.BenchmarkRun, error) {
	return s.db.GetBestBenchmarkRun(benchmarkType)
}

// GetProgress returns progress of active run
func (s *Service) GetProgress(ctx context.Context) (*Progress, error) {
	s.mu.RLock()
	runID := s.activeRunID
	s.mu.RUnlock()

	if runID == "" {
		return nil, ErrRunNotFound
	}

	return s.runner.GetProgress(ctx, runID)
}

// CancelRun cancels an active run
func (s *Service) CancelRun(ctx context.Context) error {
	s.mu.RLock()
	runID := s.activeRunID
	s.mu.RUnlock()

	if runID == "" {
		return ErrRunNotFound
	}

	if err := s.runner.Cancel(ctx, runID); err != nil {
		return err
	}

	// Update database
	run, err := s.db.GetBenchmarkRun(runID)
	if err == nil && run != nil {
		run.Status = string(StatusCancelled)
		now := time.Now()
		run.CompletedAt = &now
		s.db.UpdateBenchmarkRun(run)
	}

	return nil
}

// IsRunning returns true if a benchmark is currently running
func (s *Service) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.activeRunID != ""
}

// GetActiveRunID returns the ID of the currently running benchmark
func (s *Service) GetActiveRunID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.activeRunID
}

// CheckBridge checks if the Python bridge is available
func (s *Service) CheckBridge() error {
	return s.runner.CheckBridge()
}

// StartLoop starts an autonomous improvement loop
func (s *Service) StartLoop(ctx context.Context, config *LoopConfig) (*LoopState, error) {
	return s.loopManager.StartLoop(ctx, config)
}

// StopLoop stops the active improvement loop
func (s *Service) StopLoop() error {
	return s.loopManager.StopLoop()
}

// GetLoopState returns the current loop state
func (s *Service) GetLoopState() *LoopState {
	return s.loopManager.GetState()
}

// IsLoopRunning returns true if a loop is active
func (s *Service) IsLoopRunning() bool {
	return s.loopManager.IsRunning()
}
