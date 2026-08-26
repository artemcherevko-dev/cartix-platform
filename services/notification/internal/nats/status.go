package nats

const orderStatusTemplateFallback = `<!DOCTYPE html>
<html lang="en">
<head><meta charset="UTF-8"><title>Order status updated</title></head>
<body style="margin:0;padding:0;background-color:#eef0f8;font-family:'Segoe UI',Roboto,Helvetica,Arial,sans-serif;">
	<table role="presentation" width="100%" cellpadding="0" cellspacing="0" bgcolor="#eef0f8">
		<tr><td align="center" style="padding:40px 16px;">
			<table role="presentation" width="600" cellpadding="0" cellspacing="0" style="width:600px;max-width:100%;">
				<tr><td align="center" style="padding:0 0 24px 0;">
					<span style="font-size:26px;font-weight:800;color:#312e81;">cartix</span>
				</td></tr>
				<tr><td bgcolor="#6366f1" style="background-color:#6366f1;border-radius:20px 20px 0 0;padding:44px 48px;text-align:center;">
					<div style="font-size:46px;line-height:1;margin-bottom:16px;">&#128230;</div>
					<h1 style="margin:0 0 10px;font-size:28px;color:#ffffff;letter-spacing:1px;text-transform:uppercase;">Order status updated</h1>
					<p style="margin:0;color:#e0e7ff;font-size:15px;">The status of your order has changed.</p>
				</td></tr>
				<tr><td bgcolor="#ffffff" style="padding:40px 48px;text-align:center;">
					<p style="margin:0 0 18px;color:#6b7280;font-size:14px;">Order <strong style="color:#111827;">#{{ .OrderID }}</strong></p>
					{{ if eq .Status "done" }}
					<span style="display:inline-block;padding:14px 40px;background-color:#dcfce7;border-radius:12px;font-size:22px;font-weight:800;color:#166534;text-transform:uppercase;letter-spacing:1px;">&#10004; {{ .StatusLabel }}</span>
					{{ else if eq .Status "paid" }}
					<span style="display:inline-block;padding:14px 40px;background-color:#dbeafe;border-radius:12px;font-size:22px;font-weight:800;color:#1e40af;text-transform:uppercase;letter-spacing:1px;">&#10004; {{ .StatusLabel }}</span>
					{{ else if eq .Status "pending" }}
					<span style="display:inline-block;padding:14px 40px;background-color:#fef3c7;border-radius:12px;font-size:22px;font-weight:800;color:#92400e;text-transform:uppercase;letter-spacing:1px;">&#9203; {{ .StatusLabel }}</span>
					{{ else if eq .Status "failed" }}
					<span style="display:inline-block;padding:14px 40px;background-color:#fee2e2;border-radius:12px;font-size:22px;font-weight:800;color:#991b1b;text-transform:uppercase;letter-spacing:1px;">&#10006; {{ .StatusLabel }}</span>
					{{ else if eq .Status "cancelled" }}
					<span style="display:inline-block;padding:14px 40px;background-color:#f3f4f6;border-radius:12px;font-size:22px;font-weight:800;color:#374151;text-transform:uppercase;letter-spacing:1px;">&#240; {{ .StatusLabel }}</span>
					{{ else }}
					<span style="display:inline-block;padding:14px 40px;background-color:#f5f6ff;border-radius:12px;font-size:22px;font-weight:800;color:#312e81;text-transform:uppercase;letter-spacing:1px;">{{ .StatusLabel }}</span>
					{{ end }}
					{{ if .HasItems }}
					<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="margin:28px 0 0;font-size:14.5px;line-height:1.6;text-align:left;">
						{{ range .Lines }}
						<tr>
							<td style="padding:7px 0;color:#374151;border-bottom:1px solid #eceef5;">{{ .Name }}{{ if .QtyLabel }} <span style="color:#9ca3af;">{{ .QtyLabel }}</span>{{ end }}</td>
							<td align="right" style="padding:7px 0;color:#111827;font-weight:600;white-space:nowrap;border-bottom:1px solid #eceef5;">{{ .AmountFormatted }}</td>
						</tr>
						{{ end }}
					</table>
					{{ end }}
					{{ if .TotalFormatted }}
					<p style="margin:22px 0 0;color:#374151;font-size:15px;">Order total: <strong>{{ .TotalFormatted }}</strong></p>
					{{ end }}
					<p style="margin:24px 0 0;color:#9ca3af;font-size:13px;">Updated {{ .DateFormatted }}</p>
				</td></tr>
				<tr><td bgcolor="#ffffff" style="border-radius:0 0 20px 20px;padding:28px 48px;border-top:1px solid #eceef5;text-align:center;">
					<p style="margin:0;color:#9ca3af;font-size:12px;">&#169; cartix &#8212; automated message, please do not reply.</p>
				</td></tr>
			</table>
		</td></tr>
	</table>
</body>
</html>
`

// OrderStatusLineView is one rendered line of the order contents.
type OrderStatusLineView struct {
	Name            string
	QtyLabel        string
	AmountFormatted string
}

type OrderStatusTemplateData struct {
	OrderID        string
	Status         string
	StatusLabel    string
	TotalFormatted string
	DateFormatted  string

	Lines    []OrderStatusLineView
	HasItems bool
}

func (m *Mailer) SendOrderStatus(from, to string, data OrderStatusTemplateData) error {
	body, err := renderTemplate(m.templateDir, orderStatusTemplateFallback, "order_status", data)
	if err != nil {
		return err
	}

	subject := "Your cartix order update — " + data.StatusLabel

	return m.SendHTML(from, to, subject, body)
}
