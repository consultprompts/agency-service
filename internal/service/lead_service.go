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
	SendMockupReadyEmail(to, projectLink string) error
	SendRevisionRequestEmail(clientEmail, businessName, feedback string) error
	SendPaymentRequestEmail(to, projectLink string) error
	SendSiteLaunchedEmail(to, siteURL, businessName string) error
	SendPaymentReceiptEmail(to, businessName, packageName string, packagePrice, totalAmount int, wantsMaintenance bool, domainRenewalDate string) error
}

// Package pricing — must stay in sync with website/src/data/content.tsx PACKAGES.
var packagePrices = map[string]int{
	"facelift":   299,
	"visibility": 499,
	"growth":     699,
}

var packageNames = map[string]string{
	"facelift":   "Digital Face-Lift",
	"visibility": "Visibility Booster",
	"growth":     "Auto-Pilot Growth",
}

const (
	domainFee    = 20
	maintenanceFee = 29
)

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
	switch status {
	case "pending", "accepted", "completed", "launched":
	default:
		return errors.New("status must be 'pending', 'accepted', 'completed', or 'launched'")
	}
	return s.leadRepo.UpdateLeadStatus(ctx, id, status)
}

func (s *LeadService) UpdateLeadMilestone(ctx context.Context, id string, milestoneIndex int) error {
	if milestoneIndex < 0 || milestoneIndex > 5 {
		return errors.New("milestone_index must be between 0 and 5")
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

// SetMockupURL saves the mockup URL, advances the milestone to "Mockup Delivered" (index 1),
// and emails the client a link to their project page.
func (s *LeadService) SetMockupURL(ctx context.Context, id, url, frontendURL string) error {
	lead, err := s.leadRepo.GetLeadByID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.leadRepo.SetMockupURL(ctx, id, url); err != nil {
		return err
	}

	// Advance to milestone 1 — "Mockup Delivered"
	if err := s.leadRepo.UpdateLeadMilestone(ctx, id, 1); err != nil {
		return err
	}

	if s.notifier != nil {
		projectLink := frontendURL + "/my-projects"
		slog.Info("Sending mockup ready email", "lead_id", lead.ID, "to", lead.Email)
		go func(l model.Lead) {
			if err := s.notifier.SendMockupReadyEmail(l.Email, projectLink); err != nil {
				slog.Error("Mockup ready email failed", "lead_id", l.ID, "to", l.Email, "error", err)
			} else {
				slog.Info("Mockup ready email sent", "lead_id", l.ID, "to", l.Email)
			}
		}(*lead)
	} else {
		slog.Warn("Mockup ready email skipped — email not configured", "lead_id", lead.ID)
	}

	return nil
}

const (
	// MilestoneRevisionsSignedOff is the index clients jump to when they accept a mockup.
	MilestoneRevisionsSignedOff = 2
	// MilestoneSiteCompleted is set by the admin when the site build is done.
	MilestoneSiteCompleted = 4
)

// SubmitReview handles the client's accept/request-changes decision on a mockup.
// decision must be "accept" or "request_changes". feedback is required for request_changes.
func (s *LeadService) SubmitReview(ctx context.Context, leadID, userID, decision, feedback string) error {
	lead, err := s.leadRepo.GetLeadByID(ctx, leadID)
	if err != nil {
		return err
	}
	if lead.UserID != userID {
		return errors.New("forbidden")
	}

	switch decision {
	case "accept":
		return s.leadRepo.UpdateLeadMilestone(ctx, leadID, MilestoneRevisionsSignedOff)

	case "request_changes":
		if feedback == "" {
			return errors.New("feedback is required when requesting changes")
		}
		if err := s.leadRepo.SetRevisionFeedback(ctx, leadID, feedback); err != nil {
			return err
		}
		if s.notifier != nil {
			go func(l model.Lead, fb string) {
				if err := s.notifier.SendRevisionRequestEmail(l.Email, l.Business, fb); err != nil {
					slog.Error("Failed to send revision request email", "lead_id", l.ID, "error", err)
				}
			}(*lead, feedback)
		}
		return nil

	default:
		return errors.New("decision must be 'accept' or 'request_changes'")
	}
}

// CompleteSite marks the site as complete (milestone 4) and emails the client to pay.
func (s *LeadService) CompleteSite(ctx context.Context, id, frontendURL string) error {
	lead, err := s.leadRepo.GetLeadByID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.leadRepo.UpdateLeadMilestone(ctx, id, MilestoneSiteCompleted); err != nil {
		return err
	}

	if s.notifier != nil {
		projectLink := frontendURL + "/my-projects"
		slog.Info("Sending payment request email", "lead_id", lead.ID, "to", lead.Email)
		go func(l model.Lead) {
			if err := s.notifier.SendPaymentRequestEmail(l.Email, projectLink); err != nil {
				slog.Error("Payment request email failed", "lead_id", l.ID, "to", l.Email, "error", err)
			} else {
				slog.Info("Payment request email sent", "lead_id", l.ID, "to", l.Email)
			}
		}(*lead)
	} else {
		slog.Warn("Payment request email skipped — email not configured", "lead_id", lead.ID)
	}

	return nil
}

// SetWantsMaintenance records the client's maintenance preference.
func (s *LeadService) SetWantsMaintenance(ctx context.Context, leadID, userID string, wants bool) error {
	lead, err := s.leadRepo.GetLeadByID(ctx, leadID)
	if err != nil {
		return err
	}
	if lead.UserID != userID {
		return errors.New("forbidden")
	}
	return s.leadRepo.SetWantsMaintenance(ctx, leadID, wants)
}

// MarkPaid records payment and emails the client a receipt.
func (s *LeadService) MarkPaid(ctx context.Context, leadID, userID string) error {
	lead, err := s.leadRepo.GetLeadByID(ctx, leadID)
	if err != nil {
		return err
	}
	if lead.UserID != userID {
		return errors.New("forbidden")
	}

	// Compute authoritative total server-side.
	pkgID := ""
	if lead.Package != nil {
		pkgID = *lead.Package
	}
	pkgPrice := packagePrices[pkgID]
	pkgName := packageNames[pkgID]
	if pkgName == "" {
		pkgName = pkgID
	}
	total := pkgPrice + domainFee
	if lead.WantsMaintenance {
		total += maintenanceFee
	}

	if err := s.leadRepo.MarkPaid(ctx, leadID, total); err != nil {
		return err
	}

	if s.notifier != nil {
		// Reload to get the DB-generated domain_renewal_date.
		updated, err := s.leadRepo.GetLeadByID(ctx, leadID)
		if err != nil {
			slog.Error("Failed to reload lead after payment", "lead_id", leadID, "error", err)
			updated = lead
		}
		renewalStr := "—"
		if updated.DomainRenewalDate != nil {
			renewalStr = updated.DomainRenewalDate.Format("Jan 2, 2006")
		}
		slog.Info("Sending payment receipt email", "lead_id", leadID, "to", updated.Email, "total", total)
		go func(l model.Lead, pName string, pPrice, tot int, renewal string) {
			if err := s.notifier.SendPaymentReceiptEmail(l.Email, l.Business, pName, pPrice, tot, l.WantsMaintenance, renewal); err != nil {
				slog.Error("Payment receipt email failed", "lead_id", l.ID, "to", l.Email, "error", err)
			} else {
				slog.Info("Payment receipt email sent", "lead_id", l.ID, "to", l.Email)
			}
		}(*updated, pkgName, pkgPrice, total, renewalStr)
	} else {
		slog.Warn("Payment receipt email skipped — email not configured", "lead_id", leadID)
	}

	return nil
}

// LaunchSite sets the site URL, marks the lead as launched, and emails the client.
func (s *LeadService) LaunchSite(ctx context.Context, id, siteURL string) error {
	lead, err := s.leadRepo.GetLeadByID(ctx, id)
	if err != nil {
		return err
	}
	if !lead.IsPaid {
		return errors.New("payment required before launch")
	}

	if err := s.leadRepo.SetLaunched(ctx, id, siteURL); err != nil {
		return err
	}

	if s.notifier != nil {
		slog.Info("Sending site launched email", "lead_id", lead.ID, "to", lead.Email)
		go func(l model.Lead) {
			if err := s.notifier.SendSiteLaunchedEmail(l.Email, siteURL, l.Business); err != nil {
				slog.Error("Site launched email failed", "lead_id", l.ID, "to", l.Email, "error", err)
			} else {
				slog.Info("Site launched email sent", "lead_id", l.ID, "to", l.Email)
			}
		}(*lead)
	} else {
		slog.Warn("Site launched email skipped — email not configured", "lead_id", lead.ID)
	}

	return nil
}
