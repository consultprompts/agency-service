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

const emailWrapper = `<!DOCTYPE html>
<html lang="en">
<head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1"></head>
<body style="margin:0;padding:0;background:#050505;font-family:Arial,Helvetica,sans-serif;color:#ffffff;">
  <table width="100%%" cellpadding="0" cellspacing="0" style="background:#050505;padding:48px 16px;">
    <tr><td align="center">
      <table width="560" cellpadding="0" cellspacing="0" style="max-width:560px;width:100%%;background:#0f0f0f;border:1px solid rgba(255,255,255,0.12);border-radius:12px;overflow:hidden;">
        <!-- header -->
        <tr>
          <td style="background:linear-gradient(135deg,#00F0FF22,#7000FF22);padding:32px 40px;border-bottom:1px solid rgba(255,255,255,0.08);">
            <span style="font-size:18px;font-weight:900;letter-spacing:0.15em;text-transform:uppercase;color:#00F0FF;">CONSULTPROMPTS</span>
          </td>
        </tr>
        <!-- body -->
        <tr><td style="padding:40px;">%s</td></tr>
        <!-- footer -->
        <tr>
          <td style="padding:24px 40px;border-top:1px solid rgba(255,255,255,0.08);text-align:center;">
            <p style="margin:0;font-size:11px;color:#555555;letter-spacing:0.08em;text-transform:uppercase;">
              consultprompts.com &nbsp;·&nbsp; Helping local businesses look world-class
            </p>
          </td>
        </tr>
      </table>
    </td></tr>
  </table>
</body>
</html>`

func row(label, value string) string {
	return fmt.Sprintf(`
    <tr>
      <td style="padding:10px 0;color:#A1A1A1;font-size:12px;letter-spacing:0.1em;text-transform:uppercase;width:120px;vertical-align:top;">%s</td>
      <td style="padding:10px 0;color:#ffffff;font-size:14px;vertical-align:top;">%s</td>
    </tr>`, label, value)
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

	body := fmt.Sprintf(`
    <h2 style="margin:0 0 8px;font-size:24px;font-weight:900;letter-spacing:-0.02em;color:#ffffff;">New Lead</h2>
    <p style="margin:0 0 32px;color:#A1A1A1;font-size:14px;">A new mockup request was submitted on consultprompts.com</p>
    <table width="100%%" cellpadding="0" cellspacing="0" style="border-top:1px solid rgba(255,255,255,0.08);">
      %s%s%s%s%s
    </table>
    <p style="margin:32px 0 0;font-size:12px;color:#555555;">Submitted %s</p>`,
		row("Name", html.EscapeString(lead.Name)),
		row("Email", html.EscapeString(lead.Email)),
		row("Business", html.EscapeString(lead.Business)),
		row("Package", pkg),
		row("Message", message),
		lead.CreatedAt.Format("2006-01-02 15:04 MST"),
	)

	params := &resend.SendEmailRequest{
		From:    e.from,
		To:      []string{e.notifyTo},
		Subject: fmt.Sprintf("New lead: %s", lead.Business),
		Html:    fmt.Sprintf(emailWrapper, body),
	}

	_, err := e.client.Emails.Send(params)
	return err
}

func (e *EmailClient) SendLeadConfirmation(lead model.Lead) error {
	pkgRow := ""
	if lead.Package != nil {
		pkgRow = fmt.Sprintf(
			`<p style="margin:0 0 8px;font-size:13px;color:#A1A1A1;">Package: <span style="color:#00F0FF;">%s</span></p>`,
			html.EscapeString(*lead.Package),
		)
	}

	body := fmt.Sprintf(`
    <h2 style="margin:0 0 8px;font-size:24px;font-weight:900;letter-spacing:-0.02em;color:#ffffff;">Transmission Received</h2>
    <p style="margin:0 0 24px;color:#A1A1A1;font-size:14px;line-height:1.6;">
      Hey %s — we've got your request for <strong style="color:#ffffff;">%s</strong> and we're already on it.<br>
      Expect your free mockup within <span style="color:#00F0FF;font-weight:700;">24–48 hours</span>.
    </p>
    %s
    <table cellpadding="0" cellspacing="0" style="margin:32px 0;">
      <tr>
        <td style="background:#00F0FF;border-radius:6px;">
          <a href="https://wa.me/13026622736" style="display:inline-block;padding:14px 28px;font-size:13px;font-weight:900;letter-spacing:0.12em;text-transform:uppercase;color:#050505;text-decoration:none;">
            Chat on WhatsApp
          </a>
        </td>
      </tr>
    </table>
    <p style="margin:0;font-size:12px;color:#555555;line-height:1.6;">
      Have questions? Just reply to this email — we read every one.
    </p>`,
		html.EscapeString(lead.Name),
		html.EscapeString(lead.Business),
		pkgRow,
	)

	params := &resend.SendEmailRequest{
		From:    e.from,
		To:      []string{lead.Email},
		Subject: "Transmission received — your mockup is in the queue",
		Html:    fmt.Sprintf(emailWrapper, body),
	}

	_, err := e.client.Emails.Send(params)
	return err
}
