package transactor

import (
	"context"
	"iter"
)

// An Op is an operation that can prepare, commit, and roll-back operations.
// It promises a clean state unless Rollback() returns an error.
// An Op can have Op children, making it a tree structure.
// A Transactor exploits this fact to perform recursive operations.
type Op interface {
	Initialize(...any) error
	Prepare(context.Context) error
	Commit(context.Context) error
	Rollback() error
	Children() iter.Seq[Op]
}
