package email

import (
	"log/slog"
	"os"

	shared "github.com/consultprompts/shared/email"
	"github.com/consultprompts/agency-service/internal/model"
)

// Client adapts the shared email client to the LeadNotifier interface expected
// by agency-service. It reads LEAD_NOTIFICATION_EMAIL from env.
type Client struct {
	shared   *shared.Client
	notifyTo string
}

// NewEmailClient returns nil when email is not configured so callers can treat
// it as optional. Logs a warning at startup.
func NewEmailClient() *Client {
	sharedClient := shared.NewClient()
	notifyTo := os.Getenv("LEAD_NOTIFICATION_EMAIL")

	if sharedClient == nil || notifyTo == "" {
		slog.Warn("Lead email notifications disabled — set RESEND_API_KEY, RESEND_FROM and LEAD_NOTIFICATION_EMAIL to enable")
		return nil
	}

	return &Client{shared: sharedClient, notifyTo: notifyTo}
}

func (c *Client) SendNewLeadNotification(lead model.Lead) error {
	return c.shared.SendNewLeadNotification(c.notifyTo, shared.LeadData{
		Name:      lead.Name,
		Email:     lead.Email,
		Business:  lead.Business,
		Package:   lead.Package,
		Message:   lead.Message,
		CreatedAt: lead.CreatedAt,
	})
}

func (c *Client) SendLeadConfirmation(lead model.Lead) error {
	return c.shared.SendLeadConfirmation(shared.LeadData{
		Name:     lead.Name,
		Email:    lead.Email,
		Business: lead.Business,
		Package:  lead.Package,
	})
}
