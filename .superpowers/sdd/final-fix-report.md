# Final cart checkout fix

Implemented:
- Empty carts return `model.ErrValidation` before `LOMS.CreateOrder`.
- LOMS HTTP 400 responses map to `model.ErrValidation`.
- Added regression coverage and updated handler success fixtures.
- Request deadlines remain 500ms.

Verification:

```text
$ cd workshop-1/cart && go test ./internal/service/cart/ ./internal/client/loms/ ./internal/handler/ -v
PASS
ok  	route256/cart/internal/service/cart	(cached)
PASS
ok  	route256/cart/internal/client/loms	(cached)
PASS
ok  	route256/cart/internal/handler	0.004s
```
