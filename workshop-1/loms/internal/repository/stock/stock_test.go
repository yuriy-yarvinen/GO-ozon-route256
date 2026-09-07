package stock_test

import (
	"testing"

	"route256/loms/internal/repository/stock"
)

func TestReserveAndAvailable(t *testing.T) {
	r := stock.New(map[uint32]uint64{1001: 100})
	if got := r.GetAvailable(1001); got != 100 {
		t.Fatalf("available=%d", got)
	}
	if err := r.Reserve(1001, 30); err != nil {
		t.Fatal(err)
	}
	if got := r.GetAvailable(1001); got != 70 {
		t.Fatalf("available=%d", got)
	}
	if err := r.Reserve(1001, 80); err == nil {
		t.Fatal("expected insufficient")
	}
}
