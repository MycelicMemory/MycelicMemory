package api

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/MycelicMemory/mycelicmemory/internal/ai"
	"github.com/MycelicMemory/mycelicmemory/internal/database"
	"github.com/MycelicMemory/mycelicmemory/internal/logging"
	"github.com/MycelicMemory/mycelicmemory/internal/memory"
	"github.com/MycelicMemory/mycelicmemory/internal/ratelimit"
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

		// Relationships
		api.POST("/relationships", s.createRelationship)
		api.POST("/relationships/discover", s.discoverRelationships)

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
