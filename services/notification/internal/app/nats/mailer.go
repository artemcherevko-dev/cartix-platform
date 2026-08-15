package nats

import (
	"bytes"
	"html/template"

	"github.com/wneessen/go-mail"
)

type Mailer struct {
	client *mail.Client
}

func NewMailer(client *mail.Client) *Mailer {
	return &Mailer{
		client: client,
	}
}

const welcomeTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<title>Welcome</title>
</head>
<body>
	<h1>Welcome, {{ .FirstName }} {{ .LastName }}!</h1>

	<p>Thank you for registering in our application.</p>

	<p>We're glad to have you with us.</p>
</body>
</html>
`

type UserTemplateData struct {
	FirstName string
	LastName  string
}

func (m *Mailer) SendMail(from, to, subject string, data UserTemplateData) error {
	tmpl, err := template.New("welcome").Parse(welcomeTemplate)
	if err != nil {
		return err
	}

	var body bytes.Buffer

	if err := tmpl.Execute(&body, data); err != nil {
		return err
	}

	msg := mail.NewMsg()

	if err := msg.From(from); err != nil {
		return err
	}

	if err := msg.To(to); err != nil {
		return err
	}

	msg.Subject(subject)
	msg.SetBodyString("text/html", body.String())

	return m.client.DialAndSend(msg)
}
