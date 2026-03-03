package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/MycelicMemory/mycelicmemory/internal/dbmanager"
	"github.com/MycelicMemory/mycelicmemory/pkg/config"
)

var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "Manage databases",
	Long:  `List, create, switch, archive, import, and export databases.`,
}

var dbListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all databases",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := loadConfigOrExit()
		mgr := dbmanager.New(cfg)

		dbs, err := mgr.ListDatabases()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("Databases:")
		fmt.Printf("  %-15s %-8s %-10s %s\n", "NAME", "ACTIVE", "SIZE", "DESCRIPTION")
		fmt.Printf("  %-15s %-8s %-10s %s\n", "----", "------", "----", "-----------")
		for _, db := range dbs {
			active := ""
			if db.IsActive {
				active = "*"
			}
			size := formatBytes(db.SizeBytes)
			fmt.Printf("  %-15s %-8s %-10s %s\n", db.Name, active, size, db.Description)
		}
	},
}

var dbCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new database",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		desc, _ := cmd.Flags().GetString("description")
		cfg := loadConfigOrExit()
		mgr := dbmanager.New(cfg)

		info, err := mgr.CreateDatabase(args[0], desc)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Created database %q at %s\n", info.Name, info.Path)
	},
}

var dbSwitchCmd = &cobra.Command{
	Use:   "switch <name>",
	Short: "Switch active database",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cfg := loadConfigOrExit()
		mgr := dbmanager.New(cfg)

		if err := mgr.SwitchDatabase(args[0]); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Switched to database %q\n", args[0])
		fmt.Println("Restart the daemon for changes to take effect.")
	},
}

var dbDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a database",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cfg := loadConfigOrExit()
		mgr := dbmanager.New(cfg)

		if err := mgr.DeleteDatabase(args[0]); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Deleted database %q\n", args[0])
	},
}

var dbArchiveCmd = &cobra.Command{
	Use:   "archive [name]",
	Short: "Archive (backup) a database",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := ""
		if len(args) > 0 {
			name = args[0]
		}
		cfg := loadConfigOrExit()
		mgr := dbmanager.New(cfg)

		path, err := mgr.ArchiveDatabase(name)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Archived to %s\n", path)
	},
}

var dbImportCmd = &cobra.Command{
	Use:   "import <file>",
	Short: "Import a database file",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name, _ := cmd.Flags().GetString("name")
		if name == "" {
			fmt.Println("Error: --name is required")
			os.Exit(1)
		}
		cfg := loadConfigOrExit()
		mgr := dbmanager.New(cfg)

		info, err := mgr.ImportDatabase(args[0], name)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Imported database %q at %s\n", info.Name, info.Path)
	},
}

var dbExportCmd = &cobra.Command{
	Use:   "export <name> <path>",
	Short: "Export a database to a file",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		cfg := loadConfigOrExit()
		mgr := dbmanager.New(cfg)

		if err := mgr.ExportDatabase(args[0], args[1]); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Exported database %q to %s\n", args[0], args[1])
	},
}

var dbInfoCmd = &cobra.Command{
	Use:   "info [name]",
	Short: "Show database info",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := "default"
		if len(args) > 0 {
			name = args[0]
		}
		cfg := loadConfigOrExit()
		mgr := dbmanager.New(cfg)

		info, err := mgr.GetDatabase(name)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		active := "no"
		if info.IsActive {
			active = "yes"
		}
		fmt.Printf("Name:        %s\n", info.Name)
		fmt.Printf("Path:        %s\n", info.Path)
		fmt.Printf("Active:      %s\n", active)
		fmt.Printf("Size:        %s\n", formatBytes(info.SizeBytes))
		if info.Description != "" {
			fmt.Printf("Description: %s\n", info.Description)
		}
		if info.CreatedAt != "" {
			fmt.Printf("Created:     %s\n", info.CreatedAt)
		}
	},
}

func init() {
	rootCmd.AddCommand(dbCmd)
	dbCmd.AddCommand(dbListCmd)
	dbCmd.AddCommand(dbCreateCmd)
	dbCmd.AddCommand(dbSwitchCmd)
	dbCmd.AddCommand(dbDeleteCmd)
	dbCmd.AddCommand(dbArchiveCmd)
	dbCmd.AddCommand(dbImportCmd)
	dbCmd.AddCommand(dbExportCmd)
	dbCmd.AddCommand(dbInfoCmd)

	dbCreateCmd.Flags().StringP("description", "d", "", "Database description")
	dbImportCmd.Flags().String("name", "", "Name for the imported database")
}

func loadConfigOrExit() *config.Config {
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}
	return cfg
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
