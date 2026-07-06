package email

import (
	"fmt"
	"html"
	"log/slog"
	"os"

	"github.com/resend/resend-go/v2"
	shared "github.com/consultprompts/shared/email"
	"github.com/consultprompts/agency-service/internal/model"
)

// Client adapts the shared email client to the LeadNotifier interface.
type Client struct {
	shared *shared.Client
	resend *resend.Client
	from   string
}

// NewEmailClient returns nil when email is not configured.
func NewEmailClient() *Client {
	apiKey := os.Getenv("RESEND_API_KEY")
	from   := os.Getenv("RESEND_FROM")
	sharedClient := shared.NewClient()

	if sharedClient == nil || apiKey == "" || from == "" {
		slog.Warn("Lead email notifications disabled — set RESEND_API_KEY and RESEND_FROM to enable")
		return nil
	}

	return &Client{
		shared: sharedClient,
		resend: resend.NewClient(apiKey),
		from:   from,
	}
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

// SendLeadAccepted emails the customer when an admin accepts their project.
// Implemented directly here (not via shared) because the shared module is
// pinned to a pre-`SendLeadAccepted` commit — uses the same template style.
func (c *Client) SendLeadAccepted(lead model.Lead) error {
	frontendURL := os.Getenv("FRONTEND_URL")

	var pkgLine string
	if lead.Package != nil {
		pkgLine = fmt.Sprintf(`Package: <span style="color:#00F0FF;">%s</span>`, html.EscapeString(*lead.Package))
	}
	body := fmt.Sprintf(
		`Great news, %s — we've accepted your project for <strong style="color:#ffffff;">%s</strong> and we're getting started. Track every milestone in real time from your project dashboard.`,
		html.EscapeString(lead.Name),
		html.EscapeString(lead.Business),
	)
	if pkgLine != "" {
		body += "<br><br>" + pkgLine
	}

	logoURL := os.Getenv("LOGO_URL")
	var header string
	if logoURL != "" {
		header = fmt.Sprintf(`<table cellpadding="0" cellspacing="0"><tr><td style="vertical-align:middle; padding-right:12px;"><img src="%s" width="32" height="32" alt="" style="display:block; border:0;" /></td><td style="vertical-align:middle;"><span style="font-family:'Space Grotesk',Georgia,serif; font-size:17px; font-weight:700; letter-spacing:0.08em; text-transform:uppercase; color:#00F0FF;">Consult Prompts</span></td></tr></table>`, logoURL)
	} else {
		header = `<span style="font-family:'Space Grotesk',Georgia,serif; font-size:17px; font-weight:700; letter-spacing:0.08em; text-transform:uppercase; color:#00F0FF;">Consult Prompts</span>`
	}

	emailHTML := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <meta name="color-scheme" content="dark">
  <meta name="supported-color-schemes" content="dark light">
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link href="https://fonts.googleapis.com/css2?family=Space+Grotesk:wght@500;700&family=Inter:wght@300;400;700;900&display=swap" rel="stylesheet">
  <style>
    :root { color-scheme: dark; }
    @media (prefers-color-scheme: dark) {
      body { background:#050505 !important; }
      .email-wrapper { background:#050505 !important; }
      .email-card { background:#0f0f0f !important; }
      .email-footer { background:#0f0f0f !important; }
    }
  </style>
</head>
<body style="margin:0; padding:0; background:#050505; font-family:'Inter',Arial,Helvetica,sans-serif; color:#ffffff;">
  <table class="email-wrapper" width="100%%" cellpadding="0" cellspacing="0" style="background:#050505; padding:48px 24px 80px;">
    <tr><td align="center">
      <table class="email-card" width="560" cellpadding="0" cellspacing="0" style="max-width:560px; width:100%%; background:#0f0f0f; border:1px solid rgba(255,255,255,0.12); border-radius:14px; overflow:hidden;">
        <tr><td style="background:linear-gradient(135deg,#00F0FF22,#7000FF22); padding:28px 40px; border-bottom:1px solid rgba(255,255,255,0.08);">%s</td></tr>
        <tr><td style="padding:44px 40px 40px;">
          <table cellpadding="0" cellspacing="0" style="margin-bottom:24px;"><tr>
            <td width="56" height="56" style="width:56px; height:56px; background:rgba(0,240,255,0.1); border-radius:28px; text-align:center; vertical-align:middle;">
              <span style="font-size:22px; line-height:56px; display:block;">&#128640;</span>
            </td>
          </tr></table>
          <h2 style="margin:0 0 10px; font-family:'Space Grotesk',Georgia,serif; font-style:italic; font-size:26px; font-weight:700; letter-spacing:-0.02em; color:#ffffff;">Project Accepted!</h2>
          <p style="margin:0 0 30px; color:#A1A1A1; font-size:14px; font-weight:300; line-height:1.7;">%s</p>
          <table cellpadding="0" cellspacing="0" style="margin-bottom:28px;"><tr>
            <td style="background:#00F0FF; border-radius:8px;">
              <a href="%s/my-projects" style="display:inline-block; padding:15px 30px; font-size:12px; font-weight:900; letter-spacing:0.14em; text-transform:uppercase; color:#050505; text-decoration:none; font-family:'Inter',Arial,Helvetica,sans-serif;">Track My Project</a>
            </td>
          </tr></table>
          <p style="margin:0; font-size:12px; color:#555555; line-height:1.6;">Have questions? Just reply to this email — we read every one.</p>
        </td></tr>
        <tr><td class="email-footer" style="padding:22px 40px; border-top:1px solid rgba(255,255,255,0.08); text-align:center; background:#0f0f0f;">
          <p style="margin:0; font-size:10px; color:#555555; letter-spacing:0.1em; text-transform:uppercase;">consultprompts.com</p>
        </td></tr>
      </table>
    </td></tr>
  </table>
</body>
</html>`, header, body, frontendURL)

	_, err := c.resend.Emails.Send(&resend.SendEmailRequest{
		From:    c.from,
		To:      []string{lead.Email},
		Subject: "Your project has been accepted — consultprompts.com",
		Html:    emailHTML,
	})
	if err != nil {
		slog.Error("Failed to send lead accepted email", "lead_id", lead.ID, "error", err)
	}
	return err
}
