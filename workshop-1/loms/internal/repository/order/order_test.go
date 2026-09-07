package order_test

import (
	"context"
	"testing"

	"route256/loms/internal/model"
	orderrepo "route256/loms/internal/repository/order"
)

func TestCompareAndSwapStatusRejectsSecondTransition(t *testing.T) {
	repo := orderrepo.New()
	ctx := context.Background()
	id := repo.Create(ctx, 1, []model.Item{{SKU: 1001, Count: 1}}, model.StatusAwaitingPayment)

	order, err := repo.CompareAndSwapStatus(ctx, id, model.StatusAwaitingPayment, model.StatusPayed)
	if err != nil {
		t.Fatal(err)
	}
	if order.Status != model.StatusPayed {
		t.Fatalf("status=%q", order.Status)
	}

	if _, err := repo.CompareAndSwapStatus(ctx, id, model.StatusAwaitingPayment, model.StatusCancelled); err != model.ErrInvalidStatus {
		t.Fatalf("err=%v", err)
	}
}
