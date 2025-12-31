package api

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/MycelicMemory/ultrathink/internal/ai"
	"github.com/MycelicMemory/ultrathink/internal/database"
	"github.com/MycelicMemory/ultrathink/internal/memory"
	"github.com/MycelicMemory/ultrathink/internal/relationships"
	"github.com/MycelicMemory/ultrathink/internal/search"
	"github.com/MycelicMemory/ultrathink/pkg/config"
)

// Server represents the REST API server
type Server struct {
	router       *gin.Engine
	db           *database.Database
	config       *config.Config
	memoryService *memory.Service
	searchEngine *search.Engine
	relService   *relationships.Service
	aiManager    *ai.Manager
	httpServer   *http.Server
	sessionID    string
}

// NewServer creates a new REST API server
func NewServer(db *database.Database, cfg *config.Config) *Server {
	// Set Gin mode based on config
	if cfg.Logging.Level != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())

	// Configure CORS
	if cfg.RestAPI.CORS {
		router.Use(cors.New(cors.Config{
			AllowOrigins:     []string{"*"},
			AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
			ExposeHeaders:    []string{"Content-Length"},
			AllowCredentials: true,
			MaxAge:           12 * time.Hour,
		}))
	}

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

	server := &Server{
		router:        router,
		db:            db,
		config:        cfg,
		memoryService: memoryService,
		searchEngine:  searchEngine,
		relService:    relService,
		aiManager:     aiManager,
		sessionID:     sessionID,
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
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	// Initialize AI manager
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := s.aiManager.Initialize(ctx); err != nil {
		// Log warning but continue
		fmt.Printf("Warning: AI initialization failed: %v\n", err)
	}

	// Determine port
	port := s.config.RestAPI.Port
	if s.config.RestAPI.AutoPort {
		availablePort, err := findAvailablePort(port)
		if err != nil {
			return fmt.Errorf("failed to find available port: %w", err)
		}
		port = availablePort
	}

	addr := fmt.Sprintf("%s:%d", s.config.RestAPI.Host, port)

	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: s.router,
	}

	fmt.Printf("Starting REST API server on %s\n", addr)
	return s.httpServer.ListenAndServe()
}

// Stop gracefully stops the server
func (s *Server) Stop(ctx context.Context) error {
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
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
