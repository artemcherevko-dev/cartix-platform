package nats

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const receiptTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<meta http-equiv="X-UA-Compatible" content="IE=edge">
	<title>Payment receipt</title>
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

					<!-- Receipt header -->
					<tr>
						<td bgcolor="#6366f1" style="background-color:#6366f1; background-image:linear-gradient(135deg,#6366f1 0%,#8b5cf6 55%,#a855f7 100%); border-radius:20px 20px 0 0; padding:44px 48px 40px; text-align:center;">
							<div style="font-size:46px; line-height:1; margin-bottom:16px;">&#129534;</div>
							<h1 style="margin:0 0 10px 0; font-size:28px; line-height:1.25; font-weight:800; color:#ffffff; letter-spacing:2px; text-transform:uppercase;">
								Payment receipt
							</h1>
							<p style="margin:0; font-size:15px; line-height:1.6; color:#e0e7ff;">
								Your payment was completed successfully.
							</p>
						</td>
					</tr>

					<!-- Body -->
					<tr>
						<td bgcolor="#ffffff" style="background-color:#ffffff; padding:40px 48px 32px;">

							<!-- Identifiers & date -->
							<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="font-size:14px; line-height:1.6;">
								{{ if .OrderID }}
								<tr>
									<td style="padding:6px 0; color:#6b7280;">Order</td>
									<td align="right" style="padding:6px 0; color:#111827; font-weight:700;">#{{ .OrderID }}</td>
								</tr>
								{{ end }}
								<tr>
									<td style="padding:6px 0; color:#6b7280;">Payment</td>
									<td align="right" style="padding:6px 0; color:#111827; font-weight:700;">#{{ .PaymentID }}</td>
								</tr>
								{{ if .StripePaymentID }}
								<tr>
									<td style="padding:6px 0; color:#6b7280;">Transaction</td>
									<td align="right" style="padding:6px 0; color:#6b7280; font-size:12.5px;">{{ .StripePaymentID }}</td>
								</tr>
								{{ end }}
								<tr>
									<td style="padding:6px 0; color:#6b7280;">Date</td>
									<td align="right" style="padding:6px 0; color:#111827; font-weight:600;">{{ .DateFormatted }}</td>
								</tr>
							</table>

							<!-- Divider -->
							<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="margin:22px 0;">
								<tr>
									<td style="border-top:1px solid #eceef5;"></td>
								</tr>
							</table>

							{{ if .HasItems }}
							<!-- Line items -->
							<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="font-size:14.5px; line-height:1.6;">
								{{ range .Lines }}
								<tr>
									<td style="padding:7px 0; color:#374151;">{{ .Name }}{{ if .QtyLabel }} <span style="color:#9ca3af;">{{ .QtyLabel }}</span>{{ end }}</td>
									<td align="right" style="padding:7px 0; color:#111827; font-weight:600; white-space:nowrap;">{{ .AmountFormatted }}</td>
								</tr>
								{{ end }}
								<tr>
									<td style="padding:10px 0 4px; color:#6b7280; font-size:13.5px;">Subtotal</td>
									<td align="right" style="padding:10px 0 4px; color:#374151; font-weight:600; white-space:nowrap;">{{ .SubtotalFormatted }}</td>
								</tr>
							</table>

							<!-- Divider -->
							<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="margin:18px 0 22px;">
								<tr>
									<td style="border-top:1px solid #eceef5;"></td>
								</tr>
							</table>
							{{ end }}

							<!-- Total -->
							<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" bgcolor="#f5f6ff" style="background-color:#f5f6ff; border-radius:12px;">
								<tr>
									<td style="padding:20px 24px; font-size:15px; font-weight:700; letter-spacing:1px; color:#312e81; text-transform:uppercase;">Total</td>
									<td align="right" style="padding:20px 24px;">
										<span style="font-size:26px; font-weight:800; color:#312e81; letter-spacing:-0.5px; white-space:nowrap;">{{ .TotalFormatted }}</span>
										<span style="font-size:13px; font-weight:600; color:#6b7280; margin-left:6px;">{{ .CurrencyCode }}</span>
									</td>
								</tr>
							</table>

							<p style="margin:20px 0 0; font-size:13.5px; line-height:1.6; color:#6b7280;">
								Payment method: <strong style="color:#374151; font-weight:600;">{{ .PaymentMethodLabel }}</strong>
							</p>

						</td>
					</tr>

					<!-- Success strip -->
					<tr>
						<td bgcolor="#ecfdf5" style="background-color:#ecfdf5; padding:18px 48px; text-align:center; border-top:1px solid #d1fae5;">
							<p style="margin:0; font-size:14px; font-weight:700; letter-spacing:1.5px; color:#059669; text-transform:uppercase;">
								&#10003;&#160;Paid &#8212; payment successful
							</p>
						</td>
					</tr>

					<!-- Thank-you strip -->
					<tr>
						<td bgcolor="#f5f6ff" style="background-color:#f5f6ff; padding:22px 48px; text-align:center; border-top:1px solid #eceef5;">
							<p style="margin:0; font-size:15px; line-height:1.6; color:#4338ca; font-weight:600;">
								Thank you for your purchase.
							</p>
							<p style="margin:8px 0 0; font-size:12px; line-height:1.6; color:#8a90a8;">
								This receipt confirms your payment in full. Keep it for your records.
							</p>
						</td>
					</tr>

					<!-- Footer -->
					<tr>
						<td bgcolor="#ffffff" style="background-color:#ffffff; border-radius:0 0 20px 20px; padding:28px 48px 36px; text-align:center; border-top:1px solid #eceef5;">
							<p style="margin:0; font-size:12px; line-height:1.6; color:#9ca3af;">
								&#169; cartix &#8212; sent because of a recent purchase at cartix.<br>
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

type ReceiptItem struct {
	Name     string
	Quantity int64
	Amount   int64 // line total in minor units, e.g. cents
}

// ReceiptItemView is a receipt line prepared for rendering.
type ReceiptItemView struct {
	Name            string
	QtyLabel        string
	AmountFormatted string
}

type ReceiptTemplateData struct {
	OrderID         string
	PaymentID       string
	StripePaymentID string
	Amount          int64 // minor units, e.g. cents
	Currency        string
	PaymentMethod   string
	PaidAt          time.Time

	Items []ReceiptItem

	Lines              []ReceiptItemView
	DateFormatted      string
	TotalFormatted     string
	SubtotalFormatted  string
	CurrencyCode       string
	PaymentMethodLabel string
	HasItems           bool
}

func renderReceipt(data ReceiptTemplateData) (string, error) {
	return renderTemplate("", receiptTemplate, "receipt", buildReceiptData(data))
}

func (m *Mailer) renderReceipt(data ReceiptTemplateData) (string, error) {
	return renderTemplate(m.templateDir, receiptTemplate, "receipt", buildReceiptData(data))
}

func buildReceiptData(data ReceiptTemplateData) ReceiptTemplateData {
	data.CurrencyCode = strings.ToUpper(data.Currency)
	if data.CurrencyCode == "" {
		data.CurrencyCode = "USD"
	}
	data.TotalFormatted = FormatAmount(data.Amount, data.Currency)

	subtotal := int64(0)
	items := make([]ReceiptItemView, 0, len(data.Items))
	for _, item := range data.Items {
		subtotal += item.Amount

		qtyLabel := ""
		if item.Quantity > 1 {
			qtyLabel = "× " + strconv.FormatInt(item.Quantity, 10)
		}

		items = append(items, ReceiptItemView{
			Name:            item.Name,
			QtyLabel:        qtyLabel,
			AmountFormatted: FormatAmount(item.Amount, data.Currency),
		})
	}
	data.Items = nil
	if len(items) > 0 {
		data.HasItems = true
		data.SubtotalFormatted = FormatAmount(subtotal, data.Currency)
		data.Lines = items
	}

	if data.PaidAt.IsZero() {
		data.PaidAt = time.Now().UTC()
	}
	data.DateFormatted = data.PaidAt.Format("2 January 2006")

	data.PaymentMethodLabel = paymentMethodLabel(data.PaymentMethod)

	return data
}

// SendReceipt renders and delivers a payment-receipt email.
func (m *Mailer) SendReceipt(from, to, subject string, data ReceiptTemplateData) error {
	body, err := m.renderReceipt(buildReceiptData(data))
	if err != nil {
		return err
	}

	return m.SendHTML(from, to, subject, body)
}

func paymentMethodLabel(method string) string {
	switch strings.ToLower(method) {
	case "", "stripe":
		return "Stripe"
	case "card":
		return "Card"
	default:
		return strings.ToUpper(method[:1]) + method[1:]
	}
}

var zeroDecimalCurrencies = map[string]bool{
	"JPY": true,
	"KRW": true,
	"VND": true,
	"CLP": true,
}

var currencySymbols = map[string]string{
	"USD": "$",
	"EUR": "€",
	"GBP": "£",
	"UAH": "₴",
	"JPY": "¥",
	"INR": "₹",
}

// FormatAmount renders an amount given in minor units, e.g.
// 3000000 USD -> "$30,000.00".
func FormatAmount(amount int64, currency string) string {
	code := strings.ToUpper(currency)

	negative := amount < 0
	if negative {
		amount = -amount
	}

	var value string
	switch {
	case zeroDecimalCurrencies[code]:
		value = insertThousands(strconv.FormatInt(amount, 10))
	default:
		whole := amount / 100
		cents := amount % 100
		value = insertThousands(strconv.FormatInt(whole, 10)) + "." + fmt.Sprintf("%02d", cents)
	}

	return sign(negative) + symbol(code) + value
}

func sign(negative bool) string {
	if negative {
		return "-"
	}
	return ""
}

func symbol(code string) string {
	if s, ok := currencySymbols[code]; ok {
		return s
	}
	return ""
}

func insertThousands(digits string) string {
	n := len(digits)
	if n <= 3 {
		return digits
	}

	var b strings.Builder
	for i, r := range digits {
		if i > 0 && (n-i)%3 == 0 {
			b.WriteRune(',')
		}
		b.WriteRune(r)
	}

	return b.String()
}
