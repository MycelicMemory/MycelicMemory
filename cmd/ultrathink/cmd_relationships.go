package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/MycelicMemory/ultrathink/internal/relationships"
	"github.com/MycelicMemory/ultrathink/pkg/config"
	"github.com/MycelicMemory/ultrathink/internal/database"
)

var (
	// relate flags
	relateType     string
	relateStrength float64
	relateContext  string

	// find_related flags
	findRelatedLimit int
	findRelatedType  string

	// map_graph flags
	graphDepth       int
	graphMinStrength float64
)

// relateCmd represents the relate command
var relateCmd = &cobra.Command{
	Use:   "relate <source-id> <target-id>",
	Short: "Create relationship between memories",
	Long: `Create a relationship between two memories.

Relationship types: references, contradicts, expands, similar, sequential, causes, enables

Examples:
  ultrathink relate <id1> <id2> --type similar
  ultrathink relate <id1> <id2> --type references --strength 0.9
  ultrathink relate <id1> <id2> --type causes --context "Root cause analysis"`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		runRelate(args[0], args[1])
	},
}

// findRelatedCmd represents the find_related command
var findRelatedCmd = &cobra.Command{
	Use:   "find_related <id>",
	Short: "Find memories related to a specific memory",
	Long: `Find all memories that are related to the specified memory.

Examples:
  ultrathink find_related <id>
  ultrathink find_related <id> --limit 20
  ultrathink find_related <id> --type similar`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		runFindRelated(args[0])
	},
}

// mapGraphCmd represents the map_graph command
var mapGraphCmd = &cobra.Command{
	Use:   "map_graph <id>",
	Short: "Generate relationship graph visualization",
	Long: `Generate a relationship graph starting from a specific memory.

Examples:
  ultrathink map_graph <id>
  ultrathink map_graph <id> --depth 3
  ultrathink map_graph <id> --min-strength 0.5`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		runMapGraph(args[0])
	},
}

// discoverCmd represents the discover command
var discoverCmd = &cobra.Command{
	Use:   "discover",
	Short: "Discover relationships using AI analysis",
	Long: `Use AI to discover potential relationships between memories.

Examples:
  ultrathink discover
  ultrathink discover --limit 10`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Discovering relationships...")
		fmt.Println("Note: This feature requires Ollama to be running.")
		fmt.Println("Use 'ultrathink doctor' to check AI availability.")
	},
}

func init() {
	rootCmd.AddCommand(relateCmd)
	rootCmd.AddCommand(findRelatedCmd)
	rootCmd.AddCommand(mapGraphCmd)
	rootCmd.AddCommand(discoverCmd)

	// Relate flags
	relateCmd.Flags().StringVarP(&relateType, "type", "t", "similar", "Relationship type (references, contradicts, expands, similar, sequential, causes, enables)")
	relateCmd.Flags().Float64VarP(&relateStrength, "strength", "s", 0.8, "Relationship strength (0.0-1.0)")
	relateCmd.Flags().StringVar(&relateContext, "context", "", "Context explaining the relationship")

	// Find related flags
	findRelatedCmd.Flags().IntVarP(&findRelatedLimit, "limit", "l", 10, "Maximum results")
	findRelatedCmd.Flags().StringVarP(&findRelatedType, "type", "t", "", "Filter by relationship type")

	// Map graph flags
	mapGraphCmd.Flags().IntVarP(&graphDepth, "depth", "d", 2, "Graph traversal depth (1-5)")
	mapGraphCmd.Flags().Float64Var(&graphMinStrength, "min-strength", 0, "Minimum relationship strength")
}

func runRelate(sourceID, targetID string) {
	// Confirmation prompt
	fmt.Printf("Are you sure you want to create a '%s' relationship between memory %s and %s? [y/N]: ", relateType, sourceID, targetID)
	var response string
	fmt.Scanln(&response)
	if response != "y" && response != "Y" {
		fmt.Println("Relationship creation cancelled.")
		return
	}

	db, cfg, err := getDB()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	svc := relationships.NewService(db, cfg)

	_, err = svc.Create(&relationships.CreateOptions{
		SourceMemoryID:   sourceID,
		TargetMemoryID:   targetID,
		RelationshipType: relateType,
		Strength:         relateStrength,
		Context:          relateContext,
	})
	if err != nil {
		fmt.Printf("Error creating relationship: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("SUCCESS: Memory relationship created successfully!")
}

func runFindRelated(memoryID string) {
	db, cfg, err := getDB()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	svc := relationships.NewService(db, cfg)

	results, err := svc.FindRelated(&relationships.FindRelatedOptions{
		MemoryID:    memoryID,
		Limit:       findRelatedLimit,
		Type:        findRelatedType,
	})
	if err != nil {
		fmt.Printf("Error finding related memories: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Related Memories for: %s\n", memoryID)
	fmt.Println("════════════════════════════════════")
	fmt.Println()

	if len(results) == 0 {
		fmt.Println("No related memories found.")
		return
	}

	fmt.Printf("Found %d related memory(ies):\n\n", len(results))

	for i, r := range results {
		fmt.Printf("%d. %s\n", i+1, truncateContent(r.Memory.Content, 60))
		fmt.Printf("   ID: %s\n", r.Memory.ID)
		fmt.Printf("   Relationship: %s (strength: %.2f)\n", r.RelationshipType, r.Strength)
		fmt.Printf("   Importance: %d/10\n", r.Memory.Importance)
		fmt.Println()
	}
}

func runMapGraph(memoryID string) {
	db, cfg, err := getDB()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	svc := relationships.NewService(db, cfg)

	result, err := svc.MapGraph(&relationships.MapGraphOptions{
		RootID:      memoryID,
		Depth:       graphDepth,
		MinStrength: graphMinStrength,
	})
	if err != nil {
		fmt.Printf("Error mapping graph: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Relationship Graph for: %s\n", memoryID)
	fmt.Println("==========================================")
	fmt.Println()
	fmt.Printf("Nodes: %d | Edges: %d | Depth: %d\n\n", result.TotalNodes, result.TotalEdges, result.MaxDepth)

	fmt.Println("Nodes:")
	for _, n := range result.Nodes {
		distMarker := strings.Repeat("  ", n.Distance)
		fmt.Printf("%s[%d] %s - %s\n", distMarker, n.Distance, n.ID[:8], truncateContent(n.Content, 40))
	}

	fmt.Println()
	fmt.Println("Edges:")
	for _, e := range result.Edges {
		fmt.Printf("  %s -[%s (%.2f)]-> %s\n", e.SourceID[:8], e.Type, e.Strength, e.TargetID[:8])
	}
}

func truncateContent(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func getRelDB() (*database.Database, *config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, nil, err
	}

	db, err := database.Open(cfg.Database.Path)
	if err != nil {
		return nil, nil, err
	}

	return db, cfg, nil
}
