package ratelimit

// Config holds rate limiting configuration
type Config struct {
	Enabled bool          `mapstructure:"enabled"`
	Global  LimitConfig   `mapstructure:"global"`
	Tools   []ToolLimit   `mapstructure:"tools"`
}

// LimitConfig defines rate limit parameters
type LimitConfig struct {
	RequestsPerSecond float64 `mapstructure:"requests_per_second"`
	BurstSize         int     `mapstructure:"burst_size"`
}

// ToolLimit defines per-tool rate limiting
type ToolLimit struct {
	Name              string  `mapstructure:"name"`
	RequestsPerSecond float64 `mapstructure:"requests_per_second"`
	BurstSize         int     `mapstructure:"burst_size"`
}

// DefaultConfig returns the default rate limiting configuration
func DefaultConfig() *Config {
	return &Config{
		Enabled: true,
		Global: LimitConfig{
			RequestsPerSecond: 100,
			BurstSize:         200,
		},
		Tools: []ToolLimit{
			{
				Name:              "analysis",
				RequestsPerSecond: 5,
				BurstSize:         10,
			},
			{
				Name:              "search",
				RequestsPerSecond: 20,
				BurstSize:         40,
			},
			{
				Name:              "benchmark_run",
				RequestsPerSecond: 0.1, // 1 every 10 seconds
				BurstSize:         2,
			},
			{
				Name:              "store_memory",
				RequestsPerSecond: 30,
				BurstSize:         60,
			},
			{
				Name:              "relationships",
				RequestsPerSecond: 20,
				BurstSize:         40,
			},
		},
	}
}

// GetToolLimit returns the limit configuration for a specific tool
// Returns nil if no specific limit is configured for the tool
func (c *Config) GetToolLimit(toolName string) *ToolLimit {
	for _, tool := range c.Tools {
		if tool.Name == toolName {
			return &tool
		}
	}
	return nil
}
