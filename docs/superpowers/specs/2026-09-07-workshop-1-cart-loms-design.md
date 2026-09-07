# Workshop-1: Cart + LOMS Design

Date: 2026-09-07  
Scope: homework base + diamond (in-memory storage with real business logic)

## Goals

- Implement `cart` and `loms` skeletons per `workshop-1/docs/README.md`.
- Layered structure; business logic decoupled from HTTP handlers and HTTP clients.
- Inter-service calls actually executed over HTTP+JSON.
- ProductService integration for `cart/item/add` and `cart/list` via a **simple stub client** (no live `http://route256.pavl.uk` calls); auth token taken from request and stored in context (stub may ignore token but wiring must pass it).
- In-memory repositories for cart and loms (diamond).
- `make precommit` and `make run-all` succeed from `workshop-1/`.

Out of scope for this homework: notifications/Kafka, gRPC ProductService client, live `route256.pavl.uk` ProductService (use an in-process stub client instead).

## Architecture

Practical layered layout (chosen over clean-architecture and flat handler+service):

```
cmd/app/main.go
internal/
  config/
  model/
  handler/           # HTTP decode/encode, status mapping
  service/...        # business logic; depends on interfaces
  repository/...     # in-memory stores
  client/...         # outbound HTTP JSON clients
  pkg/ctxkey/        # request-scoped values (auth token)
```

LOMS additionally has `internal/worker/` for unpaid-order auto-cancel.

Interfaces for repositories/clients are declared in the `service` packages (dependency inversion at point of use).

## HTTP API

Protocol: HTTP POST with JSON request/response bodies (not JSON-RPC 2.0).  
Containers listen on `:8080`. Host ports via docker-compose: cart `8082`, loms `8083`.

### Cart

| Method | Path | Behavior |
|--------|------|----------|
| POST | `/cart/item/add` | Validate SKU via ProductService; check stocks via LOMS; add to cart |
| POST | `/cart/item/delete` | Remove SKU from user cart |
| POST | `/cart/list` | List cart items with name/price from ProductService; `totalPrice` |
| POST | `/cart/clear` | Clear user cart |
| POST | `/cart/checkout` | Create order via LOMS; clear cart on success; return `orderID` |

### LOMS

| Method | Path | Behavior |
|--------|------|----------|
| POST | `/order/create` | Create order, reserve stocks |
| POST | `/order/info` | Return order status/user/items |
| POST | `/order/pay` | Mark paid; convert reserved → purchased |
| POST | `/order/cancel` | Cancel; release reserved stocks |
| POST | `/stock/info` | Return available count (`total - reserved`) |

### Error mapping

- `400` — invalid request / validation
- `404` — order/item not found where applicable
- `412` — insufficient stocks / failed precondition (e.g. reserve failure, wrong status transition)

## Data model

### Cart repository

```
userID (int64) → map[sku uint32]count uint16
```

Protected by mutex. Concurrent-safe.

### LOMS stocks repository

```
sku → { total uint64, reserved uint64 }
available = total - reserved
```

Seeded at startup with a fixed small set of SKUs so local/manual checks work without an external DB.

### LOMS orders repository

```
orderID (int64, auto-increment) → {
  user int64
  items []{ sku uint32, count uint16 }
  status string
  createdAt time.Time
}
```

Statuses: `new` | `awaiting payment` | `failed` | `payed` | `cancelled`.

## Business flows

### order/create

1. Persist order with status `new`.
2. Attempt stock reserve for all items.
3. On success → status `awaiting payment`, return `orderID`.
4. On failure → status `failed`, respond `412`.

### order/pay

- Allowed only from `awaiting payment`.
- Decrease `total` and `reserved` by item counts (purchased).
- Status → `payed`.

### order/cancel

- Allowed only from `awaiting payment` (and auto-cancel path).
- Decrease `reserved` only (stocks free again).
- Status → `cancelled`.

### Auto-cancel

Background goroutine periodically finds orders in `awaiting payment` older than 10 minutes and cancels them (same as cancel flow).

### cart/item/add

1. Read auth token from `Authorization` header into context (optional `Bearer ` prefix stripped).
2. Call ProductService `get_product` (token from context put into JSON `token` field) — invalid SKU → error.
3. Call LOMS `/stock/info`; if `existing_in_cart + requested count > available` → `412`.
4. Add/increment item in cart storage.

### cart/list

- Load cart items; for each SKU call ProductService `get_product`.
- Return items with `name`/`price` and `totalPrice`.

### cart/checkout

1. Load cart; call LOMS `/order/create`.
2. On success clear cart and return `orderID`.
3. On LOMS failure propagate error (no clear).

## Clients and config

Env-driven:

- Cart: `LOMS_URL` (default `http://loms:8080` in compose)
- Both services: listen `:8080`

ProductService: **stub client** implementing the same interface as a real HTTP client (`GetProduct(ctx, sku) → name, price`). Behavior:
- reads token from context (must be wired; stub does not call network);
- returns fixed name/price for known seeded SKUs;
- returns not-found error for unknown SKUs.
Cart middleware still reads `Authorization`, stores raw token in context so the call path matches the homework.

LOMS client: thin `POST` wrappers for create/info/pay/cancel/stock info (real HTTP to loms service).

### Context and timeouts

- Propagate `context.Context` through handlers → services → repositories/clients (never drop or replace with `context.Background()` on the request path).
- Outbound HTTP calls (cart → LOMS) use per-request timeout **≤ 500ms** via `context.WithTimeout(parent, 500*time.Millisecond)` (or an `http.Client.Timeout` of 500ms). Do not use longer timeouts.
- Inbound handler: use `r.Context()`; optionally wrap with ≤500ms deadline for the whole handler if no parent deadline is set.

## Wiring

`main.go` constructs: config → repositories → clients → services → handlers → `http.ListenAndServe(":8080", mux)`.

docker-compose: cart `depends_on: loms`; cart gets `LOMS_URL=http://loms:8080`.

## Testing / verification

- Unit tests for service-layer reserve/pay/cancel and cart add validation (interfaces mocked or fake repos).
- `make precommit` (format, build, test, lint) per service.
- `make run-all` builds images and starts compose stack.

## Non-goals

- Real DB / migrations
- Kafka / notifications service behavior
- gRPC ProductService
- Idempotency keys / distributed transactions beyond in-process consistency
