package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/MycelicMemory/ultrathink/benchmark/locomo"
	"github.com/MycelicMemory/ultrathink/internal/ai"
	"github.com/MycelicMemory/ultrathink/internal/database"
	"github.com/MycelicMemory/ultrathink/internal/search"
	"github.com/MycelicMemory/ultrathink/pkg/config"
)

var (
	benchmarkDataPath string
	benchmarkStrategy string
	benchmarkTopK     int
	benchmarkCategory string
	benchmarkVerbose  bool
	benchmarkQuick    int
)

// benchmarkCmd represents the benchmark command
var benchmarkCmd = &cobra.Command{
	Use:   "benchmark",
	Short: "Run benchmarks to evaluate memory capabilities",
	Long: `Run benchmarks to evaluate Ultrathink's memory retrieval and QA capabilities.

Currently supported benchmarks:
  - locomo: LoCoMo long-term conversational memory benchmark (ACL 2024)

Examples:
  ultrathink benchmark ingest locomo           # Ingest LoCoMo dataset
  ultrathink benchmark run locomo              # Run full evaluation
  ultrathink benchmark run locomo --quick 10   # Quick test with 10 questions
  ultrathink benchmark results locomo          # View results
  ultrathink benchmark status locomo           # Check ingestion status`,
}

// benchmarkIngestCmd represents the benchmark ingest command
var benchmarkIngestCmd = &cobra.Command{
	Use:   "ingest [benchmark]",
	Short: "Ingest benchmark data",
	Long:  `Ingest benchmark data into Ultrathink's memory system.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		benchmarkName := args[0]
		switch benchmarkName {
		case "locomo":
			runLocomoIngest()
		default:
			fmt.Printf("Unknown benchmark: %s\n", benchmarkName)
			fmt.Println("Supported benchmarks: locomo")
			os.Exit(1)
		}
	},
}

// benchmarkRunCmd represents the benchmark run command
var benchmarkRunCmd = &cobra.Command{
	Use:   "run [benchmark]",
	Short: "Run benchmark evaluation",
	Long: `Run benchmark evaluation to test memory retrieval and QA capabilities.

Retrieval Strategies:
  - direct:          Use all memories as context (limited by context size)
  - dialog-rag:      Semantic search over dialogue turns (default)
  - observation-rag: Search over pre-generated observations
  - summary-rag:     Search over session summaries`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		benchmarkName := args[0]
		switch benchmarkName {
		case "locomo":
			runLocomoEval()
		default:
			fmt.Printf("Unknown benchmark: %s\n", benchmarkName)
			fmt.Println("Supported benchmarks: locomo")
			os.Exit(1)
		}
	},
}

// benchmarkResultsCmd represents the benchmark results command
var benchmarkResultsCmd = &cobra.Command{
	Use:   "results [benchmark]",
	Short: "View benchmark results",
	Long:  `View saved benchmark results and generate reports.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		benchmarkName := args[0]
		switch benchmarkName {
		case "locomo":
			runLocomoResults()
		default:
			fmt.Printf("Unknown benchmark: %s\n", benchmarkName)
			os.Exit(1)
		}
	},
}

// benchmarkStatusCmd represents the benchmark status command
var benchmarkStatusCmd = &cobra.Command{
	Use:   "status [benchmark]",
	Short: "Check benchmark ingestion status",
	Long:  `Check if benchmark data has been ingested.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		benchmarkName := args[0]
		switch benchmarkName {
		case "locomo":
			runLocomoStatus()
		default:
			fmt.Printf("Unknown benchmark: %s\n", benchmarkName)
			os.Exit(1)
		}
	},
}

// benchmarkClearCmd represents the benchmark clear command
var benchmarkClearCmd = &cobra.Command{
	Use:   "clear [benchmark]",
	Short: "Clear benchmark data",
	Long:  `Remove all ingested benchmark data from the memory system.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		benchmarkName := args[0]
		switch benchmarkName {
		case "locomo":
			runLocomoClear()
		default:
			fmt.Printf("Unknown benchmark: %s\n", benchmarkName)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(benchmarkCmd)
	benchmarkCmd.AddCommand(benchmarkIngestCmd)
	benchmarkCmd.AddCommand(benchmarkRunCmd)
	benchmarkCmd.AddCommand(benchmarkResultsCmd)
	benchmarkCmd.AddCommand(benchmarkStatusCmd)
	benchmarkCmd.AddCommand(benchmarkClearCmd)

	// Ingest flags
	benchmarkIngestCmd.Flags().StringVar(&benchmarkDataPath, "data-path", "", "Path to benchmark data file (or 'auto' to download)")

	// Run flags
	benchmarkRunCmd.Flags().StringVar(&benchmarkStrategy, "strategy", "dialog-rag", "Retrieval strategy (direct, dialog-rag, observation-rag, summary-rag)")
	benchmarkRunCmd.Flags().IntVar(&benchmarkTopK, "top-k", 10, "Number of memories to retrieve")
	benchmarkRunCmd.Flags().StringVar(&benchmarkCategory, "category", "", "Filter to specific question category")
	benchmarkRunCmd.Flags().BoolVarP(&benchmarkVerbose, "verbose", "v", false, "Enable verbose output")
	benchmarkRunCmd.Flags().IntVar(&benchmarkQuick, "quick", 0, "Quick evaluation with limited questions (0 = full)")
}

func getLocomoComponents() (*database.Database, *search.Engine, *ai.Manager, *locomo.Ingester, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to load config: %w", err)
	}

	db, err := database.Open(cfg.Database.Path)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Initialize schema if needed
	if err := db.InitSchema(); err != nil {
		db.Close()
		return nil, nil, nil, nil, fmt.Errorf("failed to init schema: %w", err)
	}

	// Create search engine
	searchEngine := search.NewEngine(db, cfg)

	// Create AI manager if enabled
	var aiManager *ai.Manager
	if cfg.Ollama.Enabled {
		aiManager = ai.NewManager(db, cfg)
		searchEngine.SetAIManager(aiManager)
	}

	// Create ingester
	ingester := locomo.NewIngester(db)

	return db, searchEngine, aiManager, ingester, nil
}

func runLocomoIngest() {
	fmt.Println("LoCoMo Benchmark - Data Ingestion")
	fmt.Println("==================================")
	fmt.Println()

	db, _, _, ingester, err := getLocomoComponents()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Load dataset
	dataPath := benchmarkDataPath
	if dataPath == "" {
		dataPath = "auto" // Auto-download
	}

	fmt.Printf("Loading dataset from: %s\n", dataPath)
	dataset, err := locomo.LoadDataset(dataPath)
	if err != nil {
		fmt.Printf("Error loading dataset: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Found %d conversations\n\n", len(dataset.Conversations))

	// Ingest
	fmt.Println("Ingesting conversations...")
	result, err := ingester.Ingest(dataset)
	if err != nil {
		fmt.Printf("Error during ingestion: %v\n", err)
		os.Exit(1)
	}

	// Print results
	fmt.Println()
	fmt.Println("Ingestion Complete")
	fmt.Println("==================")
	fmt.Printf("Conversations: %d\n", result.ConversationsIngested)
	fmt.Printf("Dialogue turns: %d\n", result.TotalTurns)
	fmt.Printf("Memories created: %d\n", result.TotalMemories)
	fmt.Printf("Persona memories: %d\n", result.PersonaMemories)
	fmt.Printf("QA questions: %d\n", result.TotalQAQuestions)
	fmt.Printf("Duration: %s\n", result.Duration.Round(100*1e6))
}

func runLocomoEval() {
	fmt.Println("LoCoMo Benchmark - QA Evaluation")
	fmt.Println("=================================")
	fmt.Println()

	db, searchEngine, aiManager, ingester, err := getLocomoComponents()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Check if data is ingested
	status, err := ingester.GetStatus()
	if err != nil {
		fmt.Printf("Error checking status: %v\n", err)
		os.Exit(1)
	}

	if !status.Ingested {
		fmt.Println("Error: No LoCoMo data ingested")
		fmt.Println("Run 'ultrathink benchmark ingest locomo' first")
		os.Exit(1)
	}

	// Load dataset for QA annotations
	dataset, err := locomo.LoadDataset("auto")
	if err != nil {
		fmt.Printf("Error loading dataset: %v\n", err)
		os.Exit(1)
	}

	// Parse strategy
	var strategy locomo.RetrievalStrategy
	switch benchmarkStrategy {
	case "direct":
		strategy = locomo.StrategyDirect
	case "dialog-rag":
		strategy = locomo.StrategyDialogRAG
	case "observation-rag":
		strategy = locomo.StrategyObservationRAG
	case "summary-rag":
		strategy = locomo.StrategySummaryRAG
	default:
		fmt.Printf("Unknown strategy: %s\n", benchmarkStrategy)
		os.Exit(1)
	}

	// Parse category
	var category locomo.QuestionCategory
	if benchmarkCategory != "" {
		category = locomo.QuestionCategory(benchmarkCategory)
	}

	// Create evaluator config
	evalConfig := &locomo.EvaluationConfig{
		Task:              "qa",
		RetrievalStrategy: strategy,
		TopK:              benchmarkTopK,
		Category:          category,
		Verbose:           benchmarkVerbose,
	}

	fmt.Printf("Strategy: %s\n", strategy)
	fmt.Printf("Top-K: %d\n", benchmarkTopK)
	if category != "" {
		fmt.Printf("Category: %s\n", category)
	}
	fmt.Println()

	// Create evaluator
	evaluator, err := locomo.NewQAEvaluator(db, searchEngine, aiManager, ingester, evalConfig)
	if err != nil {
		fmt.Printf("Error creating evaluator: %v\n", err)
		os.Exit(1)
	}

	// Run evaluation
	var results *locomo.BenchmarkResults
	if benchmarkQuick > 0 {
		fmt.Printf("Running quick evaluation (%d questions)...\n\n", benchmarkQuick)
		results, err = evaluator.QuickEval(dataset, benchmarkQuick)
	} else {
		fmt.Println("Running full evaluation...")
		fmt.Println()
		results, err = evaluator.Evaluate(dataset)
	}

	if err != nil {
		fmt.Printf("Error during evaluation: %v\n", err)
		os.Exit(1)
	}

	// Print results
	locomo.PrintResults(results)

	// Save results
	cfg, _ := config.Load()
	resultsDir := filepath.Join(filepath.Dir(cfg.Database.Path), "benchmark_results")
	store := locomo.NewResultsStore(resultsDir)
	path, err := store.Save(results)
	if err != nil {
		fmt.Printf("Warning: Failed to save results: %v\n", err)
	} else {
		fmt.Printf("\nResults saved to: %s\n", path)
	}
}

func runLocomoResults() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	resultsDir := filepath.Join(filepath.Dir(cfg.Database.Path), "benchmark_results")
	store := locomo.NewResultsStore(resultsDir)

	summaries, err := store.List()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if len(summaries) == 0 {
		fmt.Println("No benchmark results found")
		fmt.Println("Run 'ultrathink benchmark run locomo' to generate results")
		return
	}

	fmt.Println("LoCoMo Benchmark Results")
	fmt.Println("========================")
	fmt.Println()
	fmt.Println("Date                Strategy      Model                F1      Questions")
	fmt.Println("----                --------      -----                --      ---------")

	for _, s := range summaries {
		model := s.Model
		if len(model) > 20 {
			model = model[:17] + "..."
		}
		fmt.Printf("%-19s %-13s %-20s %5.2f   %d\n",
			s.Timestamp.Format("2006-01-02 15:04"),
			s.Strategy,
			model,
			s.F1,
			s.Questions)
	}

	// Show latest result details
	fmt.Println()
	latest, err := store.GetLatest()
	if err == nil {
		gen := locomo.NewReportGenerator()
		fmt.Println("Latest Result Summary")
		fmt.Println("---------------------")
		fmt.Println(gen.GenerateSummary(latest))
	}
}

func runLocomoStatus() {
	db, _, _, ingester, err := getLocomoComponents()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	status, err := ingester.GetStatus()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("LoCoMo Benchmark Status")
	fmt.Println("=======================")
	fmt.Println()

	if status.Ingested {
		fmt.Println("Status: ✅ Data Ingested")
		fmt.Printf("Conversations: %d\n", status.ConversationCount)
		fmt.Printf("Memories: %d\n", status.MemoryCount)
	} else {
		fmt.Println("Status: ❌ No Data")
		fmt.Println()
		fmt.Println("Run 'ultrathink benchmark ingest locomo' to ingest the dataset")
	}
}

func runLocomoClear() {
	db, _, _, ingester, err := getLocomoComponents()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	fmt.Println("Clearing LoCoMo benchmark data...")

	if err := ingester.ClearBenchmarkData(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Benchmark data cleared successfully")
}
