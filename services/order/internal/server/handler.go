package server

import (
	"context"
	"errors"
	"log"
	"time"

	orderpb "proto/order"

	"order/internal/db"
)

type Handler struct {
	orderpb.UnimplementedOrderServer
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) CreateOrder(ctx context.Context, req *orderpb.CreateOrderReq) (*orderpb.CreateOrderRes, error) {
	if req.UserId == "" {
		return nil, statusError(errors.New("user_id is required"))
	}

	order, err := h.service.CreateOrder(ctx, CreateOrderInput{
		UserID: req.UserId,
		Email:  req.Email,
		Items:  req.Items,
	})
	if err != nil {
		return nil, statusError(err)
	}

	return &orderpb.CreateOrderRes{
		OrderId:       order.ID.String(),
		Status:        string(order.Status),
		Currency:      order.Currency,
		SubtotalMinor: order.SubtotalMinor,
		TotalMinor:    order.TotalMinor,
		PaymentId:     order.PaymentID,
		CheckoutUrl:   order.CheckoutURL,
	}, nil
}

func (h *Handler) GetOrder(ctx context.Context, req *orderpb.GetOrderReq) (*orderpb.OrderRes, error) {
	order, err := h.service.GetOrder(ctx, req.OrderId, req.UserId)
	if err != nil {
		return nil, statusError(err)
	}

	return orderToProto(order), nil
}

func (h *Handler) ListOrders(ctx context.Context, req *orderpb.ListOrdersReq) (*orderpb.ListOrdersRes, error) {
	if req.UserId == "" {
		return nil, statusError(errors.New("user_id is required"))
	}

	orders, err := h.service.ListOrders(ctx, req.UserId)
	if err != nil {
		return nil, statusError(err)
	}

	res := &orderpb.ListOrdersRes{Orders: make([]*orderpb.OrderRes, 0, len(orders))}
	for i := range orders {
		res.Orders = append(res.Orders, orderToProto(&orders[i]))
	}

	return res, nil
}

func (h *Handler) AdminListOrders(ctx context.Context, req *orderpb.AdminListOrdersReq) (*orderpb.ListOrdersRes, error) {
	orders, err := h.service.AdminListOrders(ctx, req.UserId)
	if err != nil {
		return nil, statusError(err)
	}

	res := &orderpb.ListOrdersRes{Orders: make([]*orderpb.OrderRes, 0, len(orders))}
	for i := range orders {
		res.Orders = append(res.Orders, orderToProto(&orders[i]))
	}

	return res, nil
}

func (h *Handler) AdminGetOrder(ctx context.Context, req *orderpb.AdminGetOrderReq) (*orderpb.OrderRes, error) {
	order, err := h.service.GetOrder(ctx, req.OrderId, "")
	if err != nil {
		return nil, statusError(err)
	}

	return orderToProto(order), nil
}

func (h *Handler) AdminDeleteOrder(ctx context.Context, req *orderpb.AdminDeleteOrderReq) (*orderpb.AdminDeleteOrderRes, error) {
	deleted, err := h.service.AdminDeleteOrder(ctx, req.OrderId)
	if err != nil {
		return nil, statusError(err)
	}

	return &orderpb.AdminDeleteOrderRes{Deleted: deleted}, nil
}

func (h *Handler) AdminUpdateOrderStatus(ctx context.Context, req *orderpb.AdminUpdateOrderStatusReq) (*orderpb.OrderRes, error) {
	order, err := h.service.AdminUpdateOrderStatus(ctx, req.OrderId, req.Status)
	if err != nil {
		return nil, statusError(err)
	}

	return orderToProto(order), nil
}

func (h *Handler) GetCart(ctx context.Context, req *orderpb.GetCartReq) (*orderpb.CartRes, error) {
	if req.UserId == "" {
		return nil, statusError(errors.New("user_id is required"))
	}

	view, err := h.service.GetCart(ctx, req.UserId)
	if err != nil {
		return nil, statusError(err)
	}

	return cartToProto(view), nil
}

func (h *Handler) AddToCart(ctx context.Context, req *orderpb.AddToCartReq) (*orderpb.CartRes, error) {
	if req.UserId == "" {
		return nil, statusError(errors.New("user_id is required"))
	}

	view, err := h.service.AddToCart(ctx, req.UserId, req.ProductId, req.Quantity)
	if err != nil {
		return nil, statusError(err)
	}

	return cartToProto(view), nil
}

func (h *Handler) UpdateCartItem(ctx context.Context, req *orderpb.UpdateCartItemReq) (*orderpb.CartRes, error) {
	if req.UserId == "" {
		return nil, statusError(errors.New("user_id is required"))
	}

	found, err := h.service.SetCartItem(ctx, req.UserId, req.ProductId, req.Quantity)
	if err != nil {
		return nil, statusError(err)
	}
	if !found {
		return nil, statusError(errors.New("product is not in the cart"))
	}

	view, viewErr := h.service.GetCart(ctx, req.UserId)
	if viewErr != nil {
		return nil, statusError(viewErr)
	}

	return cartToProto(view), nil
}

func (h *Handler) ClearCart(ctx context.Context, req *orderpb.ClearCartReq) (*orderpb.ClearCartRes, error) {
	if req.UserId == "" {
		return nil, statusError(errors.New("user_id is required"))
	}

	if err := h.service.ClearCart(ctx, req.UserId); err != nil {
		return nil, statusError(err)
	}

	return &orderpb.ClearCartRes{Cleared: true}, nil
}

func (h *Handler) CheckoutCart(ctx context.Context, req *orderpb.CheckoutCartReq) (*orderpb.CreateOrderRes, error) {
	if req.UserId == "" {
		return nil, statusError(errors.New("user_id is required"))
	}

	order, err := h.service.CheckoutCart(ctx, req.UserId, req.Email)
	if err != nil {
		return nil, statusError(err)
	}

	return &orderpb.CreateOrderRes{
		OrderId:       order.ID.String(),
		Status:        string(order.Status),
		Currency:      order.Currency,
		SubtotalMinor: order.SubtotalMinor,
		TotalMinor:    order.TotalMinor,
		PaymentId:     order.PaymentID,
		CheckoutUrl:   order.CheckoutURL,
	}, nil
}

func cartToProto(v *CartView) *orderpb.CartRes {
	items := make([]*orderpb.CartItemRes, 0, len(v.Items))
	for _, item := range v.Items {
		items = append(items, &orderpb.CartItemRes{
			ProductId:      item.ProductID.String(),
			Name:           item.Name,
			Quantity:       item.Quantity,
			UnitPriceMinor: item.UnitPriceMinor,
		})
	}

	return &orderpb.CartRes{Items: items, SubtotalMinor: v.SubtotalMinor}
}

func orderToProto(o *db.Order) *orderpb.OrderRes {
	items := make([]*orderpb.OrderItemRes, 0, len(o.Items))
	for _, item := range o.Items {
		items = append(items, &orderpb.OrderItemRes{
			ProductId:      item.ProductID.String(),
			Name:           item.Name,
			Quantity:       item.Quantity,
			UnitPriceMinor: item.UnitPriceMinor,
		})
	}

	paidAt := ""
	if o.PaidAt != nil {
		paidAt = o.PaidAt.UTC().Format(time.RFC3339)
	}

	return &orderpb.OrderRes{
		Id:            o.ID.String(),
		UserId:        o.UserID,
		Status:        string(o.Status),
		Currency:      o.Currency,
		SubtotalMinor: o.SubtotalMinor,
		TotalMinor:    o.TotalMinor,
		PaymentId:     o.PaymentID,
		CheckoutUrl:   o.CheckoutURL,
		Items:         items,
		CreatedAt:     o.CreatedAt.UTC().Format(time.RFC3339),
		PaidAt:        paidAt,
	}
}

func statusError(err error) error {
	switch {
	case errors.Is(err, db.ErrOrderNotFound):
		return errors.New("order not found")
	case errors.Is(err, ErrInvalidStatus),
		errors.Is(err, ErrProductsMissing),
		errors.Is(err, ErrInvalidItems),
		errors.Is(err, ErrCurrencyMixed):
		return err
	}

	log.Printf("[ORDER] internal error: %v", err)
	return errors.New("internal error")
}
