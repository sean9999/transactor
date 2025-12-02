package transactor

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

type Transactor struct {
	rootOp Op
}

func NewTransactor(rootOp Op) *Transactor {
	return &Transactor{
		rootOp: rootOp,
	}
}

// prepare the root Op and all descendents
func (tran *Transactor) prepare(ctx context.Context) error {

	fn := func(ctx context.Context, op Op) error {
		return op.Prepare(ctx)
	}

	ch := make(chan error)
	go func() {
		err := walkFromRoot(ctx, tran.rootOp, fn)
		ch <- err
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-ch:
		return err
	}

}

// commit the rootOp and all descendents
func (tran *Transactor) commit(ctx context.Context) error {

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	fn := func(ctx context.Context, op Op) error {
		return op.Commit(ctx)
	}

	ch := make(chan error)
	go func() {
		err := walkFromRoot(ctx, tran.rootOp, fn)
		ch <- err
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-ch:
		return err
	}
}

// rollback rootOp and all descendents.
func (tran *Transactor) rollback() error {

	fn := func(op Op) error {
		return op.Rollback()
	}

	errs := reap(tran.rootOp, nil, fn)
	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// Transact does prepare() and then rollback(), or prepare() and then commit(),
// Or prepare() and then commit() and then rollback(), depending on what happens.
func (tran *Transactor) Transact(ctx context.Context) error {

	ctx = context.WithValue(ctx, "txId", uuid.New())
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	err := tran.prepare(ctx)
	if err != nil {
		//	if prepare fails, roll back
		cancel() // cancel before rollback so that in-flight operations can be killed
		err2 := tran.rollback()
		if err2 != nil {
			return fmt.Errorf("prepare failed: %w. then rollback failed: %w", err, err2)
		}
		return fmt.Errorf("prepare failed: %w. rollback success", err)
	}

	//	prepare worked, let's commit.
	err = tran.commit(ctx)
	if err != nil {
		//	commit didn't work. Roll back.
		cancel() // cancel before rollback so that in-flight operations can be killed
		err2 := tran.rollback()
		if err2 != nil {
			return fmt.Errorf("commit failed: %w. then rollback failed: %w", err, err2)
		}
		return fmt.Errorf("commit failed: %w. rollback success", err)
	}

	//	commit() worked. Our whole thing worked.
	return nil

}
