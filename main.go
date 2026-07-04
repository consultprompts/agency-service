package main

import (
	"log/slog"
	"net/http"
	"os"

	database "github.com/consultprompts/agency-service/database"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		slog.Warn("No .env file found, using existing environment variables")
	}

	database.RunMigrations()

	pool := database.Connect()
	defer pool.Close()

	if os.Getenv(gin.EnvGinMode) == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.SetTrustedProxies(nil)

	router.GET("/healthz", func(c *gin.Context) {
		if err := pool.Ping(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "message": "database connection failed"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	slog.Info("Starting server", "addr", port)
	if err := router.Run(":" + port); err != nil {
		slog.Error("Server stopped", "error", err)
		os.Exit(1)
	}
}
