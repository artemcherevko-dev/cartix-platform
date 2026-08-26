package db

import (
	"time"

	"github.com/google/uuid"
)

type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "pending"
	PaymentStatusSucceeded PaymentStatus = "succeeded"
	PaymentStatusFailed    PaymentStatus = "failed"
)

const ProviderStripe = "stripe"

type Payment struct {
	ID      uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	OrderID string    `json:"order_id" gorm:"size:64;index"`

	Status   PaymentStatus `json:"status" gorm:"type:varchar(20);not null;default:'pending'"`
	Provider string        `json:"provider" gorm:"type:varchar(20);not null;default:'stripe'"`

	// Nullable so several pending payments (no Stripe ids yet) can coexist
	// under the unique indexes; Postgres treats NULLs as distinct.
	StripeSessionID *string `json:"stripe_session_id" gorm:"size:255;uniqueIndex"`
	StripePaymentID *string `json:"stripe_payment_id" gorm:"size:255;uniqueIndex"`
	CheckoutURL     string  `json:"checkout_url" gorm:"type:text"`

	Email    string `json:"email" gorm:"size:255;index"`
	Amount   int64  `json:"amount"`
	Currency string `json:"currency" gorm:"size:10"`

	PaidAt *time.Time `json:"paid_at"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
