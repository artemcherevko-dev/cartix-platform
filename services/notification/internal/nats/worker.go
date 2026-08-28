package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"notification/internal/orders"
	notificationstore "notification/internal/store"

	"github.com/nats-io/nats.go/jetstream"
)

type HandlerFunc func(msg jetstream.Msg) error

type Worker struct {
	consumer jetstream.Consumer
	handle   HandlerFunc
}

func NewWorker(consumer jetstream.Consumer, handle HandlerFunc) Worker {
	return Worker{
		consumer: consumer,
		handle:   handle,
	}
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

// WelcomeHandler sends the email-verification mail for user.registered events.
func WelcomeHandler(mailer *Mailer, from string) HandlerFunc {
	return func(msg jetstream.Msg) error {
		var event struct {
			FirstName   string `json:"first_name"`
			LastName    string `json:"last_name"`
			Email       string `json:"email"`
			VerifyToken string `json:"verify_token"`
		}

		if err := json.Unmarshal(msg.Data(), &event); err != nil {
			return fmt.Errorf("unmarshal user.created event: %w", err)
		}

		if err := mailer.SendMail(from, event.Email, "Confirm your email", UserTemplateData{
			FirstName:        event.FirstName,
			LastName:         event.LastName,
			VerificationLink: mailer.VerificationLink(event.VerifyToken),
		}); err != nil {
			return fmt.Errorf("send welcome mail: %w", err)
		}

		return nil
	}
}

type orderStatusUpdatedEvent struct {
	OrderID        string `json:"order_id"`
	UserID         string `json:"user_id"`
	Email          string `json:"email"`
	Status         string `json:"status"`
	PreviousStatus string `json:"previous_status"`
	TotalMinor     int64  `json:"total_minor"`
	Currency       string `json:"currency"`
	Items          []struct {
		Name        string `json:"name"`
		Quantity    int32  `json:"quantity"`
		AmountMinor int64  `json:"amount_minor"`
	} `json:"items"`
	UpdatedAt time.Time `json:"updated_at"`
}

// StatusUpdatedHandler informs the customer that their order status changed.
func StatusUpdatedHandler(mailer *Mailer, from string) HandlerFunc {
	return func(msg jetstream.Msg) error {
		var event orderStatusUpdatedEvent
		if err := json.Unmarshal(msg.Data(), &event); err != nil {
			return fmt.Errorf("unmarshal order.status_updated event: %w", err)
		}

		if event.OrderID == "" || event.Email == "" {
			log.Printf("[NOTIFICATION] order.status_updated without order id or email, skipping")
			return nil
		}

		data := OrderStatusTemplateData{
			OrderID:     event.OrderID,
			Status:      event.Status,
			StatusLabel: statusLabel(event.Status),
		}
		if event.TotalMinor > 0 {
			data.TotalFormatted = FormatAmount(event.TotalMinor, event.Currency)
		}
		for _, item := range event.Items {
			qty := ""
			if item.Quantity > 1 {
				qty = "× " + strconv.FormatInt(int64(item.Quantity), 10)
			}

			data.Lines = append(data.Lines, OrderStatusLineView{
				Name:            item.Name,
				QtyLabel:        qty,
				AmountFormatted: FormatAmount(item.AmountMinor, event.Currency),
			})
		}
		data.HasItems = len(data.Lines) > 0
		date := event.UpdatedAt
		if date.IsZero() {
			date = time.Now().UTC()
		}
		data.DateFormatted = date.UTC().Format("2 January 2006, 15:04 MST")

		if err := mailer.SendOrderStatus(from, event.Email, data); err != nil {
			return fmt.Errorf("send order status mail: %w", err)
		}

		log.Printf("[NOTIFICATION] status update for order %s (%s) sent to %s", event.OrderID, event.Status, event.Email)

		return nil
	}
}

func statusLabel(status string) string {
	switch status {
	case "paid":
		return "Paid"
	case "pending":
		return "Pending"
	case "done":
		return "Done"
	case "failed":
		return "Failed"
	case "cancelled":
		return "Cancelled"
	default:
		return strings.ToUpper(status[:1]) + status[1:]
	}
}

type paymentSucceededEvent struct {
	PaymentID       string    `json:"payment_id"`
	OrderID         string    `json:"order_id"`
	StripePaymentID string    `json:"stripe_payment_id"`
	Email           string    `json:"email"`
	Amount          int64     `json:"amount"`
	Currency        string    `json:"currency"`
	PaymentMethod   string    `json:"payment_method"`
	PaidAt          time.Time `json:"paid_at"`
}

type ReceiptOrders interface {
	GetOrderItems(ctx context.Context, orderID string) ([]orders.Item, error)
}

// ReceiptHandler builds and sends the payment receipt for payment.succeeded
// events. Idempotency is enforced per payment via the DedupeStore so repeated
// webhooks or NATS redeliveries never produce a second receipt. Line items are
// fetched from the order service when available.
func ReceiptHandler(mailer *Mailer, dedupe *notificationstore.DedupeStore, from string, ordersClient ReceiptOrders) HandlerFunc {
	return func(msg jetstream.Msg) error {
		var event paymentSucceededEvent
		if err := json.Unmarshal(msg.Data(), &event); err != nil {
			return fmt.Errorf("unmarshal payment.succeeded event: %w", err)
		}

		if event.PaymentID == "" || event.Email == "" {
			log.Printf("[NOTIFICATION] payment.succeeded without payment id or email, skipping")
			return nil
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		first, err := dedupe.MarkProcessed(ctx, event.PaymentID)
		if err != nil {
			return fmt.Errorf("mark receipt processed: %w", err)
		}
		if !first {
			log.Printf("[NOTIFICATION] receipt for payment %s already sent, skipping duplicate", event.PaymentID)
			return nil
		}

		subject := "Your cartix payment receipt"
		if event.OrderID != "" {
			subject = fmt.Sprintf("Your cartix receipt — Order #%s", event.OrderID)
		}

		data := ReceiptTemplateData{
			OrderID:         event.OrderID,
			PaymentID:       event.PaymentID,
			StripePaymentID: event.StripePaymentID,
			Amount:          event.Amount,
			Currency:        event.Currency,
			PaymentMethod:   event.PaymentMethod,
			PaidAt:          event.PaidAt,
		}
		data.Items = fetchReceiptItems(ctx, ordersClient, event.OrderID)

		err = mailer.SendReceipt(from, event.Email, subject, data)
		if err != nil {
			if releaseErr := dedupe.Release(ctx, event.PaymentID); releaseErr != nil {
				log.Printf("[NOTIFICATION] release receipt lock for payment %s: %v", event.PaymentID, releaseErr)
			}
			return fmt.Errorf("send receipt mail: %w", err)
		}

		log.Printf("[NOTIFICATION] receipt for payment %s sent to %s", event.PaymentID, event.Email)

		return nil
	}
}

// fetchReceiptItems enriches the receipt with line items; failures are logged
// and the receipt is sent without lines rather than lost.
func fetchReceiptItems(ctx context.Context, client ReceiptOrders, orderID string) []ReceiptItem {
	if client == nil || orderID == "" {
		return nil
	}

	items, err := client.GetOrderItems(ctx, orderID)
	if err != nil {
		log.Printf("[NOTIFICATION] fetch items for receipt of order %s: %v", orderID, err)
		return nil
	}

	lines := make([]ReceiptItem, 0, len(items))
	for _, item := range items {
		lines = append(lines, ReceiptItem{
			Name:     item.Name,
			Quantity: int64(item.Quantity),
			Amount:   item.UnitPriceMinor * int64(item.Quantity),
		})
	}

	return lines
}
