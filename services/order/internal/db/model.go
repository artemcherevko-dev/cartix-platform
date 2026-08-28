package db

import (
	"time"

	"github.com/google/uuid"
)

type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusDone      OrderStatus = "done"
	OrderStatusPaid      OrderStatus = "paid"
	OrderStatusFailed    OrderStatus = "failed"
	OrderStatusCancelled OrderStatus = "cancelled"
)

type Order struct {
	ID     uuid.UUID   `json:"id" gorm:"type:uuid;primaryKey"`
	UserID string      `json:"user_id" gorm:"size:64;index;not null"`
	Status OrderStatus `json:"status" gorm:"type:varchar(20);not null;default:'pending'"`

	Email         string `json:"email" gorm:"size:255"`
	Currency      string `json:"currency" gorm:"size:10;not null"`
	SubtotalMinor int64  `json:"subtotal_minor" gorm:"not null"`
	TotalMinor    int64  `json:"total_minor" gorm:"not null"`

	PaymentID   string     `json:"payment_id" gorm:"size:64;index"`
	CheckoutURL string     `json:"checkout_url" gorm:"type:text"`
	PaidAt      *time.Time `json:"paid_at"`

	Items []OrderItem `json:"items" gorm:"foreignKey:OrderID;constraint:OnDelete:CASCADE;"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type OrderItem struct {
	ID             uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	OrderID        uuid.UUID `json:"order_id" gorm:"type:uuid;index;not null"`
	ProductID      uuid.UUID `json:"product_id" gorm:"type:uuid;not null"`
	Name           string    `json:"name" gorm:"size:255;not null"`
	Quantity       int32     `json:"quantity" gorm:"not null"`
	UnitPriceMinor int64     `json:"unit_price_minor" gorm:"not null"`
}

// Cart is a persistent per-user shopping cart. One user has at most one cart.
type Cart struct {
	ID        uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey"`
	UserID    string     `json:"user_id" gorm:"size:64;uniqueIndex;not null"`
	Items     []CartItem `json:"items" gorm:"foreignKey:CartID;constraint:OnDelete:CASCADE;"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type CartItem struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	CartID    uuid.UUID `json:"cart_id" gorm:"type:uuid;index;not null"`
	ProductID uuid.UUID `json:"product_id" gorm:"type:uuid;not null"`
	Quantity  int32     `json:"quantity" gorm:"not null"`
}
