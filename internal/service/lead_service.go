package service

import (
	"context"
	"errors"

	"github.com/consultprompts/agency-service/internal/model"
	"github.com/consultprompts/agency-service/internal/repository"
)

type LeadService struct {
	leadRepo *repository.LeadRepository
}

func NewLeadService(leadRepo *repository.LeadRepository) *LeadService {
	return &LeadService{leadRepo: leadRepo}
}

func (s *LeadService) CreateLead(ctx context.Context, userID string, lead model.Lead) (*model.Lead, error) {
	lead.UserID = userID
	lead.Status = "pending"
	return s.leadRepo.CreateLead(ctx, lead)
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
