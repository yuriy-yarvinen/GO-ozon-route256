package loms_test

import (
	"context"
	"testing"

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
