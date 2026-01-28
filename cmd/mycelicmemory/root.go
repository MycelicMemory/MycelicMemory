package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/MycelicMemory/mycelicmemory/internal/database"
	"github.com/MycelicMemory/mycelicmemory/internal/mcp"
	"github.com/MycelicMemory/mycelicmemory/pkg/config"
)

var (
	// Version is set during build
	Version = "1.2.0"

	// Global flags
	mcpMode bool
	quiet   bool
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "mycelicmemory",
	Short: "AI-powered persistent memory system (open source)",
	Long: `MyclicMemory provides persistent memory capabilities through natural language commands.
Works as both a standalone CLI tool and MCP server for AI agents.

Examples:
  mycelicmemory remember "Go channels are like pipes between goroutines"
  mycelicmemory search "concurrency patterns"
  mycelicmemory relate <id1> <id2> --type similar
  mycelicmemory forget <memory-id>

  mycelicmemory start     # Start daemon
  mycelicmemory status    # Check daemon status

Parameter Help:
  Add --help_parameters to any command for detailed parameter documentation:
  mycelicmemory remember --help_parameters             # Smart parameter selection
  mycelicmemory search --help_parameters --show_all    # Show all parameters

Progressive Discovery:
  --basic_only     Show only essential parameters (beginner-friendly)
  --show_advanced  Show basic + advanced parameters (power users)
  --show_all       Show all parameters including expert options`,
	Version: Version,
	Run: func(cmd *cobra.Command, args []string) {
		if mcpMode {
			runMCPServer()
		} else {
			_ = cmd.Help()
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringP("config", "c", "", "config file path")
	rootCmd.PersistentFlags().String("log_level", "info", "log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().BoolVar(&mcpMode, "mcp", false, "run as MCP server (JSON-RPC over stdin/stdout)")
	rootCmd.PersistentFlags().BoolVar(&quiet, "quiet", false, "suppress output")
}

// runMCPServer starts the MCP server mode
func runMCPServer() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Open database
	db, err := database.Open(cfg.Database.Path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Create MCP server
	server := mcp.NewServer(db, cfg)

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		cancel()
	}()

	// Run MCP server
	if err := server.Run(ctx); err != nil && err != context.Canceled {
		fmt.Fprintf(os.Stderr, "MCP server error: %v\n", err)
		os.Exit(1)
	}
}
