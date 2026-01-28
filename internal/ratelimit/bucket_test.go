package ratelimit

import (
	"testing"
	"time"
)

func TestNewBucket(t *testing.T) {
	bucket := NewBucket(10, 5)

	if bucket.Capacity() != 10 {
		t.Errorf("expected capacity 10, got %f", bucket.Capacity())
	}
	if bucket.RefillRate() != 5 {
		t.Errorf("expected refill rate 5, got %f", bucket.RefillRate())
	}
	if bucket.Tokens() < 9.9 { // Allow small time drift
		t.Errorf("expected ~10 tokens, got %f", bucket.Tokens())
	}
}

func TestTryConsume(t *testing.T) {
	bucket := NewBucket(10, 1)

	// Should succeed - have 10 tokens
	if !bucket.TryConsume(5) {
		t.Error("expected consume to succeed")
	}

	// Should succeed - have 5 tokens
	if !bucket.TryConsume(3) {
		t.Error("expected consume to succeed")
	}

	// Should fail - only have ~2 tokens
	if bucket.TryConsume(5) {
		t.Error("expected consume to fail")
	}
}

func TestRefill(t *testing.T) {
	bucket := NewBucket(10, 100) // 100 tokens/sec

	// Consume all tokens
	bucket.TryConsume(10)
	if bucket.Tokens() > 0.5 {
		t.Errorf("expected ~0 tokens after consume, got %f", bucket.Tokens())
	}

	// Wait for refill
	time.Sleep(50 * time.Millisecond) // Should refill ~5 tokens

	tokens := bucket.Tokens()
	if tokens < 4 || tokens > 6 {
		t.Errorf("expected ~5 tokens after refill, got %f", tokens)
	}
}

func TestTimeToWait(t *testing.T) {
	bucket := NewBucket(10, 10) // 10 tokens/sec

	// Consume all tokens
	bucket.TryConsume(10)

	// Need 5 tokens = 0.5 seconds
	waitTime := bucket.TimeToWait(5)
	if waitTime < 400*time.Millisecond || waitTime > 600*time.Millisecond {
		t.Errorf("expected ~500ms wait time, got %v", waitTime)
	}
}

func TestReset(t *testing.T) {
	bucket := NewBucket(10, 1)

	bucket.TryConsume(8)
	bucket.Reset()

	if bucket.Tokens() < 9.9 {
		t.Errorf("expected ~10 tokens after reset, got %f", bucket.Tokens())
	}
}

func TestCapacityLimit(t *testing.T) {
	bucket := NewBucket(10, 100)

	// Wait to accumulate more than capacity
	time.Sleep(200 * time.Millisecond)

	// Should still be capped at capacity
	if bucket.Tokens() > 10.1 {
		t.Errorf("expected tokens <= 10, got %f", bucket.Tokens())
	}
}
