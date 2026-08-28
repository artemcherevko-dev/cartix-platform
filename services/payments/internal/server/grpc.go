package server

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	paymentspb "proto/payments"

	"payments/internal/checkout"
	"payments/internal/db"

	"github.com/google/uuid"
)

type Handler struct {
	paymentspb.UnimplementedPaymentsServer
	repo    *db.Repo
	creator *checkout.Creator
}

func NewHandler(repo *db.Repo, creator *checkout.Creator) *Handler {
	return &Handler{
		repo:    repo,
		creator: creator,
	}
}

func (h *Handler) CreatePayment(ctx context.Context, req *paymentspb.CreatePaymentReq) (*paymentspb.CreatePaymentRes, error) {
	if req.OrderId == "" || req.AmountMinor <= 0 || req.Currency == "" {
		return nil, errors.New("order_id, amount_minor and currency are required")
	}

	payment := db.Payment{
		ID:       uuid.Must(uuid.NewV7()),
		OrderID:  req.OrderId,
		Status:   db.PaymentStatusPending,
		Provider: db.ProviderStripe,
		Email:    req.CustomerEmail,
		Amount:   req.AmountMinor,
		Currency: strings.ToLower(req.Currency),
	}

	session, err := h.creator.CreateCheckoutSession(ctx, checkout.CreateInput{
		OrderID:       payment.OrderID,
		PaymentID:     payment.ID.String(),
		AmountMinor:   payment.Amount,
		Currency:      payment.Currency,
		CustomerEmail: payment.Email,
		Description:   req.Description,
	})
	if err != nil {
		return nil, err
	}

	payment.StripeSessionID = &session.ID
	payment.CheckoutURL = session.URL

	if err := h.repo.CreatePending(ctx, payment); err != nil {
		return nil, err
	}

	log.Printf("[PAYMENTS] checkout session %s created for order %s", session.ID, req.OrderId)

	return &paymentspb.CreatePaymentRes{
		PaymentId:   payment.ID.String(),
		CheckoutUrl: session.URL,
	}, nil
}

func (h *Handler) GetPayment(ctx context.Context, req *paymentspb.GetPaymentReq) (*paymentspb.PaymentRes, error) {
	var (
		payment *db.Payment
		err     error
	)

	switch {
	case req.PaymentId != "":
		payment, err = h.repo.GetByPaymentID(ctx, req.PaymentId)
	case req.OrderId != "":
		payment, err = h.repo.GetByOrderID(ctx, req.OrderId)
	default:
		return nil, errors.New("payment_id or order_id is required")
	}
	if err != nil {
		return nil, err
	}

	return paymentToProto(payment), nil
}

// AdminListPayments returns payments of every order, optionally filtered by
// order_id. Statuses (pending/succeeded/failed) let the admin see what was paid.
func (h *Handler) AdminListPayments(ctx context.Context, req *paymentspb.AdminListPaymentsReq) (*paymentspb.AdminListPaymentsRes, error) {
	payments, err := h.repo.List(ctx, req.OrderId)
	if err != nil {
		return nil, errors.New("internal error")
	}

	res := &paymentspb.AdminListPaymentsRes{Payments: make([]*paymentspb.PaymentRes, 0, len(payments))}
	for i := range payments {
		res.Payments = append(res.Payments, paymentToProto(&payments[i]))
	}

	return res, nil
}

func (h *Handler) AdminDeletePayment(ctx context.Context, req *paymentspb.AdminDeletePaymentReq) (*paymentspb.AdminDeletePaymentRes, error) {
	if req.PaymentId == "" {
		return nil, errors.New("payment_id is required")
	}

	deleted, err := h.repo.Delete(ctx, req.PaymentId)
	if err != nil {
		return nil, errors.New("internal error")
	}
	if !deleted {
		return nil, db.ErrNotFound
	}

	return &paymentspb.AdminDeletePaymentRes{Deleted: true}, nil
}

func paymentToProto(p *db.Payment) *paymentspb.PaymentRes {
	res := &paymentspb.PaymentRes{
		Id:          p.ID.String(),
		OrderId:     p.OrderID,
		Status:      string(p.Status),
		Provider:    p.Provider,
		AmountMinor: p.Amount,
		Currency:    p.Currency,
		CheckoutUrl: p.CheckoutURL,
		CreatedAt:   p.CreatedAt.UTC().Format(time.RFC3339),
	}
	if p.PaidAt != nil {
		res.PaidAt = p.PaidAt.UTC().Format(time.RFC3339)
	}

	return res
}
