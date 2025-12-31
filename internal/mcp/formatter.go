package mcp

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Formatter handles UX-friendly output formatting for MCP responses
type Formatter struct{}

// NewFormatter creates a new formatter
func NewFormatter() *Formatter {
	return &Formatter{}
}

// FormatToolResponse formats a tool response with rich UX elements
func (f *Formatter) FormatToolResponse(toolName string, result interface{}, duration time.Duration) string {
	var sb strings.Builder

	// Tool header with icon
	icon := f.getToolIcon(toolName)
	sb.WriteString(fmt.Sprintf("\n%s **%s**\n", icon, f.formatToolName(toolName)))
	sb.WriteString(f.getToolTagline(toolName))
	sb.WriteString("\n")

	// Add visual separator
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	// Format based on tool type
	switch toolName {
	case "store_memory":
		sb.WriteString(f.formatStoreMemory(result))
	case "search":
		sb.WriteString(f.formatSearch(result))
	case "get_memory_by_id":
		sb.WriteString(f.formatGetMemory(result))
	case "update_memory":
		sb.WriteString(f.formatUpdateMemory(result))
	case "delete_memory":
		sb.WriteString(f.formatDeleteMemory(result))
	case "analysis":
		sb.WriteString(f.formatAnalysis(result))
	case "relationships":
		sb.WriteString(f.formatRelationships(result))
	case "sessions":
		sb.WriteString(f.formatSessions(result))
	case "domains":
		sb.WriteString(f.formatDomains(result))
	case "categories":
		sb.WriteString(f.formatCategories(result))
	case "stats":
		sb.WriteString(f.formatStats(result))
	default:
		// Fallback to JSON
		jsonBytes, _ := json.MarshalIndent(result, "", "  ")
		sb.WriteString(string(jsonBytes))
	}

	// Performance footer
	sb.WriteString("\n\n")
	sb.WriteString(f.formatPerformance(duration))

	// Contextual suggestions
	suggestions := f.getSuggestions(toolName, result)
	if len(suggestions) > 0 {
		sb.WriteString("\n\n")
		sb.WriteString("💡 **Next Steps**\n")
		for _, s := range suggestions {
			sb.WriteString(fmt.Sprintf("   → %s\n", s))
		}
	}

	// JSON data section
	sb.WriteString("\n\n")
	sb.WriteString("<details>\n<summary>📋 Raw JSON Response</summary>\n\n```json\n")
	jsonBytes, _ := json.MarshalIndent(result, "", "  ")
	sb.WriteString(string(jsonBytes))
	sb.WriteString("\n```\n</details>")

	return sb.String()
}

func (f *Formatter) getToolIcon(toolName string) string {
	icons := map[string]string{
		"store_memory":     "💾",
		"search":           "🔍",
		"get_memory_by_id": "📖",
		"update_memory":    "✏️",
		"delete_memory":    "🗑️",
		"analysis":         "🧠",
		"relationships":    "🕸️",
		"sessions":         "📊",
		"domains":          "🏷️",
		"categories":       "📂",
		"stats":            "📈",
	}
	if icon, ok := icons[toolName]; ok {
		return icon
	}
	return "⚡"
}

func (f *Formatter) formatToolName(name string) string {
	parts := strings.Split(name, "_")
	for i, p := range parts {
		parts[i] = strings.Title(p)
	}
	return strings.Join(parts, " ")
}

func (f *Formatter) getToolTagline(toolName string) string {
	taglines := map[string]string{
		"store_memory":     "Persisting knowledge for future recall",
		"search":           "Finding relevant memories across your knowledge base",
		"get_memory_by_id": "Retrieving specific memory details",
		"update_memory":    "Evolving your stored knowledge",
		"delete_memory":    "Removing outdated information",
		"analysis":         "AI-powered insights from your memories",
		"relationships":    "Mapping connections in your knowledge graph",
		"sessions":         "Viewing your memory sessions",
		"domains":          "Organizing knowledge by domain",
		"categories":       "Hierarchical memory organization",
		"stats":            "System metrics and analytics",
	}
	if tagline, ok := taglines[toolName]; ok {
		return fmt.Sprintf("*%s*", tagline)
	}
	return ""
}

func (f *Formatter) formatStoreMemory(result interface{}) string {
	var sb strings.Builder

	data, ok := result.(*StoreMemoryResponse)
	if !ok {
		return f.fallbackJSON(result)
	}

	sb.WriteString("✅ **Memory Stored Successfully**\n\n")
	sb.WriteString(fmt.Sprintf("```\n%s\n```\n\n", data.Content))
	sb.WriteString("┌─────────────────────────────────────┐\n")
	sb.WriteString(fmt.Sprintf("│ 🆔 ID: `%s`\n", f.truncateID(data.MemoryID)))
	sb.WriteString(fmt.Sprintf("│ 📅 Created: %s\n", f.formatTime(data.CreatedAt)))
	sb.WriteString(fmt.Sprintf("│ 🔗 Session: %s\n", f.truncateID(data.SessionID)))
	sb.WriteString("└─────────────────────────────────────┘")

	return sb.String()
}

func (f *Formatter) formatSearch(result interface{}) string {
	var sb strings.Builder

	data, ok := result.(*SearchResponse)
	if !ok {
		return f.fallbackJSON(result)
	}

	// Results summary
	sb.WriteString(fmt.Sprintf("📊 **Found %d result(s)**", data.Count))
	if data.SearchMetadata != nil {
		sb.WriteString(fmt.Sprintf(" for query: `%s`\n", data.SearchMetadata.Query))
	} else {
		sb.WriteString("\n")
	}

	if data.Count == 0 {
		sb.WriteString("\n```\nNo memories match your search criteria.\n```\n")
		sb.WriteString("\n💡 Try broadening your search terms or checking different tags.")
		return sb.String()
	}

	sb.WriteString("\n")

	// Visual results
	for i, r := range data.Results {
		sb.WriteString(f.formatSearchResult(i+1, r))
	}

	// Token optimization info
	if data.SizeMetadata != nil {
		sb.WriteString("\n📦 **Response Metrics**\n")
		sb.WriteString(fmt.Sprintf("   • Tokens: ~%d | Chars: %d | Within Budget: %s\n",
			data.SizeMetadata.EstimatedTokens,
			data.SizeMetadata.EstimatedChars,
			f.boolToEmoji(data.SizeMetadata.IsWithinTokenBudget)))
	}

	return sb.String()
}

func (f *Formatter) formatSearchResult(num int, r SearchResultLM) string {
	var sb strings.Builder

	relevanceBar := f.makeProgressBar(r.RelevanceScore, 10)
	relevancePercent := int(r.RelevanceScore * 100)

	sb.WriteString(fmt.Sprintf("### %d. Memory `%s`\n", num, f.truncateID(r.Memory.ID)))
	sb.WriteString(fmt.Sprintf("**Relevance:** %s %d%%\n\n", relevanceBar, relevancePercent))
	sb.WriteString(fmt.Sprintf("> %s\n\n", f.truncateContent(r.Memory.Content, 200)))

	// Metadata row
	sb.WriteString("```yaml\n")
	sb.WriteString(fmt.Sprintf("importance: %d/10\n", r.Memory.Importance))
	if len(r.Memory.Tags) > 0 {
		sb.WriteString(fmt.Sprintf("tags: [%s]\n", strings.Join(r.Memory.Tags, ", ")))
	}
	if r.Memory.Domain != "" {
		sb.WriteString(fmt.Sprintf("domain: %s\n", r.Memory.Domain))
	}
	sb.WriteString(fmt.Sprintf("created: %s\n", f.formatTime(r.Memory.CreatedAt)))
	sb.WriteString("```\n\n")

	return sb.String()
}

func (f *Formatter) formatGetMemory(result interface{}) string {
	var sb strings.Builder

	data, ok := result.(*MemoryResponse)
	if !ok {
		return f.fallbackJSON(result)
	}

	if !data.Success || data.Memory == nil {
		sb.WriteString("❌ **Memory Not Found**\n")
		if data.Message != "" {
			sb.WriteString(fmt.Sprintf("\n%s", data.Message))
		}
		return sb.String()
	}

	m := data.Memory
	importanceBar := f.makeProgressBar(float64(m.Importance)/10.0, 10)

	sb.WriteString("📖 **Memory Details**\n\n")
	sb.WriteString(fmt.Sprintf("**ID:** `%s`\n\n", m.ID))
	sb.WriteString("**Content:**\n")
	sb.WriteString(fmt.Sprintf("```\n%s\n```\n\n", m.Content))

	sb.WriteString("┌──────────────── Metadata ────────────────┐\n")
	sb.WriteString(fmt.Sprintf("│ ⭐ Importance: %s %d/10\n", importanceBar, m.Importance))
	if len(m.Tags) > 0 {
		sb.WriteString(fmt.Sprintf("│ 🏷️  Tags: %s\n", strings.Join(m.Tags, ", ")))
	}
	if m.Domain != "" {
		sb.WriteString(fmt.Sprintf("│ 📁 Domain: %s\n", m.Domain))
	}
	sb.WriteString(fmt.Sprintf("│ 📅 Created: %s\n", f.formatTime(m.CreatedAt)))
	sb.WriteString(fmt.Sprintf("│ 🔄 Updated: %s\n", f.formatTime(m.UpdatedAt)))
	sb.WriteString(fmt.Sprintf("│ 🔗 Session: %s\n", f.truncateID(m.SessionID)))
	sb.WriteString("└──────────────────────────────────────────┘")

	return sb.String()
}

func (f *Formatter) formatUpdateMemory(result interface{}) string {
	var sb strings.Builder

	data, ok := result.(*MemoryResponse)
	if !ok {
		return f.fallbackJSON(result)
	}

	if !data.Success {
		sb.WriteString("❌ **Update Failed**\n")
		if data.Message != "" {
			sb.WriteString(fmt.Sprintf("\n%s", data.Message))
		}
		return sb.String()
	}

	sb.WriteString("✅ **Memory Updated Successfully**\n\n")
	if data.Memory != nil {
		m := data.Memory
		sb.WriteString(fmt.Sprintf("**ID:** `%s`\n\n", m.ID))
		sb.WriteString("**Current State:**\n")
		sb.WriteString("```yaml\n")
		sb.WriteString(fmt.Sprintf("content: \"%s\"\n", f.truncateContent(m.Content, 100)))
		sb.WriteString(fmt.Sprintf("importance: %d\n", m.Importance))
		if len(m.Tags) > 0 {
			sb.WriteString(fmt.Sprintf("tags: [%s]\n", strings.Join(m.Tags, ", ")))
		}
		sb.WriteString(fmt.Sprintf("updated_at: %s\n", f.formatTime(m.UpdatedAt)))
		sb.WriteString("```")
	}

	return sb.String()
}

func (f *Formatter) formatDeleteMemory(result interface{}) string {
	var sb strings.Builder

	data, ok := result.(*MemoryResponse)
	if !ok {
		return f.fallbackJSON(result)
	}

	if data.Success {
		sb.WriteString("🗑️ **Memory Deleted**\n\n")
		sb.WriteString(fmt.Sprintf("```\n%s\n```", data.Message))
	} else {
		sb.WriteString("❌ **Delete Failed**\n\n")
		sb.WriteString(fmt.Sprintf("```\n%s\n```", data.Message))
	}

	return sb.String()
}

func (f *Formatter) formatAnalysis(result interface{}) string {
	// Try question response first
	if data, ok := result.(*AnalysisQuestionResponse); ok {
		return f.formatQuestionAnalysis(data)
	}

	// Try summarize response
	if data, ok := result.(*AnalysisSummarizeResponse); ok {
		return f.formatSummarizeAnalysis(data)
	}

	return f.fallbackJSON(result)
}

func (f *Formatter) formatQuestionAnalysis(data *AnalysisQuestionResponse) string {
	var sb strings.Builder

	confidenceBar := f.makeProgressBar(data.Confidence, 10)
	confidencePercent := int(data.Confidence * 100)

	sb.WriteString("🧠 **AI Analysis Complete**\n\n")
	sb.WriteString(fmt.Sprintf("**Confidence:** %s %d%%\n\n", confidenceBar, confidencePercent))

	sb.WriteString("### Answer\n")
	sb.WriteString(fmt.Sprintf("> %s\n\n", data.Answer))

	if data.Reasoning != "" {
		sb.WriteString("### Reasoning\n")
		sb.WriteString(fmt.Sprintf("*%s*\n\n", data.Reasoning))
	}

	if len(data.Sources) > 0 {
		sb.WriteString(fmt.Sprintf("### Sources (%d memories)\n", len(data.Sources)))
		for i, src := range data.Sources {
			if i >= 3 {
				sb.WriteString(fmt.Sprintf("\n*...and %d more sources*", len(data.Sources)-3))
				break
			}
			sb.WriteString(fmt.Sprintf("- `%s`: %s\n", f.truncateID(src.ID), f.truncateContent(src.Content, 60)))
		}
	}

	return sb.String()
}

func (f *Formatter) formatSummarizeAnalysis(data *AnalysisSummarizeResponse) string {
	var sb strings.Builder

	sb.WriteString("🧠 **Knowledge Summary**\n\n")
	sb.WriteString(fmt.Sprintf("📊 Analyzed **%d memories** from timeframe: `%s`\n\n", data.MemoryCount, data.Timeframe))

	sb.WriteString("### Summary\n")
	sb.WriteString(fmt.Sprintf("> %s\n\n", data.Summary))

	if len(data.KeyThemes) > 0 {
		sb.WriteString("### Key Themes\n")
		for _, theme := range data.KeyThemes {
			sb.WriteString(fmt.Sprintf("  🔹 %s\n", theme))
		}
	}

	if len(data.Sources) > 0 {
		sb.WriteString(fmt.Sprintf("\n### Top Sources (%d)\n", len(data.Sources)))
		for i, src := range data.Sources {
			if i >= 5 {
				break
			}
			sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, f.truncateContent(src.Content, 50)))
		}
	}

	return sb.String()
}

func (f *Formatter) formatRelationships(result interface{}) string {
	// Handle discover response
	if data, ok := result.(*DiscoverRelationshipsResponse); ok {
		return f.formatDiscoverRelationships(data)
	}

	// Handle find_related (array response)
	if data, ok := result.([]FindRelatedResultLM); ok {
		return f.formatFindRelated(data)
	}

	// Handle map_graph
	if data, ok := result.(*MapGraphResponseLM); ok {
		return f.formatMapGraph(data)
	}

	// Handle create relationship
	if data, ok := result.(*RelationshipDetail); ok {
		return f.formatCreatedRelationship(data)
	}

	return f.fallbackJSON(result)
}

func (f *Formatter) formatDiscoverRelationships(data *DiscoverRelationshipsResponse) string {
	var sb strings.Builder

	sb.WriteString("🕸️ **Relationship Discovery**\n\n")
	sb.WriteString(fmt.Sprintf("⚡ Found **%d connections** in %dms\n\n", data.TotalFound, data.ProcessingTimeMs))

	if len(data.Relationships) == 0 {
		sb.WriteString("```\nNo new relationships discovered.\n```\n")
		sb.WriteString("\n💡 Add more memories to enable relationship discovery.")
		return sb.String()
	}

	for i, rel := range data.Relationships {
		sb.WriteString(fmt.Sprintf("### Connection %d: %s\n", i+1, f.formatRelType(rel.Relationship.RelationshipType)))
		strengthBar := f.makeProgressBar(rel.Relationship.Strength, 8)
		sb.WriteString(fmt.Sprintf("**Strength:** %s %.0f%%\n\n", strengthBar, rel.Relationship.Strength*100))

		sb.WriteString("```\n")
		sb.WriteString(fmt.Sprintf("FROM: %s\n", f.truncateContent(rel.SourceMemory.Content, 60)))
		sb.WriteString(fmt.Sprintf("  TO: %s\n", f.truncateContent(rel.TargetMemory.Content, 60)))
		sb.WriteString("```\n\n")

		sb.WriteString(fmt.Sprintf("*%s*\n\n", rel.Explanation))
	}

	return sb.String()
}

func (f *Formatter) formatFindRelated(data []FindRelatedResultLM) string {
	var sb strings.Builder

	sb.WriteString("🔗 **Related Memories**\n\n")

	if len(data) == 0 {
		sb.WriteString("```\nNo related memories found.\n```")
		return sb.String()
	}

	sb.WriteString(fmt.Sprintf("Found **%d** related memories:\n\n", len(data)))

	for i, r := range data {
		relevanceBar := f.makeProgressBar(r.RelevanceScore/10, 8)
		sb.WriteString(fmt.Sprintf("%d. **%s** %s\n", i+1, f.truncateContent(r.Memory.Content, 50), relevanceBar))
		sb.WriteString(fmt.Sprintf("   `%s` | Relevance: %.1f\n\n", f.truncateID(r.Memory.ID), r.RelevanceScore))
	}

	return sb.String()
}

func (f *Formatter) formatMapGraph(data *MapGraphResponseLM) string {
	var sb strings.Builder

	sb.WriteString("🗺️ **Knowledge Graph**\n\n")
	sb.WriteString(fmt.Sprintf("**Central Node:** `%s`\n", f.truncateID(data.CentralMemory.ID)))
	sb.WriteString(fmt.Sprintf("> %s\n\n", f.truncateContent(data.CentralMemory.Content, 80)))

	sb.WriteString(fmt.Sprintf("📊 **Graph Stats:** %d nodes | %d edges | Depth: %d\n\n",
		data.TotalNodes, len(data.Edges), data.Depth))

	if len(data.Nodes) > 0 {
		sb.WriteString("### Connected Nodes\n")
		for i, node := range data.Nodes {
			if i >= 5 {
				sb.WriteString(fmt.Sprintf("\n*...and %d more nodes*", len(data.Nodes)-5))
				break
			}
			sb.WriteString(fmt.Sprintf("  📍 [Distance %d] %s\n", node.Distance, f.truncateContent(node.Memory.Content, 50)))
		}
	}

	return sb.String()
}

func (f *Formatter) formatCreatedRelationship(data *RelationshipDetail) string {
	var sb strings.Builder

	sb.WriteString("✅ **Relationship Created**\n\n")
	sb.WriteString(fmt.Sprintf("**Type:** %s\n", f.formatRelType(data.RelationshipType)))
	sb.WriteString(fmt.Sprintf("**Strength:** %s %.0f%%\n\n", f.makeProgressBar(data.Strength, 8), data.Strength*100))

	sb.WriteString("```yaml\n")
	sb.WriteString(fmt.Sprintf("source: %s\n", f.truncateID(data.SourceMemoryID)))
	sb.WriteString(fmt.Sprintf("target: %s\n", f.truncateID(data.TargetMemoryID)))
	if data.Context != "" {
		sb.WriteString(fmt.Sprintf("context: \"%s\"\n", f.truncateContent(data.Context, 60)))
	}
	sb.WriteString("```")

	return sb.String()
}

func (f *Formatter) formatSessions(result interface{}) string {
	var sb strings.Builder

	data, ok := result.([]SessionInfoLM)
	if !ok {
		return f.fallbackJSON(result)
	}

	sb.WriteString("📊 **Memory Sessions**\n\n")

	if len(data) == 0 {
		sb.WriteString("```\nNo sessions found.\n```")
		return sb.String()
	}

	sb.WriteString(fmt.Sprintf("Found **%d** session(s):\n\n", len(data)))

	totalMemories := 0
	for _, s := range data {
		totalMemories += s.MemoryCount
	}

	sb.WriteString("```\n")
	sb.WriteString(fmt.Sprintf("%-40s │ %8s │ %s\n", "SESSION ID", "MEMORIES", "LAST ACTIVE"))
	sb.WriteString("─────────────────────────────────────────┼──────────┼─────────────────\n")
	for _, s := range data {
		sb.WriteString(fmt.Sprintf("%-40s │ %8d │ %s\n",
			f.truncateID(s.ID), s.MemoryCount, f.formatTime(s.LastAccessed)))
	}
	sb.WriteString("```\n\n")

	sb.WriteString(fmt.Sprintf("**Total:** %d memories across %d sessions", totalMemories, len(data)))

	return sb.String()
}

func (f *Formatter) formatDomains(result interface{}) string {
	var sb strings.Builder

	data, ok := result.([]DomainFullLM)
	if !ok {
		return f.fallbackJSON(result)
	}

	sb.WriteString("🏷️ **Knowledge Domains**\n\n")

	if len(data) == 0 {
		sb.WriteString("```\nNo domains configured.\n```\n")
		sb.WriteString("\n💡 Create domains to organize memories by topic area.")
		return sb.String()
	}

	for _, d := range data {
		sb.WriteString(fmt.Sprintf("### 📁 %s\n", d.Name))
		if d.Description != "" {
			sb.WriteString(fmt.Sprintf("*%s*\n\n", d.Description))
		}
		sb.WriteString(fmt.Sprintf("`ID: %s`\n\n", f.truncateID(d.ID)))
	}

	return sb.String()
}

func (f *Formatter) formatCategories(result interface{}) string {
	var sb strings.Builder

	data, ok := result.([]CategoryFullLM)
	if !ok {
		return f.fallbackJSON(result)
	}

	sb.WriteString("📂 **Memory Categories**\n\n")

	if len(data) == 0 {
		sb.WriteString("```\nNo categories defined.\n```\n")
		sb.WriteString("\n💡 Create categories for hierarchical memory organization.")
		return sb.String()
	}

	for _, c := range data {
		autoIcon := ""
		if c.AutoGenerated {
			autoIcon = " 🤖"
		}
		sb.WriteString(fmt.Sprintf("### 📁 %s%s\n", c.Name, autoIcon))
		if c.Description != "" {
			sb.WriteString(fmt.Sprintf("*%s*\n", c.Description))
		}
		sb.WriteString(fmt.Sprintf("Confidence Threshold: %.0f%%\n\n", c.ConfidenceThreshold*100))
	}

	return sb.String()
}

func (f *Formatter) formatStats(result interface{}) string {
	var sb strings.Builder

	data, ok := result.(*StatsResponse)
	if !ok {
		return f.fallbackJSON(result)
	}

	sb.WriteString("📈 **System Statistics**\n\n")

	// Visual stats boxes
	sb.WriteString("┌────────────────────────────────────────┐\n")
	sb.WriteString(fmt.Sprintf("│  📝 Memories:     %6d               │\n", data.MemoryCount))
	sb.WriteString(fmt.Sprintf("│  📊 Sessions:     %6d               │\n", data.SessionCount))
	sb.WriteString(fmt.Sprintf("│  📋 Stats Type:   %-20s │\n", data.StatsType))
	sb.WriteString("└────────────────────────────────────────┘\n")

	// Memory distribution visualization
	if data.MemoryCount > 0 {
		sb.WriteString("\n**Memory Distribution:**\n")
		bar := f.makeProgressBar(float64(data.MemoryCount)/100.0, 20)
		sb.WriteString(fmt.Sprintf("%s %d memories\n", bar, data.MemoryCount))
	}

	return sb.String()
}

func (f *Formatter) formatPerformance(duration time.Duration) string {
	ms := duration.Milliseconds()
	var speedIcon string
	switch {
	case ms < 100:
		speedIcon = "⚡"
	case ms < 500:
		speedIcon = "🚀"
	case ms < 1000:
		speedIcon = "✓"
	default:
		speedIcon = "🐢"
	}
	return fmt.Sprintf("%s *Completed in %dms*", speedIcon, ms)
}

func (f *Formatter) getSuggestions(toolName string, result interface{}) []string {
	suggestions := map[string][]string{
		"store_memory": {
			"Use `search` to verify the memory was indexed",
			"Use `relationships(discover)` to find connections",
			"Add more memories to build your knowledge base",
		},
		"search": {
			"Use `get_memory_by_id` for full details on a result",
			"Use `relationships(find_related)` to explore connections",
			"Try `analysis(question)` to ask questions about results",
		},
		"analysis": {
			"Refine your question for more specific answers",
			"Use `search` to find additional context",
			"Try `relationships(discover)` to map knowledge connections",
		},
		"relationships": {
			"Use `analysis` to understand relationship patterns",
			"Try `map_graph` to visualize the knowledge network",
			"Create manual relationships for important connections",
		},
	}

	if s, ok := suggestions[toolName]; ok {
		return s
	}
	return nil
}

// Helper functions

func (f *Formatter) makeProgressBar(value float64, width int) string {
	if value < 0 {
		value = 0
	}
	if value > 1 {
		value = 1
	}
	filled := int(value * float64(width))
	empty := width - filled
	return "[" + strings.Repeat("█", filled) + strings.Repeat("░", empty) + "]"
}

func (f *Formatter) truncateID(id string) string {
	if len(id) <= 12 {
		return id
	}
	return id[:8] + "..."
}

func (f *Formatter) truncateContent(content string, maxLen int) string {
	content = strings.ReplaceAll(content, "\n", " ")
	if len(content) <= maxLen {
		return content
	}
	return content[:maxLen-3] + "..."
}

func (f *Formatter) formatTime(timeStr string) string {
	// Try parsing various formats
	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05-07:00",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timeStr); err == nil {
			return t.Format("Jan 02, 2006 15:04")
		}
	}
	return timeStr
}

func (f *Formatter) boolToEmoji(b bool) string {
	if b {
		return "✅"
	}
	return "❌"
}

func (f *Formatter) formatRelType(relType string) string {
	icons := map[string]string{
		"references":  "📚 References",
		"contradicts": "⚔️ Contradicts",
		"expands":     "📈 Expands",
		"similar":     "🔄 Similar",
		"sequential":  "➡️ Sequential",
		"causes":      "💥 Causes",
		"enables":     "🔓 Enables",
	}
	if icon, ok := icons[relType]; ok {
		return icon
	}
	return relType
}

func (f *Formatter) fallbackJSON(result interface{}) string {
	jsonBytes, _ := json.MarshalIndent(result, "", "  ")
	return string(jsonBytes)
}
