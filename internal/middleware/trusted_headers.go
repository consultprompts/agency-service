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

func RequireAdminRole() gin.HandlerFunc {
	return func(c *gin.Context) {
		rolesHeader := c.GetHeader("X-User-Roles")
		roles := strings.Split(rolesHeader, ",")

		isAdmin := false
		for _, r := range roles {
			if strings.TrimSpace(r) == "admin" {
				isAdmin = true
				break
			}
		}

		if !isAdmin {
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
