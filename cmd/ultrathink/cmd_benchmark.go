package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/MycelicMemory/ultrathink/internal/benchmark"
	"github.com/MycelicMemory/ultrathink/internal/database"
	"github.com/MycelicMemory/ultrathink/pkg/config"
)

var (
	benchmarkQuestionType string
	benchmarkVerbose      bool
	benchmarkQuick        int
	benchmarkChangeDesc   string
)

// benchmarkCmd represents the benchmark command
var benchmarkCmd = &cobra.Command{
	Use:   "benchmark",
	Short: "Run and manage LoCoMo benchmark evaluations",
	Long: `Run benchmarks to evaluate Ultrathink's memory retrieval and QA capabilities.

The benchmark uses the LoCoMo-MC10 dataset with ~2000 questions across categories:
  - Single-Hop:   Direct fact retrieval
  - Multi-Hop:    Connecting multiple pieces of information
  - Temporal:     Time-based reasoning
  - Open-Domain:  External knowledge requirements

IMPORTANT: The Python bridge server must be running for benchmarks.
Start it with: make server  (in benchmark/locomo/ directory)

Examples:
  ultrathink benchmark run --quick 20       # Quick test with 20 questions
  ultrathink benchmark run                  # Full benchmark (all questions)
  ultrathink benchmark status               # Check recent runs and status
  ultrathink benchmark results              # View historical results
  ultrathink benchmark compare <id1> <id2>  # Compare two runs`,
}

// benchmarkRunCmd represents the benchmark run command
var benchmarkRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Execute a benchmark evaluation",
	Long: `Run a LoCoMo benchmark evaluation.

The benchmark connects to the Python bridge server (port 9876) which handles:
- Loading questions from HuggingFace
- Generating answers using DeepSeek
- Evaluating with LLM judge, F1, and BLEU-1 metrics

Results are stored in the database and cataloged to ~/.ultrathink/benchmark_results/`,
	Run: func(cmd *cobra.Command, args []string) {
		runBenchmark()
	},
}

// benchmarkStatusCmd represents the benchmark status command
var benchmarkStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check benchmark status and recent runs",
	Run: func(cmd *cobra.Command, args []string) {
		showBenchmarkStatus()
	},
}

// benchmarkResultsCmd represents the benchmark results command
var benchmarkResultsCmd = &cobra.Command{
	Use:   "results",
	Short: "View historical benchmark results",
	Run: func(cmd *cobra.Command, args []string) {
		showBenchmarkResults()
	},
}

// benchmarkCompareCmd represents the benchmark compare command
var benchmarkCompareCmd = &cobra.Command{
	Use:   "compare <run_id_a> <run_id_b>",
	Short: "Compare two benchmark runs",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		compareBenchmarkRuns(args[0], args[1])
	},
}

func init() {
	rootCmd.AddCommand(benchmarkCmd)
	benchmarkCmd.AddCommand(benchmarkRunCmd)
	benchmarkCmd.AddCommand(benchmarkStatusCmd)
	benchmarkCmd.AddCommand(benchmarkResultsCmd)
	benchmarkCmd.AddCommand(benchmarkCompareCmd)

	// Run flags
	benchmarkRunCmd.Flags().IntVar(&benchmarkQuick, "quick", 20, "Number of questions to evaluate (0 = all)")
	benchmarkRunCmd.Flags().BoolVarP(&benchmarkVerbose, "verbose", "v", false, "Enable verbose output")
	benchmarkRunCmd.Flags().StringVar(&benchmarkChangeDesc, "desc", "", "Description of changes being tested")
}

func getBenchmarkService() (*benchmark.Service, *database.Database, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load config: %w", err)
	}

	db, err := database.Open(cfg.Database.Path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.InitSchema(); err != nil {
		db.Close()
		return nil, nil, fmt.Errorf("failed to init schema: %w", err)
	}

	// Get repo path from environment or default
	repoPath := os.Getenv("ULTRATHINK_REPO_PATH")

	svc := benchmark.NewService(db, repoPath)
	return svc, db, nil
}

func runBenchmark() {
	fmt.Println("LoCoMo-MC10 Benchmark")
	fmt.Println("=====================")
	fmt.Println()

	svc, db, err := getBenchmarkService()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Check if bridge is available
	if err := svc.CheckBridge(); err != nil {
		fmt.Println("Error: Python benchmark bridge is not running!")
		fmt.Println()
		fmt.Println("To start the bridge server:")
		fmt.Println("  cd benchmark/locomo/")
		fmt.Println("  make server")
		fmt.Println()
		fmt.Println("Then re-run this command.")
		os.Exit(1)
	}

	// Build config
	runConfig := &benchmark.RunConfig{
		BenchmarkType: "locomo",
		MaxQuestions:  benchmarkQuick,
		Verbose:       benchmarkVerbose,
		ChangeDesc:    benchmarkChangeDesc,
	}

	if benchmarkQuick == 0 {
		fmt.Println("Running full benchmark (all questions)...")
	} else {
		fmt.Printf("Running quick benchmark (%d questions)...\n", benchmarkQuick)
	}

	if benchmarkChangeDesc != "" {
		fmt.Printf("Change: %s\n", benchmarkChangeDesc)
	}
	fmt.Println()

	// Run benchmark
	ctx := context.Background()
	startTime := time.Now()

	results, err := svc.Run(ctx, runConfig)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	elapsed := time.Since(startTime)

	// Display results
	fmt.Println()
	fmt.Println("Results")
	fmt.Println("-------")
	fmt.Printf("Run ID: %s\n", results.RunID[:8])
	fmt.Printf("Git: %s (%s)%s\n", results.Git.ShortHash, results.Git.Branch, dirtyIndicator(results.Git.Dirty))
	fmt.Printf("Duration: %s\n", elapsed.Round(time.Second))
	fmt.Println()

	fmt.Println("Overall Scores:")
	fmt.Printf("  LLM Judge Accuracy: %.1f%%\n", results.Overall.LLMJudgeAccuracy)
	fmt.Printf("  F1 Score: %.4f\n", results.Overall.F1Score)
	fmt.Printf("  BLEU-1 Score: %.4f\n", results.Overall.BLEU1Score)
	fmt.Printf("  Questions: %d\n", results.Overall.TotalQuestions)
	fmt.Println()

	if len(results.ByCategory) > 0 {
		fmt.Println("By Category:")
		for cat, scores := range results.ByCategory {
			fmt.Printf("  %s: %.1f%% (%d questions)\n", cat, scores.LLMJudgeAccuracy, scores.TotalQuestions)
		}
	}

	fmt.Println()
	fmt.Printf("Results saved to database and cataloged.\n")
}

func showBenchmarkStatus() {
	svc, db, err := getBenchmarkService()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	fmt.Println("Benchmark Status")
	fmt.Println("================")
	fmt.Println()

	// Check bridge
	if err := svc.CheckBridge(); err != nil {
		fmt.Println("Bridge: Not running")
	} else {
		fmt.Println("Bridge: Running (localhost:9876)")
	}
	fmt.Println()

	// Check for active run
	if svc.IsRunning() {
		ctx := context.Background()
		progress, err := svc.GetProgress(ctx)
		if err == nil {
			fmt.Println("Active Run:")
			fmt.Printf("  ID: %s\n", progress.RunID[:8])
			fmt.Printf("  Progress: %d/%d (%.1f%%)\n", progress.CompletedCount, progress.TotalQuestions, progress.PercentComplete)
			fmt.Printf("  Elapsed: %.0fs\n", progress.ElapsedSecs)
		}
		fmt.Println()
	}

	// Show recent runs
	runs, err := svc.ListRuns(&database.BenchmarkRunFilters{Limit: 5})
	if err != nil {
		fmt.Printf("Error listing runs: %v\n", err)
		return
	}

	if len(runs) == 0 {
		fmt.Println("No benchmark runs found.")
		fmt.Println("Run 'ultrathink benchmark run --quick 20' to start.")
		return
	}

	fmt.Println("Recent Runs:")
	fmt.Println("  ID        Date                 Accuracy  Questions  Status")
	fmt.Println("  --        ----                 --------  ---------  ------")

	for _, run := range runs {
		accuracy := "-"
		if run.OverallScore != nil {
			accuracy = fmt.Sprintf("%.1f%%", *run.OverallScore)
		}
		questions := "-"
		if run.TotalQuestions != nil {
			questions = fmt.Sprintf("%d", *run.TotalQuestions)
		}
		best := ""
		if run.IsBestRun {
			best = " (best)"
		}
		fmt.Printf("  %s  %s  %8s  %9s  %s%s\n",
			run.ID[:8],
			run.StartedAt.Format("2006-01-02 15:04"),
			accuracy,
			questions,
			run.Status,
			best)
	}
}

func showBenchmarkResults() {
	svc, db, err := getBenchmarkService()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	fmt.Println("Benchmark Results")
	fmt.Println("=================")
	fmt.Println()

	// Show best run
	best, err := svc.GetBestRun("locomo")
	if err == nil && best != nil {
		fmt.Println("Best Run:")
		fmt.Printf("  ID: %s\n", best.ID[:8])
		fmt.Printf("  Date: %s\n", best.StartedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("  Git: %s (%s)\n", best.GitCommitHash[:7], best.GitBranch)
		if best.OverallScore != nil {
			fmt.Printf("  Accuracy: %.1f%%\n", *best.OverallScore)
		}
		if best.TotalQuestions != nil && best.TotalCorrect != nil {
			fmt.Printf("  Questions: %d/%d correct\n", *best.TotalCorrect, *best.TotalQuestions)
		}
		fmt.Println()
	}

	// Show all runs
	runs, err := svc.ListRuns(&database.BenchmarkRunFilters{Limit: 20})
	if err != nil {
		fmt.Printf("Error listing runs: %v\n", err)
		return
	}

	if len(runs) == 0 {
		fmt.Println("No benchmark runs found.")
		return
	}

	fmt.Println("All Runs:")
	fmt.Println("  ID        Git      Date                 Accuracy  F1       Duration")
	fmt.Println("  --        ---      ----                 --------  --       --------")

	for _, run := range runs {
		accuracy := "-"
		if run.OverallScore != nil {
			accuracy = fmt.Sprintf("%.1f%%", *run.OverallScore)
		}
		f1 := "-"
		if run.OverallF1 != nil {
			f1 = fmt.Sprintf("%.4f", *run.OverallF1)
		}
		duration := "-"
		if run.DurationSeconds != nil {
			duration = fmt.Sprintf("%.0fs", *run.DurationSeconds)
		}
		best := ""
		if run.IsBestRun {
			best = " *"
		}
		fmt.Printf("  %s  %s  %s  %8s  %6s  %8s%s\n",
			run.ID[:8],
			run.GitCommitHash[:7],
			run.StartedAt.Format("2006-01-02 15:04"),
			accuracy,
			f1,
			duration,
			best)
	}
	fmt.Println()
	fmt.Println("  * = best run")
}

func compareBenchmarkRuns(runIDA, runIDB string) {
	svc, db, err := getBenchmarkService()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	comparison, err := svc.Compare(runIDA, runIDB)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Benchmark Comparison")
	fmt.Println("====================")
	fmt.Println()

	fmt.Printf("Run A (baseline): %s\n", runIDA[:8])
	fmt.Printf("Run B (comparison): %s\n", runIDB[:8])
	fmt.Println()

	// Overall
	fmt.Println("Overall:")
	symbol := "→"
	if comparison.OverallDiff.Improved {
		symbol = "↑"
	} else if comparison.OverallDiff.Diff < 0 {
		symbol = "↓"
	}
	fmt.Printf("  Accuracy: %.1f%% %s %.1f%% (%+.1f%%)\n",
		comparison.OverallDiff.Before,
		symbol,
		comparison.OverallDiff.After,
		comparison.OverallDiff.Diff)
	fmt.Println()

	// Improvements
	if len(comparison.Improvements) > 0 {
		fmt.Println("Improvements:")
		for _, imp := range comparison.Improvements {
			fmt.Printf("  ✓ %s\n", imp)
		}
		fmt.Println()
	}

	// Regressions
	if len(comparison.Regressions) > 0 {
		fmt.Println("Regressions:")
		for _, reg := range comparison.Regressions {
			fmt.Printf("  ✗ %s\n", reg)
		}
		fmt.Println()
	}

	// Category breakdown
	if len(comparison.CategoryDiffs) > 0 {
		fmt.Println("By Category:")
		for cat, diff := range comparison.CategoryDiffs {
			symbol := "→"
			if diff.Improved {
				symbol = "↑"
			} else if diff.Diff < 0 {
				symbol = "↓"
			}
			fmt.Printf("  %s: %.1f%% %s %.1f%% (%+.1f%%)\n",
				cat, diff.Before, symbol, diff.After, diff.Diff)
		}
	}
}

func dirtyIndicator(dirty bool) string {
	if dirty {
		return " [dirty]"
	}
	return ""
}
