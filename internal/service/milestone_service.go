package service

import (
	"context"
	"errors"
	"time"

	"github.com/consultprompts/agency-service/internal/model"
	"github.com/consultprompts/agency-service/internal/repository"
	"github.com/jackc/pgx/v5"
)

var (
	ErrNotFound  = errors.New("not found")
	ErrForbidden = errors.New("forbidden")
)

func isValidMilestoneStatus(status string) bool {
	return status == "pending" || status == "in_progress" || status == "completed"
}

type MilestoneService struct {
	milestoneRepo *repository.MilestoneRepository
	leadRepo      *repository.LeadRepository
}

func NewMilestoneService(milestoneRepo *repository.MilestoneRepository, leadRepo *repository.LeadRepository) *MilestoneService {
	return &MilestoneService{milestoneRepo: milestoneRepo, leadRepo: leadRepo}
}

func (s *MilestoneService) CreateMilestone(ctx context.Context, leadID string, milestone model.Milestone) (*model.Milestone, error) {
	if _, err := s.leadRepo.GetLeadByID(ctx, leadID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	milestone.LeadID = leadID
	return s.milestoneRepo.CreateMilestone(ctx, milestone)
}

// GetMilestones returns the milestones for a lead. Non-admin callers may
// only view milestones for leads they submitted.
func (s *MilestoneService) GetMilestones(ctx context.Context, leadID, userID string, isAdmin bool) ([]model.Milestone, error) {
	lead, err := s.leadRepo.GetLeadByID(ctx, leadID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if !isAdmin && lead.UserID != userID {
		return nil, ErrForbidden
	}

	return s.milestoneRepo.GetMilestonesByLead(ctx, leadID)
}

func (s *MilestoneService) UpdateMilestone(ctx context.Context, id string, title, description, status *string, sortOrder *int, dueDate *time.Time) (*model.Milestone, error) {
	if status != nil && !isValidMilestoneStatus(*status) {
		return nil, errors.New("status must be 'pending', 'in_progress' or 'completed'")
	}

	milestone, err := s.milestoneRepo.UpdateMilestone(ctx, id, title, description, status, sortOrder, dueDate)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return milestone, nil
}

func (s *MilestoneService) DeleteMilestone(ctx context.Context, id string) error {
	deleted, err := s.milestoneRepo.DeleteMilestone(ctx, id)
	if err != nil {
		return err
	}
	if !deleted {
		return ErrNotFound
	}
	return nil
}
