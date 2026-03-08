package main

import (
	"context"
	"fmt"
	"iter"
	"time"

	"github.com/sean9999/transactor"
)

// IngredientOp is a child Op representing a single ingredient being sourced
// for the cursed potion. It has no children of its own.
var _ transactor.Op = (*IngredientOp)(nil)

type IngredientOp struct {
	name     string
	symbol   string
	color    string
	sourced  bool
	spoiled  bool // if true, Prepare will fail
	prepared bool
	committed bool
	rolledBack bool
}

func NewIngredient(name, symbol, color string, spoiled bool) *IngredientOp {
	return &IngredientOp{
		name:    name,
		symbol:  symbol,
		color:   color,
		spoiled: spoiled,
	}
}

func (ing *IngredientOp) Initialize(args ...any) error {
	// Nothing to initialize — all config comes from the constructor.
	return nil
}

func (ing *IngredientOp) Prepare(ctx context.Context) error {
	// Simulate sourcing the ingredient from the cursed ingredient market.
	time.Sleep(40 * time.Millisecond)

	if ing.spoiled {
		logf(colorRed, "  %s  SOURCING FAILED: %s is rotten and unusable! The whole batch is cursed.", ing.symbol, ing.name)
		return fmt.Errorf("ingredient %q is spoiled", ing.name)
	}

	ing.prepared = true
	logf(ing.color, "  %s  Sourced: %-28s [PREPARED]", ing.symbol, ing.name)
	return nil
}

func (ing *IngredientOp) Commit(ctx context.Context) error {
	if !ing.prepared {
		return fmt.Errorf("cannot commit %q — it was never prepared", ing.name)
	}
	ing.committed = true
	logf(ing.color, "  %s  Added to cauldron: %-22s [COMMITTED]", ing.symbol, ing.name)
	return nil
}

func (ing *IngredientOp) Rollback() error {
	ing.rolledBack = true
	ing.prepared = false
	ing.committed = false
	logf(colorYellow, "  %s  Discarded: %-26s [ROLLED BACK]", ing.symbol, ing.name)
	return nil
}

// Children returns nil — ingredients are leaf nodes in the Op tree.
func (ing *IngredientOp) Children() iter.Seq[transactor.Op] {
	return nil
}
