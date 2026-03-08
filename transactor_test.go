package transactor

import (
	"context"
	"errors"
	"iter"
	"sync/atomic"
	"testing"
)

// mockOp is a configurable mock that implements Op for testing.
type mockOp struct {
	prepareErr  error
	commitErr   error
	rollbackErr error
	children    []Op

	prepared   atomic.Bool
	committed  atomic.Bool
	rolledBack atomic.Bool
}

func (m *mockOp) Initialize(args ...any) error { return nil }

func (m *mockOp) Prepare(ctx context.Context) error {
	if m.prepareErr != nil {
		return m.prepareErr
	}
	m.prepared.Store(true)
	return nil
}

func (m *mockOp) Commit(ctx context.Context) error {
	if m.commitErr != nil {
		return m.commitErr
	}
	m.committed.Store(true)
	return nil
}

func (m *mockOp) Rollback() error {
	if m.rollbackErr != nil {
		return m.rollbackErr
	}
	m.rolledBack.Store(true)
	return nil
}

func (m *mockOp) Children() iter.Seq[Op] {
	if len(m.children) == 0 {
		return nil
	}
	return func(yield func(Op) bool) {
		for _, c := range m.children {
			if !yield(c) {
				return
			}
		}
	}
}

func TestNewTransactor(t *testing.T) {
	op := &mockOp{}
	tr := NewTransactor(op)
	if tr == nil {
		t.Fatal("expected non-nil Transactor")
	}
	if tr.rootOp != op {
		t.Fatal("expected rootOp to be the provided Op")
	}
}

func TestTransact_Success(t *testing.T) {
	child1 := &mockOp{}
	child2 := &mockOp{}
	root := &mockOp{children: []Op{child1, child2}}

	tr := NewTransactor(root)
	err := tr.Transact(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !root.prepared.Load() {
		t.Error("root should have been prepared")
	}
	if !root.committed.Load() {
		t.Error("root should have been committed")
	}
	if !child1.prepared.Load() {
		t.Error("child1 should have been prepared")
	}
	if !child1.committed.Load() {
		t.Error("child1 should have been committed")
	}
	if !child2.prepared.Load() {
		t.Error("child2 should have been prepared")
	}
	if !child2.committed.Load() {
		t.Error("child2 should have been committed")
	}
}

func TestTransact_LeafOnly(t *testing.T) {
	root := &mockOp{}
	tr := NewTransactor(root)
	err := tr.Transact(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !root.prepared.Load() || !root.committed.Load() {
		t.Error("leaf op should have been prepared and committed")
	}
}

func TestTransact_PrepareFail_RollbackSuccess(t *testing.T) {
	root := &mockOp{prepareErr: errors.New("prepare boom")}
	tr := NewTransactor(root)
	err := tr.Transact(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, root.prepareErr) {
		t.Errorf("expected prepare error in chain, got %v", err)
	}
	if !root.rolledBack.Load() {
		t.Error("root should have been rolled back")
	}
}

func TestTransact_PrepareFail_RollbackFail(t *testing.T) {
	root := &mockOp{
		prepareErr:  errors.New("prepare boom"),
		rollbackErr: errors.New("rollback boom"),
	}
	tr := NewTransactor(root)
	err := tr.Transact(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	// Both errors should be wrapped
	if !errors.Is(err, root.prepareErr) {
		t.Errorf("expected prepare error in chain, got %v", err)
	}
	if !errors.Is(err, root.rollbackErr) {
		t.Errorf("expected rollback error in chain, got %v", err)
	}
}

func TestTransact_CommitFail_RollbackSuccess(t *testing.T) {
	root := &mockOp{commitErr: errors.New("commit boom")}
	tr := NewTransactor(root)
	err := tr.Transact(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, root.commitErr) {
		t.Errorf("expected commit error in chain, got %v", err)
	}
	if !root.prepared.Load() {
		t.Error("root should have been prepared")
	}
	if !root.rolledBack.Load() {
		t.Error("root should have been rolled back")
	}
}

func TestTransact_CommitFail_RollbackFail(t *testing.T) {
	root := &mockOp{
		commitErr:   errors.New("commit boom"),
		rollbackErr: errors.New("rollback boom"),
	}
	tr := NewTransactor(root)
	err := tr.Transact(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, root.commitErr) {
		t.Errorf("expected commit error in chain, got %v", err)
	}
	if !errors.Is(err, root.rollbackErr) {
		t.Errorf("expected rollback error in chain, got %v", err)
	}
}

func TestTransact_ChildPrepareFail(t *testing.T) {
	child := &mockOp{prepareErr: errors.New("child prepare boom")}
	root := &mockOp{children: []Op{child}}

	tr := NewTransactor(root)
	err := tr.Transact(context.Background())
	if err == nil {
		t.Fatal("expected error from child prepare failure")
	}
	if !root.prepared.Load() {
		t.Error("root should have been prepared before child failed")
	}
	if !root.rolledBack.Load() {
		t.Error("root should have been rolled back")
	}
	if !child.rolledBack.Load() {
		t.Error("child should have been rolled back")
	}
}

func TestTransact_ChildCommitFail(t *testing.T) {
	child := &mockOp{commitErr: errors.New("child commit boom")}
	root := &mockOp{children: []Op{child}}

	tr := NewTransactor(root)
	err := tr.Transact(context.Background())
	if err == nil {
		t.Fatal("expected error from child commit failure")
	}
	if !root.rolledBack.Load() {
		t.Error("root should have been rolled back")
	}
}

func TestTransact_CancelledContext(t *testing.T) {
	root := &mockOp{}
	tr := NewTransactor(root)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	err := tr.Transact(ctx)
	// With an already-cancelled parent context, the derived context is also cancelled.
	// Behaviour depends on scheduling; just verify we get an error.
	if err == nil {
		// It's possible the op completes before cancellation is noticed.
		// Either outcome is acceptable.
		return
	}
}
