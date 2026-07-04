package handler

import (
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
	c.JSON(status, APIResponse{
		Success: false,
		Error:   &ErrorDetail{Code: code, Message: message},
	})
}
