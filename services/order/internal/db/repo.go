package db

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrOrderNotFound = errors.New("order not found")

type Repo struct {
	gorm *gorm.DB
}

func NewRepo(gormDB *gorm.DB) *Repo {
	return &Repo{gorm: gormDB}
}

// CreateOrder persists the order with its items in a single transaction.
func (r *Repo) CreateOrder(ctx context.Context, order Order) (*Order, error) {
	err := r.gorm.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return tx.Create(&order).Error
	})
	if err != nil {
		return nil, err
	}

	return &order, nil
}

func (r *Repo) SetPaymentInfo(ctx context.Context, orderID uuid.UUID, paymentID, checkoutURL string) error {
	return r.gorm.WithContext(ctx).Model(&Order{}).
		Where("id = ?", orderID).
		Updates(map[string]any{
			"payment_id":   paymentID,
			"checkout_url": checkoutURL,
		}).Error
}

// MarkPaid transitions a pending order to paid. Redelivered events and
// already-paid orders are ignored (RowsAffected == 0), which keeps the
// consumer idempotent.
func (r *Repo) MarkPaid(ctx context.Context, orderID uuid.UUID, paidAt time.Time) (bool, error) {
	result := r.gorm.WithContext(ctx).Model(&Order{}).
		Where("id = ? AND status <> ?", orderID, OrderStatusPaid).
		Updates(map[string]any{
			"status":  OrderStatusPaid,
			"paid_at": paidAt,
		})
	if result.Error != nil {
		return false, result.Error
	}

	return result.RowsAffected > 0, nil
}

func (r *Repo) GetForUser(ctx context.Context, orderID, userID string) (*Order, error) {
	var order Order
	query := r.gorm.WithContext(ctx).Preload("Items")

	if userID != "" {
		query = query.Where("user_id = ?", userID)
	}

	if err := query.First(&order, "id = ?", orderID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOrderNotFound
		}
		return nil, err
	}

	return &order, nil
}

func (r *Repo) ListForUser(ctx context.Context, userID string) ([]Order, error) {
	var orders []Order
	if err := r.gorm.WithContext(ctx).
		Preload("Items").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&orders).Error; err != nil {
		return nil, err
	}

	return orders, nil
}

// ListAll returns orders ordered by creation date. When userID is empty all
// orders are returned, otherwise it filters to a single user.
func (r *Repo) ListAll(ctx context.Context, userID string) ([]Order, error) {
	var orders []Order
	query := r.gorm.WithContext(ctx).Preload("Items")

	if userID != "" {
		query = query.Where("user_id = ?", userID)
	}

	if err := query.Order("created_at DESC").Find(&orders).Error; err != nil {
		return nil, err
	}

	return orders, nil
}

// UpdateStatus replaces an order status. Switching to paid stamps paid_at
// unless the order was already paid before. Returns ErrOrderNotFound when the
// order does not exist.
func (r *Repo) UpdateStatus(ctx context.Context, orderID uuid.UUID, status OrderStatus) (*Order, error) {
	order, err := r.GetForUser(ctx, orderID.String(), "")
	if err != nil {
		return nil, err
	}

	updates := map[string]any{"status": status}
	if status == OrderStatusPaid && order.PaidAt == nil {
		updates["paid_at"] = time.Now()
	}

	if err := r.gorm.WithContext(ctx).Model(&Order{}).
		Where("id = ?", orderID).
		Updates(updates).Error; err != nil {
		return nil, err
	}

	return r.GetForUser(ctx, orderID.String(), "")
}

// Delete removes an order together with its items in one transaction.
func (r *Repo) Delete(ctx context.Context, orderID uuid.UUID) (bool, error) {
	var deleted int64

	err := r.gorm.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Delete(&OrderItem{}, "order_id = ?", orderID)
		if res.Error != nil {
			return res.Error
		}

		res = tx.Delete(&Order{}, "id = ?", orderID)
		if res.Error != nil {
			return res.Error
		}
		deleted = res.RowsAffected

		return nil
	})
	if err != nil {
		return false, err
	}

	return deleted > 0, nil
}

// GetCart returns the user's cart creating it on first access.
func (r *Repo) GetCart(ctx context.Context, userID string) (*Cart, error) {
	var cart Cart

	newID, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}

	err = r.gorm.WithContext(ctx).
		Preload("Items").
		Where(Cart{UserID: userID}).
		Attrs(Cart{ID: newID}).
		FirstOrCreate(&cart).Error
	if err != nil {
		return nil, err
	}

	return &cart, nil
}

// AddCartItem inserts a product into the cart or adds to its quantity when it
// is already present.
func (r *Repo) AddCartItem(ctx context.Context, userID string, productID uuid.UUID, quantity int32) (*Cart, error) {
	cart, err := r.GetCart(ctx, userID)
	if err != nil {
		return nil, err
	}

	var existing CartItem
	err = r.gorm.WithContext(ctx).
		First(&existing, "cart_id = ? AND product_id = ?", cart.ID, productID).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		item := CartItem{
			ID:        uuid.Must(uuid.NewV7()),
			CartID:    cart.ID,
			ProductID: productID,
			Quantity:  quantity,
		}
		if err := r.gorm.WithContext(ctx).Create(&item).Error; err != nil {
			return nil, err
		}
	case err != nil:
		return nil, err
	default:
		if err := r.gorm.WithContext(ctx).Model(&CartItem{}).
			Where("id = ?", existing.ID).
			Update("quantity", gorm.Expr("quantity + ?", quantity)).Error; err != nil {
			return nil, err
		}
	}

	return r.reloadCart(cart.ID)
}

// SetCartItem replaces the quantity of a product in the cart. A quantity of
// zero (or less) removes the line. Returns false when the product was not in
// the cart.
func (r *Repo) SetCartItem(ctx context.Context, userID string, productID uuid.UUID, quantity int32) (bool, error) {
	cart, err := r.GetCart(ctx, userID)
	if err != nil {
		return false, err
	}

	if quantity <= 0 {
		result := r.gorm.WithContext(ctx).
			Delete(&CartItem{}, "cart_id = ? AND product_id = ?", cart.ID, productID)

		return result.Error == nil && result.RowsAffected > 0, result.Error
	}

	result := r.gorm.WithContext(ctx).Model(&CartItem{}).
		Where("cart_id = ? AND product_id = ?", cart.ID, productID).
		Update("quantity", quantity)
	if result.Error != nil {
		return false, result.Error
	}
	if result.RowsAffected == 0 {
		return false, nil
	}

	return true, nil
}

// ClearCart removes every item from the user's cart.
func (r *Repo) ClearCart(ctx context.Context, userID string) error {
	cart, err := r.GetCart(ctx, userID)
	if err != nil {
		return err
	}

	return r.gorm.WithContext(ctx).Delete(&CartItem{}, "cart_id = ?", cart.ID).Error
}

func (r *Repo) reloadCart(cartID uuid.UUID) (*Cart, error) {
	var cart Cart
	if err := r.gorm.Preload("Items").First(&cart, "id = ?", cartID).Error; err != nil {
		return nil, err
	}

	return &cart, nil
}
