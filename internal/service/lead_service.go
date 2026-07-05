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
}

type LeadService struct {
	leadRepo *repository.LeadRepository
	notifier LeadNotifier
}

func NewLeadService(leadRepo *repository.LeadRepository, notifier LeadNotifier) *LeadService {
	return &LeadService{leadRepo: leadRepo, notifier: notifier}
}

func (s *LeadService) CreateLead(ctx context.Context, userID string, lead model.Lead) (*model.Lead, error) {
	lead.UserID = userID
	lead.Status = "pending"

	created, err := s.leadRepo.CreateLead(ctx, lead)
	if err != nil {
		return nil, err
	}

	// Notify asynchronously — a failed email must not fail lead creation.
	if s.notifier != nil {
		go func(l model.Lead) {
			if err := s.notifier.SendNewLeadNotification(l); err != nil {
				slog.Error("Failed to send new lead notification", "lead_id", l.ID, "error", err)
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

func (s *LeadService) UpdateLeadStatus(ctx context.Context, id string, status string) error {
	if status != "pending" && status != "completed" {
		return errors.New("status must be 'pending' or 'completed'")
	}
	return s.leadRepo.UpdateLeadStatus(ctx, id, status)
}
