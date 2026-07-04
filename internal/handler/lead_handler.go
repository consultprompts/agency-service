package handler

import (
	"net/http"

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
	Name     string  `json:"name" binding:"required"`
	Email    string  `json:"email" binding:"required,email"`
	Business string  `json:"business" binding:"required"`
	Message  *string `json:"message"`
	Package  *string `json:"package"`
}

func (h *LeadHandler) CreateLead(c *gin.Context) {
	userID, _ := c.Get(middleware.ContextUserID)

	var req CreateLeadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_INPUT", err.Error())
		return
	}

	lead := model.Lead{
		Name:     req.Name,
		Email:    req.Email,
		Business: req.Business,
		Message:  req.Message,
		Package:  req.Package,
	}

	created, err := h.leadService.CreateLead(c.Request.Context(), userID.(string), lead)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	respondCreated(c, created)
}

func (h *LeadHandler) GetLeads(c *gin.Context) {
	leads, err := h.leadService.GetAllLeads(c.Request.Context())
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
