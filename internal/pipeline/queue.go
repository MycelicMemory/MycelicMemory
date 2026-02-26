package pipeline

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/MycelicMemory/mycelicmemory/internal/database"
	"github.com/MycelicMemory/mycelicmemory/internal/logging"
	"github.com/MycelicMemory/mycelicmemory/internal/relationships"
	"github.com/google/uuid"
)

var log = logging.GetLogger("pipeline")

// QueueConfig configures the ingestion queue behavior.
type QueueConfig struct {
	MaxConcurrent  int           // Worker pool size (default: 4)
	BatchSize      int           // Items per batch before flush (default: 100)
	RetryAttempts  int           // Max retries per failed item (default: 3)
	RetryDelay     time.Duration // Base delay between retries (default: 1s, exponential)
	BackfillMode   bool          // Process all history vs incremental from checkpoint
	ProgressReport time.Duration // How often to emit progress updates (default: 5s)
}

// DefaultQueueConfig returns sensible defaults.
func DefaultQueueConfig() QueueConfig {
	return QueueConfig{
		MaxConcurrent:  4,
		BatchSize:      100,
		RetryAttempts:  3,
		RetryDelay:     time.Second,
		BackfillMode:   false,
		ProgressReport: 5 * time.Second,
	}
}

// JobStatus represents the current state of a queued ingestion job.
type JobStatus struct {
	ID         string        `json:"id"`
	SourceID   string        `json:"source_id"`
	SourceType string        `json:"source_type"`
	Status     string        `json:"status"` // "queued", "running", "completed", "failed", "cancelled"
	Progress   *ProgressUpdate `json:"progress,omitempty"`
	Result     *IngestResult   `json:"result,omitempty"`
	Error      string        `json:"error,omitempty"`
	StartedAt  *time.Time    `json:"started_at,omitempty"`
	CompletedAt *time.Time   `json:"completed_at,omitempty"`
}

// Queue manages the ingestion pipeline, coordinating adapters, transformers, and storage.
type Queue struct {
	db       *database.Database
	relSvc   *relationships.Service
	adapters map[string]SourceAdapter
	config   QueueConfig

	// Active jobs
	mu   sync.RWMutex
	jobs map[string]*JobStatus

	// Progress reporting
	progress chan ProgressUpdate
}

// NewQueue creates a new ingestion queue.
func NewQueue(db *database.Database, relSvc *relationships.Service, config QueueConfig) *Queue {
	if config.MaxConcurrent <= 0 {
		config.MaxConcurrent = DefaultQueueConfig().MaxConcurrent
	}
	if config.BatchSize <= 0 {
		config.BatchSize = DefaultQueueConfig().BatchSize
	}
	if config.RetryAttempts <= 0 {
		config.RetryAttempts = DefaultQueueConfig().RetryAttempts
	}
	if config.RetryDelay <= 0 {
		config.RetryDelay = DefaultQueueConfig().RetryDelay
	}
	if config.ProgressReport <= 0 {
		config.ProgressReport = DefaultQueueConfig().ProgressReport
	}

	return &Queue{
		db:       db,
		relSvc:   relSvc,
		adapters: make(map[string]SourceAdapter),
		config:   config,
		jobs:     make(map[string]*JobStatus),
		progress: make(chan ProgressUpdate, 100),
	}
}

// RegisterAdapter registers a source adapter for a given source type.
func (q *Queue) RegisterAdapter(adapter SourceAdapter) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.adapters[adapter.Type()] = adapter
	log.Info("registered source adapter", "type", adapter.Type())
}

// GetAdapter returns the registered adapter for a source type, or nil.
func (q *Queue) GetAdapter(sourceType string) SourceAdapter {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.adapters[sourceType]
}

// ListAdapters returns all registered adapter type names.
func (q *Queue) ListAdapters() []string {
	q.mu.RLock()
	defer q.mu.RUnlock()
	types := make([]string, 0, len(q.adapters))
	for t := range q.adapters {
		types = append(types, t)
	}
	return types
}

// Enqueue starts an ingestion job for the given data source.
// It runs asynchronously and returns the job ID immediately.
func (q *Queue) Enqueue(ctx context.Context, sourceID string, mode string) (string, error) {
	// Look up the data source
	ds, err := q.db.GetDataSource(sourceID)
	if err != nil {
		return "", fmt.Errorf("failed to get data source: %w", err)
	}
	if ds == nil {
		return "", fmt.Errorf("data source not found: %s", sourceID)
	}

	// Find adapter for this source type
	adapter := q.GetAdapter(ds.SourceType)
	if adapter == nil {
		return "", fmt.Errorf("no adapter registered for source type: %s", ds.SourceType)
	}

	// Configure adapter
	if err := adapter.Configure([]byte(ds.Config)); err != nil {
		return "", fmt.Errorf("failed to configure adapter: %w", err)
	}

	// Create job
	jobID := uuid.New().String()
	now := time.Now()
	job := &JobStatus{
		ID:         jobID,
		SourceID:   sourceID,
		SourceType: ds.SourceType,
		Status:     "queued",
		StartedAt:  &now,
	}

	q.mu.Lock()
	q.jobs[jobID] = job
	q.mu.Unlock()

	// Determine checkpoint
	checkpoint := ""
	if mode != "backfill" {
		checkpoint = ds.LastSyncPosition
	}

	// Create sync history record
	syncHistory := &database.DataSourceSyncHistory{
		ID:       jobID,
		SourceID: sourceID,
		Status:   "running",
	}
	if err := q.db.CreateSyncHistory(syncHistory); err != nil {
		log.Warn("failed to create sync history", "error", err)
	}

	// Run the job asynchronously
	go q.runJob(ctx, job, adapter, ds, checkpoint)

	return jobID, nil
}

// EnqueueDirect runs an ingestion job synchronously using a pre-configured adapter.
// This is used by the Claude ingester to maintain backward compatibility.
func (q *Queue) EnqueueDirect(ctx context.Context, adapter SourceAdapter, sourceID string, checkpoint string) (*IngestResult, error) {
	jobID := uuid.New().String()
	now := time.Now()
	job := &JobStatus{
		ID:         jobID,
		SourceID:   sourceID,
		SourceType: adapter.Type(),
		Status:     "running",
		StartedAt:  &now,
	}

	q.mu.Lock()
	q.jobs[jobID] = job
	q.mu.Unlock()

	// Create sync history record
	syncHistory := &database.DataSourceSyncHistory{
		ID:       jobID,
		SourceID: sourceID,
		Status:   "running",
	}
	if err := q.db.CreateSyncHistory(syncHistory); err != nil {
		log.Warn("failed to create sync history", "error", err)
	}

	q.executeJob(ctx, job, adapter, sourceID, checkpoint)

	if job.Error != "" {
		return job.Result, fmt.Errorf("%s", job.Error)
	}
	return job.Result, nil
}

// Status returns the current status of a job.
func (q *Queue) Status(jobID string) *JobStatus {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.jobs[jobID]
}

// StatusBySource returns the most recent job status for a source.
func (q *Queue) StatusBySource(sourceID string) *JobStatus {
	q.mu.RLock()
	defer q.mu.RUnlock()

	var latest *JobStatus
	for _, job := range q.jobs {
		if job.SourceID == sourceID {
			if latest == nil || (job.StartedAt != nil && latest.StartedAt != nil && job.StartedAt.After(*latest.StartedAt)) {
				latest = job
			}
		}
	}
	return latest
}

// Progress returns the progress channel for monitoring.
func (q *Queue) Progress() <-chan ProgressUpdate {
	return q.progress
}

func (q *Queue) runJob(ctx context.Context, job *JobStatus, adapter SourceAdapter, ds *database.DataSource, checkpoint string) {
	q.executeJob(ctx, job, adapter, ds.ID, checkpoint)
}

func (q *Queue) executeJob(ctx context.Context, job *JobStatus, adapter SourceAdapter, sourceID string, checkpoint string) {
	start := time.Now()

	q.mu.Lock()
	job.Status = "running"
	q.mu.Unlock()

	transformer := NewTransformer(q.db, q.relSvc)

	// Read items from adapter
	itemsCh, errCh := adapter.ReadItems(ctx, checkpoint)

	result := &IngestResult{
		SourceID:   sourceID,
		SourceType: adapter.Type(),
	}

	// Collect items in batches
	batch := make([]ConversationItem, 0, q.config.BatchSize)

	for item := range itemsCh {
		batch = append(batch, item)

		if len(batch) >= q.config.BatchSize {
			batchResult, err := transformer.TransformBatch(ctx, batch, sourceID)
			if err != nil {
				log.Warn("batch transform error", "error", err)
				result.Errors++
			} else {
				mergeResults(result, batchResult)
			}
			batch = batch[:0]

			// Emit progress
			q.emitProgress(job, result, "transforming")
		}
	}

	// Process remaining items
	if len(batch) > 0 {
		batchResult, err := transformer.TransformBatch(ctx, batch, sourceID)
		if err != nil {
			log.Warn("final batch transform error", "error", err)
			result.Errors++
		} else {
			mergeResults(result, batchResult)
		}
	}

	// Check for read errors
	if readErr := <-errCh; readErr != nil {
		log.Warn("adapter read error", "error", readErr)
		result.Errors++
	}

	result.Duration = time.Since(start)
	result.Checkpoint = adapter.Checkpoint()

	// Update job status
	now := time.Now()
	q.mu.Lock()
	if result.Errors > 0 && result.SessionsCreated == 0 && result.MessagesCreated == 0 {
		job.Status = "failed"
		job.Error = "ingestion completed with errors and no results"
	} else {
		job.Status = "completed"
	}
	job.Result = result
	job.CompletedAt = &now
	q.mu.Unlock()

	// Update data source sync position
	if result.Checkpoint != "" {
		if err := q.db.UpdateDataSourceSyncTime(sourceID, now, result.Checkpoint); err != nil {
			log.Warn("failed to update sync position", "error", err)
		}
	}

	// Update sync history
	status := "completed"
	if job.Status == "failed" {
		status = "failed"
	}
	if err := q.db.UpdateSyncHistory(
		job.ID,
		result.SessionsProcessed,
		result.MemoriesCreated,
		result.DuplicatesSkipped,
		status,
		job.Error,
	); err != nil {
		log.Warn("failed to update sync history", "error", err)
	}

	q.emitProgress(job, result, "completed")

	log.Info("ingestion job completed",
		"job_id", job.ID,
		"source_type", job.SourceType,
		"sessions_created", result.SessionsCreated,
		"messages_created", result.MessagesCreated,
		"memories_created", result.MemoriesCreated,
		"duration", result.Duration,
	)
}

func (q *Queue) emitProgress(job *JobStatus, result *IngestResult, phase string) {
	update := ProgressUpdate{
		SourceID:        job.SourceID,
		SourceType:      job.SourceType,
		Phase:           phase,
		ItemsProcessed:  result.SessionsProcessed,
		ItemsTotal:      -1,
		SessionsCreated: result.SessionsCreated,
		MessagesCreated: result.MessagesCreated,
		MemoriesCreated: result.MemoriesCreated,
		Errors:          result.Errors,
		Checkpoint:      result.Checkpoint,
	}

	q.mu.Lock()
	job.Progress = &update
	q.mu.Unlock()

	select {
	case q.progress <- update:
	default:
		// Channel full, skip
	}
}

func mergeResults(dst, src *IngestResult) {
	dst.SessionsProcessed += src.SessionsProcessed
	dst.SessionsCreated += src.SessionsCreated
	dst.SessionsUpdated += src.SessionsUpdated
	dst.MessagesCreated += src.MessagesCreated
	dst.ActionsCreated += src.ActionsCreated
	dst.MemoriesCreated += src.MemoriesCreated
	dst.DuplicatesSkipped += src.DuplicatesSkipped
	dst.Errors += src.Errors
}
