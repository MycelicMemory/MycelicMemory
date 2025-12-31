package api

import (
	"github.com/gin-gonic/gin"

	"github.com/MycelicMemory/ultrathink/internal/ai"
)

// AnalyzeRequest represents an analysis request
type AnalyzeRequest struct {
	AnalysisType         string `json:"analysis_type"` // question, summarize, analyze, temporal_patterns
	Question             string `json:"question"`
	Query                string `json:"query"`
	Timeframe            string `json:"timeframe"` // today, week, month, all
	Concept              string `json:"concept"`
	TemporalAnalysisType string `json:"temporal_analysis_type"`
	Limit                int    `json:"limit"`
	SessionID            string `json:"session_id"`
	Domain               string `json:"domain"`
	ContextLimit         int    `json:"context_limit"`
	ResponseFormat       string `json:"response_format"`
}

// AnalyzeResponse represents an analysis response - matches local-memory format
type AnalyzeResponse struct {
	Analysis        string   `json:"analysis,omitempty"`
	Query           string   `json:"query,omitempty"`
	Insights        []string `json:"insights,omitempty"`
	Patterns        []string `json:"patterns,omitempty"`
	Recommendations []string `json:"recommendations,omitempty"`
}

// analyze handles POST /api/v1/analyze
func (s *Server) analyze(c *gin.Context) {
	var req AnalyzeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequestError(c, "Invalid request body: "+err.Error())
		return
	}

	// Validate analysis type
	analysisType := req.AnalysisType
	if analysisType == "" {
		analysisType = "question"
	}

	validTypes := map[string]bool{
		"question":         true,
		"summarize":        true,
		"analyze":          true,
		"temporal_patterns": true,
	}

	if !validTypes[analysisType] {
		BadRequestError(c, "Invalid analysis_type. Valid types: question, summarize, analyze, temporal_patterns")
		return
	}

	// Check if AI is available
	if s.aiManager == nil {
		InternalError(c, "AI services not configured")
		return
	}

	status := s.aiManager.GetStatus()
	if !status.OllamaAvailable {
		InternalError(c, "AI services not available")
		return
	}

	// Use query as alias for question (local-memory compatibility)
	question := req.Question
	if question == "" {
		question = req.Query
	}

	// Validate requirements per type
	if analysisType == "question" && question == "" {
		BadRequestError(c, "validation failed: query is required")
		return
	}

	ctx := c.Request.Context()

	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}
	if req.ContextLimit > 0 {
		limit = req.ContextLimit
	}

	opts := &ai.AnalysisOptions{
		Type:      analysisType,
		Question:  question,
		Query:     req.Query,
		Timeframe: req.Timeframe,
		Limit:     limit,
		SessionID: req.SessionID,
		Domain:    req.Domain,
	}

	result, err := s.aiManager.Analyze(ctx, opts)
	if err != nil {
		InternalError(c, "Analysis failed: "+err.Error())
		return
	}

	// Build response matching local-memory format
	insights := result.Insights
	if insights == nil || len(insights) == 0 {
		insights = []string{"parsing_failed"}
	}

	patterns := make([]string, len(result.Patterns))
	for i, p := range result.Patterns {
		patterns[i] = p.Name
	}
	if len(patterns) == 0 {
		patterns = []string{"nopatternsdetected"}
	}

	recommendations := result.Sources
	if recommendations == nil || len(recommendations) == 0 {
		recommendations = []string{"norecommendationsavailable"}
	}

	response := &AnalyzeResponse{
		Analysis:        result.Answer,
		Query:           question,
		Insights:        insights,
		Patterns:        patterns,
		Recommendations: recommendations,
	}

	SuccessResponse(c, "Memories analyzed successfully", response)
}
