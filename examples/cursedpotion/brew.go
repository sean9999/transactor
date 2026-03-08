package main

import (
	"context"
	"fmt"
	"iter"
	"time"

	"github.com/sean9999/transactor"
)

// BrewOp is the root Op representing the entire potion-brewing ritual.
// It holds IngredientOps as children, which are prepared concurrently.
var _ transactor.Op = (*BrewOp)(nil)

type BrewOp struct {
	potionName  string
	ingredients []*IngredientOp
	prepared    bool
	committed   bool
	rolledBack  bool
}

func NewBrewOp(potionName string, ingredients ...*IngredientOp) *BrewOp {
	return &BrewOp{
		potionName:  potionName,
		ingredients: ingredients,
	}
}

func (b *BrewOp) Initialize(args ...any) error {
	// Nothing to initialize dynamically — constructed with all needed data.
	return nil
}

func (b *BrewOp) Prepare(ctx context.Context) error {
	// The root ritual: inscribe the summoning circle, light the candles,
	// whisper the forbidden words. This runs BEFORE the children are prepared.
	time.Sleep(60 * time.Millisecond)
	logf(colorMagenta, "  @@  Inscribing summoning circle for: %s", b.potionName)
	logf(colorMagenta, "  @@  Candles: lit. Chanting: begun. Cats: suspicious.")
	b.prepared = true
	return nil
}

func (b *BrewOp) Commit(ctx context.Context) error {
	if !b.prepared {
		return fmt.Errorf("cannot commit brew — ritual circle was never inscribed")
	}
	// The final act: stir three times counter-clockwise, add a dramatic monologue.
	time.Sleep(30 * time.Millisecond)
	logf(colorCyan, "  @@  Stirring counter-clockwise (x3)...")
	logf(colorCyan, "  @@  Potion sealed. %s is COMPLETE.", b.potionName)
	b.committed = true
	return nil
}

func (b *BrewOp) Rollback() error {
	b.rolledBack = true
	b.prepared = false
	b.committed = false
	logf(colorYellow, "  @@  Summoning circle erased. Candles extinguished. Cats unimpressed.")
	logf(colorYellow, "  @@  Ritual aborted: %s", b.potionName)
	return nil
}

// Children returns the ingredient Ops as a sequence.
// The Transactor will prepare all of them concurrently after the root is prepared.
func (b *BrewOp) Children() iter.Seq[transactor.Op] {
	return transactor.AsOps(b.ingredients)
}
