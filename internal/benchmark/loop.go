package benchmark

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/MycelicMemory/ultrathink/internal/database"
)

// LoopManager manages autonomous improvement loops
type LoopManager struct {
	service *Service
	db      *database.Database

	mu       sync.RWMutex
	active   *database.AutonomousLoop
	stopChan chan struct{}
}

// NewLoopManager creates a new loop manager
func NewLoopManager(service *Service, db *database.Database) *LoopManager {
	return &LoopManager{
		service: service,
		db:      db,
	}
}

// StartLoop begins an autonomous improvement loop
func (m *LoopManager) StartLoop(ctx context.Context, config *LoopConfig) (*LoopState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.active != nil && m.active.Status == "running" {
		return nil, ErrLoopAlreadyRunning
	}

	// Create loop record
	loop := &database.AutonomousLoop{
		ID:                      uuid.New().String(),
		StartedAt:               time.Now(),
		Status:                  "running",
		MaxIterations:           config.MaxIterations,
		MinImprovementThreshold: config.MinImprovementThreshold,
		ConvergenceThreshold:    config.ConvergenceThreshold,
	}

	if err := m.db.CreateAutonomousLoop(loop); err != nil {
		return nil, fmt.Errorf("failed to create loop record: %w", err)
	}

	m.active = loop
	m.stopChan = make(chan struct{})

	// Start loop in background
	go m.runLoop(ctx, config)

	return m.getState(), nil
}

// StopLoop stops the active loop
func (m *LoopManager) StopLoop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.active == nil || m.active.Status != "running" {
		return ErrNoLoopRunning
	}

	close(m.stopChan)
	return nil
}

// GetState returns the current loop state
func (m *LoopManager) GetState() *LoopState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.getState()
}

func (m *LoopManager) getState() *LoopState {
	if m.active == nil {
		return nil
	}

	elapsed := time.Since(m.active.StartedAt).Minutes()

	state := &LoopState{
		ID:               m.active.ID,
		Status:           LoopStatus(m.active.Status),
		CurrentIteration: m.active.TotalIterations,
		MaxIterations:    m.active.MaxIterations,
		StartedAt:        m.active.StartedAt,
		ElapsedMinutes:   elapsed,
	}

	if m.active.BaselineScore != nil {
		state.BaselineScore = *m.active.BaselineScore
	}
	if m.active.BestScore != nil {
		state.BestScore = *m.active.BestScore
	}
	if m.active.FinalScore != nil {
		state.CurrentScore = *m.active.FinalScore
	}
	if m.active.BestRunID != "" {
		state.BestRunID = m.active.BestRunID
	}
	if m.active.StopReason != "" {
		state.StopReason = m.active.StopReason
	}

	return state
}

// runLoop executes the improvement loop
func (m *LoopManager) runLoop(ctx context.Context, config *LoopConfig) {
	defer func() {
		m.mu.Lock()
		if m.active != nil {
			now := time.Now()
			m.active.CompletedAt = &now
			m.db.UpdateAutonomousLoop(m.active)
		}
		m.mu.Unlock()
	}()

	// Run baseline
	baselineConfig := &RunConfig{
		BenchmarkType: "locomo",
		MaxQuestions:  100, // Use smaller set for faster iteration
	}

	baselineResults, err := m.service.Run(ctx, baselineConfig)
	if err != nil {
		m.setFailed(fmt.Sprintf("baseline run failed: %v", err))
		return
	}

	m.mu.Lock()
	baselineScore := baselineResults.Overall.LLMJudgeAccuracy
	m.active.BaselineScore = &baselineScore
	m.active.BestScore = &baselineScore
	m.active.BestRunID = baselineResults.RunID
	m.db.UpdateAutonomousLoop(m.active)
	m.mu.Unlock()

	// Track changes for analysis
	changesAttempted := []string{}
	changesAccepted := []string{}
	changesRejected := []string{}
	noImprovementCount := 0

	// Improvement loop
	for i := 0; i < config.MaxIterations; i++ {
		select {
		case <-m.stopChan:
			m.setCompleted("stopped by user")
			return
		case <-ctx.Done():
			m.setCompleted("context cancelled")
			return
		default:
		}

		m.mu.Lock()
		m.active.TotalIterations = i + 1
		m.db.UpdateAutonomousLoop(m.active)
		m.mu.Unlock()

		// In a full implementation, Claude would analyze weak areas and propose changes
		// For now, we just run benchmarks to track progression
		runConfig := &RunConfig{
			BenchmarkType: "locomo",
			MaxQuestions:  100,
			ChangeDesc:    fmt.Sprintf("Iteration %d", i+1),
		}

		results, err := m.service.Run(ctx, runConfig)
		if err != nil {
			changesAttempted = append(changesAttempted, fmt.Sprintf("iteration %d: failed", i+1))
			continue
		}

		currentScore := results.Overall.LLMJudgeAccuracy
		changesAttempted = append(changesAttempted, fmt.Sprintf("iteration %d: %.1f%%", i+1, currentScore))

		m.mu.Lock()
		m.active.FinalScore = &currentScore

		// Check improvement
		improvement := currentScore - *m.active.BestScore
		if improvement > config.MinImprovementThreshold*100 {
			// Improved!
			m.active.BestScore = &currentScore
			m.active.BestRunID = results.RunID
			changesAccepted = append(changesAccepted, fmt.Sprintf("iteration %d: +%.1f%%", i+1, improvement))
			noImprovementCount = 0
		} else {
			changesRejected = append(changesRejected, fmt.Sprintf("iteration %d: %.1f%%", i+1, improvement))
			noImprovementCount++
		}

		// Check convergence
		if improvement < config.ConvergenceThreshold*100 && improvement > -config.ConvergenceThreshold*100 {
			m.active.Status = "completed"
			m.active.StopReason = fmt.Sprintf("converged after %d iterations", i+1)
			m.saveChanges(changesAttempted, changesAccepted, changesRejected)
			m.db.UpdateAutonomousLoop(m.active)
			m.mu.Unlock()
			return
		}

		// Check no improvement streak
		if noImprovementCount >= 5 {
			m.active.Status = "completed"
			m.active.StopReason = "no improvement after 5 consecutive iterations"
			m.saveChanges(changesAttempted, changesAccepted, changesRejected)
			m.db.UpdateAutonomousLoop(m.active)
			m.mu.Unlock()
			return
		}

		m.db.UpdateAutonomousLoop(m.active)
		m.mu.Unlock()
	}

	m.setCompleted("max iterations reached")
	m.saveChanges(changesAttempted, changesAccepted, changesRejected)
}

func (m *LoopManager) setCompleted(reason string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.active != nil {
		m.active.Status = "completed"
		m.active.StopReason = reason
		now := time.Now()
		m.active.CompletedAt = &now
		m.db.UpdateAutonomousLoop(m.active)
	}
}

func (m *LoopManager) setFailed(reason string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.active != nil {
		m.active.Status = "failed"
		m.active.StopReason = reason
		now := time.Now()
		m.active.CompletedAt = &now
		m.db.UpdateAutonomousLoop(m.active)
	}
}

func (m *LoopManager) saveChanges(attempted, accepted, rejected []string) {
	if m.active == nil {
		return
	}

	if data, err := json.Marshal(attempted); err == nil {
		m.active.ChangesAttempted = string(data)
	}
	if data, err := json.Marshal(accepted); err == nil {
		m.active.ChangesAccepted = string(data)
	}
	if data, err := json.Marshal(rejected); err == nil {
		m.active.ChangesRejected = string(data)
	}
}

// IsRunning returns true if a loop is currently running
func (m *LoopManager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.active != nil && m.active.Status == "running"
}
