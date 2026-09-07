package cart_test

import (
	"context"
	"errors"
	"testing"

	"route256/cart/internal/model"
	cartsvc "route256/cart/internal/service/cart"
)

type repositoryFake struct {
	items   map[int64]map[uint32]uint16
	cleared bool
	ctxs    []context.Context
}

func newRepositoryFake() *repositoryFake {
	return &repositoryFake{items: make(map[int64]map[uint32]uint16)}
}

func (r *repositoryFake) Add(ctx context.Context, userID int64, sku uint32, count uint16) error {
	r.ctxs = append(r.ctxs, ctx)
	if r.items[userID] == nil {
		r.items[userID] = make(map[uint32]uint16)
	}
	r.items[userID][sku] += count
	return nil
}

func (r *repositoryFake) Delete(ctx context.Context, userID int64, sku uint32) error {
	r.ctxs = append(r.ctxs, ctx)
	delete(r.items[userID], sku)
	return nil
}

func (r *repositoryFake) List(ctx context.Context, userID int64) []model.Item {
	r.ctxs = append(r.ctxs, ctx)
	items := make([]model.Item, 0, len(r.items[userID]))
	for sku, count := range r.items[userID] {
		items = append(items, model.Item{SKU: sku, Count: count})
	}
	return items
}

func (r *repositoryFake) Clear(ctx context.Context, userID int64) {
	r.ctxs = append(r.ctxs, ctx)
	r.cleared = true
	delete(r.items, userID)
}

func (r *repositoryFake) GetCount(ctx context.Context, userID int64, sku uint32) uint16 {
	r.ctxs = append(r.ctxs, ctx)
	return r.items[userID][sku]
}

type productResult struct {
	name  string
	price uint32
	err   error
}

type productFake struct {
	results map[uint32]productResult
	ctxs    []context.Context
}

func (p *productFake) GetProduct(ctx context.Context, sku uint32) (string, uint32, error) {
	p.ctxs = append(p.ctxs, ctx)
	result := p.results[sku]
	return result.name, result.price, result.err
}

type lomsFake struct {
	stock       uint64
	stockErr    error
	orderID     int64
	createErr   error
	createCalls int
	orderItems  []model.Item
	ctxs        []context.Context
}

func (l *lomsFake) StockInfo(ctx context.Context, _ uint32) (uint64, error) {
	l.ctxs = append(l.ctxs, ctx)
	return l.stock, l.stockErr
}

func (l *lomsFake) CreateOrder(ctx context.Context, _ int64, items []model.Item) (int64, error) {
	l.ctxs = append(l.ctxs, ctx)
	l.createCalls++
	l.orderItems = append([]model.Item(nil), items...)
	return l.orderID, l.createErr
}

func TestAddItemSuccessUsesCallerContext(t *testing.T) {
	repo := newRepositoryFake()
	repo.items[42] = map[uint32]uint16{1001: 2}
	products := &productFake{results: map[uint32]productResult{1001: {name: "T-Shirt", price: 1000}}}
	loms := &lomsFake{stock: 5}
	service := cartsvc.New(repo, products, loms)
	ctx := context.WithValue(context.Background(), struct{}{}, "caller")

	if err := service.AddItem(ctx, 42, 1001, 3); err != nil {
		t.Fatal(err)
	}
	if got := repo.items[42][1001]; got != 5 {
		t.Fatalf("count=%d, want 5", got)
	}
	assertContexts(t, ctx, append(append(repo.ctxs, products.ctxs...), loms.ctxs...))
}

func TestAddItemValidatesArgumentsBeforeCallingDependencies(t *testing.T) {
	tests := []struct {
		name   string
		userID int64
		sku    uint32
		count  uint16
	}{
		{name: "user", userID: 0, sku: 1001, count: 1},
		{name: "sku", userID: 42, sku: 0, count: 1},
		{name: "count", userID: 42, sku: 1001, count: 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := newRepositoryFake()
			products := &productFake{results: make(map[uint32]productResult)}
			loms := &lomsFake{stock: 10}
			service := cartsvc.New(repo, products, loms)

			err := service.AddItem(context.Background(), test.userID, test.sku, test.count)
			if !errors.Is(err, model.ErrValidation) {
				t.Fatalf("err=%v, want validation error", err)
			}
			if len(products.ctxs) != 0 || len(loms.ctxs) != 0 || len(repo.ctxs) != 0 {
				t.Fatal("dependency called for invalid input")
			}
		})
	}
}

func TestAddItemProductNotFound(t *testing.T) {
	repo := newRepositoryFake()
	products := &productFake{results: map[uint32]productResult{9999: {err: model.ErrProductNotFound}}}
	loms := &lomsFake{stock: 10}
	service := cartsvc.New(repo, products, loms)

	err := service.AddItem(context.Background(), 42, 9999, 1)
	if !errors.Is(err, model.ErrProductNotFound) {
		t.Fatalf("err=%v, want product not found", err)
	}
	if len(loms.ctxs) != 0 {
		t.Fatal("StockInfo called after product failure")
	}
}

func TestAddItemRejectsExistingPlusRequestedAboveStock(t *testing.T) {
	repo := newRepositoryFake()
	repo.items[42] = map[uint32]uint16{1001: 4}
	products := &productFake{results: map[uint32]productResult{1001: {name: "T-Shirt", price: 1000}}}
	loms := &lomsFake{stock: 5}
	service := cartsvc.New(repo, products, loms)

	err := service.AddItem(context.Background(), 42, 1001, 2)
	if !errors.Is(err, model.ErrInsufficient) {
		t.Fatalf("err=%v, want insufficient stocks", err)
	}
	if got := repo.items[42][1001]; got != 4 {
		t.Fatalf("count=%d, want unchanged count 4", got)
	}
}

func TestListCartReturnsProductDetailsAndTotal(t *testing.T) {
	repo := newRepositoryFake()
	repo.items[42] = map[uint32]uint16{1001: 2, 1002: 3}
	products := &productFake{results: map[uint32]productResult{
		1001: {name: "T-Shirt", price: 1000},
		1002: {name: "Jeans", price: 3000},
	}}
	service := cartsvc.New(repo, products, &lomsFake{})

	cart, err := service.ListCart(context.Background(), 42)
	if err != nil {
		t.Fatal(err)
	}
	if len(cart.Items) != 2 {
		t.Fatalf("items=%+v", cart.Items)
	}
	if cart.TotalPrice != 11000 {
		t.Fatalf("totalPrice=%d, want 11000", cart.TotalPrice)
	}
	for _, item := range cart.Items {
		if item.Name == "" || item.Price == 0 {
			t.Fatalf("item lacks product details: %+v", item)
		}
	}
}

func TestCheckoutClearsOnlyAfterCreateOrderSuccess(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := newRepositoryFake()
		repo.items[42] = map[uint32]uint16{1001: 2}
		loms := &lomsFake{orderID: 77}
		service := cartsvc.New(repo, &productFake{}, loms)

		orderID, err := service.Checkout(context.Background(), 42)
		if err != nil {
			t.Fatal(err)
		}
		if orderID != 77 || loms.createCalls != 1 || !repo.cleared {
			t.Fatalf("orderID=%d calls=%d cleared=%v", orderID, loms.createCalls, repo.cleared)
		}
	})

	t.Run("create error", func(t *testing.T) {
		repo := newRepositoryFake()
		repo.items[42] = map[uint32]uint16{1001: 2}
		loms := &lomsFake{createErr: model.ErrInsufficient}
		service := cartsvc.New(repo, &productFake{}, loms)

		_, err := service.Checkout(context.Background(), 42)
		if !errors.Is(err, model.ErrInsufficient) {
			t.Fatalf("err=%v, want insufficient stocks", err)
		}
		if repo.cleared {
			t.Fatal("cart cleared after CreateOrder failure")
		}
	})
}

func assertContexts(t *testing.T, want context.Context, got []context.Context) {
	t.Helper()
	for _, ctx := range got {
		if ctx != want {
			t.Fatal("dependency did not receive caller context")
		}
	}
}
