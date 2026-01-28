package ratelimit

import (
	"sync"
	"sync/atomic"
	"time"
)

// Metrics tracks rate limiting statistics
type Metrics struct {
	mu sync.RWMutex

	// Counters
	totalAllowed  uint64
	totalRejected uint64

	// Per-tool counters
	allowedByTool  map[string]*uint64
	rejectedByTool map[string]*uint64

	// Per-limit-type rejections (global vs tool-specific)
	rejectionsByType map[string]*uint64

	// Timing
	startTime time.Time
}

// NewMetrics creates a new metrics tracker
func NewMetrics() *Metrics {
	return &Metrics{
		allowedByTool:    make(map[string]*uint64),
		rejectedByTool:   make(map[string]*uint64),
		rejectionsByType: make(map[string]*uint64),
		startTime:        time.Now(),
	}
}

// RecordAllowed records an allowed request
func (m *Metrics) RecordAllowed(toolName string) {
	atomic.AddUint64(&m.totalAllowed, 1)

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.allowedByTool[toolName]; !exists {
		var zero uint64
		m.allowedByTool[toolName] = &zero
	}
	atomic.AddUint64(m.allowedByTool[toolName], 1)
}

// RecordRejection records a rejected request
func (m *Metrics) RecordRejection(limitType, toolName string) {
	atomic.AddUint64(&m.totalRejected, 1)

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.rejectedByTool[toolName]; !exists {
		var zero uint64
		m.rejectedByTool[toolName] = &zero
	}
	atomic.AddUint64(m.rejectedByTool[toolName], 1)

	if _, exists := m.rejectionsByType[limitType]; !exists {
		var zero uint64
		m.rejectionsByType[limitType] = &zero
	}
	atomic.AddUint64(m.rejectionsByType[limitType], 1)
}

// MetricsSnapshot represents a point-in-time snapshot of metrics
type MetricsSnapshot struct {
	TotalAllowed     uint64             `json:"total_allowed"`
	TotalRejected    uint64             `json:"total_rejected"`
	AllowedByTool    map[string]uint64  `json:"allowed_by_tool"`
	RejectedByTool   map[string]uint64  `json:"rejected_by_tool"`
	RejectionsByType map[string]uint64  `json:"rejections_by_type"`
	Uptime           time.Duration      `json:"uptime"`
	RequestsPerSec   float64            `json:"requests_per_second"`
}

// Snapshot returns a point-in-time snapshot of all metrics
func (m *Metrics) Snapshot() *MetricsSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	snapshot := &MetricsSnapshot{
		TotalAllowed:     atomic.LoadUint64(&m.totalAllowed),
		TotalRejected:    atomic.LoadUint64(&m.totalRejected),
		AllowedByTool:    make(map[string]uint64),
		RejectedByTool:   make(map[string]uint64),
		RejectionsByType: make(map[string]uint64),
		Uptime:           time.Since(m.startTime),
	}

	for tool, count := range m.allowedByTool {
		snapshot.AllowedByTool[tool] = atomic.LoadUint64(count)
	}
	for tool, count := range m.rejectedByTool {
		snapshot.RejectedByTool[tool] = atomic.LoadUint64(count)
	}
	for limitType, count := range m.rejectionsByType {
		snapshot.RejectionsByType[limitType] = atomic.LoadUint64(count)
	}

	// Calculate requests per second
	totalRequests := snapshot.TotalAllowed + snapshot.TotalRejected
	if snapshot.Uptime.Seconds() > 0 {
		snapshot.RequestsPerSec = float64(totalRequests) / snapshot.Uptime.Seconds()
	}

	return snapshot
}

// TotalAllowed returns the total number of allowed requests
func (m *Metrics) TotalAllowed() uint64 {
	return atomic.LoadUint64(&m.totalAllowed)
}

// TotalRejected returns the total number of rejected requests
func (m *Metrics) TotalRejected() uint64 {
	return atomic.LoadUint64(&m.totalRejected)
}

// RejectionRate returns the current rejection rate (0.0 to 1.0)
func (m *Metrics) RejectionRate() float64 {
	allowed := atomic.LoadUint64(&m.totalAllowed)
	rejected := atomic.LoadUint64(&m.totalRejected)
	total := allowed + rejected
	if total == 0 {
		return 0
	}
	return float64(rejected) / float64(total)
}

// Reset resets all metrics
func (m *Metrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	atomic.StoreUint64(&m.totalAllowed, 0)
	atomic.StoreUint64(&m.totalRejected, 0)
	m.allowedByTool = make(map[string]*uint64)
	m.rejectedByTool = make(map[string]*uint64)
	m.rejectionsByType = make(map[string]*uint64)
	m.startTime = time.Now()
}
