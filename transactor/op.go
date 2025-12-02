package transactor

import (
	"context"
)

// An Op is an operation that can prepare, commit, and roll-back operations.
// It promises a clean state unless Rollback() returns an error.
// An Op can have children. A Transactor may exploit this to perform recursive operations.
type Op interface {
	Prepare(context.Context) error
	Commit(context.Context) error
	Rollback() error
	Revert() error // undo an already committed operation. should be rare
	PrepareChildren() error
	CommitChildren() error
	RollbackChildren() error
}
