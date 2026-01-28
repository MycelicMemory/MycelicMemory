package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/MycelicMemory/mycelicmemory/internal/database"
	"github.com/MycelicMemory/mycelicmemory/internal/dependencies"
	"github.com/MycelicMemory/mycelicmemory/pkg/config"
)

// doctorCmd represents the doctor command
var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Comprehensive system check",
	Long:  `Run a comprehensive system check to verify all components are working correctly.`,
	Run: func(cmd *cobra.Command, args []string) {
		runDoctor()
	},
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

func runDoctor() {
	fmt.Println("MycelicMemory System Check")
	fmt.Println("==========================")
	fmt.Println()

	allOk := true
	hasWarnings := false

	// Check configuration
	fmt.Print("Configuration... ")
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		allOk = false
	} else {
		fmt.Println("OK")
	}

	// Check database
	fmt.Print("Database... ")
	if cfg != nil {
		if _, err := os.Stat(cfg.Database.Path); os.IsNotExist(err) {
			fmt.Println("NOT INITIALIZED (will be created on first use)")
		} else {
			db, err := database.Open(cfg.Database.Path)
			if err != nil {
				fmt.Printf("ERROR: %v\n", err)
				allOk = false
			} else {
				stats, err := db.GetStats()
				if err != nil {
					fmt.Printf("ERROR: %v\n", err)
					allOk = false
				} else {
					fmt.Printf("OK (%d memories, %d sessions)\n", stats.MemoryCount, stats.SessionCount)
				}
				db.Close()
			}
		}
		fmt.Printf("  Path: %s\n", cfg.Database.Path)
	}
	fmt.Println()

	// Check optional dependencies (Ollama and Qdrant)
	if cfg != nil {
		depResult := dependencies.Check(cfg)

		// Print detailed report
		fmt.Print(dependencies.FormatDoctorReport(depResult, cfg))

		// Track warnings
		if depResult.Ollama.Status != dependencies.StatusAvailable || len(depResult.Ollama.MissingItems) > 0 {
			hasWarnings = true
		}
		if cfg.Qdrant.Enabled && depResult.Qdrant.Status != dependencies.StatusAvailable {
			hasWarnings = true
		}
	}

	fmt.Println()

	// Summary
	if allOk && !hasWarnings {
		fmt.Println("✅ All systems operational!")
	} else if allOk && hasWarnings {
		fmt.Println("⚠️  Core systems operational with optional features unavailable.")
		fmt.Println("   MyclicMemory will work but some AI features are disabled.")
	} else {
		fmt.Println("❌ Some issues detected. Please review the errors above.")
	}

	// Print configuration details
	fmt.Println()
	fmt.Println("Configuration:")
	if cfg != nil {
		fmt.Printf("  Config Dir: %s\n", config.ConfigPath())
		fmt.Printf("  REST API: %s:%d (enabled: %v)\n", cfg.RestAPI.Host, cfg.RestAPI.Port, cfg.RestAPI.Enabled)
	}

	// Feature availability summary
	if cfg != nil {
		fmt.Println()
		fmt.Println("Feature Availability:")
		depResult := dependencies.Check(cfg)

		if depResult.AIFeaturesAvailable() {
			fmt.Println("  ✅ AI Analysis (analyze, categorize)")
		} else {
			fmt.Println("  ❌ AI Analysis (analyze, categorize) - requires Ollama with models")
		}

		if depResult.SemanticSearchAvailable() {
			fmt.Println("  ✅ Semantic Search (AI-powered search)")
		} else {
			fmt.Println("  ❌ Semantic Search - requires Ollama + Qdrant")
		}

		fmt.Println("  ✅ Basic Search (keyword matching)")
		fmt.Println("  ✅ Memory Storage (remember, get, list)")
	}
}
