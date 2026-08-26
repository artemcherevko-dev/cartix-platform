package checkout

import (
	"context"
	"fmt"

	"github.com/stripe/stripe-go/v85"
)

// Session wraps the fields the payments service needs from a created
// Checkout Session.
type Session struct {
	ID  string
	URL string
}

// SessionCreator creates Stripe Checkout Sessions. *stripe.Client's
// V1CheckoutSessions service satisfies this; tests can supply fakes.
type SessionCreator interface {
	Create(ctx context.Context, params *stripe.CheckoutSessionCreateParams) (*stripe.CheckoutSession, error)
}

type Creator struct {
	sessions SessionCreator
	appURL   string
}

func NewCreator(sessions SessionCreator, appURL string) *Creator {
	return &Creator{
		sessions: sessions,
		appURL:   appURL,
	}
}

type CreateInput struct {
	OrderID       string
	PaymentID     string
	AmountMinor   int64
	Currency      string
	CustomerEmail string
	Description   string
}

func (c *Creator) CreateCheckoutSession(ctx context.Context, in CreateInput) (*Session, error) {
	successURL := fmt.Sprintf("%s/orders/%s?payment=success", c.appURL, in.OrderID)
	cancelURL := fmt.Sprintf("%s/orders/%s?payment=cancelled", c.appURL, in.OrderID)

	params := &stripe.CheckoutSessionCreateParams{
		Mode:          stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL:    stripe.String(successURL),
		CancelURL:     stripe.String(cancelURL),
		CustomerEmail: stripe.String(in.CustomerEmail),
		Metadata: map[string]string{
			"order_id":   in.OrderID,
			"payment_id": in.PaymentID,
		},
		LineItems: []*stripe.CheckoutSessionCreateLineItemParams{
			{
				Quantity: stripe.Int64(1),
				PriceData: &stripe.CheckoutSessionCreateLineItemPriceDataParams{
					Currency:   stripe.String(in.Currency),
					UnitAmount: stripe.Int64(in.AmountMinor),
					ProductData: &stripe.CheckoutSessionCreateLineItemPriceDataProductDataParams{
						Name:        stripe.String("cartix order"),
						Description: stripe.String(in.Description),
					},
				},
			},
		},
	}

	session, err := c.sessions.Create(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("create stripe checkout session: %w", err)
	}
	if session.ID == "" || session.URL == "" {
		return nil, fmt.Errorf("stripe returned incomplete checkout session")
	}

	return &Session{ID: session.ID, URL: session.URL}, nil
}
