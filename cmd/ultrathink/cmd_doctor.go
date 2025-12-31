package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/MycelicMemory/ultrathink/internal/ai"
	"github.com/MycelicMemory/ultrathink/internal/database"
	"github.com/MycelicMemory/ultrathink/pkg/config"
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
	fmt.Println("Ultrathink System Check")
	fmt.Println("=======================")
	fmt.Println()

	allOk := true

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

	// Check AI services
	fmt.Print("Ollama... ")
	if cfg != nil {
		db, _ := database.Open(cfg.Database.Path)
		if db != nil {
			aiManager := ai.NewManager(db, cfg)
			status := aiManager.GetStatus()
			if status.OllamaAvailable {
				fmt.Println("OK")
				fmt.Printf("  URL: %s\n", cfg.Ollama.BaseURL)
				fmt.Printf("  Chat Model: %s\n", cfg.Ollama.ChatModel)
				fmt.Printf("  Embedding Model: %s\n", cfg.Ollama.EmbeddingModel)
			} else {
				fmt.Println("NOT AVAILABLE")
				fmt.Println("  AI features will be disabled.")
				fmt.Println("  Install Ollama: https://ollama.ai")
			}
			db.Close()
		}
	}

	// Check Qdrant (optional)
	fmt.Print("Qdrant... ")
	if cfg != nil && cfg.Qdrant.URL != "" {
		fmt.Println("NOT CHECKED (optional)")
		fmt.Printf("  URL: %s\n", cfg.Qdrant.URL)
	} else {
		fmt.Println("NOT CONFIGURED (optional)")
	}

	fmt.Println()

	// Summary
	if allOk {
		fmt.Println("All core systems operational!")
	} else {
		fmt.Println("Some issues detected. Please review the errors above.")
	}

	// Print configuration details
	fmt.Println()
	fmt.Println("Configuration:")
	if cfg != nil {
		fmt.Printf("  Config Dir: %s\n", config.ConfigPath())
		fmt.Printf("  REST API: %s:%d (enabled: %v)\n", cfg.RestAPI.Host, cfg.RestAPI.Port, cfg.RestAPI.Enabled)
	}
}
