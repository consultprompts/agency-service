package service

import (
	"context"
	"errors"
	"log/slog"

	"github.com/consultprompts/agency-service/internal/model"
	"github.com/consultprompts/agency-service/internal/repository"
)

// LeadNotifier is implemented by email.EmailClient. A nil notifier disables
// notifications without changing lead creation behavior.
type LeadNotifier interface {
	SendNewLeadNotification(lead model.Lead) error
	SendLeadConfirmation(lead model.Lead) error
	SendLeadAccepted(lead model.Lead) error
}

var ErrActiveLeadExists = errors.New("you already have an active lead; a new one can be submitted once the current lead is completed")

type LeadService struct {
	leadRepo *repository.LeadRepository
	notifier LeadNotifier
}

func NewLeadService(leadRepo *repository.LeadRepository, notifier LeadNotifier) *LeadService {
	return &LeadService{leadRepo: leadRepo, notifier: notifier}
}

func (s *LeadService) CreateLead(ctx context.Context, userID string, lead model.Lead) (*model.Lead, error) {
	active, err := s.leadRepo.HasActiveLead(ctx, userID)
	if err != nil {
		return nil, err
	}
	if active {
		return nil, ErrActiveLeadExists
	}

	lead.UserID = userID
	lead.Status = "pending"

	created, err := s.leadRepo.CreateLead(ctx, lead)
	if err != nil {
		return nil, err
	}

	// Notify asynchronously — email failures must not fail lead creation.
	if s.notifier != nil {
		go func(l model.Lead) {
			if err := s.notifier.SendNewLeadNotification(l); err != nil {
				slog.Error("Failed to send new lead notification", "lead_id", l.ID, "error", err)
			}
			if err := s.notifier.SendLeadConfirmation(l); err != nil {
				slog.Error("Failed to send lead confirmation to submitter", "lead_id", l.ID, "error", err)
			}
		}(*created)
	}

	return created, nil
}

func (s *LeadService) GetLeads(ctx context.Context, page, limit int) ([]model.Lead, int, error) {
	offset := (page - 1) * limit

	leads, err := s.leadRepo.GetLeads(ctx, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.leadRepo.CountLeads(ctx)
	if err != nil {
		return nil, 0, err
	}

	return leads, total, nil
}

func (s *LeadService) GetUserLeads(ctx context.Context, userID string) ([]model.Lead, error) {
	return s.leadRepo.GetLeadsByUserID(ctx, userID)
}

func (s *LeadService) UpdateLeadStatus(ctx context.Context, id string, status string) error {
	if status != "pending" && status != "accepted" && status != "completed" {
		return errors.New("status must be 'pending', 'accepted', or 'completed'")
	}
	return s.leadRepo.UpdateLeadStatus(ctx, id, status)
}

func (s *LeadService) UpdateLeadMilestone(ctx context.Context, id string, milestoneIndex int) error {
	if milestoneIndex < 0 || milestoneIndex > 4 {
		return errors.New("milestone_index must be between 0 and 4")
	}

	// Detect pending→accepted transition to send acceptance email.
	var lead *model.Lead
	if milestoneIndex == 0 && s.notifier != nil {
		var err error
		lead, err = s.leadRepo.GetLeadByID(ctx, id)
		if err != nil {
			return err
		}
	}

	if err := s.leadRepo.UpdateLeadMilestone(ctx, id, milestoneIndex); err != nil {
		return err
	}

	if lead != nil && lead.Status == "pending" {
		go func(l model.Lead) {
			if err := s.notifier.SendLeadAccepted(l); err != nil {
				slog.Error("Failed to send lead accepted email", "lead_id", l.ID, "error", err)
			}
		}(*lead)
	}

	return nil
}
