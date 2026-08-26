package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	paymentsnats "payments/internal/nats"

	"payments/internal/db"
)

const (
	testSecret  = "whsec_test_secret"
	testEventID = "evt_1G88b2eZvKYlo2C0"
)

func checkoutSessionPayload() []byte {
	event := map[string]any{
		"id":      testEventID,
		"type":    EventCheckoutSessionCompleted,
		"created": 1756080000,
		"data": map[string]any{
			"object": map[string]any{
				"id":             "cs_test_1",
				"payment_intent": "pi_test_1",
				"amount_total":   3000000,
				"currency":       "usd",
				"customer_details": map[string]any{
					"email": "buyer@example.com",
				},
				"metadata": map[string]any{
					"order_id": "12345",
				},
			},
		},
	}

	payload, _ := json.Marshal(event)
	return payload
}

func signPayload(t *testing.T, timestamp time.Time, payload []byte, secret string) string {
	t.Helper()

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(fmt.Sprintf("%d", timestamp.Unix())))
	mac.Write([]byte("."))
	mac.Write(payload)

	return hex.EncodeToString(mac.Sum(nil))
}

type fakeStore struct {
	inputs []db.SuccessInput
	result db.SuccessResult
	err    error
}

func (f *fakeStore) UpsertSucceeded(_ context.Context, in db.SuccessInput) (db.SuccessResult, error) {
	f.inputs = append(f.inputs, in)
	return f.result, f.err
}

func newTestHandler(store *fakeStore, published *[]paymentsnats.PaymentSucceeded) *Handler {
	return NewHandler(
		testSecret,
		store,
		func(_ context.Context, event paymentsnats.PaymentSucceeded) error {
			*published = append(*published, event)
			return nil
		},
	)
}

func postWebhook(t *testing.T, handler *Handler, payload []byte, signatureHeader string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/v1/payments/webhook", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Stripe-Signature", signatureHeader)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	return recorder
}

func TestServeHTTPAcceptsValidSignatureAndPublishes(t *testing.T) {
	store := &fakeStore{result: db.SuccessResult{
		PaymentID:       "0198c0de-0000-7000-8000-000000000001",
		OrderID:         "12345",
		StripePaymentID: "pi_test_1",
		Email:           "buyer@example.com",
		Amount:          3000000,
		Currency:        "usd",
	}}
	var published []paymentsnats.PaymentSucceeded

	handler := newTestHandler(store, &published)
	payload := checkoutSessionPayload()
	now := time.Now()
	signature := fmt.Sprintf("t=%d,v1=%s", now.Unix(), signPayload(t, now, payload, testSecret))

	recorder := postWebhook(t, handler, payload, signature)

	if recorder.Code != http.StatusOK {
		t.Fatalf("got status %d, want 200: %s", recorder.Code, recorder.Body.String())
	}
	if len(store.inputs) != 1 {
		t.Fatalf("store called %d times, want 1", len(store.inputs))
	}
	if len(published) != 1 {
		t.Fatalf("published %d events, want 1", len(published))
	}

	in := store.inputs[0]
	if in.StripeSessionID != "cs_test_1" || in.StripePaymentID != "pi_test_1" {
		t.Errorf("stripe identifiers not extracted: %+v", in)
	}
	if in.OrderID != "12345" {
		t.Errorf("order_id not read from metadata, got %q", in.OrderID)
	}
	if in.Email != "buyer@example.com" {
		t.Errorf("email not extracted, got %q", in.Email)
	}
	if in.Amount != 3000000 || in.Currency != "usd" {
		t.Errorf("amount/currency wrong: %d %s", in.Amount, in.Currency)
	}
	if in.PaidAt.Unix() != 1756080000 {
		t.Errorf("paid_at not taken from event created, got %v", in.PaidAt)
	}
	if in.StripeEventID != testEventID {
		t.Errorf("event id not propagated, got %q", in.StripeEventID)
	}

	event := published[0]
	if event.PaymentID != store.result.PaymentID {
		t.Errorf("event payment_id = %q, want %q", event.PaymentID, store.result.PaymentID)
	}
	if event.Subject() != "payment.succeeded" {
		t.Errorf("unexpected subject %q", event.Subject())
	}
	if event.PaymentMethod != paymentsnats.PaymentMethodStripe {
		t.Errorf("payment method = %q", event.PaymentMethod)
	}
	if event.Amount != 3000000 || event.Currency != "usd" {
		t.Errorf("event amount/currency wrong: %d %s", event.Amount, event.Currency)
	}
}

func TestServeHTTPRejectsBadSignature(t *testing.T) {
	var published []paymentsnats.PaymentSucceeded
	handler := newTestHandler(&fakeStore{}, &published)
	payload := checkoutSessionPayload()
	now := time.Now()

	cases := map[string]string{
		"wrong signature": fmt.Sprintf("t=%d,v1=%s", now.Unix(), signPayload(t, now, payload, "other-secret")),
		"tampered body":   fmt.Sprintf("t=%d,v1=%s", now.Unix(), signPayload(t, now, []byte(`{"evil":true}`), testSecret)),
		"missing header":  "",
		"garbage header":  "not-a-signature",
	}

	for name, header := range cases {
		recorder := postWebhook(t, handler, payload, header)
		if recorder.Code != http.StatusBadRequest {
			t.Errorf("%s: got status %d, want 400", name, recorder.Code)
		}
	}

	if len(published) != 0 {
		t.Errorf("published %d events on rejected requests", len(published))
	}
}

func TestServeHTTPRejectsExpiredSignature(t *testing.T) {
	var published []paymentsnats.PaymentSucceeded
	handler := newTestHandler(&fakeStore{}, &published)

	payload := checkoutSessionPayload()
	old := time.Now().Add(-time.Hour)
	signature := fmt.Sprintf("t=%d,v1=%s", old.Unix(), signPayload(t, old, payload, testSecret))

	recorder := postWebhook(t, handler, payload, signature)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("got status %d, want 400 for replayed webhook", recorder.Code)
	}
}

func TestServeHTTPAcknowledgesIrrelevantEvents(t *testing.T) {
	store := &fakeStore{}
	var published []paymentsnats.PaymentSucceeded
	handler := newTestHandler(store, &published)

	payload := []byte(`{"id":"evt_2","type":"charge.refunded","created":1756080000,"data":{"object":{"id":"ch_1"}}}`)
	now := time.Now()
	signature := fmt.Sprintf("t=%d,v1=%s", now.Unix(), signPayload(t, now, payload, testSecret))

	recorder := postWebhook(t, handler, payload, signature)

	if recorder.Code != http.StatusOK {
		t.Fatalf("got status %d, want 200 for unrelated event", recorder.Code)
	}
	if len(store.inputs) != 0 || len(published) != 0 {
		t.Error("unrelated event must not touch store or publisher")
	}
}

func TestParseSucceededEventPaymentIntent(t *testing.T) {
	payload := []byte(`{"id":"pi_1","amount":9900,"currency":"eur","receipt_email":"p@example.com","metadata":{"order_id":"77"}}`)

	info, ok := ParseSucceededEvent(Event{
		ID:      "evt_3",
		Type:    EventPaymentIntentSucceeded,
		Data:    Data{Object: payload},
		Created: 1756080100,
	})
	if !ok {
		t.Fatal("payment_intent.succeeded must be recognized")
	}

	if info.StripePaymentID != "pi_1" || info.Amount != 9900 || info.Currency != "eur" ||
		info.Email != "p@example.com" || info.OrderID != "77" {
		t.Errorf("wrong info parsed: %+v", info)
	}
}
