package worker

import (
	"context"
	"time"

	"route256/loms/internal/model"
)

const (
	cancelInterval = 30 * time.Second
	paymentTimeout = 10 * time.Minute
)

type orderLister interface {
	ListAwaitingOlderThan(ctx context.Context, cutoff time.Time) []model.Order
}

type orderCanceler interface {
	CancelOrder(ctx context.Context, orderID int64) error
}

type Worker struct {
	orders   orderLister
	service  orderCanceler
	interval time.Duration
}

func New(orders orderLister, service orderCanceler) *Worker {
	return newWithInterval(orders, service, cancelInterval)
}

func newWithInterval(orders orderLister, service orderCanceler, interval time.Duration) *Worker {
	return &Worker{
		orders:   orders,
		service:  service,
		interval: interval,
	}
}

func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			orders := w.orders.ListAwaitingOlderThan(ctx, now.Add(-paymentTimeout))
			for _, order := range orders {
				if ctx.Err() != nil {
					return
				}
				_ = w.service.CancelOrder(ctx, order.OrderID)
			}
		}
	}
}
