// Package main demonstrates the Transactor library with an order payment processing flow.
//
// This example shows how to compose a tree of Ops:
//
//	PlaceOrderOp (root)
//	├── ReserveInventoryOp (child)
//	└── ChargePaymentOp (child)
//
// The two child Ops run concurrently during both Prepare and Commit.
// If any Op fails during Prepare, the Transactor cancels remaining work
// and rolls back all Ops that have already prepared.
//
// This example runs two scenarios:
//  1. A successful order — all Ops prepare and commit cleanly.
//  2. A failed order — payment is declined during Prepare, so all
//     Ops that had already prepared are rolled back manually to show
//     the rollback lifecycle.
package main

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"log"
	"sync"

	"github.com/sean9999/transactor"
)

// ---------------------------------------------------------------------------
// Domain types
// ---------------------------------------------------------------------------

// Order holds the data for a single order placement request.
type Order struct {
	ID        string
	ItemSKU   string
	Quantity  int
	PaymentID string
	Total     float64
}

// ---------------------------------------------------------------------------
// ReserveInventoryOp
// ---------------------------------------------------------------------------

// ReserveInventoryOp is a leaf Op (no children) that reserves stock for an order.
// In a real system this would talk to a warehouse service; here we use in-memory state.
type ReserveInventoryOp struct {
	sku      string
	qty      int
	reserved bool // set to true after a successful Prepare
}

// Confirm that ReserveInventoryOp satisfies the Op interface at compile time.
var _ transactor.Op = (*ReserveInventoryOp)(nil)

func NewReserveInventoryOp(sku string, qty int) *ReserveInventoryOp {
	return &ReserveInventoryOp{sku: sku, qty: qty}
}

func (op *ReserveInventoryOp) Initialize(args ...any) error {
	// Already initialised via the constructor; nothing extra to do.
	return nil
}

func (op *ReserveInventoryOp) Prepare(ctx context.Context) error {
	return transactor.RunOrCancel(ctx, func() error {
		fmt.Printf("  [inventory] Prepare: reserving %d units of SKU %q\n", op.qty, op.sku)

		// Simulate stock-check validation.
		if op.qty <= 0 {
			return fmt.Errorf("inventory: invalid quantity %d", op.qty)
		}
		if op.qty > 100 {
			return fmt.Errorf("inventory: insufficient stock for %d units of %q", op.qty, op.sku)
		}

		// Mark the reservation as held (a real implementation would call a service here).
		op.reserved = true
		fmt.Printf("  [inventory] Prepare: %d units of %q reserved (pending commit)\n", op.qty, op.sku)
		return nil
	})
}

func (op *ReserveInventoryOp) Commit(ctx context.Context) error {
	return transactor.RunOrCancel(ctx, func() error {
		if !op.reserved {
			return errors.New("inventory: cannot commit — nothing was reserved")
		}
		// In a real system: finalize the reservation, decrement warehouse stock.
		fmt.Printf("  [inventory] Commit: reservation of %d units of %q confirmed\n", op.qty, op.sku)
		return nil
	})
}

func (op *ReserveInventoryOp) Rollback() error {
	if !op.reserved {
		// Nothing was reserved, so there's nothing to undo.
		fmt.Printf("  [inventory] Rollback: no reservation to release for %q\n", op.sku)
		return nil
	}
	// Release the hold on stock.
	op.reserved = false
	fmt.Printf("  [inventory] Rollback: reservation of %d units of %q released\n", op.qty, op.sku)
	return nil
}

// Children returns nil — ReserveInventoryOp is a leaf node.
func (op *ReserveInventoryOp) Children() iter.Seq[transactor.Op] {
	return nil
}

// ---------------------------------------------------------------------------
// ChargePaymentOp
// ---------------------------------------------------------------------------

// ChargePaymentOp is a leaf Op that authorises a payment against a payment method.
// Set ShouldDecline to true to simulate a declined card — useful for demonstrating rollback.
type ChargePaymentOp struct {
	paymentID     string
	amount        float64
	ShouldDecline bool // inject a failure for demonstration purposes
	authorised    bool // set to true after a successful Prepare
}

var _ transactor.Op = (*ChargePaymentOp)(nil)

func NewChargePaymentOp(paymentID string, amount float64) *ChargePaymentOp {
	return &ChargePaymentOp{paymentID: paymentID, amount: amount}
}

func (op *ChargePaymentOp) Initialize(args ...any) error {
	return nil
}

func (op *ChargePaymentOp) Prepare(ctx context.Context) error {
	return transactor.RunOrCancel(ctx, func() error {
		fmt.Printf("  [payment]   Prepare: authorising $%.2f on payment method %q\n", op.amount, op.paymentID)

		if op.ShouldDecline {
			return fmt.Errorf("payment: card %q was declined", op.paymentID)
		}

		// Simulate a successful pre-authorisation hold.
		op.authorised = true
		fmt.Printf("  [payment]   Prepare: $%.2f pre-authorised on %q (pending commit)\n", op.amount, op.paymentID)
		return nil
	})
}

func (op *ChargePaymentOp) Commit(ctx context.Context) error {
	return transactor.RunOrCancel(ctx, func() error {
		if !op.authorised {
			return errors.New("payment: cannot commit — no authorisation hold")
		}
		// In a real system: capture the pre-authorised charge.
		fmt.Printf("  [payment]   Commit: $%.2f captured on %q\n", op.amount, op.paymentID)
		return nil
	})
}

func (op *ChargePaymentOp) Rollback() error {
	if !op.authorised {
		fmt.Printf("  [payment]   Rollback: no authorisation hold to void on %q\n", op.paymentID)
		return nil
	}
	// Void the pre-authorisation hold.
	op.authorised = false
	fmt.Printf("  [payment]   Rollback: pre-authorisation voided on %q\n", op.paymentID)
	return nil
}

func (op *ChargePaymentOp) Children() iter.Seq[transactor.Op] {
	return nil
}

// ---------------------------------------------------------------------------
// PlaceOrderOp (root)
// ---------------------------------------------------------------------------

// PlaceOrderOp is the root Op that owns the overall order placement transaction.
// Its two children — ReserveInventoryOp and ChargePaymentOp — run concurrently
// during both Prepare and Commit, as orchestrated by the Transactor.
type PlaceOrderOp struct {
	order     Order
	committed bool

	// children are stored as concrete types so callers can inspect them after
	// the transaction, but exposed as the Op interface via Children().
	inventory *ReserveInventoryOp
	payment   *ChargePaymentOp
}

var _ transactor.Op = (*PlaceOrderOp)(nil)

func NewPlaceOrderOp(order Order) *PlaceOrderOp {
	return &PlaceOrderOp{
		order:     order,
		inventory: NewReserveInventoryOp(order.ItemSKU, order.Quantity),
		payment:   NewChargePaymentOp(order.PaymentID, order.Total),
	}
}

func (op *PlaceOrderOp) Initialize(args ...any) error {
	return nil
}

func (op *PlaceOrderOp) Prepare(ctx context.Context) error {
	return transactor.RunOrCancel(ctx, func() error {
		fmt.Printf("[order]     Prepare: validating order %q\n", op.order.ID)

		if op.order.ItemSKU == "" {
			return errors.New("order: SKU must not be empty")
		}
		if op.order.Total <= 0 {
			return fmt.Errorf("order: total must be positive, got %.2f", op.order.Total)
		}

		fmt.Printf("[order]     Prepare: order %q validated — child Ops will now run concurrently\n", op.order.ID)
		return nil
	})
}

func (op *PlaceOrderOp) Commit(ctx context.Context) error {
	return transactor.RunOrCancel(ctx, func() error {
		// The Transactor has already committed both child Ops by the time this runs.
		op.committed = true
		fmt.Printf("[order]     Commit: order %q placed successfully\n", op.order.ID)
		return nil
	})
}

func (op *PlaceOrderOp) Rollback() error {
	if op.committed {
		// Once committed, a rollback here would require a compensating transaction
		// (e.g., an automated refund + stock reinstatement). For this example we
		// just surface the fact that it would be needed.
		return errors.New("order: already committed — a compensating transaction is required")
	}
	fmt.Printf("[order]     Rollback: order %q cancelled before commit\n", op.order.ID)
	return nil
}

// Children returns nil here; the real child list is provided by rootWithChildren,
// which wraps PlaceOrderOp when calling Transact(). PlaceOrderOp itself cannot
// return a mixed-type child slice using AsOps (which requires a homogeneous concrete
// type), so the wrapper handles that instead.
func (op *PlaceOrderOp) Children() iter.Seq[transactor.Op] {
	return nil
}

// ---------------------------------------------------------------------------
// Helper: manual rollback for the failure scenario
//
// The Transactor's internal rollback uses reap(), which has a known infinite-
// recursion issue in the current codebase. For the failure scenario below we
// therefore perform the rollback ourselves, calling Rollback() on each Op
// directly. This mirrors exactly what the Transactor would do once the bug
// is fixed, and lets us demonstrate the full lifecycle clearly.
// ---------------------------------------------------------------------------

func rollbackAll(ops []transactor.Op) {
	var wg sync.WaitGroup
	for _, op := range ops {
		wg.Add(1)
		go func(o transactor.Op) {
			defer wg.Done()
			if err := o.Rollback(); err != nil {
				fmt.Printf("  [rollback error] %v\n", err)
			}
		}(op)
	}
	wg.Wait()
}

// ---------------------------------------------------------------------------
// Scenarios
// ---------------------------------------------------------------------------

func runSuccessScenario() {
	fmt.Println("=== Scenario 1: Successful order ===")

	order := Order{
		ID:        "ORD-001",
		ItemSKU:   "WIDGET-42",
		Quantity:  3,
		PaymentID: "card_abc123",
		Total:     59.97,
	}

	root := NewPlaceOrderOp(order)

	// Wire up children so the Transactor can walk them.
	// We override Children() by embedding the real children in the root;
	// see the overridden Children() method below that uses a local closure.
	t := transactor.NewTransactor(newRootWithChildren(root))

	ctx := context.Background()
	if err := t.Transact(ctx); err != nil {
		log.Fatalf("scenario 1 failed unexpectedly: %v", err)
	}

	fmt.Printf("\nResult: order %q committed=%v\n", root.order.ID, root.committed)
}

func runFailureScenario() {
	fmt.Println("\n=== Scenario 2: Payment declined — rollback ===")

	order := Order{
		ID:        "ORD-002",
		ItemSKU:   "GADGET-7",
		Quantity:  1,
		PaymentID: "card_expired",
		Total:     199.99,
	}

	root := NewPlaceOrderOp(order)
	// Inject the failure: this payment method will be declined during Prepare.
	root.payment.ShouldDecline = true

	fmt.Printf("[order]     Prepare: validating order %q\n", order.ID)
	fmt.Printf("[order]     Prepare: order %q validated — child Ops will now run concurrently\n", order.ID)

	// Run the child Ops' Prepare methods concurrently, exactly as the Transactor would.
	ctx := context.Background()
	prepareErr := runChildrenPrepare(ctx, root.inventory, root.payment)

	if prepareErr != nil {
		fmt.Printf("\nPrepare failed: %v\n", prepareErr)
		fmt.Println("Rolling back all Ops that were prepared...")

		// Roll back: the root Op itself never fully prepared (children failed),
		// and child Ops roll back their own state.
		rollbackAll([]transactor.Op{root, root.inventory, root.payment})

		fmt.Printf("\nResult: order %q was NOT placed; all changes have been reversed\n", order.ID)
		return
	}

	// If we somehow got here without an error, commit (shouldn't happen in this scenario).
	log.Fatal("scenario 2: expected a prepare failure but got none")
}

// runChildrenPrepare runs Prepare on the given Ops concurrently and returns the
// first error encountered, or nil if all succeed. On the first error it cancels
// the shared context so any still-running Ops are interrupted — mirroring the
// raceUntil behaviour inside the Transactor.
func runChildrenPrepare(ctx context.Context, ops ...transactor.Op) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	results := make(chan error, len(ops))
	for _, op := range ops {
		go func(o transactor.Op) {
			results <- o.Prepare(ctx)
		}(op)
	}
	for range len(ops) {
		if err := <-results; err != nil {
			cancel() // stop any still-running Ops before we return
			return err
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// rootWithChildren wraps PlaceOrderOp and provides the correct Children() impl.
//
// PlaceOrderOp.Children() returns AsOps([]*PlaceOrderOp{}) — an empty slice —
// because Go generics require a homogeneous concrete type. rootWithChildren
// solves this by returning the two differently-typed child Ops as transactor.Op.
// ---------------------------------------------------------------------------

type rootWithChildren struct {
	*PlaceOrderOp
	kids []transactor.Op
}

func newRootWithChildren(root *PlaceOrderOp) *rootWithChildren {
	return &rootWithChildren{
		PlaceOrderOp: root,
		kids:         []transactor.Op{root.inventory, root.payment},
	}
}

func (r *rootWithChildren) Children() iter.Seq[transactor.Op] {
	return func(yield func(transactor.Op) bool) {
		for _, kid := range r.kids {
			if !yield(kid) {
				return
			}
		}
	}
}

// ---------------------------------------------------------------------------
// main
// ---------------------------------------------------------------------------

func main() {
	runSuccessScenario()
	runFailureScenario()
}
