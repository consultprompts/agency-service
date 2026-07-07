package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type APIResponse struct {
	Success bool         `json:"success"`
	Data    interface{}  `json:"data,omitempty"`
	Error   *ErrorDetail `json:"error,omitempty"`
}

func respondOK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, APIResponse{Success: true, Data: data})
}

func respondCreated(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, APIResponse{Success: true, Data: data})
}

func respondError(c *gin.Context, status int, code, message string) {
	// Never leak internal details (DB errors, stack info) to clients.
	if status >= http.StatusInternalServerError {
		slog.Error("request failed", "code", code, "error", message, "method", c.Request.Method, "path", c.Request.URL.Path)
		message = "an internal error occurred"
	}
	c.JSON(status, APIResponse{
		Success: false,
		Error:   &ErrorDetail{Code: code, Message: message},
	})
}
