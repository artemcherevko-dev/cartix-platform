package nats

import (
	"context"
	"encoding/json"
	natsclient "nats"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

const (
	OrdersStreamName          = "ORDERS"
	OrderStatusUpdatedSubject = "order.status_updated"
)

// InitStreams declares the ORDERS stream. Idempotent so startup order does
// not matter (the notification service declares the same stream).
func InitStreams(ctx context.Context, client *natsclient.Client) error {
	_, err := client.CreateOrUpdateStream(
		ctx,
		jetstream.StreamConfig{
			Name:      OrdersStreamName,
			Subjects:  []string{"order.>"},
			Retention: jetstream.LimitsPolicy,
		},
	)

	return err
}

// OrderStatusUpdated is published whenever an admin changes an order status.
type OrderStatusUpdated struct {
	OrderID        string            `json:"order_id"`
	UserID         string            `json:"user_id"`
	Email          string            `json:"email"`
	Status         string            `json:"status"`
	PreviousStatus string            `json:"previous_status,omitempty"`
	TotalMinor     int64             `json:"total_minor,omitempty"`
	Currency       string            `json:"currency,omitempty"`
	Items          []OrderStatusItem `json:"items,omitempty"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

// OrderStatusItem is one order line inside the notification event.
type OrderStatusItem struct {
	Name        string `json:"name"`
	Quantity    int32  `json:"quantity"`
	AmountMinor int64  `json:"amount_minor"`
}

func PublishOrderStatusUpdated(
	ctx context.Context,
	client *natsclient.Client,
	event OrderStatusUpdated,
) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return client.Publish(ctx, OrderStatusUpdatedSubject, data)
}
