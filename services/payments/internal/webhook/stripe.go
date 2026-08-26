package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const DefaultSignatureTolerance = 5 * time.Minute

var (
	ErrInvalidSignature = errors.New("invalid stripe webhook signature")
	ErrExpiredSignature = errors.New("stripe webhook signature timestamp outside tolerance")
)

// VerifySignature checks the Stripe-Signature header (scheme described at
// https://docs.stripe.com/webhooks#verify-manually) against the raw payload.
func VerifySignature(payload []byte, header, secret string, tolerance time.Duration) error {
	timestamp, signatures, err := parseSignatureHeader(header)
	if err != nil {
		return err
	}

	if tolerance > 0 && time.Since(time.Unix(timestamp, 0)) > tolerance {
		return ErrExpiredSignature
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strconv.FormatInt(timestamp, 10)))
	mac.Write([]byte("."))
	mac.Write(payload)
	expected := mac.Sum(nil)

	for _, candidate := range signatures {
		got, err := hex.DecodeString(candidate)
		if err != nil {
			continue
		}

		if hmac.Equal(expected, got) {
			return nil
		}
	}

	return ErrInvalidSignature
}

func parseSignatureHeader(header string) (int64, []string, error) {
	var (
		timestamp  int64
		signatures []string
		hasTime    bool
	)

	for _, part := range strings.Split(header, ",") {
		k, v, found := strings.Cut(strings.TrimSpace(part), "=")
		if !found {
			continue
		}

		switch k {
		case "t":
			parsed, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				return 0, nil, fmt.Errorf("parse signature timestamp: %w", err)
			}
			timestamp = parsed
			hasTime = true
		case "v1":
			signatures = append(signatures, v)
		}
	}

	if !hasTime || len(signatures) == 0 {
		return 0, nil, ErrInvalidSignature
	}

	return timestamp, signatures, nil
}

// Event is a minimal Stripe webhook envelope; only the fields needed to
// confirm a payment are decoded.
type Event struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Data    Data   `json:"data"`
	Created int64  `json:"created"`
}

type Data struct {
	Object json.RawMessage `json:"object"`
}

const (
	EventCheckoutSessionCompleted = "checkout.session.completed"
	EventPaymentIntentSucceeded   = "payment_intent.succeeded"
)

type CheckoutSession struct {
	ID            string            `json:"id"`
	PaymentIntent string            `json:"payment_intent"`
	AmountTotal   int64             `json:"amount_total"`
	Currency      string            `json:"currency"`
	CustomerEmail string            `json:"customer_email"`
	Customer      *CustomerDetails  `json:"customer_details"`
	Metadata      map[string]string `json:"metadata"`
	Created       int64             `json:"created"`
}

type CustomerDetails struct {
	Email string `json:"email"`
}

type PaymentIntent struct {
	ID           string            `json:"id"`
	Amount       int64             `json:"amount"`
	Currency     string            `json:"currency"`
	ReceiptEmail string            `json:"receipt_email"`
	Metadata     map[string]string `json:"metadata"`
	Created      int64             `json:"created"`
}

// PaymentInfo is the normalized successful-payment data extracted from an event.
type PaymentInfo struct {
	StripeSessionID string
	StripePaymentID string
	OrderID         string
	Email           string
	Amount          int64
	Currency        string
	PaidAt          time.Time
}

// ParseSucceededEvent extracts payment information from the supported
// success events. It returns ok=false for events that are not about a
// succeeded payment and should simply be acknowledged.
func ParseSucceededEvent(event Event) (info PaymentInfo, ok bool) {
	switch event.Type {
	case EventCheckoutSessionCompleted:
		var session CheckoutSession
		if err := json.Unmarshal(event.Data.Object, &session); err != nil {
			return PaymentInfo{}, false
		}

		email := session.CustomerEmail
		if session.Customer != nil && session.Customer.Email != "" {
			email = session.Customer.Email
		}

		return PaymentInfo{
			StripeSessionID: session.ID,
			StripePaymentID: session.PaymentIntent,
			OrderID:         metadataValue(session.Metadata, "order_id"),
			Email:           email,
			Amount:          session.AmountTotal,
			Currency:        session.Currency,
			PaidAt:          eventTime(event),
		}, true
	case EventPaymentIntentSucceeded:
		var intent PaymentIntent
		if err := json.Unmarshal(event.Data.Object, &intent); err != nil {
			return PaymentInfo{}, false
		}

		return PaymentInfo{
			StripePaymentID: intent.ID,
			OrderID:         metadataValue(intent.Metadata, "order_id"),
			Email:           intent.ReceiptEmail,
			Amount:          intent.Amount,
			Currency:        intent.Currency,
			PaidAt:          eventTime(event),
		}, true
	default:
		return PaymentInfo{}, false
	}
}

func metadataValue(metadata map[string]string, key string) string {
	if metadata == nil {
		return ""
	}
	return metadata[key]
}

func eventTime(event Event) time.Time {
	if event.Created > 0 {
		return time.Unix(event.Created, 0).UTC()
	}
	return time.Now().UTC()
}
