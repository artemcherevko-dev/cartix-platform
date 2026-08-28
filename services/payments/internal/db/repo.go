package db

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SuccessInput struct {
	StripeEventID   string
	StripeSessionID string
	StripePaymentID string
	OrderID         string
	Email           string
	Amount          int64
	Currency        string
	PaidAt          time.Time
}

type SuccessResult struct {
	PaymentID        string
	OrderID          string
	StripePaymentID  string
	Email            string
	Amount           int64
	Currency         string
	AlreadySucceeded bool
}

var ErrNotFound = errors.New("payment not found")

type Repo struct {
	gorm *gorm.DB
}

func NewRepo(gormDB *gorm.DB) *Repo {
	return &Repo{gorm: gormDB}
}

// CreatePending stores a newly created checkout payment before the customer
// has paid.
func (r *Repo) CreatePending(ctx context.Context, payment Payment) error {
	return r.gorm.WithContext(ctx).Create(&payment).Error
}

// GetByPaymentID returns a payment by its internal id (empty string ignored).
func (r *Repo) GetByPaymentID(ctx context.Context, paymentID string) (*Payment, error) {
	if paymentID == "" {
		return nil, ErrNotFound
	}

	var payment Payment
	if err := r.gorm.WithContext(ctx).First(&payment, "id = ?", paymentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &payment, nil
}

// GetByOrderID returns the newest payment attached to an order.
func (r *Repo) GetByOrderID(ctx context.Context, orderID string) (*Payment, error) {
	if orderID == "" {
		return nil, ErrNotFound
	}

	var payment Payment
	if err := r.gorm.WithContext(ctx).
		Order("created_at DESC").
		First(&payment, "order_id = ?", orderID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &payment, nil
}

// List returns payments ordered by creation date. When orderID is empty all
// payments are returned, otherwise it filters to a single order.
func (r *Repo) List(ctx context.Context, orderID string) ([]Payment, error) {
	var payments []Payment
	query := r.gorm.WithContext(ctx)

	if orderID != "" {
		query = query.Where("order_id = ?", orderID)
	}

	if err := query.Order("created_at DESC").Find(&payments).Error; err != nil {
		return nil, err
	}

	return payments, nil
}

// Delete removes a payment. Returns false when the payment does not exist.
func (r *Repo) Delete(ctx context.Context, paymentID string) (bool, error) {
	result := r.gorm.WithContext(ctx).Delete(&Payment{}, "id = ?", paymentID)
	if result.Error != nil {
		return false, result.Error
	}

	return result.RowsAffected > 0, nil
}

// UpsertSucceeded marks the payment as succeeded. Webhooks can be delivered
// more than once, so the operation is idempotent: if the payment is already
// succeeded it is returned as-is with AlreadySucceeded set.
func (r *Repo) UpsertSucceeded(ctx context.Context, in SuccessInput) (SuccessResult, error) {
	var payment Payment

	sessionID := nullIfEmpty(in.StripeSessionID)
	paymentID := nullIfEmpty(in.StripePaymentID)

	err := r.gorm.WithContext(ctx).
		Where("stripe_payment_id = ? OR stripe_session_id = ?", paymentID, sessionID).
		First(&payment).Error

	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		payment = Payment{
			ID:              uuid.Must(uuid.NewV7()),
			Status:          PaymentStatusSucceeded,
			Provider:        ProviderStripe,
			StripeSessionID: sessionID,
			StripePaymentID: paymentID,
		}
	case err != nil:
		return SuccessResult{}, err
	}

	alreadySucceeded := payment.Status == PaymentStatusSucceeded

	if payment.OrderID == "" {
		payment.OrderID = in.OrderID
	}
	if payment.Email == "" {
		payment.Email = in.Email
	}
	if payment.Amount == 0 {
		payment.Amount = in.Amount
	}
	if payment.Currency == "" {
		payment.Currency = in.Currency
	}

	payment.Status = PaymentStatusSucceeded
	if paidID := nullIfEmpty(in.StripePaymentID); paidID != nil && payment.StripePaymentID == nil {
		payment.StripePaymentID = paidID
	}
	if session := nullIfEmpty(in.StripeSessionID); session != nil && payment.StripeSessionID == nil {
		payment.StripeSessionID = session
	}
	if payment.PaidAt == nil {
		paidAt := in.PaidAt
		payment.PaidAt = &paidAt
	}

	if err := r.gorm.WithContext(ctx).Save(&payment).Error; err != nil {
		return SuccessResult{}, err
	}

	return SuccessResult{
		PaymentID:        payment.ID.String(),
		OrderID:          payment.OrderID,
		StripePaymentID:  in.StripePaymentID,
		Email:            payment.Email,
		Amount:           payment.Amount,
		Currency:         payment.Currency,
		AlreadySucceeded: alreadySucceeded,
	}, nil
}

func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
