package webhook

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"time"

	paymentsnats "payments/internal/nats"

	"payments/internal/db"
)

const maxPayloadSize = 1 << 20 // 1 MiB, Stripe payloads are far smaller

type Store interface {
	UpsertSucceeded(ctx context.Context, in db.SuccessInput) (db.SuccessResult, error)
}

type PublisherFunc func(ctx context.Context, event paymentsnats.PaymentSucceeded) error

type Handler struct {
	secret    []byte
	tolerance time.Duration
	store     Store
	publish   PublisherFunc
}

func NewHandler(secret string, store Store, publish PublisherFunc) *Handler {
	return &Handler{
		secret:    []byte(secret),
		tolerance: DefaultSignatureTolerance,
		store:     store,
		publish:   publish,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	payload, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxPayloadSize))
	if err != nil {
		http.Error(w, "unable to read request body", http.StatusBadRequest)
		return
	}

	if err := VerifySignature(payload, r.Header.Get("Stripe-Signature"), string(h.secret), h.tolerance); err != nil {
		log.Printf("[PAYMENTS] Webhook signature verification failed: %v", err)
		http.Error(w, "invalid signature", http.StatusBadRequest)
		return
	}

	var event Event
	if err := json.Unmarshal(payload, &event); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	if err := h.process(r.Context(), event); err != nil {
		log.Printf("[PAYMENTS] Webhook %s processing failed: %v", event.ID, err)
		http.Error(w, "temporary error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// process verifies the event, marks the payment succeeded and publishes
// payment.succeeded. The Notification Service is reached only through NATS.
func (h *Handler) process(ctx context.Context, event Event) error {
	info, ok := ParseSucceededEvent(event)
	if !ok {
		log.Printf("[PAYMENTS] Webhook %s (%s): ignoring event", event.ID, event.Type)
		return nil
	}

	result, err := h.store.UpsertSucceeded(ctx, db.SuccessInput{
		StripeEventID:   event.ID,
		StripeSessionID: info.StripeSessionID,
		StripePaymentID: info.StripePaymentID,
		OrderID:         info.OrderID,
		Email:           info.Email,
		Amount:          info.Amount,
		Currency:        info.Currency,
		PaidAt:          info.PaidAt,
	})
	if err != nil {
		return err
	}

	if result.AlreadySucceeded {
		log.Printf("[PAYMENTS] Payment %s already succeeded, re-publishing is safe (deduplicated downstream)", result.PaymentID)
	}

	err = h.publish(ctx, paymentsnats.PaymentSucceeded{
		PaymentID:       result.PaymentID,
		OrderID:         result.OrderID,
		StripePaymentID: result.StripePaymentID,
		StripeEventID:   event.ID,
		Email:           result.Email,
		Amount:          result.Amount,
		Currency:        result.Currency,
		PaymentMethod:   paymentsnats.PaymentMethodStripe,
		PaidAt:          info.PaidAt,
	})
	if err != nil {
		return errors.New("publish payment.succeeded: " + err.Error())
	}

	log.Printf(
		"[PAYMENTS] Payment %s succeeded (order %s), published %s",
		result.PaymentID, result.OrderID, paymentsnats.PaymentSucceededSubject,
	)

	return nil
}
