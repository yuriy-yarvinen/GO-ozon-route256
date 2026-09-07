# Workshop-1 Cart + LOMS Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement layered `cart` and `loms` services with real in-memory business logic, HTTP JSON between them, and a stub ProductService client.

**Architecture:** Handler → service (interfaces at point of use) → in-memory repository / HTTP or stub clients. LOMS owns orders+stocks and auto-cancels unpaid orders after 10 minutes. Cart validates via Product stub + LOMS stocks, checkouts via LOMS order/create.

**Tech Stack:** Go 1.21, stdlib `net/http` + `encoding/json`, docker-compose, existing Makefiles (`make precommit`, `make run-all`).

**Spec:** `docs/superpowers/specs/2026-09-07-workshop-1-cart-loms-design.md`

## Global Constraints

- Modules: `route256/cart`, `route256/loms` (already in `workshop-1/go.work`).
- Listen `:8080` inside containers; compose maps cart `8082`, loms `8083`.
- HTTP POST + JSON bodies; paths exactly as in docs (`/cart/...`, `/order/...`, `/stock/info`).
- ProductService: **in-process stub only** — no network to `route256.pavl.uk`.
- Token: read `Authorization` (strip optional `Bearer `), put in context; stub may ignore value but must receive call through interface.
- Statuses: `new` | `awaiting payment` | `failed` | `payed` | `cancelled`.
- Errors: 400 validation, 404 not found, 412 failed precondition / insufficient stocks.
- Seed stocks for SKUs `1001`, `1002`, `1003` with totals `100`, `50`, `10` (same SKUs known to product stub).
- Notifications service stays empty stub; do not expand Kafka scope.
- **Context:** pass `context.Context` on every handler → service → repo/client call; do not replace request context with `context.Background()` on the hot path.
- **Timeouts:** outbound HTTP and request-scoped work use deadlines **≤ 500ms** (`context.WithTimeout(..., 500*time.Millisecond)` and/or `http.Client{Timeout: 500 * time.Millisecond}`).

---

## File map

### LOMS (`workshop-1/loms/`)

| Path | Responsibility |
|------|----------------|
| `internal/model/model.go` | Order, Item, statuses, errors |
| `internal/repository/stock/stock.go` | In-memory stocks |
| `internal/repository/order/order.go` | In-memory orders |
| `internal/service/loms/service.go` | Create/Info/Pay/Cancel/StockInfo |
| `internal/service/loms/service_test.go` | Unit tests |
| `internal/handler/handler.go` | HTTP handlers + router |
| `internal/worker/cancel.go` | Auto-cancel unpaid >10m |
| `internal/config/config.go` | Addr (optional) |
| `cmd/app/main.go` | Wiring |

### Cart (`workshop-1/cart/`)

| Path | Responsibility |
|------|----------------|
| `internal/model/model.go` | Cart item / list DTOs, errors |
| `internal/pkg/ctxkey/ctxkey.go` | Token context key + helpers |
| `internal/repository/cart/cart.go` | In-memory cart |
| `internal/client/product/stub.go` | ProductService stub |
| `internal/client/loms/client.go` | HTTP client to LOMS |
| `internal/service/cart/service.go` | Add/Delete/List/Clear/Checkout |
| `internal/service/cart/service_test.go` | Unit tests with fakes |
| `internal/handler/handler.go` | HTTP + auth middleware |
| `internal/config/config.go` | `LOMS_URL` |
| `cmd/app/main.go` | Wiring |

### Compose

| Path | Change |
|------|--------|
| `workshop-1/docker-compose.yml` | Env `LOMS_URL=http://loms:8080` for cart |

---

### Task 1: LOMS models + stock repository

**Files:**
- Create: `workshop-1/loms/internal/model/model.go`
- Create: `workshop-1/loms/internal/repository/stock/stock.go`
- Test: `workshop-1/loms/internal/repository/stock/stock_test.go`

**Interfaces:**
- Produces: `model.Item`, stock repo methods `GetAvailable`, `Reserve`, `Release`, `RemoveReserved`

- [ ] **Step 1: Write failing stock tests**

```go
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
```

- [ ] **Step 2: Run test — expect fail (package missing)**

Run: `cd workshop-1/loms && go test ./internal/repository/stock/ -v`  
Expected: FAIL cannot find package / undefined

- [ ] **Step 3: Implement model + stock repo**

```go
// workshop-1/loms/internal/model/model.go
package model

import "errors"

const (
	StatusNew             = "new"
	StatusAwaitingPayment = "awaiting payment"
	StatusFailed          = "failed"
	StatusPayed           = "payed"
	StatusCancelled       = "cancelled"
)

type Item struct {
	SKU   uint32 `json:"sku"`
	Count uint16 `json:"count"`
}

type Order struct {
	OrderID int64
	User    int64
	Items   []Item
	Status  string
}

var (
	ErrNotFound      = errors.New("not found")
	ErrInsufficient  = errors.New("insufficient stocks")
	ErrInvalidStatus = errors.New("invalid status")
	ErrValidation    = errors.New("validation error")
)
```

```go
// workshop-1/loms/internal/repository/stock/stock.go
package stock

import (
	"sync"

	"route256/loms/internal/model"
)

type entry struct {
	total    uint64
	reserved uint64
}

type Repository struct {
	mu   sync.Mutex
	data map[uint32]*entry
}

func New(seed map[uint32]uint64) *Repository {
	data := make(map[uint32]*entry, len(seed))
	for sku, total := range seed {
		data[sku] = &entry{total: total}
	}
	return &Repository{data: data}
}

func (r *Repository) GetAvailable(sku uint32) uint64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	e, ok := r.data[sku]
	if !ok {
		return 0
	}
	return e.total - e.reserved
}

func (r *Repository) Reserve(sku uint32, count uint16) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	e, ok := r.data[sku]
	if !ok || e.total-e.reserved < uint64(count) {
		return model.ErrInsufficient
	}
	e.reserved += uint64(count)
	return nil
}

func (r *Repository) Release(sku uint32, count uint16) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	e, ok := r.data[sku]
	if !ok || e.reserved < uint64(count) {
		return model.ErrInsufficient
	}
	e.reserved -= uint64(count)
	return nil
}

func (r *Repository) RemoveReserved(sku uint32, count uint16) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	e, ok := r.data[sku]
	if !ok || e.reserved < uint64(count) || e.total < uint64(count) {
		return model.ErrInsufficient
	}
	e.reserved -= uint64(count)
	e.total -= uint64(count)
	return nil
}
```

- [ ] **Step 4: Run tests — expect PASS**

Run: `cd workshop-1/loms && go test ./internal/repository/stock/ -v`

- [ ] **Step 5: Commit**

```bash
git add workshop-1/loms/internal/model workshop-1/loms/internal/repository/stock
git commit -m "feat(loms): add models and in-memory stock repository"
```

---

### Task 2: LOMS order repository + service

**Files:**
- Create: `workshop-1/loms/internal/repository/order/order.go`
- Create: `workshop-1/loms/internal/service/loms/service.go`
- Test: `workshop-1/loms/internal/service/loms/service_test.go`

**Interfaces:**
- Consumes: stock repo from Task 1
- Produces: `Service.CreateOrder`, `OrderInfo`, `PayOrder`, `CancelOrder`, `StockInfo`

- [ ] **Step 1: Write failing service tests**

```go
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
```

- [ ] **Step 2: Run — expect FAIL**

Run: `cd workshop-1/loms && go test ./internal/service/loms/ -v`

- [ ] **Step 3: Implement order repo + service**

Order repo: mutex map, auto-increment `orderID`, store `createdAt time.Time`, methods `Create`, `Get`, `SetStatus`, `ListAwaitingOlderThan(time.Time)`.

Service logic:
- `CreateOrder`: validate non-empty items → create `new` → reserve all (on any fail: set `failed`, return `ErrInsufficient`) → set `awaiting payment` → return id. Reserve should be all-or-nothing (release already reserved items on partial failure).
- `PayOrder`: only `awaiting payment` → `RemoveReserved` each item → `payed`.
- `CancelOrder`: only `awaiting payment` → `Release` each → `cancelled`.
- `StockInfo`: `GetAvailable`.
- `OrderInfo`: get or `ErrNotFound`.

- [ ] **Step 4: Run tests — PASS**

Run: `cd workshop-1/loms && go test ./internal/service/loms/ -v`

- [ ] **Step 5: Commit**

```bash
git add workshop-1/loms/internal/repository/order workshop-1/loms/internal/service
git commit -m "feat(loms): add order repo and core service logic"
```

---

### Task 3: LOMS HTTP handlers + auto-cancel worker + main

**Files:**
- Create: `workshop-1/loms/internal/handler/handler.go`
- Create: `workshop-1/loms/internal/worker/cancel.go`
- Modify: `workshop-1/loms/cmd/app/main.go`

**Interfaces:**
- Consumes: `*lomssvc.Service`
- Produces: HTTP routes matching docs; worker calls `CancelOrder`

- [ ] **Step 1: Implement handlers**

Use `r.Context()` (optionally `WithTimeout(..., 500*time.Millisecond)`). Pass ctx into service methods. Decode JSON POST bodies; map errors → status codes (`ErrValidation`→400, `ErrNotFound`→404, `ErrInsufficient`/`ErrInvalidStatus`→412).

Routes:
- `POST /order/create` → `{user, items}` → `{orderID}`
- `POST /order/info` → `{orderID}` → `{status,user,items}`
- `POST /order/pay` → `{orderID}` → `{}`
- `POST /order/cancel` → `{orderID}` → `{}`
- `POST /stock/info` → `{sku}` → `{count}`

- [ ] **Step 2: Implement worker**

```go
// every 30s: list awaiting older than 10m, CancelOrder each
func (w *Worker) Run(ctx context.Context) { ... }
```

- [ ] **Step 3: Wire main**

```go
seed := map[uint32]uint64{1001: 100, 1002: 50, 1003: 10}
// repos -> service -> worker goroutine -> handler mux on :8080
```

Replace current stub `main.go`.

- [ ] **Step 4: Smoke build**

Run: `cd workshop-1/loms && go build -o /tmp/loms ./cmd/app`  
Expected: success

- [ ] **Step 5: Commit**

```bash
git add workshop-1/loms
git commit -m "feat(loms): HTTP handlers, auto-cancel worker, main wiring"
```

---

### Task 4: Cart models, ctxkey, cart repository

**Files:**
- Create: `workshop-1/cart/internal/model/model.go`
- Create: `workshop-1/cart/internal/pkg/ctxkey/ctxkey.go`
- Create: `workshop-1/cart/internal/repository/cart/cart.go`
- Test: `workshop-1/cart/internal/repository/cart/cart_test.go`

**Interfaces:**
- Produces: `ctxkey.WithToken` / `TokenFromContext`; cart `Add`, `Delete`, `List`, `Clear`, `GetCount`

- [ ] **Step 1: Write cart repo test**

Add item, increment same SKU, delete, clear — assert counts.

- [ ] **Step 2: Implement**

```go
// ctxkey
type key int
const tokenKey key = 1
func WithToken(ctx context.Context, token string) context.Context
func TokenFromContext(ctx context.Context) (string, bool)
```

Cart repo: `map[int64]map[uint32]uint16` + mutex.

Model errors: `ErrValidation`, `ErrNotFound`, `ErrInsufficient`, `ErrProductNotFound`.

- [ ] **Step 3: Test PASS + commit**

```bash
git add workshop-1/cart/internal/model workshop-1/cart/internal/pkg workshop-1/cart/internal/repository
git commit -m "feat(cart): models, token context, in-memory cart repo"
```

---

### Task 5: Product stub + LOMS HTTP client

**Files:**
- Create: `workshop-1/cart/internal/client/product/stub.go`
- Create: `workshop-1/cart/internal/client/loms/client.go`
- Test: `workshop-1/cart/internal/client/product/stub_test.go`

**Interfaces:**
- Produces product: `GetProduct(ctx, sku) (name string, price uint32, err error)`
- Produces loms client: `StockInfo`, `CreateOrder` (minimal needed by cart service; optional Info/Pay/Cancel unused by cart)

- [ ] **Step 1: Product stub**

```go
var catalog = map[uint32]struct{ Name string; Price uint32 }{
	1001: {"T-Shirt", 1000},
	1002: {"Jeans", 3000},
	1003: {"Socks", 500},
}

func (s *Stub) GetProduct(ctx context.Context, sku uint32) (string, uint32, error) {
	_ = ctxkey.TokenFromContext(ctx) // wired; stub ignores value
	p, ok := catalog[sku]
	if !ok {
		return "", 0, model.ErrProductNotFound
	}
	return p.Name, p.Price, nil
}
```

- [ ] **Step 2: LOMS HTTP client**

`New(baseURL string, httpClient *http.Client)`; default client `Timeout: 500 * time.Millisecond` if nil. Each method: `ctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond); defer cancel()` then `http.NewRequestWithContext`. POST JSON to `/stock/info` and `/order/create`; map non-2xx: 412→`ErrInsufficient`, 404→`ErrNotFound`, else wrap error.

- [ ] **Step 3: Stub unit test + commit**

```bash
git add workshop-1/cart/internal/client
git commit -m "feat(cart): product stub and LOMS HTTP client"
```

---

### Task 6: Cart service

**Files:**
- Create: `workshop-1/cart/internal/service/cart/service.go`
- Test: `workshop-1/cart/internal/service/cart/service_test.go`

**Interfaces:**
- Consumes: cart repo, product stub interface, loms client interface
- Produces: `AddItem`, `DeleteItem`, `ListCart`, `ClearCart`, `Checkout`

- [ ] **Step 1: Write tests with fake product + fake loms**

Cases:
- Add success (product ok, stocks enough)
- Add fails product not found
- Add fails insufficient (existing+requested > available)
- List returns names/prices/totalPrice
- Checkout calls CreateOrder and clears cart on success; does not clear on error

- [ ] **Step 2: Implement service**

`AddItem`: validate user/sku/count > 0 → GetProduct → StockInfo → compare `repo.GetCount+count` → Add.  
`Checkout`: List items → CreateOrder → Clear → return orderID.

- [ ] **Step 3: PASS + commit**

```bash
git add workshop-1/cart/internal/service
git commit -m "feat(cart): cart service with stock and product checks"
```

---

### Task 7: Cart HTTP handlers + main + compose

**Files:**
- Create: `workshop-1/cart/internal/handler/handler.go`
- Create: `workshop-1/cart/internal/config/config.go`
- Modify: `workshop-1/cart/cmd/app/main.go`
- Modify: `workshop-1/docker-compose.yml`

**Interfaces:**
- Consumes: cart service
- Produces: routes `/cart/item/add|delete`, `/cart/list`, `/cart/clear`, `/cart/checkout`

- [ ] **Step 1: Middleware + handlers**

Middleware: extract `Authorization`, `WithToken`, continue.  
JSON handlers; error mapping same as LOMS.

- [ ] **Step 2: main wiring**

```go
lomsURL := envOr("LOMS_URL", "http://localhost:8083")
product := product.NewStub()
lomsClient := lomsclient.New(lomsURL, http.DefaultClient)
// repo -> service -> handler on :8080
```

- [ ] **Step 3: docker-compose**

Under `cart:` add:

```yaml
environment:
  - LOMS_URL=http://loms:8080
```

- [ ] **Step 4: Build both**

Run: `cd workshop-1 && make build-all`  
Expected: binaries built

- [ ] **Step 5: Commit**

```bash
git add workshop-1/cart workshop-1/docker-compose.yml
git commit -m "feat(cart): HTTP API wiring and compose LOMS_URL"
```

---

### Task 8: precommit green + smoke

**Files:** possibly lint fixes only

- [ ] **Step 1: Run precommit**

Run: `cd workshop-1 && make precommit`  
Fix format/lint/test until OK.

- [ ] **Step 2: Optional local smoke (if docker available)**

Run: `cd workshop-1 && make run-all`  
Then curl examples:

```bash
curl -s -X POST localhost:8083/stock/info -d '{"sku":1001}'
curl -s -X POST localhost:8082/cart/item/add -H 'Authorization: stub-token' -d '{"user":1,"sku":1001,"count":2}'
curl -s -X POST localhost:8082/cart/list -H 'Authorization: stub-token' -d '{"user":1}'
curl -s -X POST localhost:8082/cart/checkout -H 'Authorization: stub-token' -d '{"user":1}'
```

- [ ] **Step 3: Final commit if fixes**

```bash
git add -u workshop-1
git commit -m "fix(workshop-1): make precommit clean"
```

---

## Spec coverage checklist

| Spec item | Task |
|-----------|------|
| LOMS order create/info/pay/cancel + stock/info | 1–3 |
| In-memory stocks/orders + seed | 1–2 |
| Auto-cancel 10m | 3 |
| Cart CRUD + checkout | 4,6,7 |
| Product via stub + token in context | 5,7 |
| Real HTTP cart→loms | 5,7 |
| docker-compose LOMS_URL | 7 |
| make precommit / run-all | 8 |

## Self-review notes

- No live ProductService; stub catalog SKUs aligned with LOMS seed (`1001–1003`).
- Path `/stock/info` (docs), not `/stocks`.
- Cart add checks `existing + requested` vs available.
