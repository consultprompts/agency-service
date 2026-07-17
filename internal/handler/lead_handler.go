package handler

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"regexp"
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
	// LogoFile is only populated by Gin's multipart binding — a plain JSON
	// request (no file attached) leaves it nil.
	LogoFile           *multipart.FileHeader `form:"logo_file"`
}

const maxLogoBytes = 5 << 20 // 5MB

var allowedLogoTypes = map[string]bool{
	"image/png":  true,
	"image/jpeg": true,
	"image/webp": true,
	"image/gif":  true,
}

// readLogoFile reads and validates an uploaded logo, enforcing a size cap
// and sniffing the real content type (never trusting the client-supplied
// one) so only actual raster images end up stored and served back out.
func readLogoFile(fh *multipart.FileHeader) ([]byte, string, error) {
	if fh.Size > maxLogoBytes {
		return nil, "", fmt.Errorf("logo file must be %dMB or smaller", maxLogoBytes>>20)
	}

	f, err := fh.Open()
	if err != nil {
		return nil, "", err
	}
	defer f.Close()

	data, err := io.ReadAll(io.LimitReader(f, maxLogoBytes+1))
	if err != nil {
		return nil, "", err
	}
	if len(data) > maxLogoBytes {
		return nil, "", fmt.Errorf("logo file must be %dMB or smaller", maxLogoBytes>>20)
	}

	contentType := http.DetectContentType(data)
	if !allowedLogoTypes[contentType] {
		return nil, "", errors.New("logo must be a PNG, JPEG, GIF, or WebP image")
	}
	return data, contentType, nil
}

// withLogo reads req.LogoFile (if present) and attaches it to lead. Shared by
// CreateLead/UpdateLead/InviteLead so a validation failure gets the same
// 400 response from all three. Returns false after writing the error
// response when the file is invalid.
func withLogo(c *gin.Context, req CreateLeadRequest, lead *model.Lead) bool {
	if req.LogoFile == nil {
		return true
	}
	data, contentType, err := readLogoFile(req.LogoFile)
	if err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_INPUT", err.Error())
		return false
	}
	lead.LogoData = data
	lead.LogoContentType = &contentType
	return true
}

func (h *LeadHandler) CreateLead(c *gin.Context) {
	userID, _ := c.Get(middleware.ContextUserID)

	var req CreateLeadRequest
	if !bindLeadRequest(c, &req) {
		return
	}

	lead := leadFromRequest(req)
	if !withLogo(c, req, &lead) {
		return
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

// bindLeadRequest binds a lead payload (JSON or multipart) and runs the URL
// validations shared by create/update/invite. Returns false after writing the
// error response when the payload is invalid.
func bindLeadRequest(c *gin.Context, req *CreateLeadRequest) bool {
	if strings.HasPrefix(c.ContentType(), "multipart/form-data") {
		if err := c.ShouldBind(req); err != nil {
			respondError(c, http.StatusBadRequest, "INVALID_INPUT", err.Error())
			return false
		}
	} else {
		if err := c.ShouldBindJSON(req); err != nil {
			respondError(c, http.StatusBadRequest, "INVALID_INPUT", err.Error())
			return false
		}
	}

	if req.ExistingWebsiteURL != nil && *req.ExistingWebsiteURL != "" && !isValidHTTPURL(*req.ExistingWebsiteURL) {
		respondError(c, http.StatusBadRequest, "INVALID_INPUT", "existing_website_url must be a valid http(s) URL")
		return false
	}
	for _, u := range req.InspirationURLs {
		if u != "" && !isValidHTTPURL(u) {
			respondError(c, http.StatusBadRequest, "INVALID_INPUT", "inspiration_urls must be valid http(s) URLs")
			return false
		}
	}
	return true
}

// leadFromRequest maps the bound payload onto the model.
func leadFromRequest(req CreateLeadRequest) model.Lead {
	return model.Lead{
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
}

// InviteLead (admin only) creates an unattached lead on a client's behalf and
// emails them a redeem link. The payload's email field is the client's
// address — both where the invite is sent and the contact stored on the lead.
func (h *LeadHandler) InviteLead(c *gin.Context) {
	var req CreateLeadRequest
	if !bindLeadRequest(c, &req) {
		return
	}

	lead := leadFromRequest(req)
	if !withLogo(c, req, &lead) {
		return
	}

	frontendURL := os.Getenv("FRONTEND_URL")
	created, redeemURL, err := h.leadService.InviteLead(c.Request.Context(), lead, frontendURL)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	respondCreated(c, gin.H{"id": created.ID, "redeem_url": redeemURL})
}

// Redeem input is user-typed in the manual flow, so a malformed UUID is an
// expected "invalid ID" case, not a database error.
var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// GetLeadLogo serves a lead's stored logo bytes. Deliberately unauthenticated
// and registered outside the JWT-protected group at both the gateway and here
// — a plain <img src> can't attach an Authorization header, and a lead's
// UUID is already treated as the bearer credential for this app (see
// RedeemLead); a business logo bound for a public website isn't sensitive
// enough to warrant a blob-fetch-and-object-URL dance on the frontend.
func (h *LeadHandler) GetLeadLogo(c *gin.Context) {
	id := c.Param("id")
	if !uuidPattern.MatchString(id) {
		c.Status(http.StatusNotFound)
		return
	}

	data, contentType, err := h.leadService.GetLeadLogo(c.Request.Context(), id)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	c.Header("Cache-Control", "public, max-age=3600")
	c.Data(http.StatusOK, contentType, data)
}

type RedeemLeadRequest struct {
	LeadID string `json:"lead_id" binding:"required"`
}

// RedeemLead attaches an unattached lead to the calling user — shared by the
// email redeem link and the manual "Redeem Project" form.
func (h *LeadHandler) RedeemLead(c *gin.Context) {
	userID, _ := c.Get(middleware.ContextUserID)

	var req RedeemLeadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_INPUT", err.Error())
		return
	}

	leadID := strings.TrimSpace(req.LeadID)
	if !uuidPattern.MatchString(leadID) {
		respondError(c, http.StatusNotFound, "INVALID_ID", service.ErrLeadNotFound.Error())
		return
	}

	lead, err := h.leadService.RedeemLead(c.Request.Context(), leadID, userID.(string))
	if err != nil {
		if errors.Is(err, service.ErrLeadNotFound) {
			respondError(c, http.StatusNotFound, "INVALID_ID", err.Error())
			return
		}
		if errors.Is(err, service.ErrAlreadyRedeemed) {
			respondError(c, http.StatusConflict, "ALREADY_REDEEMED", err.Error())
			return
		}
		respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	respondOK(c, lead)
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
	if !bindLeadRequest(c, &req) {
		return
	}

	lead := leadFromRequest(req)
	if !withLogo(c, req, &lead) {
		return
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

// GetLeadActivity returns the client-visible activity timeline for a lead,
// newest first.
func (h *LeadHandler) GetLeadActivity(c *gin.Context) {
	id := c.Param("id")
	userID, _ := c.Get(middleware.ContextUserID)

	activity, err := h.leadService.GetLeadActivity(c.Request.Context(), id, userID.(string))
	if err != nil {
		if err.Error() == "forbidden" {
			respondError(c, http.StatusForbidden, "FORBIDDEN", "you do not own this lead")
			return
		}
		respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	respondOK(c, activity)
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
