package nats

import (
	"context"
	"encoding/json"
	natsclient "nats"
	"time"
)

const (
	PaymentSucceededSubject = "payment.succeeded"

	PaymentMethodStripe = "stripe"
)

type PaymentSucceeded struct {
	PaymentID       string    `json:"payment_id"`
	OrderID         string    `json:"order_id,omitempty"`
	StripePaymentID string    `json:"stripe_payment_id,omitempty"`
	StripeEventID   string    `json:"stripe_event_id,omitempty"`
	Email           string    `json:"email"`
	Amount          int64     `json:"amount"`
	Currency        string    `json:"currency"`
	PaymentMethod   string    `json:"payment_method"`
	PaidAt          time.Time `json:"paid_at"`
}

// Subject is the JetStream subject this event is published to.
func (PaymentSucceeded) Subject() string {
	return PaymentSucceededSubject
}

func PublishPaymentSucceeded(
	ctx context.Context,
	client *natsclient.Client,
	event PaymentSucceeded,
) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return client.Publish(
		ctx,
		PaymentSucceededSubject,
		data,
	)
}
