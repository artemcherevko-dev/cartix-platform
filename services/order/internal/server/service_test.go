package server

import (
	"errors"
	"testing"

	orderpb "proto/order"

	"github.com/google/uuid"
)

func testProducts() []ProductInfo {
	return []ProductInfo{
		{ID: uuid.MustParse("0198c0de-0000-7000-8000-000000000001"), Name: "Train ticket", PriceMinor: 4900, Currency: "EUR", Active: true},
		{ID: uuid.MustParse("0198c0de-0000-7000-8000-000000000002"), Name: "City tour", PriceMinor: 1500, Currency: "EUR", Active: true},
		{ID: uuid.MustParse("0198c0de-0000-7000-8000-000000000003"), Name: "Old pass", PriceMinor: 6500, Currency: "EUR", Active: false},
		{ID: uuid.MustParse("0198c0de-0000-7000-8000-000000000004"), Name: "Baggage", PriceMinor: 3500, Currency: "USD", Active: true},
	}
}

func TestResolveItemsComputesSubtotalAndCurrency(t *testing.T) {
	id1 := "0198c0de-0000-7000-8000-000000000001"
	id2 := "0198c0de-0000-7000-8000-000000000002"

	items, subtotal, currency, err := resolveItems(
		[]*orderpb.OrderItemInput{
			{ProductId: id1, Quantity: 2},
			{ProductId: id2, Quantity: 1},
		},
		testProducts(),
	)
	if err != nil {
		t.Fatalf("resolveItems: %v", err)
	}

	if subtotal != 4900*2+1500 {
		t.Errorf("subtotal = %d, want %d", subtotal, 4900*2+1500)
	}
	if currency != "EUR" {
		t.Errorf("currency = %q", currency)
	}
	if len(items) != 2 || items[0].Name != "Train ticket" || items[0].UnitPriceMinor != 4900 {
		t.Errorf("items snapshot wrong: %+v", items)
	}
	if items[0].Quantity != 2 {
		t.Errorf("quantity = %d", items[0].Quantity)
	}
}

func TestResolveItemsRejectsBadRequests(t *testing.T) {
	good := "0198c0de-0000-7000-8000-000000000001"
	inactive := "0198c0de-0000-7000-8000-000000000003"
	usd := "0198c0de-0000-7000-8000-000000000004"
	unknown := uuid.Must(uuid.NewV7()).String()

	cases := []struct {
		name    string
		inputs  []*orderpb.OrderItemInput
		wantErr error
	}{
		{"no items", nil, ErrInvalidItems},
		{"zero quantity", []*orderpb.OrderItemInput{{ProductId: good, Quantity: 0}}, ErrInvalidItems},
		{"unknown product", []*orderpb.OrderItemInput{{ProductId: unknown, Quantity: 1}}, ErrProductsMissing},
		{"inactive product", []*orderpb.OrderItemInput{{ProductId: inactive, Quantity: 1}}, ErrProductsMissing},
		{
			"mixed currencies",
			[]*orderpb.OrderItemInput{
				{ProductId: good, Quantity: 1},
				{ProductId: usd, Quantity: 1},
			},
			ErrCurrencyMixed,
		},
	}

	for _, tc := range cases {
		_, _, _, err := resolveItems(tc.inputs, testProducts())
		if !errors.Is(err, tc.wantErr) {
			t.Errorf("%s: got %v, want %v", tc.name, err, tc.wantErr)
		}
	}
}
