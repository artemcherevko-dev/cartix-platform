package nats

import (
	"bytes"
	"html/template"
	"os"
	"path/filepath"

	"github.com/wneessen/go-mail"
)

// Mailer renders and delivers emails. Templates are read from templateDir on
// every send, so editing the HTML files takes effect without a restart. When
// a file is missing the built-in fallback is used.
type Mailer struct {
	client      *mail.Client
	appURL      string
	templateDir string
}

func NewMailer(client *mail.Client, appURL, templateDir string) *Mailer {
	return &Mailer{
		client:      client,
		appURL:      appURL,
		templateDir: templateDir,
	}
}

// loadTemplate returns the file from templateDir, falling back to the
// embedded default when the directory is unset or the file is absent.
func (m *Mailer) loadTemplate(file, fallback string) string {
	if m.templateDir == "" {
		return fallback
	}

	raw, err := os.ReadFile(filepath.Join(m.templateDir, file))
	if err != nil || len(bytes.TrimSpace(raw)) == 0 {
		return fallback
	}

	return string(raw)
}

func (m *Mailer) VerificationLink(token string) string {
	return m.appURL + "/v1/auth/verify?token=" + token
}

const welcomeTemplate = `<!DOCTYPE html>
<html lang="en" xmlns:v="urn:schemas-microsoft-com:vml">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<meta http-equiv="X-UA-Compatible" content="IE=edge">
	<title>Confirm your email</title>
</head>
<body style="margin:0; padding:0; background-color:#eef0f8; -webkit-text-size-adjust:100%; -ms-text-size-adjust:100%;">
	<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" bgcolor="#eef0f8" style="background-color:#eef0f8;">
		<tr>
			<td align="center" style="padding:40px 16px;">

				<table role="presentation" width="600" cellpadding="0" cellspacing="0" border="0" style="width:600px; max-width:100%; font-family:'Segoe UI', Roboto, Helvetica, Arial, sans-serif;">

					<!-- Logo -->
					<tr>
						<td align="center" style="padding:0 0 24px 0;">
							<span style="font-size:26px; font-weight:800; letter-spacing:-0.5px; color:#312e81;">cartix&#9992;&#65039;</span>
						</td>
					</tr>

					<!-- Hero -->
					<tr>
						<td bgcolor="#6366f1" background="" style="background-color:#6366f1; background-image:linear-gradient(135deg,#6366f1 0%,#8b5cf6 55%,#a855f7 100%); border-radius:20px 20px 0 0; padding:48px 48px 44px; text-align:center;">
							<div style="font-size:52px; line-height:1; margin-bottom:18px;">&#128274;</div>
							<h1 style="margin:0 0 14px 0; font-size:30px; line-height:1.25; font-weight:800; color:#ffffff; letter-spacing:-0.5px;">
								Confirm your email
							</h1>
							<p style="margin:0; font-size:17px; line-height:1.6; color:#e0e7ff;">
								Hi {{ .FirstName }} {{ .LastName }}, welcome to cartix!<br>
								You're one click away from activating your account.
							</p>
						</td>
					</tr>

					<!-- Body -->
					<tr>
						<td bgcolor="#ffffff" style="background-color:#ffffff; padding:44px 48px; text-align:center;">

							<p style="margin:0 0 28px 0; font-size:15px; line-height:1.7; color:#4b5563;">
								Tap the button below to verify this email address.<br>
								It's quick &#8212; and it keeps your account secure.
							</p>

							<!-- CTA button -->
							<table role="presentation" cellpadding="0" cellspacing="0" border="0" align="center" style="margin:0 auto;">
								<tr>
									<td align="center" bgcolor="#6366f1" style="border-radius:12px; box-shadow:0 6px 18px rgba(99,102,241,0.35);">
										<a href="{{ .VerificationLink }}"
										   target="_blank"
										   style="display:inline-block; padding:16px 48px; font-family:'Segoe UI', Roboto, Helvetica, Arial, sans-serif; font-size:16px; font-weight:700; color:#ffffff; text-decoration:none; border-radius:12px; background-color:#6366f1; background-image:linear-gradient(135deg,#6366f1,#8b5cf6); mso-padding-alt:0;">
											<!--[if mso]><i style="letter-spacing:24px;mso-font-width:-100%;">&nbsp;</i><![endif]-->
											Verify my email &#8594;
											<!--[if mso]><i style="letter-spacing:24px;mso-font-width:-100%;">&nbsp;</i><![endif]-->
										</a>
									</td>
								</tr>
							</table>

							<p style="margin:28px 0 0 0; font-size:13px; line-height:1.6; color:#9ca3af;">
								This link is valid for <strong style="color:#6b7280;">15 minutes</strong> and can be used only once.
							</p>

							<!-- Divider -->
							<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="margin:32px 0;">
								<tr>
									<td style="border-top:1px solid #eceef5;"></td>
								</tr>
							</table>

							<p style="margin:0 0 8px 0; font-size:13px; line-height:1.7; color:#6b7280;">
								Button not working? Paste this link into your browser:
							</p>
							<p style="margin:0; word-break:break-all;">
								<a href="{{ .VerificationLink }}" style="color:#6366f1; text-decoration:underline; font-size:12px; line-height:1.6;">{{ .VerificationLink }}</a>
							</p>

						</td>
					</tr>

					<!-- Tip strip -->
					<tr>
						<td bgcolor="#f5f6ff" style="background-color:#f5f6ff; padding:20px 48px; text-align:center; border-top:1px solid #eceef5;">
							<p style="margin:0; font-size:12.5px; line-height:1.6; color:#8a90a8;">
								&#128161; Didn't create a cartix account? You can safely ignore this email.
							</p>
						</td>
					</tr>

					<!-- Footer -->
					<tr>
						<td bgcolor="#ffffff" style="background-color:#ffffff; border-radius:0 0 20px 20px; padding:28px 48px 36px; text-align:center; border-top:1px solid #eceef5;">
							<p style="margin:0; font-size:12px; line-height:1.6; color:#9ca3af;">
								&#169; cartix &#8212; sent because you signed up at cartix.<br>
								This is an automated message, please do not reply.
							</p>
						</td>
					</tr>

				</table>

			</td>
		</tr>
	</table>
</body>
</html>
`

type UserTemplateData struct {
	FirstName        string
	LastName         string
	VerificationLink string
}

func (m *Mailer) renderWelcome(data UserTemplateData) (string, error) {
	return renderTemplate(m.templateDir, welcomeTemplate, "welcome", data)
}

func renderTemplate(dir, fallback, name string, data any) (string, error) {
	source := fallback
	if dir != "" {
		if raw, err := os.ReadFile(filepath.Join(dir, name+".html")); err == nil && len(bytes.TrimSpace(raw)) > 0 {
			source = string(raw)
		}
	}

	tmpl, err := template.New(name).Parse(source)
	if err != nil {
		return "", err
	}

	var body bytes.Buffer

	if err := tmpl.Execute(&body, data); err != nil {
		return "", err
	}

	return body.String(), nil
}

func (m *Mailer) SendMail(from, to, subject string, data UserTemplateData) error {
	body, err := m.renderWelcome(data)
	if err != nil {
		return err
	}

	return m.SendHTML(from, to, subject, body)
}

// SendHTML delivers a rendered HTML email. Template rendering stays with each
// feature (welcome, receipt, ...), delivery and SMTP handling stay here.
func (m *Mailer) SendHTML(from, to, subject, body string) error {
	msg := mail.NewMsg()

	if err := msg.From(from); err != nil {
		return err
	}

	if err := msg.To(to); err != nil {
		return err
	}

	msg.Subject(subject)
	msg.SetBodyString("text/html", body)

	return m.client.DialAndSend(msg)
}
