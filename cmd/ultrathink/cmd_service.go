package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/MycelicMemory/ultrathink/internal/api"
	"github.com/MycelicMemory/ultrathink/internal/daemon"
	"github.com/MycelicMemory/ultrathink/internal/database"
	"github.com/MycelicMemory/ultrathink/pkg/config"
)

var (
	startPort       int
	startHost       string
	startBackground bool
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the daemon",
	Long:  `Start the Ultrathink daemon which provides REST API and MCP services.`,
	Run: func(cmd *cobra.Command, args []string) {
		runStart()
	},
}

// stopCmd represents the stop command
var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the daemon",
	Long:  `Stop the running Ultrathink daemon.`,
	Run: func(cmd *cobra.Command, args []string) {
		runStop()
	},
}

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show daemon status",
	Long:  `Show the current status of the Ultrathink daemon.`,
	Run: func(cmd *cobra.Command, args []string) {
		runStatus()
	},
}

// psCmd represents the ps command
var psCmd = &cobra.Command{
	Use:   "ps",
	Short: "List running ultrathink processes",
	Long:  `List all running Ultrathink processes.`,
	Run: func(cmd *cobra.Command, args []string) {
		runPS()
	},
}

// killAllCmd represents the kill_all command
var killAllCmd = &cobra.Command{
	Use:   "kill_all",
	Short: "Kill all ultrathink processes",
	Long:  `Kill all running Ultrathink processes.`,
	Run: func(cmd *cobra.Command, args []string) {
		runKillAll()
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(psCmd)
	rootCmd.AddCommand(killAllCmd)

	// Start command flags
	startCmd.Flags().IntVarP(&startPort, "port", "p", 0, "Port to listen on (overrides config)")
	startCmd.Flags().StringVar(&startHost, "host", "", "Host to bind to (overrides config)")
	startCmd.Flags().BoolVarP(&startBackground, "background", "b", false, "Run in background (daemonize)")
}

func getDaemon() *daemon.Daemon {
	return daemon.New(config.ConfigPath(), Version)
}

func runStart() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	d := daemon.New(config.ConfigPath(), Version)

	// Check if already running
	if d.IsRunning() {
		status := d.Status()
		fmt.Printf("Ultrathink daemon is already running (PID: %d)\n", status.PID)
		fmt.Println("Use 'ultrathink stop' to stop it first")
		os.Exit(1)
	}

	// Handle background mode
	if startBackground {
		args := []string{"start"}
		if startPort > 0 {
			args = append(args, "--port", fmt.Sprintf("%d", startPort))
		}
		if startHost != "" {
			args = append(args, "--host", startHost)
		}
		// Don't pass -b again to avoid infinite loop

		_, err := d.Daemonize(args)
		if err != nil {
			fmt.Printf("Error starting daemon: %v\n", err)
			os.Exit(1)
		}

		// Wait for daemon to start (poll for up to 5 seconds)
		fmt.Println("Starting daemon...")
		for i := 0; i < 50; i++ {
			time.Sleep(100 * time.Millisecond)
			if d.IsRunning() {
				status := d.Status()
				fmt.Printf("Ultrathink daemon started (PID: %d)\n", status.PID)
				if status.RESTEnabled {
					fmt.Printf("REST API: http://%s:%d\n", status.RESTHost, status.RESTPort)
				}
				return
			}
		}

		fmt.Println("Failed to start daemon (timeout)")
		os.Exit(1)
	}

	// Foreground mode
	fmt.Printf("Ultrathink v%s\n", Version)
	fmt.Println("AI-powered persistent memory system (open source)")
	fmt.Println()

	// Override with command line flags
	if startPort > 0 {
		cfg.RestAPI.Port = startPort
	}
	if startHost != "" {
		cfg.RestAPI.Host = startHost
	}

	// Ensure config directory exists
	if err := cfg.EnsureConfigDir(); err != nil {
		fmt.Printf("Error creating config directory: %v\n", err)
		os.Exit(1)
	}

	// Initialize database
	db, err := database.Open(cfg.Database.Path)
	if err != nil {
		fmt.Printf("Error initializing database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Initialize schema
	if err := db.InitSchema(); err != nil {
		fmt.Printf("Error initializing schema: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Database: %s\n", cfg.Database.Path)

	// Register daemon state
	if err := d.Start(cfg.RestAPI.Enabled, cfg.RestAPI.Host, cfg.RestAPI.Port, false); err != nil {
		fmt.Printf("Warning: Could not register daemon state: %v\n", err)
	}
	defer d.Cleanup()

	// Create and start REST API server
	if cfg.RestAPI.Enabled {
		server := api.NewServer(db, cfg)

		// Handle shutdown signals
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		go func() {
			sig := <-sigChan
			fmt.Printf("\nReceived %v, shutting down...\n", sig)
			d.Cleanup()
			os.Exit(0)
		}()

		fmt.Printf("\nStarting REST API on %s:%d\n", cfg.RestAPI.Host, cfg.RestAPI.Port)
		fmt.Println("Press Ctrl+C to stop")
		if err := server.Start(); err != nil {
			fmt.Printf("Error starting server: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Println("REST API is disabled in configuration")
	}
}

func runStop() {
	d := getDaemon()

	if !d.IsRunning() {
		fmt.Println("Ultrathink daemon is not running")
		return
	}

	status := d.Status()
	fmt.Printf("Stopping Ultrathink daemon (PID: %d)...\n", status.PID)

	if err := d.Stop(); err != nil {
		fmt.Printf("Error stopping daemon: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Daemon stopped successfully")
}

func runStatus() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	d := daemon.New(config.ConfigPath(), Version)
	status := d.Status()

	fmt.Println("Ultrathink Status")
	fmt.Println("═════════════════")
	fmt.Println()

	if status.Running {
		fmt.Printf("🟢 Daemon: Running (PID: %d) - Uptime: %s\n", status.PID, formatDuration(status.Uptime))
		fmt.Printf("Version: %s\n", status.Version)
		fmt.Println()
		fmt.Println("Services:")
		fmt.Println("  🟢 MCP Server: Enabled")
		if status.RESTEnabled {
			fmt.Printf("  🟢 REST API: Running on port %d\n", status.RESTPort)
		} else {
			fmt.Println("  ⚪ REST API: Disabled")
		}
	} else {
		fmt.Println("🔴 Daemon: Stopped")
		fmt.Printf("Version: %s\n", Version)
	}

	fmt.Println()
	fmt.Println("Configuration:")
	fmt.Printf("  Config: %s/config.yaml\n", config.ConfigPath())
	fmt.Printf("  Database: %s\n", cfg.Database.Path)
}

func runPS() {
	d := getDaemon()
	processes, err := d.ListProcesses()
	if err != nil {
		fmt.Printf("Error listing processes: %v\n", err)
		os.Exit(1)
	}

	if len(processes) == 0 {
		fmt.Println("No Ultrathink processes running")
		return
	}

	fmt.Println("Running Ultrathink processes:")
	fmt.Println("PID\tTYPE\t\tUPTIME\t\tVERSION")
	fmt.Println("---\t----\t\t------\t\t-------")
	for _, p := range processes {
		fmt.Printf("%d\t%s\t\t%s\t\t%s\n", p.PID, p.Type, formatDuration(p.Uptime), p.Version)
	}
}

func runKillAll() {
	d := getDaemon()

	if !d.IsRunning() {
		fmt.Println("No Ultrathink processes running")
		return
	}

	fmt.Println("Killing all Ultrathink processes...")
	killed, err := d.KillAll()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Killed %d process(es)\n", killed)
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
	}
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	return fmt.Sprintf("%dd%dh", days, hours)
}
