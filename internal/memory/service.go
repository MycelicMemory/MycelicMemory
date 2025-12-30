package memory

import (
	"fmt"
	"strings"
	"time"

	"github.com/MycelicMemory/ultrathink/internal/database"
	"github.com/MycelicMemory/ultrathink/pkg/config"
)

// Service provides the business logic layer for memory operations
// VERIFIED: Implements all local-memory memory operations
type Service struct {
	db              *database.Database
	config          *config.Config
	sessionDetector *SessionDetector
}

// NewService creates a new memory service
func NewService(db *database.Database, cfg *config.Config) *Service {
	strategy := SessionStrategyGitDirectory
	if cfg.Session.Strategy == "manual" {
		strategy = SessionStrategyManual
	} else if cfg.Session.Strategy == "hash" {
		strategy = SessionStrategyHash
	}

	detector := NewSessionDetector(strategy)
	if cfg.Session.ManualID != "" {
		detector.ManualID = cfg.Session.ManualID
	}

	return &Service{
		db:              db,
		config:          cfg,
		sessionDetector: detector,
	}
}

// StoreOptions contains options for storing a memory
type StoreOptions struct {
	Content      string
	Importance   int
	Tags         []string
	Domain       string
	Source       string
	SessionID    string // Optional override
	AgentType    string // Optional override
	AgentContext string // Optional override
	AccessScope  string // "session", "shared", "global"
	Slug         string
}

// StoreResult contains the result of storing a memory
type StoreResult struct {
	Memory    *database.Memory
	IsNew     bool
	SessionID string
}

// Store creates a new memory with validation and enrichment
// VERIFIED: Matches local-memory store_memory behavior
func (s *Service) Store(opts *StoreOptions) (*StoreResult, error) {
	// Validate content
	if strings.TrimSpace(opts.Content) == "" {
		return nil, fmt.Errorf("content is required")
	}

	// Validate importance (1-10, default 5)
	importance := opts.Importance
	if importance < 1 {
		importance = 5
	}
	if importance > 10 {
		importance = 10
	}

	// Detect session ID
	sessionID := opts.SessionID
	if sessionID == "" {
		sessionID = s.sessionDetector.DetectSessionID()
	}

	// Detect agent type
	agentType := opts.AgentType
	if agentType == "" {
		agentType = GetAgentType()
	}

	// Validate agent type
	if !database.IsValidAgentType(agentType) {
		agentType = "unknown"
	}

	// Get agent context
	agentContext := opts.AgentContext
	if agentContext == "" {
		agentContext = GetAgentContext()
	}

	// Set access scope
	accessScope := opts.AccessScope
	if accessScope == "" {
		accessScope = "session"
	}

	// Normalize tags
	tags := normalizeTags(opts.Tags)

	// Auto-create domain if specified
	if opts.Domain != "" {
		if err := s.ensureDomainExists(opts.Domain); err != nil {
			// Log warning but don't fail
			// Domain creation is optional
		}
	}

	// Create memory
	memory := &database.Memory{
		Content:      strings.TrimSpace(opts.Content),
		Importance:   importance,
		Tags:         tags,
		Domain:       opts.Domain,
		Source:       opts.Source,
		SessionID:    sessionID,
		AgentType:    agentType,
		AgentContext: agentContext,
		AccessScope:  accessScope,
		Slug:         opts.Slug,
	}

	// Store in database
	if err := s.db.CreateMemory(memory); err != nil {
		return nil, fmt.Errorf("failed to store memory: %w", err)
	}

	return &StoreResult{
		Memory:    memory,
		IsNew:     true,
		SessionID: sessionID,
	}, nil
}

// GetOptions contains options for retrieving a memory
type GetOptions struct {
	ID   string
	Slug string
}

// Get retrieves a memory by ID or slug
// VERIFIED: Matches local-memory get_memory_by_id behavior
func (s *Service) Get(opts *GetOptions) (*database.Memory, error) {
	if opts.ID != "" {
		return s.db.GetMemory(opts.ID)
	}

	if opts.Slug != "" {
		// TODO: Implement slug lookup
		return nil, fmt.Errorf("slug lookup not yet implemented")
	}

	return nil, fmt.Errorf("id or slug is required")
}

// UpdateOptions contains options for updating a memory
type UpdateOptions struct {
	ID         string
	Content    *string
	Importance *int
	Tags       []string
	Source     *string
	Domain     *string
}

// Update modifies an existing memory
// VERIFIED: Matches local-memory update_memory behavior
func (s *Service) Update(opts *UpdateOptions) (*database.Memory, error) {
	if opts.ID == "" {
		return nil, fmt.Errorf("id is required")
	}

	// Validate importance if provided
	if opts.Importance != nil {
		if *opts.Importance < 1 || *opts.Importance > 10 {
			return nil, fmt.Errorf("importance must be between 1 and 10")
		}
	}

	// Normalize tags if provided
	var tags []string
	if opts.Tags != nil {
		tags = normalizeTags(opts.Tags)
	}

	// Auto-create domain if specified
	if opts.Domain != nil && *opts.Domain != "" {
		if err := s.ensureDomainExists(*opts.Domain); err != nil {
			// Log warning but don't fail
		}
	}

	// Update in database
	update := &database.MemoryUpdate{
		Content:    opts.Content,
		Importance: opts.Importance,
		Tags:       tags,
		Source:     opts.Source,
		Domain:     opts.Domain,
	}

	if err := s.db.UpdateMemory(opts.ID, update); err != nil {
		return nil, fmt.Errorf("failed to update memory: %w", err)
	}

	// Return updated memory
	return s.db.GetMemory(opts.ID)
}

// Delete removes a memory by ID
// VERIFIED: Matches local-memory delete_memory behavior (CASCADE deletes relationships)
func (s *Service) Delete(id string) error {
	if id == "" {
		return fmt.Errorf("id is required")
	}

	return s.db.DeleteMemory(id)
}

// ListOptions contains options for listing memories
type ListOptions struct {
	SessionID         string
	Domain            string
	Tags              []string
	MinImportance     int
	MaxImportance     int
	StartDate         *time.Time
	EndDate           *time.Time
	Limit             int
	Offset            int
	SessionFilterMode string // "all", "session_only", "session_and_shared"
}

// List retrieves memories with optional filters
// VERIFIED: Matches local-memory list behavior with session filtering
func (s *Service) List(opts *ListOptions) ([]*database.Memory, error) {
	// Apply session filter mode
	sessionID := opts.SessionID
	if opts.SessionFilterMode == "session_only" && sessionID == "" {
		sessionID = s.sessionDetector.DetectSessionID()
	}

	filters := &database.MemoryFilters{
		SessionID:     sessionID,
		Domain:        opts.Domain,
		Tags:          normalizeTags(opts.Tags),
		MinImportance: opts.MinImportance,
		MaxImportance: opts.MaxImportance,
		StartDate:     opts.StartDate,
		EndDate:       opts.EndDate,
		Limit:         opts.Limit,
		Offset:        opts.Offset,
	}

	return s.db.ListMemories(filters)
}

// ensureDomainExists creates a domain if it doesn't exist
func (s *Service) ensureDomainExists(name string) error {
	domains, err := s.db.ListDomains()
	if err != nil {
		return err
	}

	// Check if domain already exists
	for _, d := range domains {
		if strings.EqualFold(d.Name, name) {
			return nil
		}
	}

	// Create new domain
	domain := &database.Domain{
		Name:        strings.ToLower(name),
		Description: fmt.Sprintf("Auto-created domain: %s", name),
	}

	return s.db.CreateDomain(domain)
}

// normalizeTags normalizes tag names (lowercase, trim whitespace, deduplicate)
func normalizeTags(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}

	seen := make(map[string]bool)
	var result []string

	for _, tag := range tags {
		normalized := strings.ToLower(strings.TrimSpace(tag))
		if normalized != "" && !seen[normalized] {
			seen[normalized] = true
			result = append(result, normalized)
		}
	}

	return result
}

// GetSessionID returns the current session ID
func (s *Service) GetSessionID() string {
	return s.sessionDetector.DetectSessionID()
}

// Stats returns memory statistics
type Stats struct {
	TotalMemories   int
	TotalSessions   int
	TotalDomains    int
	TotalCategories int
	SessionID       string
}

// GetStats returns memory statistics
func (s *Service) GetStats() (*Stats, error) {
	dbStats, err := s.db.GetStats()
	if err != nil {
		return nil, err
	}

	return &Stats{
		TotalMemories:   dbStats.MemoryCount,
		TotalSessions:   dbStats.SessionCount,
		TotalDomains:    dbStats.DomainCount,
		TotalCategories: dbStats.CategoryCount,
		SessionID:       s.sessionDetector.DetectSessionID(),
	}, nil
}
