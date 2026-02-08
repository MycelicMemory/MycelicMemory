package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/MycelicMemory/mycelicmemory/internal/claude"
	"github.com/MycelicMemory/mycelicmemory/internal/database"
)

// ingestConversations handles POST /api/v1/chats/ingest
func (s *Server) ingestConversations(c *gin.Context) {
	var req struct {
		ProjectPath     string `json:"project_path"`
		CreateSummaries bool   `json:"create_summaries"`
		MinMessages     int    `json:"min_messages"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		// Allow empty body with defaults
		req.CreateSummaries = true
		req.MinMessages = 3
	}

	if req.MinMessages <= 0 {
		req.MinMessages = 3
	}

	reader := claude.NewReader("")
	ingester := claude.NewIngester(reader, s.db, s.relService)

	result, err := ingester.IngestAll(c.Request.Context(), &claude.IngestOptions{
		ProjectPath:     req.ProjectPath,
		MinMessages:     req.MinMessages,
		CreateSummaries: req.CreateSummaries,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Conversations ingested successfully",
		"data":    result,
	})
}

// listChatSessions handles GET /api/v1/chats
func (s *Server) listChatSessions(c *gin.Context) {
	projectPath := c.Query("project_path")
	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)

	if limit <= 0 {
		limit = 50
	}

	sessions, err := s.db.ListCCSessions(&database.CCSessionFilters{
		ProjectPath: projectPath,
		Limit:       limit,
		Offset:      offset,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Listed chat sessions",
		"data":    sessions,
		"count":   len(sessions),
	})
}

// searchChatSessions handles GET /api/v1/chats/search
func (s *Server) searchChatSessions(c *gin.Context) {
	query := c.Query("query")
	projectPath := c.Query("project_path")
	limitStr := c.DefaultQuery("limit", "20")

	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 {
		limit = 20
	}

	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "query parameter is required",
		})
		return
	}

	// Search sessions by title/first_prompt
	sessions, err := s.db.ListCCSessions(&database.CCSessionFilters{
		ProjectPath: projectPath,
		Limit:       100,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Search messages
	messages, err := s.db.SearchCCMessages(query, projectPath, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Build results
	type result struct {
		Session   *database.CCSession `json:"session"`
		MatchType string              `json:"match_type"`
		Snippet   string              `json:"snippet,omitempty"`
	}

	var results []result
	seen := make(map[string]bool)

	// Match sessions
	for _, sess := range sessions {
		if containsInsensitive(sess.Title, query) || containsInsensitive(sess.FirstPrompt, query) {
			seen[sess.ID] = true
			results = append(results, result{
				Session:   sess,
				MatchType: "session",
			})
		}
	}

	// Match from messages
	for _, msg := range messages {
		if seen[msg.SessionID] {
			continue
		}
		seen[msg.SessionID] = true

		sess, err := s.db.GetCCSession(msg.SessionID)
		if err != nil {
			continue
		}

		snippet := msg.Content
		if len(snippet) > 200 {
			snippet = snippet[:200] + "..."
		}

		results = append(results, result{
			Session:   sess,
			MatchType: "message",
			Snippet:   snippet,
		})

		if len(results) >= limit {
			break
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    results,
		"count":   len(results),
	})
}

// getChatSession handles GET /api/v1/chats/:id
func (s *Server) getChatSession(c *gin.Context) {
	id := c.Param("id")

	sess, err := s.db.GetCCSession(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Session not found",
		})
		return
	}

	response := gin.H{
		"success": true,
		"data": gin.H{
			"session": sess,
		},
	}

	// Include messages if requested
	if c.DefaultQuery("include_messages", "true") == "true" {
		messages, err := s.db.GetCCMessages(id, 0, 0)
		if err == nil {
			response["data"].(gin.H)["messages"] = messages
			response["data"].(gin.H)["message_count"] = len(messages)
		}
	}

	// Include tool calls if requested
	if c.DefaultQuery("include_tool_calls", "false") == "true" {
		toolCalls, err := s.db.GetCCToolCalls(id)
		if err == nil {
			response["data"].(gin.H)["tool_calls"] = toolCalls
			response["data"].(gin.H)["tool_call_count"] = len(toolCalls)
		}
	}

	// Include linked memories
	memories, err := s.db.GetSessionMemories(id, 50, 0)
	if err == nil && len(memories) > 0 {
		response["data"].(gin.H)["linked_memories"] = memories
	}

	c.JSON(http.StatusOK, response)
}

// getChatMessages handles GET /api/v1/chats/:id/messages
func (s *Server) getChatMessages(c *gin.Context) {
	id := c.Param("id")
	limitStr := c.DefaultQuery("limit", "0")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)

	messages, err := s.db.GetCCMessages(id, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    messages,
		"count":   len(messages),
	})
}

// getChatToolCalls handles GET /api/v1/chats/:id/tool-calls
func (s *Server) getChatToolCalls(c *gin.Context) {
	id := c.Param("id")

	toolCalls, err := s.db.GetCCToolCalls(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    toolCalls,
		"count":   len(toolCalls),
	})
}

// traceMemorySource handles GET /api/v1/memories/:id/trace
func (s *Server) traceMemorySource(c *gin.Context) {
	id := c.Param("id")

	mem, err := s.db.GetMemory(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Memory not found",
		})
		return
	}

	if mem.CCSessionID == "" {
		c.JSON(http.StatusOK, gin.H{
			"success":    true,
			"has_source": false,
			"message":    "Memory has no linked conversation",
		})
		return
	}

	sess, err := s.db.GetCCSession(mem.CCSessionID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success":    true,
			"has_source": false,
			"message":    "Linked session not found",
		})
		return
	}

	messages, _ := s.db.GetCCMessages(mem.CCSessionID, 20, 0)

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"has_source": true,
		"session":    sess,
		"messages":   messages,
	})
}

// chatProjects handles GET /api/v1/chats/projects
func (s *Server) chatProjects(c *gin.Context) {
	// Get unique project paths from ingested sessions
	sessions, err := s.db.ListCCSessions(&database.CCSessionFilters{
		Limit: 1000,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	type projectInfo struct {
		ProjectPath  string `json:"project_path"`
		ProjectHash  string `json:"project_hash"`
		SessionCount int    `json:"session_count"`
	}

	projectMap := make(map[string]*projectInfo)
	for _, sess := range sessions {
		if p, ok := projectMap[sess.ProjectHash]; ok {
			p.SessionCount++
		} else {
			projectMap[sess.ProjectHash] = &projectInfo{
				ProjectPath:  sess.ProjectPath,
				ProjectHash:  sess.ProjectHash,
				SessionCount: 1,
			}
		}
	}

	var projects []projectInfo
	for _, p := range projectMap {
		projects = append(projects, *p)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    projects,
		"count":   len(projects),
	})
}

// containsInsensitive checks if s contains substr (case-insensitive)
func containsInsensitive(s, substr string) bool {
	return len(s) >= len(substr) &&
		len(substr) > 0 &&
		strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
