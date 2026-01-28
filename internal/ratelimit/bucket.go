package ratelimit

import (
	"sync"
	"time"
)

// Bucket implements a token bucket rate limiter
// Thread-safe with automatic token refill based on elapsed time
type Bucket struct {
	mu         sync.Mutex
	tokens     float64
	capacity   float64
	refillRate float64   // tokens per second
	lastRefill time.Time
}

// NewBucket creates a new token bucket
// capacity: maximum tokens the bucket can hold (burst size)
// refillRate: tokens added per second
func NewBucket(capacity, refillRate float64) *Bucket {
	return &Bucket{
		tokens:     capacity, // Start full
		capacity:   capacity,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// TryConsume attempts to consume n tokens from the bucket
// Returns true if successful, false if insufficient tokens
func (b *Bucket) TryConsume(n float64) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.refill()

	if b.tokens >= n {
		b.tokens -= n
		return true
	}
	return false
}

// refill adds tokens based on elapsed time since last refill
// Must be called with mutex held
func (b *Bucket) refill() {
	now := time.Now()
	elapsed := now.Sub(b.lastRefill).Seconds()
	b.lastRefill = now

	// Add tokens based on elapsed time
	b.tokens += elapsed * b.refillRate
	if b.tokens > b.capacity {
		b.tokens = b.capacity
	}
}

// Tokens returns the current number of available tokens
func (b *Bucket) Tokens() float64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.refill()
	return b.tokens
}

// TimeToWait returns the duration to wait until n tokens are available
// Returns 0 if tokens are already available
func (b *Bucket) TimeToWait(n float64) time.Duration {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.refill()

	if b.tokens >= n {
		return 0
	}

	needed := n - b.tokens
	seconds := needed / b.refillRate
	return time.Duration(seconds * float64(time.Second))
}

// Reset resets the bucket to full capacity
func (b *Bucket) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.tokens = b.capacity
	b.lastRefill = time.Now()
}

// Capacity returns the bucket's maximum capacity
func (b *Bucket) Capacity() float64 {
	return b.capacity
}

// RefillRate returns the bucket's refill rate in tokens/second
func (b *Bucket) RefillRate() float64 {
	return b.refillRate
}
