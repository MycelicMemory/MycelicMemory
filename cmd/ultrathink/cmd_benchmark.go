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
	benchmarkQuestionType   string
	benchmarkTopK           int
	benchmarkVerbose        bool
	benchmarkQuick          int
	benchmarkUseSummaries   bool
)

// benchmarkCmd represents the benchmark command
var benchmarkCmd = &cobra.Command{
	Use:   "benchmark",
	Short: "Run benchmarks to evaluate memory capabilities",
	Long: `Run benchmarks to evaluate Ultrathink's memory retrieval and QA capabilities.

Currently supported benchmarks:
  - locomo: LoCoMo-MC10 long-term conversational memory benchmark

The benchmark evaluates 5 types of questions:
  - single_hop:    Direct fact retrieval
  - multi_hop:     Connecting multiple pieces of information
  - temporal:      Time-based reasoning
  - open_domain:   External knowledge requirements
  - adversarial:   Challenging/tricky questions

Examples:
  ultrathink benchmark run locomo              # Run full evaluation
  ultrathink benchmark run locomo --quick 20   # Quick test with 20 questions
  ultrathink benchmark run locomo --type single_hop  # Test single-hop only
  ultrathink benchmark results locomo          # View results
  ultrathink benchmark status locomo           # Check status`,
}

// benchmarkRunCmd represents the benchmark run command
var benchmarkRunCmd = &cobra.Command{
	Use:   "run [benchmark]",
	Short: "Run benchmark evaluation",
	Long: `Run benchmark evaluation to test memory retrieval and QA capabilities.

Question Types:
  - single_hop:    Direct fact retrieval from conversations
  - multi_hop:     Connecting multiple pieces of information
  - temporal:      Understanding time-based relationships
  - open_domain:   External knowledge requirements
  - adversarial:   Challenging/tricky questions`,
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
	Short: "Check benchmark status",
	Long:  `Check the status of benchmark data and previous runs.`,
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
	benchmarkCmd.AddCommand(benchmarkRunCmd)
	benchmarkCmd.AddCommand(benchmarkResultsCmd)
	benchmarkCmd.AddCommand(benchmarkStatusCmd)
	benchmarkCmd.AddCommand(benchmarkClearCmd)

	// Run flags
	benchmarkRunCmd.Flags().StringVar(&benchmarkQuestionType, "type", "", "Filter to specific question type (single_hop, multi_hop, temporal, open_domain, adversarial)")
	benchmarkRunCmd.Flags().IntVar(&benchmarkTopK, "top-k", 10, "Number of memories to retrieve for context")
	benchmarkRunCmd.Flags().BoolVarP(&benchmarkVerbose, "verbose", "v", false, "Enable verbose output")
	benchmarkRunCmd.Flags().IntVar(&benchmarkQuick, "quick", 0, "Quick evaluation with limited questions (0 = full)")
	benchmarkRunCmd.Flags().BoolVar(&benchmarkUseSummaries, "summaries", false, "Use session summaries instead of full dialogues")
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

func runLocomoEval() {
	fmt.Println("LoCoMo-MC10 Benchmark - Evaluation")
	fmt.Println("===================================")
	fmt.Println()

	db, searchEngine, aiManager, ingester, err := getLocomoComponents()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	if aiManager == nil {
		fmt.Println("Error: AI/Ollama must be enabled for benchmark evaluation")
		fmt.Println("Configure Ollama in your config file")
		os.Exit(1)
	}

	// Parse question type filter
	var qType locomo.QuestionType
	if benchmarkQuestionType != "" {
		switch benchmarkQuestionType {
		case "single_hop", "sh":
			qType = locomo.TypeSingleHop
		case "multi_hop", "mh":
			qType = locomo.TypeMultiHop
		case "temporal", "tr":
			qType = locomo.TypeTemporal
		case "open_domain", "od":
			qType = locomo.TypeOpenDomain
		case "adversarial", "adv":
			qType = locomo.TypeAdversarial
		default:
			fmt.Printf("Unknown question type: %s\n", benchmarkQuestionType)
			fmt.Println("Valid types: single_hop, multi_hop, temporal, open_domain, adversarial")
			os.Exit(1)
		}
	}

	// Create evaluator config
	evalConfig := &locomo.EvaluationConfig{
		QuestionType:        qType,
		TopK:                benchmarkTopK,
		Verbose:             benchmarkVerbose,
		UseSessionSummaries: benchmarkUseSummaries,
	}

	fmt.Printf("Top-K: %d\n", benchmarkTopK)
	fmt.Printf("Use Summaries: %t\n", benchmarkUseSummaries)
	if qType != "" {
		fmt.Printf("Question Type: %s\n", qType)
	}
	fmt.Println()

	// Load dataset
	maxQuestions := 0 // Load all
	if benchmarkQuick > 0 {
		maxQuestions = benchmarkQuick
		fmt.Printf("Loading %d questions (quick mode)...\n", maxQuestions)
	} else {
		fmt.Println("Loading full dataset from HuggingFace...")
	}

	dataset, err := locomo.LoadDataset(maxQuestions)
	if err != nil {
		fmt.Printf("Error loading dataset: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Loaded %d questions\n\n", len(dataset.Questions))

	// Create evaluator
	evaluator, err := locomo.NewMCEvaluator(db, searchEngine, aiManager, ingester, evalConfig)
	if err != nil {
		fmt.Printf("Error creating evaluator: %v\n", err)
		os.Exit(1)
	}

	// Run evaluation
	fmt.Println("Running evaluation...")
	fmt.Println()

	results, err := evaluator.Evaluate(dataset)
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

	fmt.Println("LoCoMo-MC10 Benchmark Results")
	fmt.Println("=============================")
	fmt.Println()
	fmt.Println("Date                Model                Accuracy  Questions  Duration")
	fmt.Println("----                -----                --------  ---------  --------")

	for _, s := range summaries {
		model := s.Model
		if len(model) > 20 {
			model = model[:17] + "..."
		}
		fmt.Printf("%-19s %-20s %6.1f%%   %4d/%4d  %s\n",
			s.Timestamp.Format("2006-01-02 15:04"),
			model,
			s.Accuracy,
			s.Correct,
			s.Total,
			s.Duration.Round(1e9))
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

	fmt.Println("LoCoMo-MC10 Benchmark Status")
	fmt.Println("============================")
	fmt.Println()

	if status.Ingested {
		fmt.Println("Status: Data Ingested")
		fmt.Printf("Questions: %d\n", status.QuestionCount)
		fmt.Printf("Memories: %d\n", status.MemoryCount)
	} else {
		fmt.Println("Status: No ingested data")
		fmt.Println()
		fmt.Println("Note: The LoCoMo-MC10 benchmark loads data directly from HuggingFace")
		fmt.Println("Run 'ultrathink benchmark run locomo' to start evaluation")
	}
}

func runLocomoClear() {
	db, _, _, ingester, err := getLocomoComponents()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	fmt.Println("Clearing LoCoMo-MC10 benchmark data...")

	if err := ingester.ClearBenchmarkData(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Benchmark data cleared successfully")
}
