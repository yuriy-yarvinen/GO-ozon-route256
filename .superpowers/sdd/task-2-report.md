# Task 2 Report: LOMS order repository + service

## Status

Complete.

## Implemented

- Added an in-memory, mutex-protected order repository with:
  - auto-incrementing order IDs;
  - creation timestamps;
  - `Create`, `Get`, `SetStatus`, and `ListAwaitingOlderThan`;
  - defensive copies of order item slices.
- Added the core LOMS service with:
  - non-empty item validation;
  - order creation and stock reservation;
  - all-or-nothing reservation rollback;
  - awaiting-payment, failed, payed, and cancelled transitions;
  - order and stock lookup.
- Threaded caller-provided `context.Context` through every service/repository request-path method.
- Updated the Task 1 stock repository API and test to accept `context.Context`.

## TDD Evidence

### RED

1. Removed the partial order repository/service drafts and ran:

   `go test ./internal/service/loms/ -v`

   The package failed because `repository/order` and the service implementation were absent.

2. Updated the stock repository test to use the required context-aware API and ran:

   `go test ./internal/repository/stock/ -v`

   Compilation failed with the expected “too many arguments” errors for `GetAvailable` and `Reserve`.

### GREEN

- Implemented the context-aware repositories and service.
- Preserved all three exact brief test cases and values.
- Added tests for empty items, partial reservation rollback, missing orders, and invalid pay transitions.
- Focused service and stock repository tests passed.

## Verification

From `workshop-1/loms`:

- `go test ./internal/service/loms/ -v` — PASS (7 tests).
- `go test ./internal/repository/stock/ -v` — PASS.
- `go test ./...` — PASS for all LOMS packages.
- `git diff --check` — PASS.

## Self-review

- Confirmed no service method substitutes `context.Background()` for the caller context.
- Confirmed all repository calls receive the same caller context.
- Confirmed failed multi-item reservation releases every earlier reservation.
- Confirmed order reads and list results cannot mutate stored item slices.
- Confirmed only Task 2 files and the binding-required stock API/test changes are included in the task commit.

## Concerns

None blocking. In-memory repository methods accept but do not actively inspect cancellation; this is intentional so caller deadlines can be propagated into later external clients.

## Important Review Fixes

- Added an atomic `CompareAndSwapStatus` repository operation. `PayOrder` and `CancelOrder` now claim an awaiting-payment order before changing stock, so only one competing transition can proceed.
- Added status rollback to awaiting payment when a claimed pay/cancel stock operation fails; rollback errors are combined with the stock error.
- `CreateOrder` now attempts every partial-reservation release, checks the failed-status update, and returns combined rollback errors instead of reporting a clean insufficient-stock result when cleanup failed.
- Added a repository test proving a second CAS transition is rejected, plus service tests for release and status rollback failures.

## Review Fix Verification

From `workshop-1/loms`:

- `go test ./internal/service/loms/ -v` — PASS (9 tests).
- `go test ./...` — PASS for all LOMS packages.
- `git diff --check` — PASS.
