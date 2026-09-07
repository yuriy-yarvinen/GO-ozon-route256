package order

import (
	"context"
	"route256/loms/internal/model"
	"sync"
	"time"
)

type entry struct {
	order     model.Order
	createdAt time.Time
}

type Repository struct {
	mu     sync.RWMutex
	nextID int64
	orders map[int64]entry
}

func New() *Repository {
	return &Repository{orders: make(map[int64]entry)}
}

func (r *Repository) Create(_ context.Context, user int64, items []model.Item, status string) int64 {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.nextID++
	orderID := r.nextID
	r.orders[orderID] = entry{
		order: model.Order{
			OrderID: orderID,
			User:    user,
			Items:   cloneItems(items),
			Status:  status,
		},
		createdAt: time.Now(),
	}
	return orderID
}

func (r *Repository) Get(_ context.Context, orderID int64) (model.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stored, ok := r.orders[orderID]
	if !ok {
		return model.Order{}, model.ErrNotFound
	}
	result := stored.order
	result.Items = cloneItems(result.Items)
	return result, nil
}

func (r *Repository) SetStatus(_ context.Context, orderID int64, status string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	stored, ok := r.orders[orderID]
	if !ok {
		return model.ErrNotFound
	}
	stored.order.Status = status
	r.orders[orderID] = stored
	return nil
}

func (r *Repository) CompareAndSwapStatus(_ context.Context, orderID int64, from, to string) (model.Order, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	stored, ok := r.orders[orderID]
	if !ok {
		return model.Order{}, model.ErrNotFound
	}
	if stored.order.Status != from {
		return model.Order{}, model.ErrInvalidStatus
	}
	stored.order.Status = to
	r.orders[orderID] = stored

	result := stored.order
	result.Items = cloneItems(result.Items)
	return result, nil
}

func (r *Repository) ListAwaitingOlderThan(_ context.Context, cutoff time.Time) []model.Order {
	r.mu.RLock()
	defer r.mu.RUnlock()

	orders := make([]model.Order, 0)
	for _, stored := range r.orders {
		if stored.order.Status != model.StatusAwaitingPayment || !stored.createdAt.Before(cutoff) {
			continue
		}
		result := stored.order
		result.Items = cloneItems(result.Items)
		orders = append(orders, result)
	}
	return orders
}

func cloneItems(items []model.Item) []model.Item {
	return append([]model.Item(nil), items...)
}
