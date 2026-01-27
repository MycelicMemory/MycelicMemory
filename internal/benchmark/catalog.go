package benchmark

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Catalog manages file-based benchmark results storage
type Catalog struct {
	baseDir string
}

// NewCatalog creates a new catalog at the specified directory
func NewCatalog(baseDir string) *Catalog {
	return &Catalog{baseDir: baseDir}
}

// DefaultCatalogDir returns the default catalog directory
func DefaultCatalogDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".mycelicmemory", "benchmark_results")
}

// EnsureDirectories creates the catalog directory structure
func (c *Catalog) EnsureDirectories() error {
	dirs := []string{
		filepath.Join(c.baseDir, "locomo", "runs"),
		filepath.Join(c.baseDir, "locomo", "comparisons"),
		filepath.Join(c.baseDir, "locomo", "best"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

// CatalogedRun represents a cataloged benchmark run
type CatalogedRun struct {
	RunID       string                    `json:"run_id"`
	Timestamp   time.Time                 `json:"timestamp"`
	Git         GitState                  `json:"git"`
	Config      RunConfig                 `json:"config"`
	Overall     OverallScores             `json:"overall"`
	ByCategory  map[string]CategoryScores `json:"by_category"`
	DurationSec float64                   `json:"duration_seconds"`
	IsBest      bool                      `json:"is_best,omitempty"`
	Comparison  *ComparisonInfo           `json:"comparison,omitempty"`
}

// ComparisonInfo holds baseline comparison data
type ComparisonInfo struct {
	BaselineRunID    string  `json:"baseline_run_id"`
	BaselineAccuracy float64 `json:"baseline_accuracy"`
	Improvement      float64 `json:"improvement"`
}

// SaveRun saves a benchmark run to the catalog
func (c *Catalog) SaveRun(results *RunResults) (string, error) {
	if err := c.EnsureDirectories(); err != nil {
		return "", err
	}

	run := &CatalogedRun{
		RunID:       results.RunID,
		Timestamp:   results.StartedAt,
		Git:         results.Git,
		Config:      results.Config,
		Overall:     results.Overall,
		ByCategory:  results.ByCategory,
		DurationSec: results.DurationSecs,
	}

	// Generate filename: YYYY-MM-DDTHH-MM-SS_<short_hash>.json
	filename := fmt.Sprintf("%s_%s.json",
		results.StartedAt.Format("2006-01-02T15-04-05"),
		results.Git.ShortHash)

	path := filepath.Join(c.baseDir, "locomo", "runs", filename)

	data, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal run: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write run file: %w", err)
	}

	return path, nil
}

// SaveComparison saves a comparison between two runs
func (c *Catalog) SaveComparison(comp *Comparison) (string, error) {
	if err := c.EnsureDirectories(); err != nil {
		return "", err
	}

	// Generate filename: <run_a_short>_vs_<run_b_short>.json
	aShort := comp.RunA
	bShort := comp.RunB
	if len(aShort) > 8 {
		aShort = aShort[:8]
	}
	if len(bShort) > 8 {
		bShort = bShort[:8]
	}

	filename := fmt.Sprintf("%s_vs_%s.json", aShort, bShort)
	path := filepath.Join(c.baseDir, "locomo", "comparisons", filename)

	data, err := json.MarshalIndent(comp, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal comparison: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write comparison file: %w", err)
	}

	return path, nil
}

// UpdateBest updates the best run marker
func (c *Catalog) UpdateBest(results *RunResults) error {
	if err := c.EnsureDirectories(); err != nil {
		return err
	}

	run := &CatalogedRun{
		RunID:       results.RunID,
		Timestamp:   results.StartedAt,
		Git:         results.Git,
		Config:      results.Config,
		Overall:     results.Overall,
		ByCategory:  results.ByCategory,
		DurationSec: results.DurationSecs,
		IsBest:      true,
	}

	path := filepath.Join(c.baseDir, "locomo", "best", "current_best.json")

	data, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal best run: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write best run file: %w", err)
	}

	return nil
}

// GetBest returns the current best run
func (c *Catalog) GetBest() (*CatalogedRun, error) {
	path := filepath.Join(c.baseDir, "locomo", "best", "current_best.json")

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read best run: %w", err)
	}

	var run CatalogedRun
	if err := json.Unmarshal(data, &run); err != nil {
		return nil, fmt.Errorf("failed to parse best run: %w", err)
	}

	return &run, nil
}

// ListRuns returns all cataloged runs
func (c *Catalog) ListRuns(limit int) ([]*CatalogedRun, error) {
	runsDir := filepath.Join(c.baseDir, "locomo", "runs")

	entries, err := os.ReadDir(runsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read runs directory: %w", err)
	}

	// Sort by name (which includes timestamp) descending
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() > entries[j].Name()
	})

	runs := make([]*CatalogedRun, 0)
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(runsDir, entry.Name()))
		if err != nil {
			continue
		}

		var run CatalogedRun
		if err := json.Unmarshal(data, &run); err != nil {
			continue
		}

		runs = append(runs, &run)

		if limit > 0 && len(runs) >= limit {
			break
		}
	}

	return runs, nil
}

// GetRunByCommit finds a run by git commit hash
func (c *Catalog) GetRunByCommit(commitHash string) (*CatalogedRun, error) {
	runs, err := c.ListRuns(0)
	if err != nil {
		return nil, err
	}

	for _, run := range runs {
		if strings.HasPrefix(run.Git.CommitHash, commitHash) ||
			strings.HasPrefix(run.Git.ShortHash, commitHash) {
			return run, nil
		}
	}

	return nil, nil
}

// CleanOldRuns removes runs older than the specified number of days
func (c *Catalog) CleanOldRuns(daysToKeep int) (int, error) {
	runsDir := filepath.Join(c.baseDir, "locomo", "runs")
	cutoff := time.Now().AddDate(0, 0, -daysToKeep)

	entries, err := os.ReadDir(runsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	removed := 0
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			path := filepath.Join(runsDir, entry.Name())
			if err := os.Remove(path); err == nil {
				removed++
			}
		}
	}

	return removed, nil
}
