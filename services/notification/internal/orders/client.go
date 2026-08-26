package orders

import (
	"context"
	"fmt"

	orderpb "proto/order"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Item struct {
	Name           string
	Quantity       int32
	UnitPriceMinor int64
}

// Client fetches order data for receipt rendering.
type Client struct {
	grpc orderpb.OrderClient
}

func New(target string) (*Client, error) {
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("connect order service: %w", err)
	}

	return &Client{grpc: orderpb.NewOrderClient(conn)}, nil
}

// GetOrderItems returns the line items of an order. AdminGetOrder is used
// because the notification service acts on behalf of the system.
func (c *Client) GetOrderItems(ctx context.Context, orderID string) ([]Item, error) {
	res, err := c.grpc.AdminGetOrder(ctx, &orderpb.AdminGetOrderReq{OrderId: orderID})
	if err != nil {
		return nil, fmt.Errorf("get order %s: %w", orderID, err)
	}

	items := make([]Item, 0, len(res.Items))
	for _, item := range res.Items {
		items = append(items, Item{
			Name:           item.Name,
			Quantity:       item.Quantity,
			UnitPriceMinor: item.UnitPriceMinor,
		})
	}

	return items, nil
}
