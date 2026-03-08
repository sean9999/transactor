# orderpayment

Demonstrates the Transactor library with an order payment processing flow.

## Overview

Placing an order requires two things to succeed atomically: stock must be
reserved and a payment must be authorised. If either step fails, neither
should take effect. This example models that with a tree of three Ops:

```
PlaceOrderOp        (root — validates the order itself)
├── ReserveInventoryOp  (leaf — holds stock for the order)
└── ChargePaymentOp     (leaf — pre-authorises the payment)
```

The Transactor orchestrates the tree: it calls Prepare root-to-leaves
(children run concurrently), then Commit root-to-leaves if every Prepare
succeeded, or triggers Rollback if any Prepare failed.

## What You'll Learn

- How to implement the full `Op` interface: `Initialize`, `Prepare`, `Commit`,
  `Rollback`, and `Children`.
- How to compose a root Op with multiple concurrent child Ops using
  `transactor.AsOps` and a custom `Children()` implementation.
- How the Transactor's prepare-then-commit (or prepare-then-rollback) lifecycle
  maps to a real-world multi-step transaction.
- How `transactor.RunOrCancel` lets an Op respect context cancellation.
- How child Ops with different concrete types can be returned from `Children()`
  by wrapping the root in a thin adapter struct.

## How to Run

```sh
go run ./examples/orderpayment
```

## Expected Output

```
=== Scenario 1: Successful order ===
[order]     Prepare: validating order "ORD-001"
[order]     Prepare: order "ORD-001" validated — child Ops will now run concurrently
  [inventory] Prepare: reserving 3 units of SKU "WIDGET-42"
  [inventory] Prepare: 3 units of "WIDGET-42" reserved (pending commit)
  [payment]   Prepare: authorising $59.97 on payment method "card_abc123"
  [payment]   Prepare: $59.97 pre-authorised on "card_abc123" (pending commit)
[order]     Commit: order "ORD-001" placed successfully
  [payment]   Commit: $59.97 captured on "card_abc123"
  [inventory] Commit: reservation of 3 units of "WIDGET-42" confirmed

Result: order "ORD-001" committed=true

=== Scenario 2: Payment declined — rollback ===
[order]     Prepare: validating order "ORD-002"
[order]     Prepare: order "ORD-002" validated — child Ops will now run concurrently
  [payment]   Prepare: authorising $199.99 on payment method "card_expired"
  [inventory] Prepare: reserving 1 units of SKU "GADGET-7"
  [inventory] Prepare: 1 units of "GADGET-7" reserved (pending commit)

Prepare failed: payment: card "card_expired" was declined
Rolling back all Ops that were prepared...
  [payment]   Rollback: no authorisation hold to void on "card_expired"
  [inventory] Rollback: reservation of 1 units of "GADGET-7" released
[order]     Rollback: order "ORD-002" cancelled before commit

Result: order "ORD-002" was NOT placed; all changes have been reversed
```

The ordering of concurrent lines (inventory vs. payment) may vary between runs.

## Code Walkthrough

### Op interface implementation

Each Op struct carries the state it needs to do its work:

```go
type ReserveInventoryOp struct {
    sku      string
    qty      int
    reserved bool // set to true by Prepare; cleared by Rollback
}
```

`Prepare` validates inputs and records intent (sets `reserved = true`).
`Commit` finalises that intent (in a real system: decrement warehouse stock).
`Rollback` undoes whatever `Prepare` did (release the hold).

### Concurrent children

`PlaceOrderOp` owns the two child Ops. The Transactor calls `Children()` to
discover them and then fans out Prepare and Commit calls concurrently using
`raceUntil` internally. The example surfaces this with a `rootWithChildren`
adapter that returns both children as `iter.Seq[transactor.Op]`:

```go
func (r *rootWithChildren) Children() iter.Seq[transactor.Op] {
    return func(yield func(transactor.Op) bool) {
        for _, kid := range r.kids {
            if !yield(kid) {
                return
            }
        }
    }
}
```

The adapter is needed because `transactor.AsOps[T]` requires a homogeneous
concrete slice type, and `ReserveInventoryOp` and `ChargePaymentOp` are
different types.

### Context cancellation

Every blocking step is wrapped in `transactor.RunOrCancel(ctx, fn)`. If the
context is cancelled (because a sibling Op failed), the Op stops immediately
rather than doing work that will only be rolled back.

### Rollback scenario

Scenario 2 injects a failure by setting `payment.ShouldDecline = true`. The
`runChildrenPrepare` helper cancels its own derived context as soon as the
first error arrives, interrupting any sibling Ops still in flight. Then
`rollbackAll` calls `Rollback()` concurrently on all three Ops (root,
inventory, payment), each of which inspects its own state to decide what to
undo.

> **Note on `reap`:** The Transactor's internal `rollback()` method uses a
> `reap()` helper that currently has an infinite-recursion bug in the
> codebase. Scenario 2 therefore performs the rollback manually to keep the
> example runnable. The rollback logic is identical to what `reap` would do
> once that bug is fixed.
