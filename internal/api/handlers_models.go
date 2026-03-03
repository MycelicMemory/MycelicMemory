package api

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/MycelicMemory/mycelicmemory/internal/database"
)

// listModels handles GET /api/v1/models
func (s *Server) listModels(c *gin.Context) {
	if s.aiManager == nil {
		InternalError(c, "AI services not configured")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	models, err := s.aiManager.GetModels(ctx)
	if err != nil {
		InternalError(c, "Failed to list models: "+err.Error())
		return
	}

	SuccessResponse(c, "Models listed", map[string]interface{}{
		"models": models,
	})
}

// pullModel handles POST /api/v1/models/pull
func (s *Server) pullModel(c *gin.Context) {
	if s.aiManager == nil {
		InternalError(c, "AI services not configured")
		return
	}

	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequestError(c, "Invalid request: "+err.Error())
		return
	}

	// Use a longer timeout for model pulls
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Minute)
	defer cancel()

	if err := s.aiManager.PullModel(ctx, req.Name); err != nil {
		InternalError(c, "Failed to pull model: "+err.Error())
		return
	}

	SuccessResponse(c, "Model pulled successfully", map[string]string{
		"model": req.Name,
	})
}

// testModel handles POST /api/v1/models/test
func (s *Server) testModel(c *gin.Context) {
	if s.aiManager == nil {
		InternalError(c, "AI services not configured")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	ok, message, err := s.aiManager.TestConnectivity(ctx)
	if err != nil {
		InternalError(c, "Test failed: "+err.Error())
		return
	}

	SuccessResponse(c, message, map[string]interface{}{
		"connected": ok,
		"message":   message,
	})
}

// modelStatus handles GET /api/v1/models/status
func (s *Server) modelStatus(c *gin.Context) {
	if s.aiManager == nil {
		SuccessResponse(c, "AI not configured", map[string]interface{}{
			"configured": false,
		})
		return
	}

	status := s.aiManager.GetStatus()
	SuccessResponse(c, "Model status", status)
}

// updateOllamaConfig handles PUT /api/v1/config/ollama
func (s *Server) updateOllamaConfig(c *gin.Context) {
	var req struct {
		BaseURL        string `json:"base_url"`
		EmbeddingModel string `json:"embedding_model"`
		ChatModel      string `json:"chat_model"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequestError(c, "Invalid request: "+err.Error())
		return
	}

	reindexRequired := false

	if req.BaseURL != "" {
		s.config.Ollama.BaseURL = req.BaseURL
	}
	if req.EmbeddingModel != "" && req.EmbeddingModel != s.config.Ollama.EmbeddingModel {
		s.config.Ollama.EmbeddingModel = req.EmbeddingModel
		reindexRequired = true
	}
	if req.ChatModel != "" {
		s.config.Ollama.ChatModel = req.ChatModel
	}

	if err := s.config.Save(); err != nil {
		InternalError(c, "Failed to save config: "+err.Error())
		return
	}

	SuccessResponse(c, "Ollama config updated", map[string]interface{}{
		"reindex_required": reindexRequired,
	})
}

// updateQdrantConfig handles PUT /api/v1/config/qdrant
func (s *Server) updateQdrantConfig(c *gin.Context) {
	var req struct {
		URL    string `json:"url"`
		APIKey string `json:"api_key"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequestError(c, "Invalid request: "+err.Error())
		return
	}

	if req.URL != "" {
		s.config.Qdrant.URL = req.URL
	}
	if req.APIKey != "" {
		s.config.Qdrant.APIKey = req.APIKey
	}

	if err := s.config.Save(); err != nil {
		InternalError(c, "Failed to save config: "+err.Error())
		return
	}

	SuccessResponse(c, "Qdrant config updated", nil)
}

// reindexMemories handles POST /api/v1/memories/reindex
func (s *Server) reindexMemories(c *gin.Context) {
	if s.aiManager == nil {
		InternalError(c, "AI services not configured")
		return
	}

	// Fetch all memories
	memories, err := s.db.ListMemories(&database.MemoryFilters{Limit: 10000})
	if err != nil {
		InternalError(c, "Failed to list memories: "+err.Error())
		return
	}

	ctx := c.Request.Context()
	indexed, errors := s.aiManager.BatchIndexMemories(ctx, memories)

	SuccessResponse(c, "Re-indexing complete", map[string]interface{}{
		"indexed": indexed,
		"errors":  len(errors),
		"total":   len(memories),
	})
}
