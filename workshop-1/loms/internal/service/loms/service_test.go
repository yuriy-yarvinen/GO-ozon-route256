package loms_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"route256/loms/internal/model"
	orderrepo "route256/loms/internal/repository/order"
	stockrepo "route256/loms/internal/repository/stock"
	lomssvc "route256/loms/internal/service/loms"
)

func newSvc() *lomssvc.Service {
	return lomssvc.New(
		orderrepo.New(),
		stockrepo.New(map[uint32]uint64{1001: 10}),
	)
}

func TestCreateReservePay(t *testing.T) {
	svc := newSvc()
	ctx := context.Background()
	id, err := svc.CreateOrder(ctx, 1, []model.Item{{SKU: 1001, Count: 3}})
	if err != nil {
		t.Fatal(err)
	}
	info, err := svc.OrderInfo(ctx, id)
	if err != nil || info.Status != model.StatusAwaitingPayment {
		t.Fatalf("info=%+v err=%v", info, err)
	}
	if err := svc.PayOrder(ctx, id); err != nil {
		t.Fatal(err)
	}
	if svc.StockInfo(ctx, 1001) != 7 {
		t.Fatalf("stock=%d", svc.StockInfo(ctx, 1001))
	}
}

func TestCreateInsufficient(t *testing.T) {
	svc := newSvc()
	_, err := svc.CreateOrder(context.Background(), 1, []model.Item{{SKU: 1001, Count: 100}})
	if err != model.ErrInsufficient {
		t.Fatalf("err=%v", err)
	}
}

func TestCancelReleases(t *testing.T) {
	svc := newSvc()
	ctx := context.Background()
	id, _ := svc.CreateOrder(ctx, 1, []model.Item{{SKU: 1001, Count: 4}})
	if err := svc.CancelOrder(ctx, id); err != nil {
		t.Fatal(err)
	}
	if svc.StockInfo(ctx, 1001) != 10 {
		t.Fatalf("stock=%d", svc.StockInfo(ctx, 1001))
	}
}

func TestCreateOrderRejectsEmptyItems(t *testing.T) {
	svc := newSvc()
	_, err := svc.CreateOrder(context.Background(), 1, nil)
	if err != model.ErrValidation {
		t.Fatalf("err=%v", err)
	}
}

func TestCreateOrderReleasesPartialReservation(t *testing.T) {
	svc := lomssvc.New(
		orderrepo.New(),
		stockrepo.New(map[uint32]uint64{1001: 10, 1002: 0}),
	)
	ctx := context.Background()

	_, err := svc.CreateOrder(ctx, 1, []model.Item{
		{SKU: 1001, Count: 4},
		{SKU: 1002, Count: 1},
	})
	if err != model.ErrInsufficient {
		t.Fatalf("err=%v", err)
	}
	if got := svc.StockInfo(ctx, 1001); got != 10 {
		t.Fatalf("stock=%d", got)
	}
}

func TestOrderInfoNotFound(t *testing.T) {
	_, err := newSvc().OrderInfo(context.Background(), 999)
	if err != model.ErrNotFound {
		t.Fatalf("err=%v", err)
	}
}

func TestPayOrderRejectsNonAwaitingOrder(t *testing.T) {
	svc := newSvc()
	ctx := context.Background()
	id, err := svc.CreateOrder(ctx, 1, []model.Item{{SKU: 1001, Count: 3}})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.PayOrder(ctx, id); err != nil {
		t.Fatal(err)
	}
	if err := svc.PayOrder(ctx, id); err != model.ErrInvalidStatus {
		t.Fatalf("err=%v", err)
	}
}

func TestCreateOrderReturnsReleaseRollbackError(t *testing.T) {
	releaseErr := errors.New("release rollback failed")
	orders := &rollbackOrderRepository{}
	stocks := &rollbackStockRepository{releaseErr: releaseErr}
	svc := lomssvc.New(orders, stocks)

	_, err := svc.CreateOrder(context.Background(), 1, []model.Item{
		{SKU: 1001, Count: 1},
		{SKU: 1002, Count: 1},
	})

	if !errors.Is(err, releaseErr) {
		t.Fatalf("err=%v", err)
	}
	if err == model.ErrInsufficient {
		t.Fatal("reported clean insufficient-stock error after failed release rollback")
	}
	if orders.order.Status != model.StatusFailed {
		t.Fatalf("status=%q", orders.order.Status)
	}
}

func TestCreateOrderReturnsStatusRollbackError(t *testing.T) {
	statusErr := errors.New("status rollback failed")
	orders := &rollbackOrderRepository{statusErr: statusErr}
	svc := lomssvc.New(orders, &rollbackStockRepository{})

	_, err := svc.CreateOrder(context.Background(), 1, []model.Item{
		{SKU: 1001, Count: 1},
		{SKU: 1002, Count: 1},
	})

	if !errors.Is(err, statusErr) {
		t.Fatalf("err=%v", err)
	}
	if err == model.ErrInsufficient {
		t.Fatal("reported clean insufficient-stock error after failed status rollback")
	}
}

type rollbackOrderRepository struct {
	order     model.Order
	statusErr error
}

func (r *rollbackOrderRepository) Create(_ context.Context, user int64, items []model.Item, status string) int64 {
	r.order = model.Order{OrderID: 1, User: user, Items: items, Status: status}
	return r.order.OrderID
}

func (r *rollbackOrderRepository) Get(_ context.Context, _ int64) (model.Order, error) {
	return r.order, nil
}

func (r *rollbackOrderRepository) SetStatus(_ context.Context, _ int64, status string) error {
	if status == model.StatusFailed && r.statusErr != nil {
		return r.statusErr
	}
	r.order.Status = status
	return nil
}

func (r *rollbackOrderRepository) CompareAndSwapStatus(_ context.Context, _ int64, from, to string) (model.Order, error) {
	if r.order.Status != from {
		return model.Order{}, model.ErrInvalidStatus
	}
	r.order.Status = to
	return r.order, nil
}

func (r *rollbackOrderRepository) ListAwaitingOlderThan(_ context.Context, _ time.Time) []model.Order {
	return nil
}

type rollbackStockRepository struct {
	reserveCalls int
	releaseErr   error
}

func (r *rollbackStockRepository) GetAvailable(context.Context, uint32) uint64 {
	return 0
}

func (r *rollbackStockRepository) Reserve(context.Context, uint32, uint16) error {
	r.reserveCalls++
	if r.reserveCalls == 2 {
		return model.ErrInsufficient
	}
	return nil
}

func (r *rollbackStockRepository) Release(context.Context, uint32, uint16) error {
	return r.releaseErr
}

func (r *rollbackStockRepository) RemoveReserved(context.Context, uint32, uint16) error {
	return nil
}
