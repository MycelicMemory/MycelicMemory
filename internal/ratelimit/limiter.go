package ratelimit

import (
	"sync"
	"time"
)

// LimitResult contains the result of a rate limit check
type LimitResult struct {
	Allowed    bool          // Whether the request is allowed
	RetryAfter time.Duration // Suggested wait time if not allowed
	LimitType  string        // "global" or tool name
	Remaining  float64       // Remaining tokens in the relevant bucket
}

// Limiter manages rate limiting with global and per-tool buckets
type Limiter struct {
	mu           sync.RWMutex
	enabled      bool
	globalBucket *Bucket
	toolBuckets  map[string]*Bucket
	config       *Config
	metrics      *Metrics
}

// NewLimiter creates a new rate limiter from configuration
func NewLimiter(cfg *Config) *Limiter {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	l := &Limiter{
		enabled:     cfg.Enabled,
		toolBuckets: make(map[string]*Bucket),
		config:      cfg,
		metrics:     NewMetrics(),
	}

	// Create global bucket
	l.globalBucket = NewBucket(
		float64(cfg.Global.BurstSize),
		cfg.Global.RequestsPerSecond,
	)

	// Create per-tool buckets
	for _, toolLimit := range cfg.Tools {
		l.toolBuckets[toolLimit.Name] = NewBucket(
			float64(toolLimit.BurstSize),
			toolLimit.RequestsPerSecond,
		)
	}

	return l
}

// Allow checks if a request for the given tool is allowed
// Returns a LimitResult with the decision and metadata
func (l *Limiter) Allow(toolName string) *LimitResult {
	if !l.enabled {
		return &LimitResult{
			Allowed:   true,
			LimitType: "disabled",
			Remaining: -1,
		}
	}

	l.mu.RLock()
	defer l.mu.RUnlock()

	// Check global limit first
	if !l.globalBucket.TryConsume(1) {
		retryAfter := l.globalBucket.TimeToWait(1)
		l.metrics.RecordRejection("global", toolName)
		return &LimitResult{
			Allowed:    false,
			RetryAfter: retryAfter,
			LimitType:  "global",
			Remaining:  l.globalBucket.Tokens(),
		}
	}

	// Check tool-specific limit if configured
	if toolBucket, exists := l.toolBuckets[toolName]; exists {
		if !toolBucket.TryConsume(1) {
			// Refund the global token since we're rejecting
			l.globalBucket.Reset() // Note: This is a simplified approach
			retryAfter := toolBucket.TimeToWait(1)
			l.metrics.RecordRejection(toolName, toolName)
			return &LimitResult{
				Allowed:    false,
				RetryAfter: retryAfter,
				LimitType:  toolName,
				Remaining:  toolBucket.Tokens(),
			}
		}
		l.metrics.RecordAllowed(toolName)
		return &LimitResult{
			Allowed:   true,
			LimitType: toolName,
			Remaining: toolBucket.Tokens(),
		}
	}

	// No tool-specific limit, global check passed
	l.metrics.RecordAllowed(toolName)
	return &LimitResult{
		Allowed:   true,
		LimitType: "global",
		Remaining: l.globalBucket.Tokens(),
	}
}

// IsEnabled returns whether rate limiting is enabled
func (l *Limiter) IsEnabled() bool {
	return l.enabled
}

// SetEnabled enables or disables rate limiting
func (l *Limiter) SetEnabled(enabled bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.enabled = enabled
}

// GetMetrics returns the current metrics
func (l *Limiter) GetMetrics() *Metrics {
	return l.metrics
}

// GetToolBucket returns the bucket for a specific tool (for testing)
func (l *Limiter) GetToolBucket(toolName string) *Bucket {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.toolBuckets[toolName]
}

// GetGlobalBucket returns the global bucket (for testing)
func (l *Limiter) GetGlobalBucket() *Bucket {
	return l.globalBucket
}

// Reset resets all buckets to full capacity
func (l *Limiter) Reset() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.globalBucket.Reset()
	for _, bucket := range l.toolBuckets {
		bucket.Reset()
	}
}

// Stats returns current limiter statistics
type Stats struct {
	Enabled      bool               `json:"enabled"`
	GlobalTokens float64            `json:"global_tokens"`
	ToolTokens   map[string]float64 `json:"tool_tokens"`
}

// GetStats returns current limiter statistics
func (l *Limiter) GetStats() *Stats {
	l.mu.RLock()
	defer l.mu.RUnlock()

	stats := &Stats{
		Enabled:      l.enabled,
		GlobalTokens: l.globalBucket.Tokens(),
		ToolTokens:   make(map[string]float64),
	}

	for name, bucket := range l.toolBuckets {
		stats.ToolTokens[name] = bucket.Tokens()
	}

	return stats
}
