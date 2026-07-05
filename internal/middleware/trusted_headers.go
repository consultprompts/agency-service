package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const ContextUserID = "userID"

func RequireUserID() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetHeader("X-User-ID")
		if userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "missing user identity",
				},
			})
			c.Abort()
			return
		}
		c.Set(ContextUserID, userID)
		c.Next()
	}
}

// IsAdmin reports whether the gateway-set X-User-Roles header includes
// the admin role.
func IsAdmin(c *gin.Context) bool {
	for _, r := range strings.Split(c.GetHeader("X-User-Roles"), ",") {
		if strings.TrimSpace(r) == "admin" {
			return true
		}
	}
	return false
}

func RequireAdminRole() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !IsAdmin(c) {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "FORBIDDEN",
					"message": "insufficient permissions",
				},
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
