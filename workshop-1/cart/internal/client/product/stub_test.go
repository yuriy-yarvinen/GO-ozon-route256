package product

import (
	"context"
	"errors"
	"route256/cart/internal/model"
	"route256/cart/internal/pkg/ctxkey"
	"testing"
)

func TestStubGetProduct(t *testing.T) {
	t.Parallel()

	stub := New()
	ctx := ctxkey.WithToken(context.Background(), "token")

	tests := []struct {
		name      string
		sku       uint32
		wantName  string
		wantPrice uint32
	}{
		{name: "t-shirt", sku: 1001, wantName: "T-Shirt", wantPrice: 1000},
		{name: "jeans", sku: 1002, wantName: "Jeans", wantPrice: 3000},
		{name: "socks", sku: 1003, wantName: "Socks", wantPrice: 500},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			name, price, err := stub.GetProduct(ctx, tt.sku)
			if err != nil {
				t.Fatalf("GetProduct() error = %v", err)
			}
			if name != tt.wantName || price != tt.wantPrice {
				t.Fatalf("GetProduct() = (%q, %d), want (%q, %d)", name, price, tt.wantName, tt.wantPrice)
			}
		})
	}
}

func TestStubGetProductNotFound(t *testing.T) {
	t.Parallel()

	name, price, err := New().GetProduct(context.Background(), 9999)
	if !errors.Is(err, model.ErrProductNotFound) {
		t.Fatalf("GetProduct() error = %v, want %v", err, model.ErrProductNotFound)
	}
	if name != "" || price != 0 {
		t.Fatalf("GetProduct() = (%q, %d), want zero values", name, price)
	}
}
