package email

import (
	"log/slog"

	shared "github.com/consultprompts/shared/email"
	"github.com/consultprompts/agency-service/internal/model"
)

// Client adapts the shared email client to the LeadNotifier interface expected
// by agency-service.
type Client struct {
	shared *shared.Client
}

// NewEmailClient returns nil when email is not configured so callers can treat
// it as optional. Logs a warning at startup.
func NewEmailClient() *Client {
	sharedClient := shared.NewClient()

	if sharedClient == nil {
		slog.Warn("Lead email notifications disabled — set RESEND_API_KEY and RESEND_FROM to enable")
		return nil
	}

	return &Client{shared: sharedClient}
}

func (c *Client) SendNewLeadNotification(lead model.Lead) error {
	return c.shared.SendNewLeadNotification(shared.LeadData{
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
