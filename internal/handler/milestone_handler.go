package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/consultprompts/agency-service/internal/middleware"
	"github.com/consultprompts/agency-service/internal/model"
	"github.com/consultprompts/agency-service/internal/service"
)

type MilestoneHandler struct {
	milestoneService *service.MilestoneService
}

func NewMilestoneHandler(milestoneService *service.MilestoneService) *MilestoneHandler {
	return &MilestoneHandler{milestoneService: milestoneService}
}

const dueDateLayout = "2006-01-02"

func parseDueDate(value *string) (*time.Time, error) {
	if value == nil {
		return nil, nil
	}
	t, err := time.Parse(dueDateLayout, *value)
	if err != nil {
		return nil, errors.New("due_date must be in YYYY-MM-DD format")
	}
	return &t, nil
}

type CreateMilestoneRequest struct {
	Title       string  `json:"title" binding:"required"`
	Description *string `json:"description"`
	SortOrder   int     `json:"sort_order"`
	DueDate     *string `json:"due_date"`
}

func (h *MilestoneHandler) CreateMilestone(c *gin.Context) {
	leadID := c.Param("id")

	var req CreateMilestoneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_INPUT", err.Error())
		return
	}

	dueDate, err := parseDueDate(req.DueDate)
	if err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_INPUT", err.Error())
		return
	}

	milestone := model.Milestone{
		Title:       req.Title,
		Description: req.Description,
		SortOrder:   req.SortOrder,
		DueDate:     dueDate,
	}

	created, err := h.milestoneService.CreateMilestone(c.Request.Context(), leadID, milestone)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			respondError(c, http.StatusNotFound, "NOT_FOUND", "lead not found")
			return
		}
		respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	respondCreated(c, created)
}

func (h *MilestoneHandler) GetMilestones(c *gin.Context) {
	leadID := c.Param("id")
	userID, _ := c.Get(middleware.ContextUserID)

	milestones, err := h.milestoneService.GetMilestones(
		c.Request.Context(), leadID, userID.(string), middleware.IsAdmin(c))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrNotFound):
			respondError(c, http.StatusNotFound, "NOT_FOUND", "lead not found")
		case errors.Is(err, service.ErrForbidden):
			respondError(c, http.StatusForbidden, "FORBIDDEN", "insufficient permissions")
		default:
			respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		}
		return
	}

	respondOK(c, gin.H{"milestones": milestones})
}

type UpdateMilestoneRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Status      *string `json:"status"`
	SortOrder   *int    `json:"sort_order"`
	DueDate     *string `json:"due_date"`
}

func (h *MilestoneHandler) UpdateMilestone(c *gin.Context) {
	id := c.Param("id")

	var req UpdateMilestoneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_INPUT", err.Error())
		return
	}

	dueDate, err := parseDueDate(req.DueDate)
	if err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_INPUT", err.Error())
		return
	}

	updated, err := h.milestoneService.UpdateMilestone(
		c.Request.Context(), id, req.Title, req.Description, req.Status, req.SortOrder, dueDate)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			respondError(c, http.StatusNotFound, "NOT_FOUND", "milestone not found")
			return
		}
		respondError(c, http.StatusBadRequest, "INVALID_INPUT", err.Error())
		return
	}

	respondOK(c, updated)
}

func (h *MilestoneHandler) DeleteMilestone(c *gin.Context) {
	id := c.Param("id")

	if err := h.milestoneService.DeleteMilestone(c.Request.Context(), id); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			respondError(c, http.StatusNotFound, "NOT_FOUND", "milestone not found")
			return
		}
		respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	respondOK(c, gin.H{"message": "milestone deleted"})
}
