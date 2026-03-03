package api

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	slackadapter "github.com/MycelicMemory/mycelicmemory/internal/adapters/slack"
	"github.com/MycelicMemory/mycelicmemory/internal/ai"
	"github.com/MycelicMemory/mycelicmemory/internal/claude"
	"github.com/MycelicMemory/mycelicmemory/internal/database"
	"github.com/MycelicMemory/mycelicmemory/internal/dbmanager"
	"github.com/MycelicMemory/mycelicmemory/internal/logging"
	"github.com/MycelicMemory/mycelicmemory/internal/memory"
	"github.com/MycelicMemory/mycelicmemory/internal/pipeline"
	"github.com/MycelicMemory/mycelicmemory/internal/ratelimit"
	"github.com/MycelicMemory/mycelicmemory/internal/recall"
	"github.com/MycelicMemory/mycelicmemory/internal/relationships"
	"github.com/MycelicMemory/mycelicmemory/internal/search"
	"github.com/MycelicMemory/mycelicmemory/pkg/config"
)

// Server represents the REST API server
type Server struct {
	router        *gin.Engine
	db            *database.Database
	config        *config.Config
	memoryService *memory.Service
	searchEngine  *search.Engine
	relService    *relationships.Service
	aiManager     *ai.Manager
	recallEngine  *recall.Engine
	pipelineQueue *pipeline.Queue
	dbManager     *dbmanager.Manager
	httpServer    *http.Server
	sessionID     string
	log           *logging.Logger
}

// NewServer creates a new REST API server
func NewServer(db *database.Database, cfg *config.Config) *Server {
	log := logging.GetLogger("api")
	log.Info("initializing REST API server")

	// Set Gin mode based on config
	if cfg.Logging.Level != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())

	// Configure CORS
	if cfg.RestAPI.CORS {
		log.Debug("enabling CORS")
		corsConfig := cors.Config{
			AllowMethods:  []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
			AllowHeaders:  []string{"Origin", "Content-Type", "Accept", "Authorization", "X-API-Key"},
			ExposeHeaders: []string{"Content-Length", "Retry-After"},
			MaxAge:        12 * time.Hour,
		}

		// Determine allowed origins
		if len(cfg.RestAPI.AllowOrigins) > 0 {
			corsConfig.AllowOrigins = cfg.RestAPI.AllowOrigins
		} else if cfg.RestAPI.APIKey != "" {
			// When auth is enabled, restrict to localhost variants
			corsConfig.AllowOrigins = []string{
				"http://localhost:*",
				"http://127.0.0.1:*",
				"https://localhost:*",
				"https://127.0.0.1:*",
				"tauri://localhost",
			}
			corsConfig.AllowWildcard = true
		} else {
			// No auth: allow all origins but without credentials
			corsConfig.AllowAllOrigins = true
		}

		router.Use(cors.New(corsConfig))
	}

	// API key authentication middleware
	if cfg.RestAPI.APIKey != "" {
		log.Info("API key authentication enabled")
		router.Use(APIKeyAuthMiddleware(cfg.RestAPI.APIKey))
	}

	// Rate limiting middleware
	if cfg.RateLimit.Enabled {
		log.Info("rate limiting enabled")
		rlCfg := &ratelimit.Config{
			Enabled: cfg.RateLimit.Enabled,
			Global: ratelimit.LimitConfig{
				RequestsPerSecond: cfg.RateLimit.Global.RequestsPerSecond,
				BurstSize:         cfg.RateLimit.Global.BurstSize,
			},
		}
		for _, tool := range cfg.RateLimit.Tools {
			rlCfg.Tools = append(rlCfg.Tools, ratelimit.ToolLimit{
				Name:              tool.Name,
				RequestsPerSecond: tool.RequestsPerSecond,
				BurstSize:         tool.BurstSize,
			})
		}
		limiter := ratelimit.NewLimiter(rlCfg)
		router.Use(RateLimitMiddleware(limiter))
	}

	// Default body size limit (1MB)
	router.Use(MaxBodySizeMiddleware(DefaultBodyLimit))

	// Create services
	memoryService := memory.NewService(db, cfg)
	searchEngine := search.NewEngine(db, cfg)
	relService := relationships.NewService(db, cfg)
	aiManager := ai.NewManager(db, cfg)

	// Connect AI manager to search engine
	searchEngine.SetAIManager(aiManager)

	// Create recall engine
	recallEngine := recall.NewEngine(db, cfg, aiManager, searchEngine, relService)

	// Initialize pipeline queue with adapters
	pipelineQueue := pipeline.NewQueue(db, relService, pipeline.DefaultQueueConfig())
	claudeReader := claude.NewReader("")
	claudeAdapter := claude.NewAdapter(claudeReader)
	pipelineQueue.RegisterAdapter(claudeAdapter)
	slackAdapter := slackadapter.NewAdapter()
	pipelineQueue.RegisterAdapter(slackAdapter)

	// Detect session ID
	strategy := memory.SessionStrategyGitDirectory
	if cfg.Session.Strategy == "manual" {
		strategy = memory.SessionStrategyManual
	}
	sessionDetector := memory.NewSessionDetector(strategy)
	sessionID := sessionDetector.DetectSessionID()

	log.Debug("session detected", "session_id", sessionID, "strategy", cfg.Session.Strategy)

	server := &Server{
		router:        router,
		db:            db,
		config:        cfg,
		memoryService: memoryService,
		searchEngine:  searchEngine,
		relService:    relService,
		aiManager:     aiManager,
		recallEngine:  recallEngine,
		pipelineQueue: pipelineQueue,
		dbManager:     dbmanager.New(cfg),
		sessionID:     sessionID,
		log:           log,
	}

	// Set up routes
	server.setupRoutes()

	return server
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	api := s.router.Group("/api/v1")
	{
		// Health
		api.GET("/health", s.healthHandler)

		// Memory Operations
		api.POST("/memories", s.createMemory)
		api.GET("/memories", s.listMemories)
		api.GET("/memories/search", s.searchMemoriesGET)
		api.POST("/memories/search", s.searchMemoriesPOST)
		api.POST("/memories/search/intelligent", s.intelligentSearch)
		api.GET("/memories/:id", s.getMemory)
		api.PUT("/memories/:id", s.updateMemory)
		api.DELETE("/memories/:id", s.deleteMemory)
		api.GET("/memories/stats", s.memoryStats)
		api.GET("/memories/:id/related", s.findRelated)
		api.GET("/memories/:id/graph", s.getGraph)

		// AI Operations
		api.POST("/analyze", s.analyze)
		api.POST("/recall", s.handleRecall)

		// Relationships
		api.GET("/relationships", s.getAllRelationships)
		api.POST("/relationships", s.createRelationship)
		api.POST("/relationships/discover", s.discoverRelationships)
		api.POST("/relationships/batch-discover", s.batchDiscoverRelationships)

		// Graph
		api.GET("/graph/stats", s.getGraphStats)

		// Categories
		api.POST("/categories", s.createCategory)
		api.GET("/categories", s.listCategories)
		api.POST("/memories/:id/categorize", s.categorizeMemory)
		api.GET("/categories/stats", s.categoryStats)

		// Domains
		api.POST("/domains", s.createDomain)
		api.GET("/domains", s.listDomains)
		api.GET("/domains/:domain/stats", s.domainStats)

		// Sessions
		api.GET("/sessions", s.listSessions)
		api.GET("/sessions/stats", s.sessionStats)

		// System Stats
		api.GET("/stats", s.systemStats)

		// Search endpoints
		api.POST("/search/tags", s.searchByTags)
		api.POST("/search/date-range", s.searchByDateRange)

		// Data Sources (Multi-source ingestion support)
		api.POST("/sources", s.createDataSource)
		api.GET("/sources", s.listDataSources)
		api.GET("/sources/:id", s.getDataSource)
		api.PATCH("/sources/:id", s.updateDataSource)
		api.DELETE("/sources/:id", s.deleteDataSource)

		// Source Control
		api.POST("/sources/:id/pause", s.pauseDataSource)
		api.POST("/sources/:id/resume", s.resumeDataSource)
		api.POST("/sources/:id/sync", s.triggerSync)

		// Ingestion (larger body size limit for bulk operations)
		api.POST("/sources/:id/ingest", MaxBodySizeMiddleware(IngestBodyLimit), s.ingestItems)

		// Source History & Stats
		api.GET("/sources/:id/history", s.getSyncHistory)
		api.GET("/sources/:id/stats", s.getSourceStats)
		api.GET("/sources/:id/memories", s.getSourceMemories)

		// Chat History (Claude Code conversations)
		api.POST("/chats/ingest", MaxBodySizeMiddleware(IngestBodyLimit), s.ingestConversations)
		api.GET("/chats", s.listChatSessions)
		api.GET("/chats/search", s.searchChatSessions)
		api.GET("/chats/projects", s.chatProjects)
		api.GET("/chats/:id", s.getChatSession)
		api.GET("/chats/:id/messages", s.getChatMessages)
		api.GET("/chats/:id/tool-calls", s.getChatToolCalls)

		// Memory tracing
		api.GET("/memories/:id/trace", s.traceMemorySource)

		// Models
		api.GET("/models", s.listModels)
		api.POST("/models/pull", s.pullModel)
		api.POST("/models/test", s.testModel)
		api.GET("/models/status", s.modelStatus)

		// Config
		api.PUT("/config/ollama", s.updateOllamaConfig)
		api.PUT("/config/qdrant", s.updateQdrantConfig)

		// Re-index
		api.POST("/memories/reindex", s.reindexMemories)

		// Seed
		api.POST("/seed", s.seedMemories)

		// Database management
		api.GET("/databases", s.listDatabases)
		api.POST("/databases", s.createDatabase)
		api.GET("/databases/:name", s.getDatabaseInfo)
		api.DELETE("/databases/:name", s.deleteDatabase)
		api.POST("/databases/:name/switch", s.switchDatabase)
		api.POST("/databases/:name/archive", s.archiveDatabase)
		api.POST("/databases/import", s.importDatabase)
		api.POST("/databases/:name/export", s.exportDatabase)
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	// Initialize AI manager
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := s.aiManager.Initialize(ctx); err != nil {
		s.log.Warn("AI initialization failed", "error", err)
	}

	// Determine port
	port := s.config.RestAPI.Port
	if s.config.RestAPI.AutoPort {
		availablePort, err := findAvailablePort(port)
		if err != nil {
			s.log.Error("failed to find available port", "error", err, "start_port", port)
			return fmt.Errorf("failed to find available port: %w", err)
		}
		port = availablePort
		s.log.Debug("found available port", "port", port)
	}

	addr := fmt.Sprintf("%s:%d", s.config.RestAPI.Host, port)

	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: s.router,
	}

	s.log.Info("starting REST API server", "address", addr)
	return s.httpServer.ListenAndServe()
}

// StartWithContext starts the HTTP server with graceful shutdown support
// It blocks until the context is cancelled or the server encounters an error
func (s *Server) StartWithContext(ctx context.Context, shutdownTimeout time.Duration) error {
	// Initialize AI manager
	initCtx, initCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer initCancel()
	if err := s.aiManager.Initialize(initCtx); err != nil {
		s.log.Warn("AI initialization failed", "error", err)
	}

	// Determine port
	port := s.config.RestAPI.Port
	if s.config.RestAPI.AutoPort {
		availablePort, err := findAvailablePort(port)
		if err != nil {
			s.log.Error("failed to find available port", "error", err, "start_port", port)
			return fmt.Errorf("failed to find available port: %w", err)
		}
		port = availablePort
		s.log.Debug("found available port", "port", port)
	}

	addr := fmt.Sprintf("%s:%d", s.config.RestAPI.Host, port)

	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: s.router,
	}

	// Channel for server errors
	errChan := make(chan error, 1)

	// Start server in goroutine
	go func() {
		s.log.Info("starting REST API server", "address", addr)
		if err := s.httpServer.ListenAndServe(); err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	// Wait for context cancellation or server error
	select {
	case <-ctx.Done():
		s.log.Info("shutdown signal received")
		// Graceful shutdown with timeout
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer shutdownCancel()
		return s.Stop(shutdownCtx)
	case err := <-errChan:
		return fmt.Errorf("server error: %w", err)
	}
}

// Stop gracefully stops the server
func (s *Server) Stop(ctx context.Context) error {
	s.log.Info("stopping REST API server")
	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(ctx); err != nil {
			s.log.Error("server shutdown error", "error", err)
			return err
		}
		s.log.Info("REST API server stopped")
	}
	return nil
}

// ReloadDatabase hot-swaps the database connection to the currently active database.
// Called after SwitchDatabase to apply the change without process restart.
func (s *Server) ReloadDatabase() error {
	newPath := s.config.GetActiveDBPath()
	s.log.Info("reloading database", "path", newPath)

	// Open new database
	newDB, err := database.Open(newPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	if err := newDB.InitSchema(); err != nil {
		newDB.Close()
		return fmt.Errorf("failed to init schema: %w", err)
	}
	if err := newDB.RunMigrations(); err != nil {
		newDB.Close()
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// Create new service instances pointing to the new DB
	memoryService := memory.NewService(newDB, s.config)
	searchEngine := search.NewEngine(newDB, s.config)
	relService := relationships.NewService(newDB, s.config)
	aiManager := ai.NewManager(newDB, s.config)
	searchEngine.SetAIManager(aiManager)
	recallEngine := recall.NewEngine(newDB, s.config, aiManager, searchEngine, relService)

	pipelineQueue := pipeline.NewQueue(newDB, relService, pipeline.DefaultQueueConfig())
	claudeReader := claude.NewReader("")
	claudeAdapter := claude.NewAdapter(claudeReader)
	pipelineQueue.RegisterAdapter(claudeAdapter)
	slackAdapter := slackadapter.NewAdapter()
	pipelineQueue.RegisterAdapter(slackAdapter)

	// Hold reference to old DB before swapping
	oldDB := s.db

	// Swap all service references
	s.db = newDB
	s.memoryService = memoryService
	s.searchEngine = searchEngine
	s.relService = relService
	s.aiManager = aiManager
	s.recallEngine = recallEngine
	s.pipelineQueue = pipelineQueue

	// Initialize new AI manager (non-fatal)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := s.aiManager.Initialize(ctx); err != nil {
		s.log.Warn("AI initialization failed after reload", "error", err)
	}

	// Close old database
	if oldDB != nil {
		oldDB.Close()
	}

	s.log.Info("database reloaded successfully", "path", newPath)
	return nil
}

// Router returns the underlying Gin router for testing
func (s *Server) Router() *gin.Engine {
	return s.router
}

// findAvailablePort finds an available port starting from the given port
func findAvailablePort(startPort int) (int, error) {
	for port := startPort; port < startPort+100; port++ {
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			ln.Close()
			return port, nil
		}
	}
	return 0, fmt.Errorf("no available port found in range %d-%d", startPort, startPort+100)
}
