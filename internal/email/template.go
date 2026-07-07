package email

import (
	"fmt"
	"os"
)

// openEmail returns the full HTML opening for every transactional email:
// DOCTYPE, head, dark-mode styles, outer wrapper, card open, brand header row,
// and the opening of the body cell. Call closeEmail() to close it.
func openEmail() string {
	logoURL := os.Getenv("LOGO_URL")

	var logoHTML string
	if logoURL != "" {
		logoHTML = fmt.Sprintf(
			`<table cellpadding="0" cellspacing="0"><tr>`+
				`<td style="vertical-align:middle; padding-right:12px;">`+
				`<img src="%s" width="32" height="32" alt="" style="display:block; border:0;" />`+
				`</td>`+
				`<td style="vertical-align:middle;">`+
				`<span style="font-family:'Space Grotesk',Georgia,serif; font-size:17px; font-weight:700; letter-spacing:0.08em; text-transform:uppercase; color:#00F0FF;">Consult Prompts</span>`+
				`</td></tr></table>`,
			logoURL,
		)
	} else {
		logoHTML = `<span style="font-family:'Space Grotesk',Georgia,serif; font-size:17px; font-weight:700; letter-spacing:0.08em; text-transform:uppercase; color:#00F0FF;">Consult Prompts</span>`
	}

	return `<!DOCTYPE html>
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
      body            { background:#050505 !important; }
      .email-wrapper  { background:#050505 !important; }
      .email-card     { background:#0f0f0f !important; }
      .email-footer   { background:#0f0f0f !important; }
    }
  </style>
</head>
<body style="margin:0; padding:0; background:#050505; font-family:'Inter',Arial,Helvetica,sans-serif; color:#ffffff;">
  <table class="email-wrapper" width="100%" cellpadding="0" cellspacing="0" style="background:#050505; padding:48px 24px 80px;">
    <tr><td align="center">
      <table class="email-card" width="560" cellpadding="0" cellspacing="0" style="max-width:560px; width:100%; background:#0f0f0f; border:1px solid rgba(255,255,255,0.12); border-radius:14px; overflow:hidden;">
        <!-- brand header -->
        <tr><td style="background:linear-gradient(135deg,#00F0FF22,#7000FF22); padding:28px 40px; border-bottom:1px solid rgba(255,255,255,0.08);">` +
		logoHTML +
		`</td></tr>
        <!-- body -->
        <tr><td style="padding:44px 40px 40px;">`
}

// closeEmail closes the body cell, appends the brand footer row, and closes all
// outer HTML. Use it after writing the body content returned by openEmail().
func closeEmail() string {
	return `</td></tr>
        <!-- footer -->
        <tr><td class="email-footer" style="padding:22px 40px; border-top:1px solid rgba(255,255,255,0.08); text-align:center; background:#0f0f0f;">
          <p style="margin:0; font-size:10px; color:#555555; letter-spacing:0.1em; text-transform:uppercase;">consultprompts.com</p>
        </td></tr>
      </table>
    </td></tr>
  </table>
</body>
</html>`
}
