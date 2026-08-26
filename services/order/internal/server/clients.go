package server

import (
	"context"
	"fmt"

	catalogpb "proto/catalog"
	paymentspb "proto/payments"
)

// CatalogGRPC adapts the Catalog gRPC stub to the CatalogClient interface.
type CatalogGRPC struct {
	client catalogpb.CatalogClient
}

func NewCatalogGRPC(client catalogpb.CatalogClient) *CatalogGRPC {
	return &CatalogGRPC{client: client}
}

func (c *CatalogGRPC) GetProducts(ctx context.Context, ids []string) ([]ProductInfo, error) {
	res, err := c.client.GetProducts(ctx, &catalogpb.GetProductsReq{Ids: ids})
	if err != nil {
		return nil, fmt.Errorf("catalog GetProducts: %w", err)
	}

	products := make([]ProductInfo, 0, len(res.Products))
	for _, p := range res.Products {
		id, err := parseUUID(p.Id)
		if err != nil {
			return nil, fmt.Errorf("catalog returned bad product id %q", p.Id)
		}

		products = append(products, ProductInfo{
			ID:         id,
			Name:       p.Name,
			PriceMinor: p.PriceMinor,
			Currency:   p.Currency,
			Active:     p.Active,
		})
	}

	return products, nil
}

// PaymentsGRPC adapts the Payments gRPC stub to the PaymentsClient interface.
type PaymentsGRPC struct {
	client paymentspb.PaymentsClient
}

func NewPaymentsGRPC(client paymentspb.PaymentsClient) *PaymentsGRPC {
	return &PaymentsGRPC{client: client}
}

func (p *PaymentsGRPC) CreatePayment(ctx context.Context, req PaymentRequest) (PaymentResult, error) {
	res, err := p.client.CreatePayment(ctx, &paymentspb.CreatePaymentReq{
		OrderId:       req.OrderID,
		AmountMinor:   req.AmountMinor,
		Currency:      req.Currency,
		CustomerEmail: req.CustomerEmail,
		Description:   req.Description,
	})
	if err != nil {
		return PaymentResult{}, fmt.Errorf("payments CreatePayment: %w", err)
	}

	return PaymentResult{
		PaymentID:   res.PaymentId,
		CheckoutURL: res.CheckoutUrl,
	}, nil
}
