package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"order/internal/db"

	orderpb "proto/order"

	"github.com/google/uuid"
)

var (
	ErrInvalidItems    = errors.New("order must contain at least one valid item")
	ErrProductsMissing = errors.New("some products are unavailable")
	ErrCurrencyMixed   = errors.New("all items must share the same currency")
	ErrInvalidStatus   = errors.New("invalid order status")
)

// ProductInfo is what Order needs to know from the Catalog service.
type ProductInfo struct {
	ID         uuid.UUID
	Name       string
	PriceMinor int64
	Currency   string
	Active     bool
}

type CatalogClient interface {
	GetProducts(ctx context.Context, ids []string) ([]ProductInfo, error)
}

type PaymentRequest struct {
	OrderID       string
	AmountMinor   int64
	Currency      string
	CustomerEmail string
	Description   string
}

type PaymentResult struct {
	PaymentID   string
	CheckoutURL string
}

type PaymentsClient interface {
	CreatePayment(ctx context.Context, req PaymentRequest) (PaymentResult, error)
}

type Service struct {
	repo     *db.Repo
	catalog  CatalogClient
	payments PaymentsClient

	publishStatus StatusPublisherFunc // optional; nil disables notifications

	now func() time.Time // injectable clock for tests
}

// OrderStatusEvent is what the notification service needs to inform the
// customer about a status change.
type OrderStatusEvent struct {
	OrderID        string
	UserID         string
	Email          string
	Status         string
	PreviousStatus string
	TotalMinor     int64
	Currency       string
	Items          []OrderStatusItem
	UpdatedAt      time.Time
}

// OrderStatusItem is one order line inside a status notification.
type OrderStatusItem struct {
	Name        string
	Quantity    int32
	AmountMinor int64 // quantity * unit price, minor units
}

// StatusPublisherFunc delivers the event to the message bus. Implementation
// lives in main wiring to keep this package free of infrastructure.
type StatusPublisherFunc func(ctx context.Context, event OrderStatusEvent) error

// SetStatusPublisher wires the NATS publisher (optional dependency).
func (s *Service) SetStatusPublisher(fn StatusPublisherFunc) {
	s.publishStatus = fn
}

func NewService(repo *db.Repo, catalog CatalogClient, payments PaymentsClient) *Service {
	return &Service{
		repo:     repo,
		catalog:  catalog,
		payments: payments,
		now:      time.Now,
	}
}

// resolveItems validates requested items against catalog data and snapshots
// names and unit prices into order line items. Pure function: unit-testable.
func resolveItems(inputs []*orderpb.OrderItemInput, products []ProductInfo) ([]db.OrderItem, int64, string, error) {
	if len(inputs) == 0 {
		return nil, 0, "", ErrInvalidItems
	}

	byID := make(map[uuid.UUID]ProductInfo, len(products))
	for _, p := range products {
		byID[p.ID] = p
	}

	items := make([]db.OrderItem, 0, len(inputs))
	subtotal := int64(0)
	currency := ""

	for _, in := range inputs {
		id, err := uuid.Parse(in.ProductId)
		if err != nil {
			return nil, 0, "", fmt.Errorf("%w: bad product id %q", ErrProductsMissing, in.ProductId)
		}

		product, ok := byID[id]
		if !ok || !product.Active {
			return nil, 0, "", fmt.Errorf("%w: %s", ErrProductsMissing, id)
		}
		if in.Quantity < 1 {
			return nil, 0, "", fmt.Errorf("%w: quantity of %s must be >= 1", ErrInvalidItems, id)
		}
		if currency == "" {
			currency = strings.ToUpper(product.Currency)
		} else if currency != strings.ToUpper(product.Currency) {
			return nil, 0, "", ErrCurrencyMixed
		}

		line := product.PriceMinor * int64(in.Quantity)
		subtotal += line

		items = append(items, db.OrderItem{
			ProductID:      id,
			Name:           product.Name,
			Quantity:       in.Quantity,
			UnitPriceMinor: product.PriceMinor,
		})
	}

	return items, subtotal, currency, nil
}

type CreateOrderInput struct {
	UserID string
	Email  string
	Items  []*orderpb.OrderItemInput
}

func (s *Service) CreateOrder(ctx context.Context, in CreateOrderInput) (*db.Order, error) {
	productIDs := make([]string, 0, len(in.Items))
	for _, item := range in.Items {
		productIDs = append(productIDs, item.ProductId)
	}

	products, err := s.catalog.GetProducts(ctx, productIDs)
	if err != nil {
		return nil, fmt.Errorf("fetch products: %w", err)
	}

	items, subtotal, currency, err := resolveItems(in.Items, products)
	if err != nil {
		return nil, err
	}

	order := db.Order{
		ID:            uuid.Must(uuid.NewV7()),
		UserID:        in.UserID,
		Status:        db.OrderStatusPending,
		Email:         in.Email,
		Currency:      currency,
		SubtotalMinor: subtotal,
		TotalMinor:    subtotal,
		Items:         items,
		CreatedAt:     s.now(),
	}
	for i := range order.Items {
		order.Items[i].OrderID = order.ID
		order.Items[i].ID = uuid.Must(uuid.NewV7())
	}

	created, err := s.repo.CreateOrder(ctx, order)
	if err != nil {
		return nil, fmt.Errorf("create order: %w", err)
	}

	result, err := s.payments.CreatePayment(ctx, PaymentRequest{
		OrderID:       created.ID.String(),
		AmountMinor:   created.TotalMinor,
		Currency:      created.Currency,
		CustomerEmail: created.Email,
		Description:   "cartix order " + created.ID.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("create payment: %w", err)
	}

	if err := s.repo.SetPaymentInfo(ctx, created.ID, result.PaymentID, result.CheckoutURL); err != nil {
		return nil, fmt.Errorf("attach payment to order: %w", err)
	}
	created.PaymentID = result.PaymentID
	created.CheckoutURL = result.CheckoutURL

	return created, nil
}

// MarkPaid applies a payment.succeeded event. Returns false when the order
// was already paid or does not exist.
func (s *Service) MarkPaid(ctx context.Context, orderID string, paidAt time.Time) (bool, error) {
	id, err := uuid.Parse(orderID)
	if err != nil {
		return false, nil
	}

	return s.repo.MarkPaid(ctx, id, paidAt)
}

func (s *Service) GetOrder(ctx context.Context, orderID, userID string) (*db.Order, error) {
	return s.repo.GetForUser(ctx, orderID, userID)
}

func (s *Service) ListOrders(ctx context.Context, userID string) ([]db.Order, error) {
	return s.repo.ListForUser(ctx, userID)
}

// AdminListOrders lists orders of every user, optionally filtered by userID.
func (s *Service) AdminListOrders(ctx context.Context, userID string) ([]db.Order, error) {
	return s.repo.ListAll(ctx, userID)
}

// AdminDeleteOrder removes an order and its items. Returns false when the
// order does not exist.
func (s *Service) AdminDeleteOrder(ctx context.Context, orderID string) (bool, error) {
	id, err := uuid.Parse(orderID)
	if err != nil {
		return false, db.ErrOrderNotFound
	}

	return s.repo.Delete(ctx, id)
}

var validOrderStatuses = map[db.OrderStatus]bool{
	db.OrderStatusPending:   true,
	db.OrderStatusDone:      true,
	db.OrderStatusPaid:      true,
	db.OrderStatusFailed:    true,
	db.OrderStatusCancelled: true,
}

// AdminUpdateOrderStatus sets an order status to one of the allowed values.
func (s *Service) AdminUpdateOrderStatus(ctx context.Context, orderID, status string) (*db.Order, error) {
	next := db.OrderStatus(status)
	if !validOrderStatuses[next] {
		return nil, fmt.Errorf("%w: %q (allowed: pending, done, paid, failed, cancelled)", ErrInvalidStatus, status)
	}

	id, err := uuid.Parse(orderID)
	if err != nil {
		return nil, db.ErrOrderNotFound
	}

	order, err := s.repo.UpdateStatus(ctx, id, next)
	if err != nil {
		return nil, err
	}

	s.notifyStatusChanged(order)

	return order, nil
}

// notifyStatusChanged publishes the status event for the customer. A
// notification failure must not fail the already-committed update.
func (s *Service) notifyStatusChanged(order *db.Order) {
	if s.publishStatus == nil || order.Email == "" {
		return
	}

	event := OrderStatusEvent{
		OrderID:    order.ID.String(),
		UserID:     order.UserID,
		Email:      order.Email,
		Status:     string(order.Status),
		TotalMinor: order.TotalMinor,
		Currency:   order.Currency,
		UpdatedAt:  time.Now(),
	}
	for _, item := range order.Items {
		event.Items = append(event.Items, OrderStatusItem{
			Name:        item.Name,
			Quantity:    item.Quantity,
			AmountMinor: item.UnitPriceMinor * int64(item.Quantity),
		})
	}
	if order.PaidAt != nil && order.Status == db.OrderStatusPaid {
		event.UpdatedAt = *order.PaidAt
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.publishStatus(ctx, event); err != nil {
		log.Printf("[ORDER] publish %s for order %s: %v", "order.status_updated", order.ID, err)
	}
}

// CartItemView is a cart line enriched with current catalog data.
type CartItemView struct {
	ProductID      uuid.UUID
	Name           string
	Quantity       int32
	UnitPriceMinor int64
}

// CartView is the user's cart priced against the live catalog.
type CartView struct {
	Items         []CartItemView
	SubtotalMinor int64
}

// cartView prices raw cart lines using current catalog data. Lines whose
// product disappeared or was deactivated are skipped silently.
func (s *Service) cartView(ctx context.Context, cart *db.Cart) (*CartView, error) {
	if len(cart.Items) == 0 {
		return &CartView{}, nil
	}

	productIDs := make([]string, 0, len(cart.Items))
	for _, item := range cart.Items {
		productIDs = append(productIDs, item.ProductID.String())
	}

	products, err := s.catalog.GetProducts(ctx, productIDs)
	if err != nil {
		return nil, fmt.Errorf("fetch products: %w", err)
	}
	byID := make(map[uuid.UUID]ProductInfo, len(products))
	for _, p := range products {
		if p.Active {
			byID[p.ID] = p
		}
	}

	view := &CartView{}
	for _, item := range cart.Items {
		product, ok := byID[item.ProductID]
		if !ok {
			continue
		}

		view.Items = append(view.Items, CartItemView{
			ProductID:      item.ProductID,
			Name:           product.Name,
			Quantity:       item.Quantity,
			UnitPriceMinor: product.PriceMinor,
		})
		view.SubtotalMinor += product.PriceMinor * int64(item.Quantity)
	}

	return view, nil
}

func (s *Service) GetCart(ctx context.Context, userID string) (*CartView, error) {
	cart, err := s.repo.GetCart(ctx, userID)
	if err != nil {
		return nil, err
	}

	return s.cartView(ctx, cart)
}

// AddToCart validates the product and stores it in the user's cart, returning
// the refreshed cart view.
func (s *Service) AddToCart(ctx context.Context, userID, productID string, quantity int32) (*CartView, error) {
	id, err := uuid.Parse(productID)
	if err != nil {
		return nil, fmt.Errorf("%w: bad product id %q", ErrProductsMissing, productID)
	}
	if quantity < 1 {
		return nil, fmt.Errorf("%w: quantity must be >= 1", ErrInvalidItems)
	}

	products, err := s.catalog.GetProducts(ctx, []string{id.String()})
	if err != nil {
		return nil, fmt.Errorf("fetch product: %w", err)
	}
	if len(products) != 1 || !products[0].Active {
		return nil, fmt.Errorf("%w: %s", ErrProductsMissing, id)
	}

	cart, err := s.repo.AddCartItem(ctx, userID, id, quantity)
	if err != nil {
		return nil, err
	}

	return s.cartView(ctx, cart)
}

// SetCartItem replaces the quantity of a line. Returns false when the product
// is not in the cart.
func (s *Service) SetCartItem(ctx context.Context, userID, productID string, quantity int32) (bool, error) {
	id, err := uuid.Parse(productID)
	if err != nil {
		return false, db.ErrOrderNotFound
	}

	return s.repo.SetCartItem(ctx, userID, id, quantity)
}

func (s *Service) ClearCart(ctx context.Context, userID string) error {
	return s.repo.ClearCart(ctx, userID)
}

// CheckoutCart converts the user's cart into an order through the regular
// CreateOrder flow and empties the cart afterwards.
func (s *Service) CheckoutCart(ctx context.Context, userID, email string) (*db.Order, error) {
	cart, err := s.repo.GetCart(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(cart.Items) == 0 {
		return nil, ErrInvalidItems
	}

	items := make([]*orderpb.OrderItemInput, 0, len(cart.Items))
	for _, item := range cart.Items {
		items = append(items, &orderpb.OrderItemInput{
			ProductId: item.ProductID.String(),
			Quantity:  item.Quantity,
		})
	}

	order, err := s.CreateOrder(ctx, CreateOrderInput{
		UserID: userID,
		Email:  email,
		Items:  items,
	})
	if err != nil {
		return nil, err
	}

	if err := s.repo.ClearCart(ctx, userID); err != nil {
		return nil, fmt.Errorf("clear cart: %w", err)
	}

	return order, nil
}
