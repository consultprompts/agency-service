package email

import (
	"fmt"
	"html"
	"os"

	"github.com/consultprompts/agency-service/internal/model"
	"github.com/resend/resend-go/v2"
)

type EmailClient struct {
	client   *resend.Client
	from     string
	notifyTo string
}

// NewEmailClient returns nil when lead notifications are not configured,
// so callers can treat email as an optional feature.
func NewEmailClient() *EmailClient {
	apiKey := os.Getenv("RESEND_API_KEY")
	from := os.Getenv("RESEND_FROM")
	notifyTo := os.Getenv("LEAD_NOTIFICATION_EMAIL")

	if apiKey == "" || from == "" || notifyTo == "" {
		return nil
	}

	return &EmailClient{
		client:   resend.NewClient(apiKey),
		from:     from,
		notifyTo: notifyTo,
	}
}

func (e *EmailClient) SendNewLeadNotification(lead model.Lead) error {
	pkg := "—"
	if lead.Package != nil {
		pkg = html.EscapeString(*lead.Package)
	}
	message := "—"
	if lead.Message != nil {
		message = html.EscapeString(*lead.Message)
	}

	params := &resend.SendEmailRequest{
		From:    e.from,
		To:      []string{e.notifyTo},
		Subject: fmt.Sprintf("New lead: %s — consultprompts.com", lead.Business),
		Html: fmt.Sprintf(`
			<h2>New lead submitted</h2>
			<p><strong>Name:</strong> %s</p>
			<p><strong>Email:</strong> %s</p>
			<p><strong>Business:</strong> %s</p>
			<p><strong>Package:</strong> %s</p>
			<p><strong>Message:</strong> %s</p>
			<p>Submitted at %s</p>
		`,
			html.EscapeString(lead.Name),
			html.EscapeString(lead.Email),
			html.EscapeString(lead.Business),
			pkg,
			message,
			lead.CreatedAt.Format("2006-01-02 15:04 MST"),
		),
	}

	_, err := e.client.Emails.Send(params)
	return err
}
