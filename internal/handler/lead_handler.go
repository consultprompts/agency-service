package handler

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/consultprompts/agency-service/internal/middleware"
	"github.com/consultprompts/agency-service/internal/model"
	"github.com/consultprompts/agency-service/internal/service"
)

type LeadHandler struct {
	leadService *service.LeadService
}

// isValidHTTPURL rejects anything that isn't an absolute http(s) URL, so
// stored URLs can never smuggle javascript:/data: into client-rendered links.
func isValidHTTPURL(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

func NewLeadHandler(leadService *service.LeadService) *LeadHandler {
	return &LeadHandler{leadService: leadService}
}

type CreateLeadRequest struct {
	Name               string   `json:"name"                form:"name"                binding:"required"`
	Email              string   `json:"email"               form:"email"               binding:"required,email"`
	Business           string   `json:"business"            form:"business"            binding:"required"`
	Message            *string  `json:"message"             form:"message"`
	ExistingWebsite    *bool    `json:"existing_website"    form:"existing_website"`
	ExistingWebsiteURL *string  `json:"existing_website_url" form:"existing_website_url"`
	Location           *string  `json:"location"            form:"location"`
	SiteGoal           *string  `json:"site_goal"           form:"site_goal"`
	PagesNeeded        []string `json:"pages_needed"        form:"pages_needed[]"`
	StyleDirection     *string  `json:"style_direction"     form:"style_direction"`
	HasLogo            *bool    `json:"has_logo"            form:"has_logo"`
	HasBrandColors     *bool    `json:"has_brand_colors"    form:"has_brand_colors"`
	PrimaryColor       *string  `json:"primary_color"       form:"primary_color"`
	SecondaryColor     *string  `json:"secondary_color"     form:"secondary_color"`
	InspirationURLs    []string `json:"inspiration_urls"    form:"inspiration_urls[]"`
	PhoneNumber        *string  `json:"phone_number"        form:"phone_number"`
	ContactMethod      *string  `json:"contact_method"      form:"contact_method"`
	Timeline           *string  `json:"timeline"            form:"timeline"`
	Package            *string  `json:"package"             form:"package"`
	WantsCall          bool     `json:"wants_call"          form:"wants_call"`
}

func (h *LeadHandler) CreateLead(c *gin.Context) {
	userID, _ := c.Get(middleware.ContextUserID)

	var req CreateLeadRequest
	if strings.HasPrefix(c.ContentType(), "multipart/form-data") {
		if err := c.ShouldBind(&req); err != nil {
			respondError(c, http.StatusBadRequest, "INVALID_INPUT", err.Error())
			return
		}
		// Logo file upload — TODO: store to object storage and set LogoURL on the lead.
		// For now the file is accepted but not persisted.
	} else {
		if err := c.ShouldBindJSON(&req); err != nil {
			respondError(c, http.StatusBadRequest, "INVALID_INPUT", err.Error())
			return
		}
	}

	if req.ExistingWebsiteURL != nil && *req.ExistingWebsiteURL != "" && !isValidHTTPURL(*req.ExistingWebsiteURL) {
		respondError(c, http.StatusBadRequest, "INVALID_INPUT", "existing_website_url must be a valid http(s) URL")
		return
	}
	for _, u := range req.InspirationURLs {
		if u != "" && !isValidHTTPURL(u) {
			respondError(c, http.StatusBadRequest, "INVALID_INPUT", "inspiration_urls must be valid http(s) URLs")
			return
		}
	}

	lead := model.Lead{
		Name:               req.Name,
		Email:              req.Email,
		Business:           req.Business,
		Message:            req.Message,
		ExistingWebsite:    req.ExistingWebsite,
		ExistingWebsiteURL: req.ExistingWebsiteURL,
		Location:           req.Location,
		SiteGoal:           req.SiteGoal,
		PagesNeeded:        req.PagesNeeded,
		StyleDirection:     req.StyleDirection,
		HasLogo:            req.HasLogo,
		HasBrandColors:     req.HasBrandColors,
		PrimaryColor:       req.PrimaryColor,
		SecondaryColor:     req.SecondaryColor,
		InspirationURLs:    req.InspirationURLs,
		PhoneNumber:        req.PhoneNumber,
		ContactMethod:      req.ContactMethod,
		Timeline:           req.Timeline,
		Package:            req.Package,
		WantsCall:          req.WantsCall,
	}

	created, err := h.leadService.CreateLead(c.Request.Context(), userID.(string), lead)
	if err != nil {
		if errors.Is(err, service.ErrActiveLeadExists) {
			respondError(c, http.StatusConflict, "ACTIVE_LEAD_EXISTS", err.Error())
			return
		}
		respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	respondCreated(c, created)
}

const (
	defaultLeadsLimit = 20
	maxLeadsLimit     = 100
)

type Pagination struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

type LeadsResponse struct {
	Leads      []model.Lead `json:"leads"`
	Pagination Pagination   `json:"pagination"`
}

func (h *LeadHandler) GetLeads(c *gin.Context) {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		respondError(c, http.StatusBadRequest, "INVALID_INPUT", "page must be a positive integer")
		return
	}

	limit, err := strconv.Atoi(c.DefaultQuery("limit", strconv.Itoa(defaultLeadsLimit)))
	if err != nil || limit < 1 || limit > maxLeadsLimit {
		respondError(c, http.StatusBadRequest, "INVALID_INPUT",
			fmt.Sprintf("limit must be between 1 and %d", maxLeadsLimit))
		return
	}

	leads, total, err := h.leadService.GetLeads(c.Request.Context(), page, limit)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	respondOK(c, LeadsResponse{
		Leads: leads,
		Pagination: Pagination{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: (total + limit - 1) / limit,
		},
	})
}

func (h *LeadHandler) UpdateLead(c *gin.Context) {
	userID, _ := c.Get(middleware.ContextUserID)
	id := c.Param("id")

	var req CreateLeadRequest
	if strings.HasPrefix(c.ContentType(), "multipart/form-data") {
		if err := c.ShouldBind(&req); err != nil {
			respondError(c, http.StatusBadRequest, "INVALID_INPUT", err.Error())
			return
		}
	} else {
		if err := c.ShouldBindJSON(&req); err != nil {
			respondError(c, http.StatusBadRequest, "INVALID_INPUT", err.Error())
			return
		}
	}

	if req.ExistingWebsiteURL != nil && *req.ExistingWebsiteURL != "" && !isValidHTTPURL(*req.ExistingWebsiteURL) {
		respondError(c, http.StatusBadRequest, "INVALID_INPUT", "existing_website_url must be a valid http(s) URL")
		return
	}
	for _, u := range req.InspirationURLs {
		if u != "" && !isValidHTTPURL(u) {
			respondError(c, http.StatusBadRequest, "INVALID_INPUT", "inspiration_urls must be valid http(s) URLs")
			return
		}
	}

	lead := model.Lead{
		Name:               req.Name,
		Business:           req.Business,
		Message:            req.Message,
		ExistingWebsite:    req.ExistingWebsite,
		ExistingWebsiteURL: req.ExistingWebsiteURL,
		Location:           req.Location,
		SiteGoal:           req.SiteGoal,
		PagesNeeded:        req.PagesNeeded,
		StyleDirection:     req.StyleDirection,
		HasLogo:            req.HasLogo,
		HasBrandColors:     req.HasBrandColors,
		PrimaryColor:       req.PrimaryColor,
		SecondaryColor:     req.SecondaryColor,
		InspirationURLs:    req.InspirationURLs,
		PhoneNumber:        req.PhoneNumber,
		ContactMethod:      req.ContactMethod,
		Timeline:           req.Timeline,
		Package:            req.Package,
		WantsCall:          req.WantsCall,
	}

	if err := h.leadService.UpdateLead(c.Request.Context(), id, userID.(string), lead); err != nil {
		if errors.Is(err, service.ErrLeadNotPending) {
			respondError(c, http.StatusConflict, "LEAD_NOT_PENDING", err.Error())
			return
		}
		if err.Error() == "forbidden" {
			respondError(c, http.StatusForbidden, "FORBIDDEN", err.Error())
			return
		}
		respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	respondOK(c, gin.H{"ok": true})
}

func (h *LeadHandler) GetUserLeads(c *gin.Context) {
	userID, _ := c.Get(middleware.ContextUserID)

	leads, err := h.leadService.GetUserLeads(c.Request.Context(), userID.(string))
	if err != nil {
		respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	respondOK(c, leads)
}

type UpdateLeadMilestoneRequest struct {
	MilestoneIndex int `json:"milestone_index"`
}

func (h *LeadHandler) UpdateLeadMilestone(c *gin.Context) {
	id := c.Param("id")

	var req UpdateLeadMilestoneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_INPUT", err.Error())
		return
	}

	if err := h.leadService.UpdateLeadMilestone(c.Request.Context(), id, req.MilestoneIndex); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_INPUT", err.Error())
		return
	}

	respondOK(c, gin.H{"message": "milestone updated"})
}

type SetMockupURLRequest struct {
	MockupURL string `json:"mockup_url" binding:"required"`
}

func (h *LeadHandler) SetMockupURL(c *gin.Context) {
	id := c.Param("id")

	var req SetMockupURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_INPUT", err.Error())
		return
	}
	if !isValidHTTPURL(req.MockupURL) {
		respondError(c, http.StatusBadRequest, "INVALID_INPUT", "mockup_url must be a valid http(s) URL")
		return
	}

	frontendURL := os.Getenv("FRONTEND_URL")
	if err := h.leadService.SetMockupURL(c.Request.Context(), id, req.MockupURL, frontendURL); err != nil {
		respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	respondOK(c, gin.H{"message": "mockup URL saved"})
}

type SubmitReviewRequest struct {
	Decision string `json:"decision" binding:"required"`
	Feedback string `json:"feedback"`
}

func (h *LeadHandler) SubmitReview(c *gin.Context) {
	id := c.Param("id")
	userID, _ := c.Get(middleware.ContextUserID)

	var req SubmitReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_INPUT", err.Error())
		return
	}

	err := h.leadService.SubmitReview(c.Request.Context(), id, userID.(string), req.Decision, req.Feedback)
	if err != nil {
		if err.Error() == "forbidden" {
			respondError(c, http.StatusForbidden, "FORBIDDEN", "you do not own this lead")
			return
		}
		respondError(c, http.StatusBadRequest, "INVALID_INPUT", err.Error())
		return
	}

	respondOK(c, gin.H{"message": "review submitted"})
}

func (h *LeadHandler) CompleteSite(c *gin.Context) {
	id := c.Param("id")

	frontendURL := os.Getenv("FRONTEND_URL")
	if err := h.leadService.CompleteSite(c.Request.Context(), id, frontendURL); err != nil {
		respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	respondOK(c, gin.H{"message": "site marked complete"})
}

type SetMaintenanceRequest struct {
	WantsMaintenance bool `json:"wants_maintenance"`
}

func (h *LeadHandler) SetWantsMaintenance(c *gin.Context) {
	id := c.Param("id")
	userID, _ := c.Get(middleware.ContextUserID)

	var req SetMaintenanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_INPUT", err.Error())
		return
	}

	err := h.leadService.SetWantsMaintenance(c.Request.Context(), id, userID.(string), req.WantsMaintenance)
	if err != nil {
		if err.Error() == "forbidden" {
			respondError(c, http.StatusForbidden, "FORBIDDEN", "you do not own this lead")
			return
		}
		respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	respondOK(c, gin.H{"message": "maintenance preference saved"})
}

type PaymentWebhookRequest struct {
	ProjectID string `json:"project_id" binding:"required"`
}

// PaymentWebhook simulates a payment provider's server-to-server success
// callback (e.g. a Stripe webhook). It is unauthenticated at the user level —
// providers can't send our JWTs — so it is guarded by a shared secret instead:
// the X-Webhook-Secret header must match PAYMENT_WEBHOOK_SECRET. Fails closed
// when the secret isn't configured.
func (h *LeadHandler) PaymentWebhook(c *gin.Context) {
	secret := os.Getenv("PAYMENT_WEBHOOK_SECRET")
	if secret == "" {
		respondError(c, http.StatusServiceUnavailable, "WEBHOOK_NOT_CONFIGURED", "PAYMENT_WEBHOOK_SECRET is not set")
		return
	}
	if subtle.ConstantTimeCompare([]byte(c.GetHeader("X-Webhook-Secret")), []byte(secret)) != 1 {
		respondError(c, http.StatusUnauthorized, "INVALID_SIGNATURE", "webhook secret mismatch")
		return
	}

	var req PaymentWebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_INPUT", err.Error())
		return
	}

	if err := h.leadService.ConfirmPaymentWebhook(c.Request.Context(), req.ProjectID); err != nil {
		respondError(c, http.StatusUnprocessableEntity, "PAYMENT_NOT_APPLICABLE", err.Error())
		return
	}

	respondOK(c, gin.H{"message": "payment confirmed"})
}

func (h *LeadHandler) MarkPaid(c *gin.Context) {
	id := c.Param("id")
	userID, _ := c.Get(middleware.ContextUserID)

	err := h.leadService.MarkPaid(c.Request.Context(), id, userID.(string))
	if err != nil {
		if err.Error() == "forbidden" {
			respondError(c, http.StatusForbidden, "FORBIDDEN", "you do not own this lead")
			return
		}
		respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	respondOK(c, gin.H{"message": "payment recorded"})
}

type LaunchSiteRequest struct {
	SiteURL string `json:"site_url" binding:"required"`
}

func (h *LeadHandler) LaunchSite(c *gin.Context) {
	id := c.Param("id")

	var req LaunchSiteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_INPUT", err.Error())
		return
	}
	if !isValidHTTPURL(req.SiteURL) {
		respondError(c, http.StatusBadRequest, "INVALID_INPUT", "site_url must be a valid http(s) URL")
		return
	}

	if err := h.leadService.LaunchSite(c.Request.Context(), id, req.SiteURL); err != nil {
		if err.Error() == "payment required before launch" {
			respondError(c, http.StatusConflict, "PAYMENT_REQUIRED", err.Error())
			return
		}
		respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	respondOK(c, gin.H{"message": "site launched"})
}

type SetSuspendedRequest struct {
	Suspended bool `json:"suspended"`
}

// SetSuspended pauses or resumes an in-flight project (admin only).
func (h *LeadHandler) SetSuspended(c *gin.Context) {
	id := c.Param("id")

	var req SetSuspendedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_INPUT", err.Error())
		return
	}

	var (
		newStatus string
		err       error
	)
	message := "project reactivated"
	if req.Suspended {
		newStatus, err = h.leadService.SuspendLead(c.Request.Context(), id)
		message = "project suspended"
	} else {
		newStatus, err = h.leadService.ReactivateLead(c.Request.Context(), id)
	}
	if err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_INPUT", err.Error())
		return
	}

	respondOK(c, gin.H{"message": message, "status": newStatus})
}

// RequestMeeting lets a client who skipped the initial 15-minute call ask for
// one after all — only available while the meeting is still marked skipped.
func (h *LeadHandler) RequestMeeting(c *gin.Context) {
	id := c.Param("id")
	userID, _ := c.Get(middleware.ContextUserID)

	err := h.leadService.RequestMeeting(c.Request.Context(), id, userID.(string))
	if err != nil {
		if err.Error() == "forbidden" {
			respondError(c, http.StatusForbidden, "FORBIDDEN", "you do not own this lead")
			return
		}
		respondError(c, http.StatusBadRequest, "INVALID_INPUT", err.Error())
		return
	}

	respondOK(c, gin.H{"message": "meeting requested"})
}
