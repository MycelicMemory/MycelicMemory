package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/MycelicMemory/mycelicmemory/internal/ratelimit"
)

// =============================================================================
// AUTH MIDDLEWARE
// =============================================================================

// APIKeyAuthMiddleware returns middleware that checks for a valid API key.
// Health endpoint is exempt. No-op if apiKey is empty.
func APIKeyAuthMiddleware(apiKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// No-op if no API key configured
		if apiKey == "" {
			c.Next()
			return
		}

		// Health endpoint is always accessible
		if c.Request.URL.Path == "/api/v1/health" {
			c.Next()
			return
		}

		// Check Authorization: Bearer <key>
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") && parts[1] == apiKey {
				c.Next()
				return
			}
		}

		// Check X-API-Key header
		if c.GetHeader("X-API-Key") == apiKey {
			c.Next()
			return
		}

		UnauthorizedError(c, "Invalid or missing API key")
		c.Abort()
	}
}

// =============================================================================
// RATE LIMIT MIDDLEWARE
// =============================================================================

// routeToToolCategory maps API routes to rate limiter tool categories
func routeToToolCategory(path, method string) string {
	switch {
	case strings.Contains(path, "/search") || strings.Contains(path, "/related") || strings.Contains(path, "/graph"):
		return "search"
	case strings.Contains(path, "/analyze"):
		return "analysis"
	case method == "POST" && strings.HasSuffix(path, "/memories"):
		return "store_memory"
	case strings.Contains(path, "/relationships") || strings.Contains(path, "/discover"):
		return "relationships"
	default:
		return ""
	}
}

// RateLimitMiddleware returns middleware that rate-limits requests using the provided limiter
func RateLimitMiddleware(limiter *ratelimit.Limiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if limiter == nil {
			c.Next()
			return
		}

		toolCategory := routeToToolCategory(c.Request.URL.Path, c.Request.Method)
		if toolCategory == "" {
			toolCategory = "default"
		}

		result := limiter.Allow(toolCategory)
		if !result.Allowed {
			retryAfter := int(result.RetryAfter.Seconds())
			if retryAfter < 1 {
				retryAfter = 1
			}
			c.Header("Retry-After", fmt.Sprintf("%d", retryAfter))
			TooManyRequestsError(c, fmt.Sprintf("Rate limit exceeded for %s. Retry after %d seconds.", result.LimitType, retryAfter))
			c.Abort()
			return
		}

		c.Next()
	}
}

// =============================================================================
// BODY SIZE MIDDLEWARE
// =============================================================================

// MaxBodySizeMiddleware returns middleware that limits request body size
func MaxBodySizeMiddleware(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body != nil && c.Request.ContentLength > maxBytes {
			PayloadTooLargeError(c, fmt.Sprintf("Request body too large. Maximum: %d bytes", maxBytes))
			c.Abort()
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}

// =============================================================================
// VALIDATION CONSTANTS
// =============================================================================

const (
	MaxContentLength = 100 * 1024 // 100KB
	MaxQueryLength   = 10 * 1024  // 10KB
	MaxTags          = 100
	MaxTagLength     = 200
	MaxLimit         = 1000
	DefaultLimit     = 50
	MaxNameLength    = 500
	MaxSourceLength  = 500
	DefaultBodyLimit = 1 * 1024 * 1024  // 1MB
	IngestBodyLimit  = 10 * 1024 * 1024 // 10MB
)

// =============================================================================
// VALIDATION HELPERS
// =============================================================================

// clampLimit ensures limit is within valid range
func clampLimit(limit int) int {
	if limit <= 0 {
		return DefaultLimit
	}
	if limit > MaxLimit {
		return MaxLimit
	}
	return limit
}

// validateImportance checks that importance is in valid range (1-10)
func validateImportance(importance int) error {
	if importance < 0 || importance > 10 {
		return fmt.Errorf("importance must be between 0 and 10, got %d", importance)
	}
	return nil
}

// validateTags checks tags array for size and content
func validateTags(tags []string) error {
	if len(tags) > MaxTags {
		return fmt.Errorf("too many tags: %d (maximum: %d)", len(tags), MaxTags)
	}
	for _, tag := range tags {
		if len(tag) > MaxTagLength {
			return fmt.Errorf("tag too long: %d characters (maximum: %d)", len(tag), MaxTagLength)
		}
	}
	return nil
}

// validateContent checks content string for length
func validateContent(content string) error {
	if len(content) > MaxContentLength {
		return fmt.Errorf("content too long: %d bytes (maximum: %d)", len(content), MaxContentLength)
	}
	return nil
}

// validateQuery checks search query for length
func validateQuery(query string) error {
	if len(query) > MaxQueryLength {
		return fmt.Errorf("query too long: %d bytes (maximum: %d)", len(query), MaxQueryLength)
	}
	return nil
}
