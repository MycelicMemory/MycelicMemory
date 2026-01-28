package ratelimit

import (
	"testing"
)

func TestNewLimiter(t *testing.T) {
	cfg := &Config{
		Enabled: true,
		Global: LimitConfig{
			RequestsPerSecond: 100,
			BurstSize:         200,
		},
		Tools: []ToolLimit{
			{Name: "search", RequestsPerSecond: 20, BurstSize: 40},
		},
	}

	limiter := NewLimiter(cfg)

	if !limiter.IsEnabled() {
		t.Error("expected limiter to be enabled")
	}

	if limiter.GetGlobalBucket() == nil {
		t.Error("expected global bucket to exist")
	}

	if limiter.GetToolBucket("search") == nil {
		t.Error("expected search bucket to exist")
	}

	if limiter.GetToolBucket("unknown") != nil {
		t.Error("expected unknown bucket to be nil")
	}
}

func TestAllowGlobalLimit(t *testing.T) {
	cfg := &Config{
		Enabled: true,
		Global: LimitConfig{
			RequestsPerSecond: 1,
			BurstSize:         2,
		},
	}

	limiter := NewLimiter(cfg)

	// First two requests should succeed (burst)
	result1 := limiter.Allow("test")
	if !result1.Allowed {
		t.Error("expected first request to be allowed")
	}

	result2 := limiter.Allow("test")
	if !result2.Allowed {
		t.Error("expected second request to be allowed")
	}

	// Third request should fail (exceeded burst)
	result3 := limiter.Allow("test")
	if result3.Allowed {
		t.Error("expected third request to be rejected")
	}
	if result3.LimitType != "global" {
		t.Errorf("expected limit type 'global', got '%s'", result3.LimitType)
	}
}

func TestAllowToolLimit(t *testing.T) {
	cfg := &Config{
		Enabled: true,
		Global: LimitConfig{
			RequestsPerSecond: 100,
			BurstSize:         200,
		},
		Tools: []ToolLimit{
			{Name: "expensive", RequestsPerSecond: 1, BurstSize: 1},
		},
	}

	limiter := NewLimiter(cfg)

	// First request to expensive tool should succeed
	result1 := limiter.Allow("expensive")
	if !result1.Allowed {
		t.Error("expected first expensive request to be allowed")
	}

	// Second request should be rejected by tool limit
	result2 := limiter.Allow("expensive")
	if result2.Allowed {
		t.Error("expected second expensive request to be rejected")
	}
	if result2.LimitType != "expensive" {
		t.Errorf("expected limit type 'expensive', got '%s'", result2.LimitType)
	}

	// Request to other tool should still succeed (global limit)
	result3 := limiter.Allow("cheap")
	if !result3.Allowed {
		t.Error("expected cheap request to be allowed")
	}
}

func TestDisabledLimiter(t *testing.T) {
	cfg := &Config{
		Enabled: false,
		Global: LimitConfig{
			RequestsPerSecond: 1,
			BurstSize:         1,
		},
	}

	limiter := NewLimiter(cfg)

	// All requests should be allowed when disabled
	for i := 0; i < 100; i++ {
		result := limiter.Allow("test")
		if !result.Allowed {
			t.Errorf("expected request %d to be allowed when disabled", i)
		}
		if result.LimitType != "disabled" {
			t.Errorf("expected limit type 'disabled', got '%s'", result.LimitType)
		}
	}
}

func TestSetEnabled(t *testing.T) {
	cfg := &Config{
		Enabled: true,
		Global: LimitConfig{
			RequestsPerSecond: 1,
			BurstSize:         1,
		},
	}

	limiter := NewLimiter(cfg)

	// Exhaust the bucket
	limiter.Allow("test")

	// Request should be rejected
	result := limiter.Allow("test")
	if result.Allowed {
		t.Error("expected request to be rejected")
	}

	// Disable limiter
	limiter.SetEnabled(false)

	// Request should now be allowed
	result = limiter.Allow("test")
	if !result.Allowed {
		t.Error("expected request to be allowed when disabled")
	}
}

func TestGetStats(t *testing.T) {
	cfg := &Config{
		Enabled: true,
		Global: LimitConfig{
			RequestsPerSecond: 100,
			BurstSize:         200,
		},
		Tools: []ToolLimit{
			{Name: "search", RequestsPerSecond: 20, BurstSize: 40},
		},
	}

	limiter := NewLimiter(cfg)
	stats := limiter.GetStats()

	if !stats.Enabled {
		t.Error("expected stats.Enabled to be true")
	}
	if stats.GlobalTokens < 199 {
		t.Errorf("expected ~200 global tokens, got %f", stats.GlobalTokens)
	}
	if _, ok := stats.ToolTokens["search"]; !ok {
		t.Error("expected search tool tokens in stats")
	}
}

func TestLimiterReset(t *testing.T) {
	cfg := &Config{
		Enabled: true,
		Global: LimitConfig{
			RequestsPerSecond: 1,
			BurstSize:         2,
		},
	}

	limiter := NewLimiter(cfg)

	// Exhaust buckets
	limiter.Allow("test")
	limiter.Allow("test")

	// Reset
	limiter.Reset()

	// Should be able to make requests again
	result := limiter.Allow("test")
	if !result.Allowed {
		t.Error("expected request to be allowed after reset")
	}
}
