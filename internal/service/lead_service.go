package service

import (
	"context"
	"errors"
	"fmt"
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
	SendRevisedMockupEmail(to, projectLink string) error
	SendRevisionRequestEmail(clientEmail, businessName, feedback string) error
	SendRevisionRequestConfirmationEmail(to, businessName string) error
	SendPaymentRequestEmail(to, projectLink string) error
	SendSiteLaunchedEmail(to, siteURL, businessName string) error
	SendPaymentReceiptEmail(to, businessName, packageName string, packagePrice, totalAmount int, wantsMaintenance bool, domainRenewalDate string) error
	SendMeetingRequestEmail(clientName, clientEmail, business string) error
	SendProjectSuspendedEmail(to, name, business string) error
	SendProjectReactivatedEmail(to, name, business string) error
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

var ErrLeadNotPending = errors.New("lead can only be edited while it is pending review")

func (s *LeadService) UpdateLead(ctx context.Context, id, userID string, lead model.Lead) error {
	existing, err := s.leadRepo.GetLeadByID(ctx, id)
	if err != nil {
		return err
	}
	if existing.UserID != userID {
		return errors.New("forbidden")
	}
	if existing.Status != "pending" {
		return ErrLeadNotPending
	}

	lead.UserID = userID
	if !lead.WantsCall {
		lead.MeetingSkipped = true
		lead.MilestoneIndex = model.MilestoneMeeting
	} else {
		lead.MeetingSkipped = false
		lead.MilestoneIndex = model.MilestoneNone
	}

	return s.leadRepo.UpdateLead(ctx, id, lead)
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

	// Skipping the 15-minute call means there's nothing to wait on — check
	// off "Meeting Completed" immediately instead of parking the project on
	// a call that will never happen. The client-facing tracker shows this
	// milestone as "Meeting Skipped" rather than "Meeting Completed".
	if !lead.WantsCall {
		lead.MeetingSkipped = true
		lead.MilestoneIndex = model.MilestoneMeeting
	}

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

// UpdateLeadMilestone handles the admin's direct interactions with the
// checklist: accepting a lead (index 0 while pending), checking off "Meeting
// Completed" (index 1), and undoing to any earlier milestone. Every other
// milestone has a dedicated flow — mockup approval (SubmitReview), site
// completion (CompleteSite), payment (webhook / MarkPaid), launch (LaunchSite)
// — and is rejected here so those flows can't be bypassed.
func (s *LeadService) UpdateLeadMilestone(ctx context.Context, id string, milestoneIndex int) error {
	lead, err := s.leadRepo.GetLeadByID(ctx, id)
	if err != nil {
		return err
	}

	if milestoneIndex < model.MilestoneNone || milestoneIndex > model.MilestoneLive {
		return fmt.Errorf("milestone_index must be between %d and %d", model.MilestoneNone, model.MilestoneLive)
	}
	if lead.IsPaid && milestoneIndex < model.MilestonePayment {
		return errors.New("cannot undo milestones past a confirmed payment")
	}
	if milestoneIndex > lead.MilestoneIndex && milestoneIndex != model.MilestoneMeeting {
		switch milestoneIndex {
		case model.MilestoneMockup, model.MilestoneApproved:
			return errors.New("Mockup Completed and Design Approved are checked automatically when the client approves the design")
		case model.MilestoneWebsite:
			return errors.New("Website Completed is set via the complete endpoint")
		case model.MilestonePayment:
			return errors.New("Payment Completed is checked automatically when payment is confirmed")
		default: // model.MilestoneLive
			return errors.New("Website is Live is set via the launch endpoint")
		}
	}

	wasPending := lead.Status == "pending"
	isAccept := milestoneIndex == model.MilestoneNone && wasPending

	// When accepting a lead that skipped the intro call, the meeting milestone
	// is already done — write MilestoneMeeting so the checklist reflects that.
	if isAccept && lead.MeetingSkipped {
		milestoneIndex = model.MilestoneMeeting
	}

	if err := s.leadRepo.UpdateLeadMilestone(ctx, id, milestoneIndex); err != nil {
		return err
	}

	if isAccept && s.notifier != nil {
		go func(l model.Lead) {
			if err := s.notifier.SendLeadAccepted(l); err != nil {
				slog.Error("Failed to send lead accepted email", "lead_id", l.ID, "error", err)
			}
		}(*lead)
	}

	return nil
}

// SetMockupURL saves the mockup URL and emails the client a review link.
// "Mockup Completed" deliberately stays UNCHECKED — the milestone only
// completes (together with "Design Approved") when the client approves the
// design via SubmitReview.
func (s *LeadService) SetMockupURL(ctx context.Context, id, url, frontendURL string) error {
	lead, err := s.leadRepo.GetLeadByID(ctx, id)
	if err != nil {
		return err
	}
	if lead.MilestoneIndex < model.MilestoneMeeting {
		return errors.New("check off Meeting Completed before sending a mockup")
	}
	if lead.MilestoneIndex >= model.MilestoneApproved {
		return errors.New("the design has already been approved")
	}

	if err := s.leadRepo.SetMockupURL(ctx, id, url); err != nil {
		return err
	}

	// Advance to MilestoneMockup on first send so the checklist reflects that
	// the mockup is ready. On re-sends after client changes the index is already
	// at MilestoneMockup, so we leave it untouched.
	if lead.MilestoneIndex == model.MilestoneMeeting {
		if err := s.leadRepo.UpdateLeadMilestone(ctx, id, model.MilestoneMockup); err != nil {
			return err
		}
	}

	if s.notifier != nil {
		projectLink := frontendURL + "/my-projects"
		isSecondRevision := lead.RevisionCount >= 2
		slog.Info("Sending mockup email", "lead_id", lead.ID, "to", lead.Email, "revision_count", lead.RevisionCount)
		go func(l model.Lead, secondRevision bool) {
			var err error
			if secondRevision {
				err = s.notifier.SendRevisedMockupEmail(l.Email, projectLink)
			} else {
				err = s.notifier.SendMockupReadyEmail(l.Email, projectLink)
			}
			if err != nil {
				slog.Error("Mockup email failed", "lead_id", l.ID, "to", l.Email, "error", err)
			} else {
				slog.Info("Mockup email sent", "lead_id", l.ID, "to", l.Email)
			}
		}(*lead, isSecondRevision)
	} else {
		slog.Warn("Mockup ready email skipped — email not configured", "lead_id", lead.ID)
	}

	return nil
}

// SubmitReview handles the client's accept/request-changes decision on a mockup.
// decision must be "accept" or "request_changes". feedback is required for request_changes.
//
// Accepting checks off "Design Approved" (index 2 -> 3); "Mockup Completed"
// was already checked when the admin sent the URL.
func (s *LeadService) SubmitReview(ctx context.Context, leadID, userID, decision, feedback string) error {
	lead, err := s.leadRepo.GetLeadByID(ctx, leadID)
	if err != nil {
		return err
	}
	if lead.UserID != userID {
		return errors.New("forbidden")
	}
	if lead.MockupURL == nil || lead.MilestoneIndex != model.MilestoneMockup {
		return errors.New("there is no mockup awaiting review")
	}

	switch decision {
	case "accept":
		return s.leadRepo.UpdateLeadMilestone(ctx, leadID, model.MilestoneApproved)

	case "request_changes":
		if feedback == "" {
			return errors.New("feedback is required when requesting changes")
		}
		// Reset to MilestoneMeeting so "Mockup Completed" unchecks — the admin
		// sends a revised URL which re-advances it to MilestoneMockup.
		if err := s.leadRepo.UpdateLeadMilestone(ctx, leadID, model.MilestoneMeeting); err != nil {
			return err
		}
		if err := s.leadRepo.SetRevisionFeedback(ctx, leadID, feedback); err != nil {
			return err
		}
		if err := s.leadRepo.IncrementRevisionCount(ctx, leadID); err != nil {
			return err
		}
		if s.notifier != nil {
			go func(l model.Lead, fb string) {
				if err := s.notifier.SendRevisionRequestEmail(l.Email, l.Business, fb); err != nil {
					slog.Error("Failed to send revision request email", "lead_id", l.ID, "error", err)
				}
				if err := s.notifier.SendRevisionRequestConfirmationEmail(l.Email, l.Business); err != nil {
					slog.Error("Failed to send revision request confirmation email", "lead_id", l.ID, "error", err)
				}
			}(*lead, feedback)
		}
		return nil

	default:
		return errors.New("decision must be 'accept' or 'request_changes'")
	}
}

// CompleteSite checks off "Website Completed" and emails the client a payment
// request — the tracker then shows "Awaiting Final Payment" with the payment UI.
func (s *LeadService) CompleteSite(ctx context.Context, id, frontendURL string) error {
	lead, err := s.leadRepo.GetLeadByID(ctx, id)
	if err != nil {
		return err
	}
	if lead.MilestoneIndex != model.MilestoneApproved {
		return errors.New("the design must be approved before the website can be marked complete")
	}

	if err := s.leadRepo.UpdateLeadMilestone(ctx, id, model.MilestoneWebsite); err != nil {
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

// MarkPaid records a payment initiated by the client from the tracker's
// payment UI. Ownership-checked, then delegates to recordPayment.
func (s *LeadService) MarkPaid(ctx context.Context, leadID, userID string) error {
	lead, err := s.leadRepo.GetLeadByID(ctx, leadID)
	if err != nil {
		return err
	}
	if lead.UserID != userID {
		return errors.New("forbidden")
	}
	return s.recordPayment(ctx, lead)
}

// ConfirmPaymentWebhook handles a payment provider's server-to-server success
// callback. Idempotent: replayed webhooks for an already-paid project succeed
// without side effects, so provider retries are safe.
func (s *LeadService) ConfirmPaymentWebhook(ctx context.Context, leadID string) error {
	lead, err := s.leadRepo.GetLeadByID(ctx, leadID)
	if err != nil {
		return err
	}
	if lead.IsPaid {
		return nil
	}
	return s.recordPayment(ctx, lead)
}

// recordPayment computes the authoritative total server-side, marks the lead
// paid, checks off "Payment Completed" (index 5), and emails a receipt.
func (s *LeadService) recordPayment(ctx context.Context, lead *model.Lead) error {
	if lead.IsPaid {
		return errors.New("payment has already been recorded")
	}
	if lead.MilestoneIndex != model.MilestoneWebsite {
		return errors.New("the project is not awaiting payment")
	}

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

	if err := s.leadRepo.MarkPaid(ctx, lead.ID, total); err != nil {
		return err
	}
	if err := s.leadRepo.UpdateLeadMilestone(ctx, lead.ID, model.MilestonePayment); err != nil {
		return err
	}

	if s.notifier != nil {
		// Reload to get the DB-generated domain_renewal_date.
		updated, err := s.leadRepo.GetLeadByID(ctx, lead.ID)
		if err != nil {
			slog.Error("Failed to reload lead after payment", "lead_id", lead.ID, "error", err)
			updated = lead
		}
		renewalStr := "—"
		if updated.DomainRenewalDate != nil {
			renewalStr = updated.DomainRenewalDate.Format("Jan 2, 2006")
		}
		slog.Info("Sending payment receipt email", "lead_id", lead.ID, "to", updated.Email, "total", total)
		go func(l model.Lead, pName string, pPrice, tot int, renewal string) {
			if err := s.notifier.SendPaymentReceiptEmail(l.Email, l.Business, pName, pPrice, tot, l.WantsMaintenance, renewal); err != nil {
				slog.Error("Payment receipt email failed", "lead_id", l.ID, "to", l.Email, "error", err)
			} else {
				slog.Info("Payment receipt email sent", "lead_id", l.ID, "to", l.Email)
			}
		}(*updated, pkgName, pkgPrice, total, renewalStr)
	} else {
		slog.Warn("Payment receipt email skipped — email not configured", "lead_id", lead.ID)
	}

	return nil
}

// LaunchSite checks off the final milestone: saves the live site URL, marks
// the lead launched, and emails the client. Payment must be confirmed first.
func (s *LeadService) LaunchSite(ctx context.Context, id, siteURL string) error {
	lead, err := s.leadRepo.GetLeadByID(ctx, id)
	if err != nil {
		return err
	}
	if !lead.IsPaid {
		return errors.New("payment required before launch")
	}

	if err := s.leadRepo.SetLaunched(ctx, id, siteURL, model.MilestoneLive); err != nil {
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

// SuspendLead pauses an in-flight project. The current status is saved to
// pre_suspend_status so ReactivateLead can restore it exactly. Returns the
// new status so callers can update their view without a refetch.
func (s *LeadService) SuspendLead(ctx context.Context, leadID string) (string, error) {
	lead, err := s.leadRepo.GetLeadByID(ctx, leadID)
	if err != nil {
		return "", err
	}
	if lead.Status == "suspended" {
		return "", errors.New("project is already suspended")
	}
	if lead.Status == "launched" || lead.Status == "completed" {
		return "", errors.New("a launched project cannot be suspended")
	}

	current := lead.Status
	if err := s.leadRepo.SetSuspended(ctx, leadID, "suspended", &current); err != nil {
		return "", err
	}

	if s.notifier != nil {
		go func(l model.Lead) {
			if err := s.notifier.SendProjectSuspendedEmail(l.Email, l.Name, l.Business); err != nil {
				slog.Error("Failed to send project suspended email", "lead_id", l.ID, "error", err)
			}
		}(*lead)
	}

	return "suspended", nil
}

// ReactivateLead resumes a suspended project, restoring whatever status it
// had before suspension. Returns the restored status.
func (s *LeadService) ReactivateLead(ctx context.Context, leadID string) (string, error) {
	lead, err := s.leadRepo.GetLeadByID(ctx, leadID)
	if err != nil {
		return "", err
	}
	if lead.Status != "suspended" {
		return "", errors.New("project is not suspended")
	}

	resumeStatus := "accepted"
	if lead.PreSuspendStatus != nil && *lead.PreSuspendStatus != "" {
		resumeStatus = *lead.PreSuspendStatus
	}
	if err := s.leadRepo.SetSuspended(ctx, leadID, resumeStatus, nil); err != nil {
		return "", err
	}

	if s.notifier != nil {
		go func(l model.Lead) {
			if err := s.notifier.SendProjectReactivatedEmail(l.Email, l.Name, l.Business); err != nil {
				slog.Error("Failed to send project reactivated email", "lead_id", l.ID, "error", err)
			}
		}(*lead)
	}

	return resumeStatus, nil
}

// RequestMeeting lets a client who initially skipped the 15-minute call ask
// for one after all. Only valid while the meeting is still marked skipped —
// once it's requested the admin follows up outside the tracker, so this is a
// one-shot notification rather than a milestone change.
func (s *LeadService) RequestMeeting(ctx context.Context, leadID, userID string) error {
	lead, err := s.leadRepo.GetLeadByID(ctx, leadID)
	if err != nil {
		return err
	}
	if lead.UserID != userID {
		return errors.New("forbidden")
	}
	if !lead.MeetingSkipped {
		return errors.New("the meeting was not skipped for this project")
	}

	if s.notifier != nil {
		go func(l model.Lead) {
			if err := s.notifier.SendMeetingRequestEmail(l.Name, l.Email, l.Business); err != nil {
				slog.Error("Failed to send meeting request email", "lead_id", l.ID, "error", err)
			}
		}(*lead)
	} else {
		slog.Warn("Meeting request email skipped — email not configured", "lead_id", lead.ID)
	}

	return nil
}
