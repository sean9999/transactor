package transactor

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"
)

// --- AsOps tests ---

func TestAsOps_Empty(t *testing.T) {
	seq := AsOps([]*mockOp{})
	collected := slices.Collect(seq)
	if len(collected) != 0 {
		t.Fatalf("expected 0 ops, got %d", len(collected))
	}
}

func TestAsOps_Multiple(t *testing.T) {
	ops := []*mockOp{{}, {}, {}}
	seq := AsOps(ops)
	collected := slices.Collect(seq)
	if len(collected) != 3 {
		t.Fatalf("expected 3 ops, got %d", len(collected))
	}
	for i, op := range collected {
		if op != ops[i] {
			t.Errorf("op at index %d doesn't match", i)
		}
	}
}

func TestAsOps_EarlyBreak(t *testing.T) {
	ops := []*mockOp{{}, {}, {}}
	seq := AsOps(ops)
	count := 0
	for range seq {
		count++
		if count == 2 {
			break
		}
	}
	if count != 2 {
		t.Errorf("expected to iterate 2 times, got %d", count)
	}
}

// --- raceAll tests ---

func TestRaceAll_Empty(t *testing.T) {
	errs := raceAll(nil, func(op Op) error { return nil })
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got %d", len(errs))
	}
}

func TestRaceAll_AllSucceed(t *testing.T) {
	ops := []Op{&mockOp{}, &mockOp{}, &mockOp{}}
	errs := raceAll(ops, func(op Op) error { return nil })
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got %d", len(errs))
	}
}

func TestRaceAll_SomeFail(t *testing.T) {
	boom := errors.New("boom")
	ops := []Op{&mockOp{}, &mockOp{}, &mockOp{}}
	errs := raceAll(ops, func(op Op) error {
		if op == ops[1] {
			return boom
		}
		return nil
	})
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
	if errs[0] != boom {
		t.Errorf("expected boom error, got %v", errs[0])
	}
}

func TestRaceAll_AllFail(t *testing.T) {
	ops := []Op{&mockOp{}, &mockOp{}}
	errs := raceAll(ops, func(op Op) error {
		return errors.New("fail")
	})
	if len(errs) != 2 {
		t.Fatalf("expected 2 errors, got %d", len(errs))
	}
}

// --- raceUntil tests ---

func TestRaceUntil_Empty(t *testing.T) {
	err := raceUntil(context.Background(), nil, func(ctx context.Context, op Op) error {
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestRaceUntil_AllSucceed(t *testing.T) {
	ops := []Op{&mockOp{}, &mockOp{}}
	err := raceUntil(context.Background(), ops, func(ctx context.Context, op Op) error {
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestRaceUntil_FirstError(t *testing.T) {
	boom := errors.New("boom")
	ops := []Op{&mockOp{}, &mockOp{}}
	err := raceUntil(context.Background(), ops, func(ctx context.Context, op Op) error {
		if op == ops[0] {
			return boom
		}
		time.Sleep(50 * time.Millisecond)
		return nil
	})
	if err != boom {
		t.Fatalf("expected boom, got %v", err)
	}
}

func TestRaceUntil_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	ops := []Op{&mockOp{}}
	err := raceUntil(ctx, ops, func(ctx context.Context, op Op) error {
		cancel()
		time.Sleep(50 * time.Millisecond)
		return nil
	})
	// Either context.Canceled or nil is acceptable depending on scheduling
	if err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("expected nil or context.Canceled, got %v", err)
	}
}

// --- reap tests ---

func TestReap_LeafNoError(t *testing.T) {
	op := &mockOp{}
	errs := reap(op, nil, func(o Op) error { return nil })
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got %d", len(errs))
	}
}

func TestReap_LeafWithError(t *testing.T) {
	boom := errors.New("boom")
	op := &mockOp{}
	errs := reap(op, nil, func(o Op) error { return boom })
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
	if errs[0] != boom {
		t.Errorf("expected boom, got %v", errs[0])
	}
}

func TestReap_WithChildren(t *testing.T) {
	child1 := &mockOp{}
	child2 := &mockOp{}
	root := &mockOp{children: []Op{child1, child2}}

	visited := make(map[Op]bool)
	errs := reap(root, nil, func(o Op) error {
		visited[o] = true
		return nil
	})
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got %d", len(errs))
	}
	if !visited[root] || !visited[child1] || !visited[child2] {
		t.Error("expected all nodes to be visited")
	}
}

func TestReap_WithChildren_CollectsErrors(t *testing.T) {
	child1 := &mockOp{}
	child2 := &mockOp{}
	root := &mockOp{children: []Op{child1, child2}}

	errs := reap(root, nil, func(o Op) error {
		return errors.New("err")
	})
	if len(errs) != 3 {
		t.Fatalf("expected 3 errors (root + 2 children), got %d", len(errs))
	}
}

func TestReap_DeepTree(t *testing.T) {
	grandchild := &mockOp{}
	child := &mockOp{children: []Op{grandchild}}
	root := &mockOp{children: []Op{child}}

	visited := make(map[Op]bool)
	errs := reap(root, nil, func(o Op) error {
		visited[o] = true
		return nil
	})
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got %d", len(errs))
	}
	if len(visited) != 3 {
		t.Errorf("expected 3 visited nodes, got %d", len(visited))
	}
}

func TestReap_ExistingErrors(t *testing.T) {
	existing := errors.New("existing")
	op := &mockOp{}
	errs := reap(op, []error{existing}, func(o Op) error { return nil })
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
	if errs[0] != existing {
		t.Errorf("expected existing error preserved")
	}
}

// --- walkFromRoot tests ---

func TestWalkFromRoot_LeafSuccess(t *testing.T) {
	op := &mockOp{}
	err := walkFromRoot(context.Background(), op, func(ctx context.Context, o Op) error {
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestWalkFromRoot_LeafError(t *testing.T) {
	boom := errors.New("boom")
	op := &mockOp{}
	err := walkFromRoot(context.Background(), op, func(ctx context.Context, o Op) error {
		return boom
	})
	if err != boom {
		t.Fatalf("expected boom, got %v", err)
	}
}

func TestWalkFromRoot_WithChildren_Success(t *testing.T) {
	child := &mockOp{}
	root := &mockOp{children: []Op{child}}

	visited := make(map[Op]bool)
	err := walkFromRoot(context.Background(), root, func(ctx context.Context, o Op) error {
		visited[o] = true
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if !visited[root] || !visited[child] {
		t.Error("expected both root and child to be visited")
	}
}

func TestWalkFromRoot_ChildError(t *testing.T) {
	boom := errors.New("child boom")
	child := &mockOp{}
	root := &mockOp{children: []Op{child}}

	err := walkFromRoot(context.Background(), root, func(ctx context.Context, o Op) error {
		if o == child {
			return boom
		}
		return nil
	})
	if err != boom {
		t.Fatalf("expected child boom, got %v", err)
	}
}

// --- RunOrCancel tests ---

func TestRunOrCancel_Success(t *testing.T) {
	err := RunOrCancel(context.Background(), func() error {
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestRunOrCancel_Error(t *testing.T) {
	boom := errors.New("boom")
	err := RunOrCancel(context.Background(), func() error {
		return boom
	})
	if err != boom {
		t.Fatalf("expected boom, got %v", err)
	}
}

func TestRunOrCancel_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := RunOrCancel(ctx, func() error {
		time.Sleep(100 * time.Millisecond)
		return nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}
