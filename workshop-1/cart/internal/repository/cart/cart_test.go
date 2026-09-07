package cart_test

import (
	"context"
	"testing"

	"route256/cart/internal/repository/cart"
)

func TestCartRepository(t *testing.T) {
	r := cart.New()
	ctx := context.Background()
	const userID int64 = 42

	if err := r.Add(ctx, userID, 1001, 2); err != nil {
		t.Fatal(err)
	}
	if got := r.GetCount(ctx, userID, 1001); got != 2 {
		t.Fatalf("count after add=%d", got)
	}

	if err := r.Add(ctx, userID, 1001, 3); err != nil {
		t.Fatal(err)
	}
	if got := r.GetCount(ctx, userID, 1001); got != 5 {
		t.Fatalf("count after increment=%d", got)
	}

	items := r.List(ctx, userID)
	if len(items) != 1 || items[0].SKU != 1001 || items[0].Count != 5 {
		t.Fatalf("list=%+v", items)
	}

	if err := r.Delete(ctx, userID, 1001); err != nil {
		t.Fatal(err)
	}
	if got := r.GetCount(ctx, userID, 1001); got != 0 {
		t.Fatalf("count after delete=%d", got)
	}

	if err := r.Add(ctx, userID, 1002, 1); err != nil {
		t.Fatal(err)
	}
	r.Clear(ctx, userID)
	if got := r.GetCount(ctx, userID, 1002); got != 0 {
		t.Fatalf("count after clear=%d", got)
	}
	if items := r.List(ctx, userID); len(items) != 0 {
		t.Fatalf("list after clear=%+v", items)
	}
}
