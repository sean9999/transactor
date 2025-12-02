package transactor

import (
	"context"
	"sync"
)

// raceAll races Op operations, finishing each one and collecting all errors.
func raceAll(ops []Op, fn func(Op) error) []error {

	errChan := make(chan error, len(ops))

	wg := sync.WaitGroup{}
	wg.Add(len(ops))

	for _, op := range ops {
		go func() {
			err := fn(op)
			if err != nil {
				errChan <- err
			}
			wg.Done()
		}()
	}
	wg.Wait()

	//	drain errChan, turning it into an []error (or nil)
	close(errChan)
	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	return errs
}

// raceUntil races Op operations, returning the first error or nil
func raceUntil(ctx context.Context, ops []Op, fn func(context.Context, Op) error) error {
	results := make(chan error, len(ops))
	for _, op := range ops {
		go func() {
			results <- fn(ctx, op)
		}()
	}
	for range len(ops) {
		select {
		case err := <-results:
			if err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

// reap calls a function on Op and all descendents, without stopping.
// it will collect and return any errors encountered along the way.
func reap(op Op, errs []error, fn func(Op) error) []error {
	if errs == nil {
		errs = make([]error, 0)
	}
	err := fn(op)
	if err != nil {
		errs = append(errs, err)
	}
	if op.Children() != nil {
		childErrs := raceAll(op.Children(), fn)
		errs = append(errs, childErrs...)
	}
	return reap(op, errs, fn)
}

// walkFromRoot walks an Op tree, starting from the root and cascading to the leaves.
// It returns early at the first error.
func walkFromRoot(ctx context.Context, op Op, fn func(context.Context, Op) error) error {
	err := fn(ctx, op)
	if err != nil {
		return err
	}
	if op.Children() != nil {
		return raceUntil(ctx, op.Children(), fn)
	}
	return nil
}

// walkFromLeaves walks an Op tree, starting from the leaves and cascading to the root.
// It returns early at the first error.
//func walkFromLeaves(ctx context.Context, op Op, fn func(context.Context, Op) error) error {
//	if op.Children() == nil {
//		return fn(ctx, op)
//	} else {
//		return walkFromLeaves(ctx, op, fn)
//	}
//}

// RunOrCancel runs and returns a synchronous function, unless the context was canceled first
func RunOrCancel(ctx context.Context, fn func() error) error {
	ch := make(chan error)
	go func() {
		ch <- fn()
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-ch:
		return err
	}
}
