package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/MycelicMemory/mycelicmemory/internal/recall"
)

func (s *Server) handleRecall(c *gin.Context) {
	var req recall.RecallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	if req.Context == "" {
		ErrorResponse(c, http.StatusBadRequest, "context is required")
		return
	}

	result, err := s.recallEngine.Recall(c.Request.Context(), &req)
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "recall failed: "+err.Error())
		return
	}

	SuccessResponse(c, "Context recall complete", result)
}
