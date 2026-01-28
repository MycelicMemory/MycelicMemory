package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/MycelicMemory/mycelicmemory/internal/database"
	"github.com/MycelicMemory/mycelicmemory/internal/dependencies"
	"github.com/MycelicMemory/mycelicmemory/internal/memory"
	"github.com/MycelicMemory/mycelicmemory/internal/search"
	"github.com/MycelicMemory/mycelicmemory/pkg/config"
)

var (
	// remember flags
	rememberImportance int
	rememberTags       []string
	rememberDomain     string
	rememberSource     string

	// search flags
	searchLimit  int
	searchDomain string
	searchTags   []string

	// update flags
	updateContent    string
	updateImportance int
	updateTags       []string
	updateDomain     string

	// list flags
	listLimit  int
	listOffset int
	listDomain string
)

// rememberCmd represents the remember command
var rememberCmd = &cobra.Command{
	Use:   "remember <content>",
	Short: "Store a memory",
	Long: `Store a new memory with the given content.

Examples:
  mycelicmemory remember "Go channels are like pipes between goroutines"
  mycelicmemory remember "Important meeting notes" --importance 9 --tags meeting,work
  mycelicmemory remember "Python tip" --domain programming`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		content := strings.Join(args, " ")
		runRemember(content)
	},
}

// searchCmd represents the search command
var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search memories",
	Long: `Search through stored memories using keywords or semantic search.

Examples:
  mycelicmemory search "concurrency patterns"
  mycelicmemory search "golang" --limit 10
  mycelicmemory search "api" --domain programming`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		query := strings.Join(args, " ")
		runSearch(query)
	},
}

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get memory by ID",
	Long: `Retrieve a specific memory by its UUID.

Examples:
  mycelicmemory get 550e8400-e29b-41d4-a716-446655440000`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		runGet(args[0])
	},
}

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all memories",
	Long: `List all stored memories with optional filtering.

Examples:
  mycelicmemory list
  mycelicmemory list --limit 20
  mycelicmemory list --domain programming`,
	Run: func(cmd *cobra.Command, args []string) {
		runList()
	},
}

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a memory",
	Long: `Update an existing memory's content, importance, or tags.

Examples:
  mycelicmemory update 550e8400-e29b-41d4-a716-446655440000 --content "Updated content"
  mycelicmemory update <id> --importance 8
  mycelicmemory update <id> --tags newtag1,newtag2`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		runUpdate(args[0])
	},
}

// forgetCmd represents the forget command
var forgetCmd = &cobra.Command{
	Use:   "forget <id>",
	Short: "Delete a memory",
	Long: `Delete a memory by its UUID.

Examples:
  mycelicmemory forget 550e8400-e29b-41d4-a716-446655440000`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		runForget(args[0])
	},
}

func init() {
	rootCmd.AddCommand(rememberCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(forgetCmd)

	// Remember flags
	rememberCmd.Flags().IntVarP(&rememberImportance, "importance", "i", 5, "Importance level (1-10)")
	rememberCmd.Flags().StringSliceVarP(&rememberTags, "tags", "t", nil, "Tags (comma-separated)")
	rememberCmd.Flags().StringVarP(&rememberDomain, "domain", "d", "", "Knowledge domain")
	rememberCmd.Flags().StringVarP(&rememberSource, "source", "s", "", "Source of the memory")

	// Search flags
	searchCmd.Flags().IntVarP(&searchLimit, "limit", "l", 10, "Maximum results to return")
	searchCmd.Flags().StringVarP(&searchDomain, "domain", "d", "", "Filter by domain")
	searchCmd.Flags().StringSliceVarP(&searchTags, "tags", "t", nil, "Filter by tags")

	// Update flags
	updateCmd.Flags().StringVar(&updateContent, "content", "", "New content")
	updateCmd.Flags().IntVarP(&updateImportance, "importance", "i", 0, "New importance (1-10)")
	updateCmd.Flags().StringSliceVarP(&updateTags, "tags", "t", nil, "New tags")
	updateCmd.Flags().StringVarP(&updateDomain, "domain", "d", "", "New domain")

	// List flags
	listCmd.Flags().IntVarP(&listLimit, "limit", "l", 50, "Maximum results to return")
	listCmd.Flags().IntVarP(&listOffset, "offset", "o", 0, "Offset for pagination")
	listCmd.Flags().StringVarP(&listDomain, "domain", "d", "", "Filter by domain")
}

func getDB() (*database.Database, *config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, nil, err
	}

	db, err := database.Open(cfg.Database.Path)
	if err != nil {
		return nil, nil, err
	}

	// Initialize schema if needed
	if err := db.InitSchema(); err != nil {
		db.Close()
		return nil, nil, err
	}

	return db, cfg, nil
}

func runRemember(content string) {
	db, cfg, err := getDB()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Check for dependency warnings
	depResult := dependencies.Check(cfg)
	if warning := dependencies.FormatShortWarning(depResult); warning != "" {
		fmt.Printf("⚠️  %s - memory stored but not indexed for AI search\n", warning)
	}

	svc := memory.NewService(db, cfg)

	result, err := svc.Store(&memory.StoreOptions{
		Content:    content,
		Importance: rememberImportance,
		Tags:       rememberTags,
		Domain:     rememberDomain,
		Source:     rememberSource,
	})
	if err != nil {
		fmt.Printf("Error storing memory: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Memory Stored Successfully")
	fmt.Println("=============================")
	fmt.Println()
	fmt.Printf("🆔 Memory ID: %s\n", result.Memory.ID)
	fmt.Println()
	fmt.Println("📝 Stored Content:")
	fmt.Printf("   %s\n", result.Memory.Content)
	fmt.Println()
	fmt.Printf("📊 Importance: %d/10\n", result.Memory.Importance)
	if len(result.Memory.Tags) > 0 {
		fmt.Printf("🏷️  Tags: %s\n", strings.Join(result.Memory.Tags, ", "))
	}
	if result.Memory.Domain != "" {
		fmt.Printf("🌍 Domain: %s\n", result.Memory.Domain)
	}
	fmt.Println()
	fmt.Println("💡 Use this memory ID in subsequent commands:")
	fmt.Printf("   mycelicmemory update %s --content \"new content\"\n", result.Memory.ID)
	fmt.Printf("   mycelicmemory relate %s <other-memory-id>\n", result.Memory.ID)
	fmt.Println()
	fmt.Println("💡 Suggestions:")
	fmt.Println("   💡 Consider setting higher importance (--importance 8-10) for critical information")
	fmt.Println("   💡 Add tags (--tags tag1,tag2) to make this memory easier to find later")
	fmt.Println("   💡 Specify a domain (--domain category) to organize related memories")

	// Show dependency warning at the end if AI not available
	if !depResult.AIFeaturesAvailable() {
		fmt.Println()
		fmt.Println("ℹ️  Note: AI features disabled. Run 'mycelicmemory doctor' to enable.")
	}
}

func runSearch(query string) {
	db, cfg, err := getDB()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Check for dependency warnings
	depResult := dependencies.Check(cfg)
	searchMode := "keyword"
	if depResult.SemanticSearchAvailable() {
		searchMode = "semantic + keyword"
	}

	engine := search.NewEngine(db, cfg)

	results, err := engine.Search(&search.SearchOptions{
		Query:  query,
		Limit:  searchLimit,
		Domain: searchDomain,
		Tags:   searchTags,
	})
	if err != nil {
		fmt.Printf("Error searching: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Search Results for: \"%s\"\n", query)
	fmt.Println("========================================")
	if !depResult.SemanticSearchAvailable() {
		fmt.Println("⚠️  Semantic search unavailable - using keyword matching only")
	}
	fmt.Println()
	fmt.Printf("Found %d result(s) [mode: %s]:\n\n", len(results), searchMode)

	for i, r := range results {
		fmt.Printf("%d. %s\n", i+1, r.Memory.Content)
		fmt.Printf("   ID: %s\n", r.Memory.ID)
		fmt.Printf("   Relevance: %.2f\n", r.Relevance)
		fmt.Printf("   Importance: %d/10\n", r.Memory.Importance)
		if len(r.Memory.Tags) > 0 {
			fmt.Printf("   Tags: %s\n", strings.Join(r.Memory.Tags, ", "))
		}
		if r.Memory.Domain != "" {
			fmt.Printf("   Domain: %s\n", r.Memory.Domain)
		}
		fmt.Printf("   Created: %s\n", r.Memory.CreatedAt.Format("2006-01-02 15:04"))
		fmt.Printf("   Updated: %s\n", r.Memory.UpdatedAt.Format("2006-01-02 15:04"))
		fmt.Printf("   Session: %s\n", r.Memory.SessionID)
		fmt.Println()
	}

	fmt.Println("Response format: detailed")
	fmt.Println()

	// Show appropriate suggestions based on AI availability
	fmt.Println("💡 Suggestions:")
	if depResult.SemanticSearchAvailable() {
		fmt.Println("   💡 Semantic search is active - natural language queries work best")
	} else {
		fmt.Println("   💡 Enable semantic search: run 'mycelicmemory doctor' for setup instructions")
	}
	fmt.Println("   💡 Combine with tags (--tags tag1,tag2) for more precise filtering")
}

func runGet(id string) {
	db, cfg, err := getDB()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	svc := memory.NewService(db, cfg)

	mem, err := svc.Get(&memory.GetOptions{ID: id})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	if mem == nil {
		fmt.Printf("Memory not found: %s\n", id)
		os.Exit(1)
	}

	fmt.Println("Memory Details")
	fmt.Println("==============")
	fmt.Println()
	fmt.Println("📝 Content:")
	fmt.Printf("   %s\n", mem.Content)
	fmt.Println()
	fmt.Println("📊 Metadata:")
	fmt.Printf("   ID: %s\n", mem.ID)
	fmt.Printf("   Importance: %d/10\n", mem.Importance)
	if len(mem.Tags) > 0 {
		fmt.Printf("   Tags: %s\n", strings.Join(mem.Tags, ", "))
	}
	if mem.Domain != "" {
		fmt.Printf("   Domain: %s\n", mem.Domain)
	}
	fmt.Printf("   Session: %s\n", mem.SessionID)
	fmt.Printf("   Created: %s\n", mem.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("   Updated: %s\n", mem.UpdatedAt.Format("2006-01-02 15:04:05"))
	fmt.Println()
	fmt.Println("💡 Suggestions:")
	fmt.Printf("   💡 Update this memory: mycelicmemory update %s --content \"new content\"\n", mem.ID)
	if len(mem.Tags) > 0 {
		fmt.Printf("   💡 Find related: mycelicmemory search --tags %s\n", strings.Join(mem.Tags, ","))
	}
	fmt.Printf("   💡 Create relationship: mycelicmemory relate %s <other-memory-id>\n", mem.ID)
}

func runList() {
	db, cfg, err := getDB()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	svc := memory.NewService(db, cfg)

	memories, err := svc.List(&memory.ListOptions{
		Limit:  listLimit,
		Offset: listOffset,
		Domain: listDomain,
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Memory List")
	fmt.Println("===========")
	fmt.Println()
	fmt.Printf("Found %d memories\n\n", len(memories))

	var firstID string
	for i, m := range memories {
		if i == 0 {
			firstID = m.ID
		}
		fmt.Printf("%d. %s\n", i+1, m.Content)
		fmt.Printf("   ID: %s | Importance: %d/10", m.ID, m.Importance)
		if len(m.Tags) > 0 {
			fmt.Printf(" | Tags: %s", strings.Join(m.Tags, ", "))
		}
		fmt.Printf(" | Created: %s\n\n", m.CreatedAt.Format("2006-01-02"))
	}

	if len(memories) > 0 {
		fmt.Println()
		fmt.Println("💡 Suggestions:")
		fmt.Printf("   💡 View details: mycelicmemory get %s\n", firstID)
		fmt.Println("   💡 Filter results: mycelicmemory list --limit 5 or use search with specific criteria")
	}
}

func runUpdate(id string) {
	db, cfg, err := getDB()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	svc := memory.NewService(db, cfg)

	opts := &memory.UpdateOptions{ID: id}

	// Show preview of changes
	fmt.Printf("🔄 Updating memory %s with the following changes:\n", id)
	if updateContent != "" {
		opts.Content = &updateContent
		fmt.Printf("   content: %s\n", updateContent)
	}
	if updateImportance > 0 {
		opts.Importance = &updateImportance
		fmt.Printf("   importance: %d\n", updateImportance)
	}
	if len(updateTags) > 0 {
		opts.Tags = updateTags
		fmt.Printf("   tags: %s\n", strings.Join(updateTags, ", "))
	}
	if updateDomain != "" {
		opts.Domain = &updateDomain
		fmt.Printf("   domain: %s\n", updateDomain)
	}
	fmt.Print("Continue? [Y/n]: ")

	mem, err := svc.Update(opts)
	if err != nil {
		fmt.Printf("Error updating memory: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Memory Updated Successfully")
	fmt.Println("================================")
	fmt.Println()
	fmt.Printf("🆔 Memory ID: %s\n", mem.ID)
	fmt.Println()
	fmt.Println("📝 Updated Fields:")
	fmt.Printf("   Content: %s\n", mem.Content)
	fmt.Printf("   Importance: %d/10\n", mem.Importance)
	if len(mem.Tags) > 0 {
		fmt.Printf("   Tags: %s\n", strings.Join(mem.Tags, ", "))
	}
	fmt.Println()
	fmt.Println("Memory updated and ready for use in subsequent commands.")
	fmt.Println()
	fmt.Println("💡 Suggestions:")
	fmt.Printf("   💡 Next: Search related memories with: mycelicmemory search \"%s\"\n", truncate(mem.Content, 30))
	if len(mem.Tags) > 0 {
		fmt.Printf("   💡 Find similar: mycelicmemory search --tags %s\n", strings.Join(mem.Tags, ","))
	}
}

func runForget(id string) {
	// Confirmation prompt
	fmt.Printf("Are you sure you want to delete memory %s? [y/N]: ", id)
	var response string
	_, _ = fmt.Scanln(&response)
	if response != "y" && response != "Y" {
		fmt.Println("Delete cancelled.")
		return
	}

	db, cfg, err := getDB()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	svc := memory.NewService(db, cfg)

	if err := svc.Delete(id); err != nil {
		fmt.Printf("Error deleting memory: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("SUCCESS: Memory deleted successfully")
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
