package locomo

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/MycelicMemory/ultrathink/internal/database"
	"github.com/MycelicMemory/ultrathink/internal/logging"
)

var log = logging.GetLogger("locomo")

const (
	// LoCoMoDataURL is the URL to download the LoCoMo dataset
	LoCoMoDataURL = "https://raw.githubusercontent.com/snap-research/locomo/main/data/locomo10.json"

	// LoCoMoDomain is the domain used for LoCoMo memories
	LoCoMoDomain = "locomo-benchmark"

	// DefaultSessionPrefix is used to create unique sessions per conversation
	DefaultSessionPrefix = "locomo-conv-"
)

// Ingester handles ingestion of LoCoMo data into ultrathink
type Ingester struct {
	db *database.Database

	// Track dialogue ID to memory ID mapping for evidence retrieval
	dialogueToMemory map[string]string // "conv_1:dia_1_1" -> memory_id
}

// NewIngester creates a new LoCoMo ingester
func NewIngester(db *database.Database) *Ingester {
	return &Ingester{
		db:               db,
		dialogueToMemory: make(map[string]string),
	}
}

// LoadDataset loads the LoCoMo dataset from a file or URL
func LoadDataset(path string) (*Dataset, error) {
	var data []byte
	var err error

	if path == "" || path == "auto" {
		log.Info("downloading LoCoMo dataset from GitHub", "url", LoCoMoDataURL)
		data, err = downloadDataset(LoCoMoDataURL)
	} else if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		log.Info("downloading LoCoMo dataset", "url", path)
		data, err = downloadDataset(path)
	} else {
		log.Info("loading LoCoMo dataset from file", "path", path)
		data, err = os.ReadFile(path)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to load dataset: %w", err)
	}

	// The LoCoMo dataset is an array of conversations, not an object with "conversations" key
	var conversations []Conversation
	if err := json.Unmarshal(data, &conversations); err != nil {
		// Try parsing as Dataset struct first (in case format changes)
		var dataset Dataset
		if err2 := json.Unmarshal(data, &dataset); err2 == nil {
			log.Info("loaded LoCoMo dataset", "conversations", len(dataset.Conversations))
			return &dataset, nil
		}
		return nil, fmt.Errorf("failed to parse dataset: %w", err)
	}

	dataset := &Dataset{Conversations: conversations}
	log.Info("loaded LoCoMo dataset", "conversations", len(dataset.Conversations))
	return dataset, nil
}

func downloadDataset(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %s", resp.Status)
	}

	return io.ReadAll(resp.Body)
}

// Ingest ingests all conversations from the dataset into ultrathink
func (i *Ingester) Ingest(dataset *Dataset) (*IngestionResult, error) {
	start := time.Now()
	result := &IngestionResult{}

	// Create the LoCoMo domain if it doesn't exist
	if err := i.ensureDomain(); err != nil {
		return nil, err
	}

	for _, conv := range dataset.Conversations {
		convResult, err := i.IngestConversation(&conv)
		if err != nil {
			log.Error("failed to ingest conversation", "id", conv.ID, "error", err)
			continue
		}

		result.ConversationsIngested++
		result.TotalTurns += convResult.TotalTurns
		result.TotalMemories += convResult.TotalMemories
		result.PersonaMemories += convResult.PersonaMemories
		result.TotalQAQuestions += len(conv.QA)
	}

	result.Duration = time.Since(start)
	log.Info("ingestion complete",
		"conversations", result.ConversationsIngested,
		"memories", result.TotalMemories,
		"qa_questions", result.TotalQAQuestions,
		"duration", result.Duration)

	return result, nil
}

// IngestConversation ingests a single conversation
func (i *Ingester) IngestConversation(conv *Conversation) (*IngestionResult, error) {
	log.Debug("ingesting conversation", "id", conv.ID, "speakers", []string{conv.SpeakerA, conv.SpeakerB})

	result := &IngestionResult{}
	sessionID := DefaultSessionPrefix + conv.ID

	// Ensure session exists
	if err := i.db.EnsureSession(sessionID, "benchmark"); err != nil {
		log.Warn("failed to ensure session", "session_id", sessionID, "error", err)
	}

	// Ingest personas as high-importance memories
	for speaker, traits := range conv.Personas {
		for _, trait := range traits {
			mem := &database.Memory{
				Content:    fmt.Sprintf("[Persona - %s] %s", speaker, trait),
				Source:     "locomo-persona",
				Importance: 10, // Maximum importance for persona information
				Tags:       []string{"locomo", "persona", "conv_" + conv.ID, speaker},
				SessionID:  sessionID,
				Domain:     LoCoMoDomain,
				AgentType:  "benchmark",
			}

			if err := i.db.CreateMemory(mem); err != nil {
				return nil, fmt.Errorf("failed to store persona: %w", err)
			}
			result.PersonaMemories++
			result.TotalMemories++
		}
	}

	// Get sorted session keys for chronological order
	sessionKeys := make([]string, 0, len(conv.Sessions))
	for k := range conv.Sessions {
		sessionKeys = append(sessionKeys, k)
	}
	sort.Strings(sessionKeys)

	// Ingest dialogue turns
	for _, sessionKey := range sessionKeys {
		turns := conv.Sessions[sessionKey]

		// Extract session number
		sessionNum := extractSessionNumber(sessionKey)

		// Get session date if available
		dateKey := sessionKey + "_date_time"
		sessionDate := conv.SessionDates[dateKey]
		var createdAt time.Time
		if sessionDate != "" {
			if t, err := parseSessionDate(sessionDate); err == nil {
				createdAt = t
			}
		}

		for _, turn := range turns {
			result.TotalTurns++

			// Calculate importance based on content
			importance := calculateTurnImportance(turn)

			content := formatTurnContent(turn, conv)

			mem := &database.Memory{
				Content:    content,
				Source:     fmt.Sprintf("locomo-%s-%s", conv.ID, turn.DiaID),
				Importance: importance,
				Tags:       buildTurnTags(conv.ID, sessionNum, turn),
				SessionID:  sessionID,
				Domain:     LoCoMoDomain,
				AgentType:  "benchmark",
			}

			if !createdAt.IsZero() {
				mem.CreatedAt = createdAt
			}

			if err := i.db.CreateMemory(mem); err != nil {
				return nil, fmt.Errorf("failed to store turn: %w", err)
			}

			// Track dialogue ID mapping
			mapKey := fmt.Sprintf("%s:%s", conv.ID, turn.DiaID)
			i.dialogueToMemory[mapKey] = mem.ID

			result.TotalMemories++
		}
	}

	return result, nil
}

// GetMemoryForDialogue returns the memory ID for a dialogue turn
func (i *Ingester) GetMemoryForDialogue(convID, diaID string) (string, bool) {
	key := fmt.Sprintf("%s:%s", convID, diaID)
	memID, ok := i.dialogueToMemory[key]
	return memID, ok
}

// GetDialogueMapping returns the full dialogue to memory mapping
func (i *Ingester) GetDialogueMapping() map[string]string {
	return i.dialogueToMemory
}

func (i *Ingester) ensureDomain() error {
	domains, err := i.db.ListDomains()
	if err != nil {
		return err
	}

	for _, d := range domains {
		if d.Name == LoCoMoDomain {
			return nil // Already exists
		}
	}

	return i.db.CreateDomain(&database.Domain{
		Name:        LoCoMoDomain,
		Description: "LoCoMo benchmark evaluation data",
	})
}

func extractSessionNumber(sessionKey string) int {
	// Parse "session_1" -> 1
	parts := strings.Split(sessionKey, "_")
	if len(parts) >= 2 {
		if n, err := strconv.Atoi(parts[1]); err == nil {
			return n
		}
	}
	return 0
}

func parseSessionDate(dateStr string) (time.Time, error) {
	// Try common formats from LoCoMo dataset
	formats := []string{
		"January 2, 2006",
		"Jan 2, 2006",
		"2006-01-02",
		"2006-01-02 15:04:05",
		time.RFC3339,
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
}

func formatTurnContent(turn Turn, conv *Conversation) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("[%s] %s", turn.Speaker, turn.Content))

	// Include image information if present
	if turn.ImageURL != "" {
		sb.WriteString(fmt.Sprintf(" [shared image: %s]", turn.ImageCaption))
	}

	return sb.String()
}

func calculateTurnImportance(turn Turn) int {
	importance := 5 // Base importance

	// Longer messages tend to be more important
	words := len(strings.Fields(turn.Content))
	if words > 50 {
		importance++
	}
	if words > 100 {
		importance++
	}

	// Images often contain important information
	if turn.ImageURL != "" {
		importance += 2
	}

	// Cap at 9 (10 is reserved for personas)
	if importance > 9 {
		importance = 9
	}

	return importance
}

func buildTurnTags(convID string, sessionNum int, turn Turn) []string {
	tags := []string{
		"locomo",
		"conv_" + convID,
		fmt.Sprintf("session_%d", sessionNum),
		turn.Speaker,
		turn.DiaID,
	}

	if turn.ImageURL != "" {
		tags = append(tags, "has_image")
	}

	return tags
}

// ClearBenchmarkData removes all LoCoMo benchmark data from the database
func (i *Ingester) ClearBenchmarkData() error {
	log.Info("clearing LoCoMo benchmark data")

	// Delete all memories in the locomo-benchmark domain
	_, err := i.db.Exec(`DELETE FROM memories WHERE domain = ?`, LoCoMoDomain)
	if err != nil {
		return fmt.Errorf("failed to clear benchmark memories: %w", err)
	}

	// Clear the mapping
	i.dialogueToMemory = make(map[string]string)

	log.Info("benchmark data cleared")
	return nil
}

// GetConversationMemories retrieves all memories for a specific conversation
func (i *Ingester) GetConversationMemories(convID string) ([]*database.Memory, error) {
	tag := "conv_" + convID
	return i.db.ListMemories(&database.MemoryFilters{
		Domain: LoCoMoDomain,
		Tags:   []string{tag},
		Limit:  1000, // Conversations can have many turns
	})
}

// GetEvidenceMemories retrieves memories matching the evidence dialogue IDs
func (i *Ingester) GetEvidenceMemories(convID string, evidence []string) ([]*database.Memory, error) {
	var memories []*database.Memory

	for _, diaID := range evidence {
		if memID, ok := i.GetMemoryForDialogue(convID, diaID); ok {
			if mem, err := i.db.GetMemory(memID); err == nil && mem != nil {
				memories = append(memories, mem)
			}
		}
	}

	return memories, nil
}

// BenchmarkStatus returns the current status of ingested benchmark data
type BenchmarkStatus struct {
	Ingested          bool
	ConversationCount int
	MemoryCount       int
	QAQuestionCount   int
}

// GetStatus returns the current benchmark ingestion status
func (i *Ingester) GetStatus() (*BenchmarkStatus, error) {
	status := &BenchmarkStatus{}

	// Count memories in the locomo domain
	stats, err := i.db.GetDomainStats(LoCoMoDomain)
	if err != nil {
		return nil, err
	}

	status.MemoryCount = stats.MemoryCount
	status.Ingested = stats.MemoryCount > 0

	// Count distinct conversations
	var convCount int
	err = i.db.QueryRow(`
		SELECT COUNT(DISTINCT session_id)
		FROM memories
		WHERE domain = ?
	`, LoCoMoDomain).Scan(&convCount)
	if err == nil {
		status.ConversationCount = convCount
	}

	return status, nil
}
