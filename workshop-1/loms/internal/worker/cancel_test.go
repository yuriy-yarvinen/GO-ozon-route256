package worker

import (
	"context"
	"sync"
	"testing"
	"time"

	"route256/loms/internal/model"
)

type orderListerStub struct {
	mu      sync.Mutex
	cutoffs []time.Time
	orders  []model.Order
}

func (s *orderListerStub) ListAwaitingOlderThan(_ context.Context, cutoff time.Time) []model.Order {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cutoffs = append(s.cutoffs, cutoff)
	return append([]model.Order(nil), s.orders...)
}

type cancelerStub struct {
	canceled chan int64
}

func (s *cancelerStub) CancelOrder(_ context.Context, orderID int64) error {
	s.canceled <- orderID
	return nil
}

func TestWorkerCancelsAwaitingOrdersOlderThanTenMinutes(t *testing.T) {
	lister := &orderListerStub{orders: []model.Order{{OrderID: 42}}}
	canceler := &cancelerStub{canceled: make(chan int64, 1)}
	worker := newWithInterval(lister, canceler, time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go worker.Run(ctx)

	select {
	case orderID := <-canceler.canceled:
		if orderID != 42 {
			t.Fatalf("orderID=%d", orderID)
		}
	case <-time.After(time.Second):
		t.Fatal("worker did not cancel order")
	}

	lister.mu.Lock()
	defer lister.mu.Unlock()
	if len(lister.cutoffs) == 0 {
		t.Fatal("worker did not list old orders")
	}
	age := time.Since(lister.cutoffs[0])
	if age < 10*time.Minute || age > 10*time.Minute+time.Second {
		t.Fatalf("cutoff age=%s", age)
	}
}

func TestWorkerStopsWhenContextIsCanceled(t *testing.T) {
	worker := newWithInterval(
		&orderListerStub{},
		&cancelerStub{canceled: make(chan int64, 1)},
		time.Hour,
	)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	done := make(chan struct{})
	go func() {
		worker.Run(ctx)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("worker did not stop")
	}
}
