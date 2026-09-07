package product

import (
	"context"

	"route256/cart/internal/model"
	"route256/cart/internal/pkg/ctxkey"
)

type Stub struct{}

func New() *Stub {
	return &Stub{}
}

var catalog = map[uint32]struct {
	Name  string
	Price uint32
}{
	1001: {Name: "T-Shirt", Price: 1000},
	1002: {Name: "Jeans", Price: 3000},
	1003: {Name: "Socks", Price: 500},
}

func (s *Stub) GetProduct(ctx context.Context, sku uint32) (string, uint32, error) {
	_, _ = ctxkey.TokenFromContext(ctx)

	product, ok := catalog[sku]
	if !ok {
		return "", 0, model.ErrProductNotFound
	}
	return product.Name, product.Price, nil
}
