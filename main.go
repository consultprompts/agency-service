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
	"github.com/consultprompts/agency-service/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	logger.Init()

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
	router.Use(middleware.RequestLogger(), gin.Recovery())

	router.GET("/healthz", func(c *gin.Context) {
		if err := pool.Ping(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "message": "database connection failed"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	emailClient := email.NewEmailClient()
	if emailClient != nil {
		slog.Info("Email notifications enabled", "from", os.Getenv("RESEND_FROM"))
	} else {
		slog.Warn("Email notifications DISABLED — set RESEND_API_KEY and RESEND_FROM in .env to enable")
	}

	leadRepo := repository.NewLeadRepository(pool)
	var notifier service.LeadNotifier
	if emailClient != nil {
		notifier = emailClient
	}
	leadService := service.NewLeadService(leadRepo, notifier)
	leadHandler := handler.NewLeadHandler(leadService)

	// Payment provider callbacks carry a shared secret, not a user JWT —
	// registered outside the RequireUserID group. See handler.PaymentWebhook.
	router.POST("/webhooks/payment-success", leadHandler.PaymentWebhook)

	// Public: a plain <img> tag can't attach an Authorization header. See
	// handler.GetLeadLogo for the trust-model rationale.
	router.GET("/agency/leads/:id/logo", leadHandler.GetLeadLogo)

	protected := router.Group("/")
	protected.Use(middleware.RequireUserID())
	{
		protected.POST("/agency/leads", leadHandler.CreateLead)
		protected.POST("/agency/leads/redeem", leadHandler.RedeemLead)
		protected.PATCH("/agency/leads/:id", leadHandler.UpdateLead)
		protected.GET("/agency/leads/mine", leadHandler.GetUserLeads)
		protected.POST("/agency/leads/:id/review", leadHandler.SubmitReview)
		protected.PATCH("/agency/leads/:id/maintenance", leadHandler.SetWantsMaintenance)
		protected.POST("/agency/leads/:id/pay", leadHandler.MarkPaid)
		protected.POST("/agency/leads/:id/request-meeting", leadHandler.RequestMeeting)
		protected.GET("/agency/leads/:id/activity", leadHandler.GetLeadActivity)

		admin := protected.Group("/")
		admin.Use(middleware.RequireAdminRole())
		{
			admin.GET("/agency/leads", leadHandler.GetLeads)
			admin.POST("/agency/leads/invite", leadHandler.InviteLead)
			admin.PATCH("/agency/leads/:id/milestone", leadHandler.UpdateLeadMilestone)
			admin.PATCH("/agency/leads/:id/mockup", leadHandler.SetMockupURL)
			admin.PATCH("/agency/leads/:id/complete", leadHandler.CompleteSite)
			admin.PATCH("/agency/leads/:id/launch", leadHandler.LaunchSite)
			admin.PATCH("/agency/leads/:id/suspend", leadHandler.SetSuspended)
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
