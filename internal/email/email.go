package email

import (
	"fmt"
	"html"
	"log/slog"
	"os"

	"github.com/resend/resend-go/v2"

	"github.com/consultprompts/agency-service/internal/model"
	shared "github.com/consultprompts/shared/email"
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
	from := os.Getenv("RESEND_FROM")
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

// SendMockupReadyEmail notifies the client that their mockup is ready to review.
func (c *Client) SendMockupReadyEmail(to, projectLink string) error {
	body := fmt.Sprintf(`
    <table cellpadding="0" cellspacing="0" style="margin-bottom:24px;"><tr>
      <td width="56" height="56" style="width:56px; height:56px; background:rgba(0,240,255,0.1); border-radius:28px; text-align:center; vertical-align:middle;">
        <span style="font-size:22px; line-height:56px; display:block;">&#127912;</span>
      </td>
    </tr></table>
    <h2 style="margin:0 0 10px; font-family:'Space Grotesk',Georgia,serif; font-style:italic; font-size:26px; font-weight:700; letter-spacing:-0.02em; color:#ffffff;">Your Mockup is Ready!</h2>
    <p style="margin:0 0 30px; color:#A1A1A1; font-size:14px; font-weight:300; line-height:1.7;">We've finished your initial mockup design. Open your project page to view it and let us know whether you'd like to move forward or request revisions.</p>
    <table cellpadding="0" cellspacing="0" style="margin-bottom:28px;"><tr>
      <td style="background:#00F0FF; border-radius:8px;">
        <a href="%s" style="display:inline-block; padding:15px 30px; font-size:12px; font-weight:900; letter-spacing:0.14em; text-transform:uppercase; color:#050505; text-decoration:none; font-family:'Inter',Arial,Helvetica,sans-serif;">Review My Mockup</a>
      </td>
    </tr></table>
    <p style="margin:0; font-size:12px; color:#555555; line-height:1.6;">Have questions? Just reply to this email — we read every one.</p>
`, projectLink)

	_, err := c.resend.Emails.Send(&resend.SendEmailRequest{
		From:    c.from,
		To:      []string{to},
		Subject: "Your mockup is ready to review — consultprompts.com",
		Html:    openEmail() + body + closeEmail(),
	})
	if err != nil {
		slog.Error("Failed to send mockup ready email", "to", to, "error", err)
	}
	return err
}

// SendRevisedMockupEmail notifies the client that their revised mockup (after a
// second or later change request) is ready to review.
func (c *Client) SendRevisedMockupEmail(to, projectLink string) error {
	body := fmt.Sprintf(`
    <table cellpadding="0" cellspacing="0" style="margin-bottom:24px;"><tr>
      <td width="56" height="56" style="width:56px; height:56px; background:rgba(112,0,255,0.1); border-radius:28px; text-align:center; vertical-align:middle;">
        <span style="font-size:22px; line-height:56px; display:block;">&#10055;</span>
      </td>
    </tr></table>
    <h2 style="margin:0 0 10px; font-family:'Space Grotesk',Georgia,serif; font-style:italic; font-size:26px; font-weight:700; letter-spacing:-0.02em; color:#ffffff;">Your Revised Mockup is Ready!</h2>
    <p style="margin:0 0 30px; color:#A1A1A1; font-size:14px; font-weight:300; line-height:1.7;">We've carefully worked through all your feedback and updated your design. Open your project page to review the changes — we believe this one hits the mark.</p>
    <table cellpadding="0" cellspacing="0" style="margin-bottom:28px;"><tr>
      <td style="background:#B98CFF; border-radius:8px;">
        <a href="%s" style="display:inline-block; padding:15px 30px; font-size:12px; font-weight:900; letter-spacing:0.14em; text-transform:uppercase; color:#050505; text-decoration:none; font-family:'Inter',Arial,Helvetica,sans-serif;">Review My Mockup</a>
      </td>
    </tr></table>
    <p style="margin:0; font-size:12px; color:#555555; line-height:1.6;">Have questions? Just reply to this email — we read every one.</p>
`, projectLink)

	_, err := c.resend.Emails.Send(&resend.SendEmailRequest{
		From:    c.from,
		To:      []string{to},
		Subject: "Your revised mockup is ready to review — consultprompts.com",
		Html:    openEmail() + body + closeEmail(),
	})
	if err != nil {
		slog.Error("Failed to send revised mockup email", "to", to, "error", err)
	}
	return err
}

// SendRevisionRequestEmail notifies the admin that a client has requested changes.
func (c *Client) SendRevisionRequestEmail(clientEmail, businessName, feedback string) error {
	adminEmail := os.Getenv("ADMIN_NOTIFICATION_EMAIL")
	if adminEmail == "" {
		slog.Warn("ADMIN_NOTIFICATION_EMAIL not set; skipping revision request email")
		return nil
	}

	body := fmt.Sprintf(`
    <table cellpadding="0" cellspacing="0" style="margin-bottom:24px;"><tr>
      <td width="56" height="56" style="width:56px; height:56px; background:rgba(245,197,66,0.1); border-radius:28px; text-align:center; vertical-align:middle;">
        <span style="font-size:22px; line-height:56px; display:block;">&#9999;&#65039;</span>
      </td>
    </tr></table>
    <h2 style="margin:0 0 10px; font-family:'Space Grotesk',Georgia,serif; font-style:italic; font-size:26px; font-weight:700; letter-spacing:-0.02em; color:#ffffff;">Revision Requested</h2>
    <p style="margin:0 0 20px; color:#A1A1A1; font-size:14px; font-weight:300; line-height:1.7;"><strong style="color:#ffffff;">%s</strong> (<a href="mailto:%s" style="color:#00F0FF;">%s</a>) has requested changes to their mockup.</p>
    <div style="background:#050505; border:1px solid rgba(255,255,255,0.08); border-radius:8px; padding:20px; margin-bottom:28px;">
      <p style="margin:0 0 8px; font-size:10px; font-weight:700; letter-spacing:0.1em; text-transform:uppercase; color:#A1A1A1;">Client Feedback</p>
      <p style="margin:0; color:#ffffff; font-size:14px; font-weight:300; line-height:1.7; white-space:pre-wrap;">%s</p>
    </div>
    <p style="margin:0; font-size:12px; color:#555555; line-height:1.6;">Log in to the admin console to update the project and deliver a revised mockup.</p>
`,
		html.EscapeString(businessName),
		html.EscapeString(clientEmail),
		html.EscapeString(clientEmail),
		html.EscapeString(feedback),
	)

	_, err := c.resend.Emails.Send(&resend.SendEmailRequest{
		From:    c.from,
		To:      []string{adminEmail},
		Subject: fmt.Sprintf("Revision requested: %s — consultprompts.com", businessName),
		Html:    openEmail() + body + closeEmail(),
	})
	if err != nil {
		slog.Error("Failed to send revision request email", "business", businessName, "error", err)
	}
	return err
}

// SendRevisionRequestConfirmationEmail confirms to the client that their revision
// request was received and a new mockup is on the way.
func (c *Client) SendRevisionRequestConfirmationEmail(to, businessName string) error {
	body := fmt.Sprintf(`
    <table cellpadding="0" cellspacing="0" style="margin-bottom:24px;"><tr>
      <td width="56" height="56" style="width:56px; height:56px; background:rgba(245,197,66,0.1); border-radius:28px; text-align:center; vertical-align:middle;">
        <span style="font-size:22px; line-height:56px; display:block;">&#9989;</span>
      </td>
    </tr></table>
    <h2 style="margin:0 0 10px; font-family:'Space Grotesk',Georgia,serif; font-style:italic; font-size:26px; font-weight:700; letter-spacing:-0.02em; color:#ffffff;">Revision Request Received</h2>
    <p style="margin:0 0 30px; color:#A1A1A1; font-size:14px; font-weight:300; line-height:1.7;">We've received your requested changes for <strong style="color:#ffffff;">%s</strong>. We'll get to work on an updated mockup and let you know as soon as it's ready to review.</p>
    <p style="margin:0; font-size:12px; color:#555555; line-height:1.6;">Have questions? Just reply to this email — we read every one.</p>
`, html.EscapeString(businessName))

	_, err := c.resend.Emails.Send(&resend.SendEmailRequest{
		From:    c.from,
		To:      []string{to},
		Subject: "We've received your revision request — consultprompts.com",
		Html:    openEmail() + body + closeEmail(),
	})
	if err != nil {
		slog.Error("Failed to send revision request confirmation email", "to", to, "error", err)
	}
	return err
}

// SendPaymentRequestEmail notifies the client that their site is complete and payment is due.
func (c *Client) SendPaymentRequestEmail(to, projectLink string) error {
	body := fmt.Sprintf(`
    <table cellpadding="0" cellspacing="0" style="margin-bottom:24px;"><tr>
      <td width="56" height="56" style="width:56px; height:56px; background:rgba(0,240,255,0.1); border-radius:28px; text-align:center; vertical-align:middle;">
        <span style="font-size:22px; line-height:56px; display:block;">&#127881;</span>
      </td>
    </tr></table>
    <h2 style="margin:0 0 10px; font-family:'Space Grotesk',Georgia,serif; font-style:italic; font-size:26px; font-weight:700; letter-spacing:-0.02em; color:#ffffff;">Your Site is Complete!</h2>
    <p style="margin:0 0 30px; color:#A1A1A1; font-size:14px; font-weight:300; line-height:1.7;">Your website is built and ready to launch. Complete your payment through your project page and we'll get it live immediately.</p>
    <table cellpadding="0" cellspacing="0" style="margin-bottom:28px;"><tr>
      <td style="background:#00F0FF; border-radius:8px;">
        <a href="%s" style="display:inline-block; padding:15px 30px; font-size:12px; font-weight:900; letter-spacing:0.14em; text-transform:uppercase; color:#050505; text-decoration:none; font-family:'Inter',Arial,Helvetica,sans-serif;">Complete Payment</a>
      </td>
    </tr></table>
    <p style="margin:0; font-size:12px; color:#555555; line-height:1.6;">Have questions? Just reply to this email — we read every one.</p>
`, projectLink)

	_, err := c.resend.Emails.Send(&resend.SendEmailRequest{
		From:    c.from,
		To:      []string{to},
		Subject: "Your site is complete — complete payment to launch — consultprompts.com",
		Html:    openEmail() + body + closeEmail(),
	})
	if err != nil {
		slog.Error("Failed to send payment request email", "to", to, "error", err)
	}
	return err
}

// SendPaymentReceiptEmail confirms payment to the client.
func (c *Client) SendPaymentReceiptEmail(to, businessName, packageName string, packagePrice, totalAmount int, wantsMaintenance bool, domainRenewalDate string) error {
	packageRow := ""
	if packageName != "" && packagePrice > 0 {
		packageRow = fmt.Sprintf(
			`<tr><td style="padding:8px 16px; border-bottom:1px solid rgba(255,255,255,0.06); color:#A1A1A1; font-size:13px;">%s</td><td style="padding:8px 16px; border-bottom:1px solid rgba(255,255,255,0.06); text-align:right; color:#ffffff; font-size:13px; font-weight:700;">$%d</td></tr>`,
			html.EscapeString(packageName), packagePrice,
		)
	}

	maintenanceRow := ""
	if wantsMaintenance {
		maintenanceRow = `<tr><td style="padding:8px 16px; border-bottom:1px solid rgba(255,255,255,0.06); color:#A1A1A1; font-size:13px;">Monthly Site Maintenance</td><td style="padding:8px 16px; border-bottom:1px solid rgba(255,255,255,0.06); text-align:right; color:#ffffff; font-size:13px; font-weight:700;">$29/mo</td></tr>`
	}

	body := fmt.Sprintf(`
    <table cellpadding="0" cellspacing="0" style="margin-bottom:24px;"><tr>
      <td width="56" height="56" style="width:56px; height:56px; background:rgba(0,240,255,0.1); border-radius:28px; text-align:center; vertical-align:middle;">
        <span style="font-size:22px; line-height:56px; display:block;">&#10003;</span>
      </td>
    </tr></table>
    <h2 style="margin:0 0 6px; font-family:'Space Grotesk',Georgia,serif; font-style:italic; font-size:26px; font-weight:700; letter-spacing:-0.02em; color:#ffffff;">Payment Received</h2>
    <p style="margin:0 0 28px; color:#A1A1A1; font-size:14px; font-weight:300; line-height:1.7;">Thank you for your payment for <strong style="color:#ffffff;">%s</strong>. Your site will be launched shortly.</p>

    <table width="100%%" cellpadding="0" cellspacing="0" style="margin-bottom:28px; border:1px solid rgba(255,255,255,0.08); border-radius:8px; overflow:hidden;">
      <tr style="background:rgba(255,255,255,0.04);">
        <td style="padding:10px 16px; font-size:10px; font-weight:700; letter-spacing:0.1em; text-transform:uppercase; color:#A1A1A1;" colspan="2">Order Summary</td>
      </tr>
      %s
      <tr>
        <td style="padding:8px 16px; border-bottom:1px solid rgba(255,255,255,0.06); color:#A1A1A1; font-size:13px;">Domain Registration</td>
        <td style="padding:8px 16px; border-bottom:1px solid rgba(255,255,255,0.06); text-align:right; color:#ffffff; font-size:13px; font-weight:700;">$20/yr</td>
      </tr>
      %s
      <tr style="background:rgba(185,140,255,0.06);">
        <td style="padding:12px 16px; color:#B98CFF; font-size:13px; font-weight:700; letter-spacing:0.04em;">Total Charged</td>
        <td style="padding:12px 16px; text-align:right; color:#B98CFF; font-size:18px; font-weight:900;">$%d</td>
      </tr>
      <tr style="background:rgba(255,255,255,0.02);">
        <td style="padding:10px 16px; font-size:11px; color:#555555; font-style:italic;" colspan="2">Domain renewal due: %s</td>
      </tr>
    </table>

    <p style="margin:0; font-size:12px; color:#555555; line-height:1.6;">Have questions? Just reply to this email — we read every one.</p>
`,
		html.EscapeString(businessName),
		packageRow,
		maintenanceRow,
		totalAmount,
		html.EscapeString(domainRenewalDate),
	)

	_, err := c.resend.Emails.Send(&resend.SendEmailRequest{
		From:    c.from,
		To:      []string{to},
		Subject: fmt.Sprintf("Payment confirmed — %s — consultprompts.com", businessName),
		Html:    openEmail() + body + closeEmail(),
	})
	if err != nil {
		slog.Error("Failed to send payment receipt email", "to", to, "error", err)
	}
	return err
}

// SendSiteLaunchedEmail notifies the client that their site is live.
func (c *Client) SendSiteLaunchedEmail(to, siteURL, businessName string) error {
	body := fmt.Sprintf(`
    <table cellpadding="0" cellspacing="0" style="margin-bottom:24px;"><tr>
      <td width="56" height="56" style="width:56px; height:56px; background:rgba(0,240,255,0.1); border-radius:28px; text-align:center; vertical-align:middle;">
        <span style="font-size:22px; line-height:56px; display:block;">&#127756;</span>
      </td>
    </tr></table>
    <h2 style="margin:0 0 10px; font-family:'Space Grotesk',Georgia,serif; font-style:italic; font-size:26px; font-weight:700; letter-spacing:-0.02em; color:#ffffff;">%s is Live!</h2>
    <p style="margin:0 0 30px; color:#A1A1A1; font-size:14px; font-weight:300; line-height:1.7;">Congratulations — your website is now live and published. Click the button below to visit it.</p>
    <table cellpadding="0" cellspacing="0" style="margin-bottom:28px;"><tr>
      <td style="background:#00F0FF; border-radius:8px;">
        <a href="%s" style="display:inline-block; padding:15px 30px; font-size:12px; font-weight:900; letter-spacing:0.14em; text-transform:uppercase; color:#050505; text-decoration:none; font-family:'Inter',Arial,Helvetica,sans-serif;">Visit My Site</a>
      </td>
    </tr></table>
    <p style="margin:0; font-size:12px; color:#555555; line-height:1.6;">Have questions or need changes? Reply to this email — we read every one.</p>
`,
		html.EscapeString(businessName),
		html.EscapeString(siteURL),
	)

	_, err := c.resend.Emails.Send(&resend.SendEmailRequest{
		From:    c.from,
		To:      []string{to},
		Subject: fmt.Sprintf("%s is live! — consultprompts.com", businessName),
		Html:    openEmail() + body + closeEmail(),
	})
	if err != nil {
		slog.Error("Failed to send site launched email", "to", to, "error", err)
	}
	return err
}

// SendLeadAccepted emails the customer when an admin accepts their project.
func (c *Client) SendLeadAccepted(lead model.Lead) error {
	frontendURL := os.Getenv("FRONTEND_URL")

	bodyText := fmt.Sprintf(
		`Great news, %s — we've accepted your project for <strong style="color:#ffffff;">%s</strong> and we're getting started. Track every milestone in real time from your project dashboard.`,
		html.EscapeString(lead.Name),
		html.EscapeString(lead.Business),
	)
	if lead.Package != nil {
		bodyText += fmt.Sprintf(`<br><br>Package: <span style="color:#00F0FF;">%s</span>`, html.EscapeString(*lead.Package))
	}

	body := fmt.Sprintf(`
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
`, bodyText, frontendURL)

	_, err := c.resend.Emails.Send(&resend.SendEmailRequest{
		From:    c.from,
		To:      []string{lead.Email},
		Subject: "Your project has been accepted — consultprompts.com",
		Html:    openEmail() + body + closeEmail(),
	})
	if err != nil {
		slog.Error("Failed to send lead accepted email", "lead_id", lead.ID, "error", err)
	}
	return err
}
