package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

// OrderPayer is implemented by the order service: consuming
// payment.succeeded transitions the matching order to paid.
type OrderPayer interface {
	MarkPaid(ctx context.Context, orderID string, paidAt time.Time) (bool, error)
}

type Worker struct {
	consumer jetstream.Consumer
	payer    OrderPayer
}

func NewWorker(consumer jetstream.Consumer, payer OrderPayer) Worker {
	return Worker{
		consumer: consumer,
		payer:    payer,
	}
}

func (w *Worker) handle(msg jetstream.Msg) error {
	var event struct {
		PaymentID string    `json:"payment_id"`
		OrderID   string    `json:"order_id"`
		PaidAt    time.Time `json:"paid_at"`
	}

	if err := json.Unmarshal(msg.Data(), &event); err != nil {
		return fmt.Errorf("unmarshal payment.succeeded event: %w", err)
	}

	if event.OrderID == "" {
		log.Printf("[ORDER] payment.succeeded without order_id (payment %s), skipping", event.PaymentID)
		return nil
	}

	paidAt := event.PaidAt
	if paidAt.IsZero() {
		paidAt = time.Now().UTC()
	}

	transitioned, err := w.payer.MarkPaid(context.Background(), event.OrderID, paidAt)
	if err != nil {
		return fmt.Errorf("mark order %s paid: %w", event.OrderID, err)
	}
	if !transitioned {
		log.Printf("[ORDER] order %s already paid or unknown, ignoring duplicate event", event.OrderID)
		return nil
	}

	log.Printf("[ORDER] order %s is now PAID (payment %s)", event.OrderID, event.PaymentID)

	return nil
}

func (w *Worker) Run(ctx context.Context) error {
	for {
		msgs, err := w.consumer.Fetch(10)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}

			log.Printf("fetch messages: %v", err)
			continue
		}

		for msg := range msgs.Messages() {
			if err := w.handle(msg); err != nil {
				log.Printf("handle message: %v", err)
				continue
			}

			if err := msg.Ack(); err != nil {
				log.Printf("ack message: %v", err)
			}
		}
	}
}
