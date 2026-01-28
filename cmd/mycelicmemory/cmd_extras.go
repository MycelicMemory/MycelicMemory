package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/MycelicMemory/mycelicmemory/internal/ai"
	"github.com/MycelicMemory/mycelicmemory/internal/dependencies"
	"github.com/MycelicMemory/mycelicmemory/pkg/config"
)

var (
	categorizeAutoCreate         bool
	categorizeConfidenceThreshold float64
)

// categorizeCmd represents the categorize command
var categorizeCmd = &cobra.Command{
	Use:   "categorize <memory-id>",
	Short: "Categorize a memory using AI",
	Long: `Categorize a memory using AI analysis.

Examples:
  mycelicmemory categorize 550e8400-e29b-41d4-a716-446655440000
  mycelicmemory categorize <id> --auto-create
  mycelicmemory categorize <id> --confidence 0.8`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		runCategorize(args[0])
	},
}

// setupCmd represents the setup command
var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Run setup wizard",
	Long:  `Run the setup wizard to configure MyclicMemory.`,
	Run: func(cmd *cobra.Command, args []string) {
		runSetup()
	},
}

// validateCmd represents the validate command
var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate installation",
	Long:  `Validate the MyclicMemory installation and configuration.`,
	Run: func(cmd *cobra.Command, args []string) {
		runValidate()
	},
}

// installCmd represents the install command
var installCmd = &cobra.Command{
	Use:   "install [component]",
	Short: "Install MyclicMemory integration",
	Long: `Install MyclicMemory integrations.

Examples:
  mycelicmemory install mcp     # Install MCP for Claude Desktop
  mycelicmemory install shell   # Install shell completion`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Println("Available installations:")
			fmt.Println("  mcp     - Install MCP for Claude Desktop")
			fmt.Println("  shell   - Install shell completion")
			return
		}
		runInstall(args[0])
	},
}

// killCmd represents the kill command
var killCmd = &cobra.Command{
	Use:   "kill <pid>",
	Short: "Kill specific mycelicmemory process",
	Long: `Kill a specific mycelicmemory process by PID.

Examples:
  mycelicmemory kill 12345`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		runKill(args[0])
	},
}


func init() {
	rootCmd.AddCommand(categorizeCmd)
	rootCmd.AddCommand(setupCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(killCmd)

	// Categorize flags
	categorizeCmd.Flags().BoolVar(&categorizeAutoCreate, "auto-create", true, "Auto-create suggested categories")
	categorizeCmd.Flags().Float64Var(&categorizeConfidenceThreshold, "confidence", 0.7, "Minimum confidence threshold")
}

func runCategorize(memoryID string) {
	db, cfg, err := getDB()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Check dependencies
	depResult := dependencies.Check(cfg)

	if !depResult.AIFeaturesAvailable() {
		fmt.Println("AI Categorization")
		fmt.Println("=================")
		fmt.Println()
		fmt.Println("❌ Error: AI features are not available.")
		fmt.Println()

		// Show specific issue
		switch depResult.Ollama.Status {
		case dependencies.StatusMissing, dependencies.StatusUnavailable:
			fmt.Println("Ollama is not running or not installed.")
		case dependencies.StatusDisabled:
			fmt.Println("Ollama is disabled in configuration.")
		case dependencies.StatusAvailable:
			if len(depResult.Ollama.MissingItems) > 0 {
				fmt.Printf("Missing required models: %s\n", strings.Join(depResult.Ollama.MissingItems, ", "))
			}
		}

		// Show installation instructions
		instructions := dependencies.GetInstallInstructions(depResult, cfg)
		if instructions.Ollama != nil {
			fmt.Println()
			fmt.Println("To enable AI categorization:")
			for _, step := range instructions.Ollama.InstallSteps {
				fmt.Println(step)
			}
			for _, step := range instructions.Ollama.ModelSteps {
				fmt.Println(step)
			}
		}

		fmt.Println()
		fmt.Println("Run 'mycelicmemory doctor' for full system status.")
		os.Exit(1)
	}

	// Check if AI is available (for unused variable warning suppression)
	aiManager := ai.NewManager(db, cfg)
	_ = aiManager // Will be used when categorization is implemented

	// TODO: Implement AI-based categorization
	fmt.Println("AI Categorization")
	fmt.Println("=================")
	fmt.Println()
	fmt.Printf("Memory ID: %s\n", memoryID)
	fmt.Printf("Auto-create: %v\n", categorizeAutoCreate)
	fmt.Printf("Confidence threshold: %.2f\n", categorizeConfidenceThreshold)
	fmt.Println()
	fmt.Println("AI-based categorization is not yet fully implemented.")
	fmt.Println("Use 'mycelicmemory analyze' for AI analysis features.")
}

func runSetup() {
	fmt.Println("MyclicMemory Setup Wizard")
	fmt.Println("======================")
	fmt.Println()

	// Check configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Creating default configuration...\n")
		cfg = config.DefaultConfig()
	}

	fmt.Println("Configuration Summary:")
	fmt.Printf("  Config Dir: %s\n", config.ConfigPath())
	fmt.Printf("  Database: %s\n", cfg.Database.Path)
	fmt.Printf("  REST API: %s:%d\n", cfg.RestAPI.Host, cfg.RestAPI.Port)
	fmt.Printf("  Ollama: %s\n", cfg.Ollama.BaseURL)
	fmt.Println()
	fmt.Println("Setup complete!")
	fmt.Println()
	fmt.Println("Run 'mycelicmemory doctor' to verify all components.")
}

func runValidate() {
	fmt.Println("MycelicMemory Installation Validation")
	fmt.Println("=====================================")
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

	// Check database path
	fmt.Print("Database Path... ")
	if cfg != nil {
		if _, err := os.Stat(cfg.Database.Path); os.IsNotExist(err) {
			fmt.Println("NOT FOUND (will be created on first use)")
		} else {
			fmt.Println("OK")
		}
	}

	// Check binary
	fmt.Print("Binary... ")
	if exe, err := os.Executable(); err == nil {
		fmt.Printf("OK (%s)\n", exe)
	} else {
		fmt.Println("ERROR")
		allOk = false
	}

	// Check optional dependencies
	if cfg != nil {
		fmt.Println()
		fmt.Println("Optional Dependencies:")

		depResult := dependencies.Check(cfg)

		// Ollama check
		fmt.Print("  Ollama... ")
		switch depResult.Ollama.Status {
		case dependencies.StatusAvailable:
			if len(depResult.Ollama.MissingItems) > 0 {
				fmt.Printf("PARTIAL (missing models: %s)\n", depResult.Ollama.MissingItems)
				hasWarnings = true
			} else {
				fmt.Println("OK")
			}
		case dependencies.StatusDisabled:
			fmt.Println("DISABLED")
		case dependencies.StatusMissing, dependencies.StatusUnavailable:
			fmt.Println("NOT AVAILABLE")
			hasWarnings = true
		}

		// Qdrant check
		fmt.Print("  Qdrant... ")
		switch depResult.Qdrant.Status {
		case dependencies.StatusAvailable:
			fmt.Println("OK")
		case dependencies.StatusDisabled:
			fmt.Println("DISABLED")
		case dependencies.StatusMissing, dependencies.StatusUnavailable:
			fmt.Println("NOT AVAILABLE")
			if cfg.Qdrant.Enabled {
				hasWarnings = true
			}
		}

		// Show warning summary if needed
		if warning := dependencies.FormatWarning(depResult); warning != "" {
			fmt.Println()
			fmt.Print(warning)
		}
	}

	fmt.Println()
	if allOk && !hasWarnings {
		fmt.Println("✅ Installation validated successfully!")
	} else if allOk && hasWarnings {
		fmt.Println("✅ Core installation validated.")
		fmt.Println("⚠️  Some optional dependencies are unavailable.")
		fmt.Println("   Run 'mycelicmemory doctor' for installation instructions.")
	} else {
		fmt.Println("❌ Some issues found. Run 'mycelicmemory doctor' for more details.")
	}
}

func runInstall(component string) {
	switch component {
	case "mcp":
		fmt.Println("Installing MCP for Claude Desktop...")
		fmt.Println()
		fmt.Println("MCP installation is not yet implemented.")
		fmt.Println("Please add mycelicmemory to your Claude Desktop config manually:")
		fmt.Println()
		fmt.Printf("  \"mycelicmemory\": {\n")
		fmt.Printf("    \"command\": \"%s\",\n", os.Args[0])
		fmt.Printf("    \"args\": [\"--mcp\"]\n")
		fmt.Printf("  }\n")

	case "shell":
		fmt.Println("To install shell completion, run one of:")
		fmt.Println()
		fmt.Println("  # Bash")
		fmt.Println("  mycelicmemory completion bash > /etc/bash_completion.d/mycelicmemory")
		fmt.Println()
		fmt.Println("  # Zsh")
		fmt.Println("  mycelicmemory completion zsh > \"${fpath[1]}/_mycelicmemory\"")
		fmt.Println()
		fmt.Println("  # Fish")
		fmt.Println("  mycelicmemory completion fish > ~/.config/fish/completions/mycelicmemory.fish")

	default:
		fmt.Printf("Unknown component: %s\n", component)
		fmt.Println("Available: mcp, shell")
		os.Exit(1)
	}
}

func runKill(pidStr string) {
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		fmt.Printf("Invalid PID: %s\n", pidStr)
		os.Exit(1)
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		fmt.Printf("Process not found: %d\n", pid)
		os.Exit(1)
	}

	if err := process.Signal(syscall.SIGTERM); err != nil {
		fmt.Printf("Error killing process: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Sent SIGTERM to process %d\n", pid)
}

