package stock_test

import (
	"context"
	"route256/loms/internal/repository/stock"
	"testing"
)

func TestReserveAndAvailable(t *testing.T) {
	r := stock.New(map[uint32]uint64{1001: 100})
	ctx := context.Background()
	if got := r.GetAvailable(ctx, 1001); got != 100 {
		t.Fatalf("available=%d", got)
	}
	if err := r.Reserve(ctx, 1001, 30); err != nil {
		t.Fatal(err)
	}
	if got := r.GetAvailable(ctx, 1001); got != 70 {
		t.Fatalf("available=%d", got)
	}
	if err := r.Reserve(ctx, 1001, 80); err == nil {
		t.Fatal("expected insufficient")
	}
}
