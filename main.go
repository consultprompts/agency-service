package main

import (
	"log/slog"
	"net/http"
	"os"

	database "github.com/consultprompts/agency-service/database"
	"github.com/consultprompts/agency-service/internal/email"
	"github.com/consultprompts/agency-service/internal/handler"
	"github.com/consultprompts/agency-service/internal/middleware"
	"github.com/consultprompts/agency-service/internal/repository"
	"github.com/consultprompts/agency-service/internal/service"
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

	emailClient := email.NewEmailClient()

	leadRepo := repository.NewLeadRepository(pool)
	var notifier service.LeadNotifier
	if emailClient != nil {
		notifier = emailClient
	}
	leadService := service.NewLeadService(leadRepo, notifier)
	leadHandler := handler.NewLeadHandler(leadService)

	milestoneRepo := repository.NewMilestoneRepository(pool)
	milestoneService := service.NewMilestoneService(milestoneRepo, leadRepo)
	milestoneHandler := handler.NewMilestoneHandler(milestoneService)

	protected := router.Group("/")
	protected.Use(middleware.RequireUserID())
	{
		protected.POST("/agency/leads", leadHandler.CreateLead)
		protected.GET("/agency/leads/mine", leadHandler.GetUserLeads)
		protected.GET("/agency/leads/:id/milestones", milestoneHandler.GetMilestones)

		admin := protected.Group("/")
		admin.Use(middleware.RequireAdminRole())
		{
			admin.GET("/agency/leads", leadHandler.GetLeads)
			admin.PATCH("/agency/leads/:id/status", leadHandler.UpdateLeadStatus)
			admin.PATCH("/agency/leads/:id/milestone", leadHandler.UpdateLeadMilestone)
			admin.POST("/agency/leads/:id/milestones", milestoneHandler.CreateMilestone)
			admin.PATCH("/agency/milestones/:id", milestoneHandler.UpdateMilestone)
			admin.DELETE("/agency/milestones/:id", milestoneHandler.DeleteMilestone)
		}
	}

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
