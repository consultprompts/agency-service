package handler

import (
	"errors"
	"fmt"
	"net/http"
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

func NewLeadHandler(leadService *service.LeadService) *LeadHandler {
	return &LeadHandler{leadService: leadService}
}

type CreateLeadRequest struct {
	Name               string   `json:"name"                form:"name"                binding:"required"`
	Email              string   `json:"email"               form:"email"               binding:"required,email"`
	Business           string   `json:"business"            form:"business"            binding:"required"`
	ExistingWebsite    *bool    `json:"existing_website"    form:"existing_website"`
	ExistingWebsiteURL *string  `json:"existing_website_url" form:"existing_website_url"`
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

	lead := model.Lead{
		Name:               req.Name,
		Email:              req.Email,
		Business:           req.Business,
		ExistingWebsite:    req.ExistingWebsite,
		ExistingWebsiteURL: req.ExistingWebsiteURL,
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

func (h *LeadHandler) GetUserLeads(c *gin.Context) {
	userID, _ := c.Get(middleware.ContextUserID)

	leads, err := h.leadService.GetUserLeads(c.Request.Context(), userID.(string))
	if err != nil {
		respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	respondOK(c, leads)
}

type UpdateStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

func (h *LeadHandler) UpdateLeadStatus(c *gin.Context) {
	id := c.Param("id")

	var req UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_INPUT", err.Error())
		return
	}

	if err := h.leadService.UpdateLeadStatus(c.Request.Context(), id, req.Status); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_INPUT", err.Error())
		return
	}

	respondOK(c, gin.H{"message": "status updated"})
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
