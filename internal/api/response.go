package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response represents a standard API response
// VERIFIED: Matches local-memory response format exactly
type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// SuccessResponse sends a success response
func SuccessResponse(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, &Response{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// CreatedResponse sends a 201 created response
func CreatedResponse(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusCreated, &Response{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// ErrorResponse sends an error response
func ErrorResponse(c *gin.Context, code int, message string) {
	c.JSON(code, &Response{
		Success: false,
		Message: message,
	})
}

// BadRequestError sends a 400 error
func BadRequestError(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusBadRequest, message)
}

// NotFoundError sends a 404 error
func NotFoundError(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusNotFound, message)
}

// NotFoundErrorWithID sends a 404 error matching local-memory format
func NotFoundErrorWithID(c *gin.Context, id string) {
	c.JSON(http.StatusNotFound, gin.H{
		"error": "not_found",
		"id":    id,
	})
}

// InternalError sends a 500 error
func InternalError(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusInternalServerError, message)
}
