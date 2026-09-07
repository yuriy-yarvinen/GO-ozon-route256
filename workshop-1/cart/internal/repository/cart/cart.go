package cart

import (
	"context"
	"sync"

	"route256/cart/internal/model"
)

type Repository struct {
	mu   sync.Mutex
	data map[int64]map[uint32]uint16
}

func New() *Repository {
	return &Repository{data: make(map[int64]map[uint32]uint16)}
}

func (r *Repository) Add(_ context.Context, userID int64, sku uint32, count uint16) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	cart, ok := r.data[userID]
	if !ok {
		cart = make(map[uint32]uint16)
		r.data[userID] = cart
	}
	cart[sku] += count
	return nil
}

func (r *Repository) Delete(_ context.Context, userID int64, sku uint32) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	cart, ok := r.data[userID]
	if !ok {
		return model.ErrNotFound
	}
	if _, ok := cart[sku]; !ok {
		return model.ErrNotFound
	}
	delete(cart, sku)
	if len(cart) == 0 {
		delete(r.data, userID)
	}
	return nil
}

func (r *Repository) List(_ context.Context, userID int64) []model.Item {
	r.mu.Lock()
	defer r.mu.Unlock()

	cart, ok := r.data[userID]
	if !ok || len(cart) == 0 {
		return nil
	}
	items := make([]model.Item, 0, len(cart))
	for sku, count := range cart {
		items = append(items, model.Item{SKU: sku, Count: count})
	}
	return items
}

func (r *Repository) Clear(_ context.Context, userID int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.data, userID)
}

func (r *Repository) GetCount(_ context.Context, userID int64, sku uint32) uint16 {
	r.mu.Lock()
	defer r.mu.Unlock()

	cart, ok := r.data[userID]
	if !ok {
		return 0
	}
	return cart[sku]
}
