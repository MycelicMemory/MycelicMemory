package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/MycelicMemory/mycelicmemory/internal/database"
)

var (
	// create_category flags
	categoryDescription string
	categoryParent      string

	// create_domain flags
	domainDescription string
)

// listCategoriesCmd represents the list_categories command
var listCategoriesCmd = &cobra.Command{
	Use:   "list_categories",
	Short: "List all categories",
	Long:  `List all memory categories.`,
	Run: func(cmd *cobra.Command, args []string) {
		runListCategories()
	},
}

// createCategoryCmd represents the create_category command
var createCategoryCmd = &cobra.Command{
	Use:   "create_category <name>",
	Short: "Create a new category",
	Long: `Create a new memory category.

Examples:
  mycelicmemory create_category "Technical Documentation"
  mycelicmemory create_category "Meeting Notes" --description "Notes from meetings"`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		runCreateCategory(args[0])
	},
}

// categoryStatsCmd represents the category_stats command
var categoryStatsCmd = &cobra.Command{
	Use:   "category_stats",
	Short: "Show category statistics",
	Long:  `Show statistics for all categories.`,
	Run: func(cmd *cobra.Command, args []string) {
		runCategoryStats()
	},
}

// listDomainsCmd represents the list_domains command
var listDomainsCmd = &cobra.Command{
	Use:   "list_domains",
	Short: "List all knowledge domains",
	Long:  `List all knowledge domains.`,
	Run: func(cmd *cobra.Command, args []string) {
		runListDomains()
	},
}

// createDomainCmd represents the create_domain command
var createDomainCmd = &cobra.Command{
	Use:   "create_domain <name>",
	Short: "Create a new knowledge domain",
	Long: `Create a new knowledge domain.

Examples:
  mycelicmemory create_domain programming
  mycelicmemory create_domain "machine-learning" --description "ML and AI concepts"`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		runCreateDomain(args[0])
	},
}

// domainStatsCmd represents the domain_stats command
var domainStatsCmd = &cobra.Command{
	Use:   "domain_stats <domain>",
	Short: "Show statistics for a knowledge domain",
	Long: `Show statistics for a specific knowledge domain.

Examples:
  mycelicmemory domain_stats programming`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		runDomainStats(args[0])
	},
}

// listSessionsCmd represents the list_sessions command
var listSessionsCmd = &cobra.Command{
	Use:   "list_sessions",
	Short: "List all memory sessions",
	Long:  `List all memory sessions.`,
	Run: func(cmd *cobra.Command, args []string) {
		runListSessions()
	},
}

// sessionStatsCmd represents the session_stats command
var sessionStatsCmd = &cobra.Command{
	Use:   "session_stats",
	Short: "Show current session statistics",
	Long:  `Show statistics for the current session.`,
	Run: func(cmd *cobra.Command, args []string) {
		runSessionStats()
	},
}

func init() {
	rootCmd.AddCommand(listCategoriesCmd)
	rootCmd.AddCommand(createCategoryCmd)
	rootCmd.AddCommand(categoryStatsCmd)
	rootCmd.AddCommand(listDomainsCmd)
	rootCmd.AddCommand(createDomainCmd)
	rootCmd.AddCommand(domainStatsCmd)
	rootCmd.AddCommand(listSessionsCmd)
	rootCmd.AddCommand(sessionStatsCmd)

	// Create category flags
	createCategoryCmd.Flags().StringVarP(&categoryDescription, "description", "d", "", "Category description")
	createCategoryCmd.Flags().StringVar(&categoryParent, "parent", "", "Parent category ID")

	// Create domain flags
	createDomainCmd.Flags().StringVarP(&domainDescription, "description", "d", "", "Domain description")
}

func runListCategories() {
	db, _, err := getDB()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	categories, err := db.ListCategories()
	if err != nil {
		fmt.Printf("Error listing categories: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Categories (%d)\n", len(categories))
	fmt.Println("===============")
	fmt.Println()

	if len(categories) == 0 {
		fmt.Println("No categories found.")
		fmt.Println("Create one with: mycelicmemory create_category <name>")
		return
	}

	for _, c := range categories {
		fmt.Printf("- %s\n", c.Name)
		fmt.Printf("  ID: %s\n", c.ID)
		if c.Description != "" {
			fmt.Printf("  Description: %s\n", c.Description)
		}
		fmt.Println()
	}
}

func runCreateCategory(name string) {
	db, _, err := getDB()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	category := &database.Category{
		Name:        name,
		Description: categoryDescription,
	}

	if err := db.CreateCategory(category); err != nil {
		fmt.Printf("Error creating category: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Category Created Successfully")
	fmt.Println("=============================")
	fmt.Println()
	fmt.Printf("ID: %s\n", category.ID)
	fmt.Printf("Name: %s\n", category.Name)
	if category.Description != "" {
		fmt.Printf("Description: %s\n", category.Description)
	}
}

func runCategoryStats() {
	db, _, err := getDB()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	stats, err := db.GetStats()
	if err != nil {
		fmt.Printf("Error getting stats: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Category Statistics")
	fmt.Println("===================")
	fmt.Println()
	fmt.Printf("Total Categories: %d\n", stats.CategoryCount)
}

func runListDomains() {
	db, _, err := getDB()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	domains, err := db.ListDomains()
	if err != nil {
		fmt.Printf("Error listing domains: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Domains (%d)\n", len(domains))
	fmt.Println("============")
	fmt.Println()

	if len(domains) == 0 {
		fmt.Println("No domains found.")
		fmt.Println("Create one with: mycelicmemory create_domain <name>")
		return
	}

	for _, d := range domains {
		fmt.Printf("- %s\n", d.Name)
		fmt.Printf("  ID: %s\n", d.ID)
		if d.Description != "" {
			fmt.Printf("  Description: %s\n", d.Description)
		}
		fmt.Println()
	}
}

func runCreateDomain(name string) {
	db, _, err := getDB()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	domain := &database.Domain{
		Name:        name,
		Description: domainDescription,
	}

	if err := db.CreateDomain(domain); err != nil {
		fmt.Printf("Error creating domain: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Domain Created Successfully")
	fmt.Println("===========================")
	fmt.Println()
	fmt.Printf("ID: %s\n", domain.ID)
	fmt.Printf("Name: %s\n", domain.Name)
	if domain.Description != "" {
		fmt.Printf("Description: %s\n", domain.Description)
	}
}

func runDomainStats(domainName string) {
	db, _, err := getDB()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	stats, err := db.GetDomainStats(domainName)
	if err != nil {
		fmt.Printf("Error getting domain stats: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Domain Statistics: %s\n", domainName)
	fmt.Println("========================")
	fmt.Println()
	fmt.Printf("Memory Count: %d\n", stats.MemoryCount)
	fmt.Printf("Average Importance: %.1f\n", stats.AverageImportance)
}

func runListSessions() {
	db, _, err := getDB()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	sessions, err := db.ListSessions()
	if err != nil {
		fmt.Printf("Error listing sessions: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Sessions (%d)\n", len(sessions))
	fmt.Println("=============")
	fmt.Println()

	if len(sessions) == 0 {
		fmt.Println("No sessions found.")
		return
	}

	for _, s := range sessions {
		fmt.Printf("- %s\n", s.SessionID)
		fmt.Printf("  Agent: %s\n", s.AgentType)
		fmt.Printf("  Created: %s\n", s.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("  Last Active: %s\n", s.LastAccessed.Format("2006-01-02 15:04:05"))
		fmt.Println()
	}
}

func runSessionStats() {
	db, cfg, err := getDB()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	stats, err := db.GetStats()
	if err != nil {
		fmt.Printf("Error getting stats: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Session Statistics")
	fmt.Println("==================")
	fmt.Println()
	fmt.Printf("Total Sessions: %d\n", stats.SessionCount)
	fmt.Printf("Total Memories: %d\n", stats.MemoryCount)
	fmt.Printf("Database: %s\n", cfg.Database.Path)
}
