package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/MycelicMemory/mycelicmemory/internal/ai"
)

var (
	analyzeType      string
	analyzeTimeframe string
	analyzeLimit     int
	analyzeDomain    string
)

// analyzeCmd represents the analyze command
var analyzeCmd = &cobra.Command{
	Use:   "analyze <question-or-query>",
	Short: "AI-powered memory analysis",
	Long: `Analyze memories using AI to answer questions, summarize, or find patterns.

Analysis Types:
  question  - Answer a question based on memories (default)
  summarize - Summarize memories over a timeframe
  patterns  - Find patterns in memories
  temporal  - Analyze learning progression over time

Examples:
  mycelicmemory analyze "What have I learned about Go?"
  mycelicmemory analyze --type summarize --timeframe week
  mycelicmemory analyze "concurrency" --type patterns
  mycelicmemory analyze --type temporal --timeframe month`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		query := ""
		if len(args) > 0 {
			query = args[0]
		}
		runAnalyze(query)
	},
}

func init() {
	rootCmd.AddCommand(analyzeCmd)

	analyzeCmd.Flags().StringVarP(&analyzeType, "type", "t", "question", "Analysis type (question, summarize, patterns, temporal)")
	analyzeCmd.Flags().StringVar(&analyzeTimeframe, "timeframe", "all", "Timeframe (today, week, month, all)")
	analyzeCmd.Flags().IntVarP(&analyzeLimit, "limit", "l", 10, "Maximum memories to analyze")
	analyzeCmd.Flags().StringVarP(&analyzeDomain, "domain", "d", "", "Filter by domain")
}

func runAnalyze(query string) {
	db, cfg, err := getDB()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Check if AI is configured
	aiManager := ai.NewManager(db, cfg)
	status := aiManager.GetStatus()

	if !status.OllamaAvailable {
		fmt.Println("AI Analysis")
		fmt.Println("===========")
		fmt.Println()
		fmt.Println("Error: Ollama is not available.")
		fmt.Println()
		fmt.Println("To use AI analysis features, please:")
		fmt.Println("1. Install Ollama: https://ollama.ai")
		fmt.Println("2. Start Ollama: ollama serve")
		fmt.Println("3. Pull required models:")
		fmt.Printf("   ollama pull %s\n", cfg.Ollama.ChatModel)
		fmt.Printf("   ollama pull %s\n", cfg.Ollama.EmbeddingModel)
		fmt.Println()
		fmt.Println("Run 'mycelicmemory doctor' to check system status.")
		os.Exit(1)
	}

	// Validate analysis type
	validTypes := map[string]bool{
		"question":  true,
		"summarize": true,
		"patterns":  true,
		"temporal":  true,
	}
	if !validTypes[analyzeType] {
		fmt.Printf("Error: Invalid analysis type '%s'\n", analyzeType)
		fmt.Println("Valid types: question, summarize, patterns, temporal")
		os.Exit(1)
	}

	// For question type, query is required
	if analyzeType == "question" && query == "" {
		fmt.Println("Error: Question is required for 'question' analysis type")
		fmt.Println("Example: mycelicmemory analyze \"What have I learned about Go?\"")
		os.Exit(1)
	}

	ctx := context.Background()

	opts := &ai.AnalysisOptions{
		Type:      analyzeType,
		Question:  query,
		Query:     query,
		Timeframe: analyzeTimeframe,
		Limit:     analyzeLimit,
		Domain:    analyzeDomain,
	}

	startTime := time.Now()

	result, err := aiManager.Analyze(ctx, opts)
	if err != nil {
		fmt.Printf("Error analyzing: %v\n", err)
		os.Exit(1)
	}

	elapsed := time.Since(startTime)

	// Display results based on type
	switch analyzeType {
	case "question":
		fmt.Println("📊 **Analysis Result** (concise format)")
		fmt.Printf("**Question**: %s\n", query)
		fmt.Printf("**Answer**: %s\n", result.Answer)
		fmt.Printf("**Memories Analyzed**: %d\n", result.MemoryCount)
		fmt.Printf("**Confidence**: %.0f%%\n", result.Confidence*100)

	case "summarize":
		fmt.Println("📊 **Memory Summary** (concise format)")
		fmt.Printf("**Summary**: %s\n", result.Summary)
		fmt.Printf("**Memories Analyzed**: %d\n", result.MemoryCount)
		if len(result.KeyThemes) > 0 {
			fmt.Printf("**Key Themes**: %s\n", strings.Join(result.KeyThemes, ", "))
		}

	case "patterns":
		fmt.Println("📊 **Pattern Analysis** (concise format)")
		if len(result.Patterns) > 0 {
			for _, p := range result.Patterns {
				fmt.Printf("**Pattern**: %s - %s\n", p.Name, p.Description)
			}
		} else {
			fmt.Println("No significant patterns found.")
		}
		fmt.Printf("**Memories Analyzed**: %d\n", result.MemoryCount)

	case "temporal":
		fmt.Println("📊 **Temporal Analysis** (concise format)")
		fmt.Printf("**Summary**: %s\n", result.Summary)
		if len(result.Insights) > 0 {
			fmt.Printf("**Insights**: %s\n", strings.Join(result.Insights, "; "))
		}
		fmt.Printf("**Memories Analyzed**: %d\n", result.MemoryCount)
	}

	fmt.Printf("\n⏱️  Analysis completed in %s\n", elapsed)
}

func truncateStr(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
