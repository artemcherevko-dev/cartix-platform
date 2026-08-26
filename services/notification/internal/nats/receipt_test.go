package nats

import (
	"strings"
	"testing"
	"time"
)

func TestReceiptTemplateRenders(t *testing.T) {
	paidAt := time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC)

	out, err := renderReceipt(ReceiptTemplateData{
		OrderID:         "12345",
		PaymentID:       "67890",
		StripePaymentID: "pi_3Ks7",
		Amount:          30000000,
		Currency:        "usd",
		PaymentMethod:   "stripe",
		PaidAt:          paidAt,
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	for _, want := range []string{
		"#12345", "#67890", "pi_3Ks7",
		"25 August 2026",
		"$300,000.00", "USD",
		"Total", "Stripe",
		"payment successful", "Thank you for your purchase.",
		"Payment receipt",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("template output missing %q", want)
		}
	}

	if strings.Contains(out, "{{") {
		t.Error("template contains unrendered placeholders")
	}
}

func TestReceiptTemplateWithItems(t *testing.T) {
	out, err := renderReceipt(ReceiptTemplateData{
		OrderID:   "42",
		PaymentID: "pmt_1",
		Currency:  "EUR",
		Amount:    30000,
		Items: []ReceiptItem{
			{Name: "Flight ticket", Quantity: 2, Amount: 20000},
			{Name: "Baggage", Quantity: 1, Amount: 10000},
		},
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	for _, want := range []string{"€300.00", "€200.00", "€100.00", "× 2", "Subtotal", "Flight ticket"} {
		if !strings.Contains(out, want) {
			t.Errorf("template output missing %q", want)
		}
	}
}

func TestReceiptTemplateWithoutItems(t *testing.T) {
	out, err := renderReceipt(ReceiptTemplateData{
		PaymentID: "pmt_1",
		Amount:    500,
		Currency:  "USD",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	if strings.Contains(out, "Subtotal") {
		t.Error("subtotal must be omitted when there are no line items")
	}
	if !strings.Contains(out, "$5.00") {
		t.Error("template output missing total $5.00")
	}
}

func TestFormatAmount(t *testing.T) {
	cases := []struct {
		amount   int64
		currency string
		want     string
	}{
		{30000000, "usd", "$300,000.00"},
		{999, "eur", "€9.99"},
		{5, "usd", "$0.05"},
		{0, "gbp", "£0.00"},
		{-150025, "USD", "-$1,500.25"},
		{1000, "jpy", "¥1,000"},
		{2500000000, "XXX", "25,000,000.00"},
	}

	for _, tc := range cases {
		if got := FormatAmount(tc.amount, tc.currency); got != tc.want {
			t.Errorf("FormatAmount(%d, %q) = %q, want %q", tc.amount, tc.currency, got, tc.want)
		}
	}
}
