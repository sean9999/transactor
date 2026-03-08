// Package main demonstrates the Transactor library with a cursed potion brewing scenario.
//
// The BrewOp is the root Op (the overall ritual). IngredientOps are child Ops
// (individual ingredients sourced concurrently). A successful brew goes through
// prepare -> commit. A spoiled ingredient causes prepare to fail.
//
// NOTE: The library's internal reap() function has an infinite recursion bug that
// causes a stack overflow whenever Transact() would normally trigger a rollback.
// This example demonstrates both paths: the happy-path success via Transact(), and
// the rollback concept via a manually-orchestrated failure scenario that calls
// Rollback() directly on each Op, showing exactly what the Transactor would do.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/sean9999/transactor"
)

// ANSI color codes
const (
	colorReset   = "\033[0m"
	colorRed     = "\033[31m"
	colorGreen   = "\033[32m"
	colorYellow  = "\033[33m"
	colorBlue    = "\033[34m"
	colorMagenta = "\033[35m"
	colorCyan    = "\033[36m"
	colorWhite   = "\033[37m"
	colorBold    = "\033[1m"
	colorDim     = "\033[2m"

	bgRed    = "\033[41m"
	bgGreen  = "\033[42m"
	bgYellow = "\033[43m"
)

func logf(color, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stdout, "%s%s%s\n", color, msg, colorReset)
}

func banner(text string) {
	width := 60
	line := ""
	for range width {
		line += "~"
	}
	fmt.Println()
	logf(colorBold+colorMagenta, line)
	logf(colorBold+colorMagenta, "  %s", text)
	logf(colorBold+colorMagenta, line)
	fmt.Println()
}

func phase(label string) {
	fmt.Println()
	logf(colorBold+colorCyan, "  >>> %s <<<", label)
	fmt.Println()
	time.Sleep(80 * time.Millisecond) // dramatic pause
}

func successBox(msg string) {
	logf(colorBold+colorGreen, "")
	logf(colorBold+colorGreen, "  +--------------------------------------------------+")
	logf(colorBold+colorGreen, "  |  *** %s ***", msg)
	logf(colorBold+colorGreen, "  +--------------------------------------------------+")
	logf(colorBold+colorGreen, "")
}

func failBox(msg string) {
	logf(colorBold+colorRed, "")
	logf(colorBold+colorRed, "  +--------------------------------------------------+")
	logf(colorBold+colorRed, "  |  !!! %s !!!", msg)
	logf(colorBold+colorRed, "  +--------------------------------------------------+")
	logf(colorBold+colorRed, "")
}

func rollbackBox(msg string) {
	logf(colorBold+colorYellow, "")
	logf(colorBold+colorYellow, "  +--------------------------------------------------+")
	logf(colorBold+colorYellow, "  |  ~~~ %s ~~~", msg)
	logf(colorBold+colorYellow, "  +--------------------------------------------------+")
	logf(colorBold+colorYellow, "")
}

// manualRollback rolls back a BrewOp and all its prepared ingredients.
// This replicates what Transactor.rollback() would do, worked around the
// reap() infinite recursion bug in util.go.
func manualRollback(brew *BrewOp) {
	rollbackBox("PREPARE FAILED — INITIATING ROLLBACK SEQUENCE")
	phase("ROLLBACK: Discarding ingredients")

	// Roll back the children first (leaves -> root is the safe order).
	for _, ing := range brew.ingredients {
		if ing.prepared {
			_ = ing.Rollback()
		} else {
			logf(colorDim, "  %s  Skipping: %-28s (was never prepared)", ing.symbol, ing.name)
		}
	}

	phase("ROLLBACK: Undoing the ritual")
	if brew.prepared {
		_ = brew.Rollback()
	}
}

func main() {
	banner("MADAME HEXWHISTLE'S TRANSACTIONAL ALCHEMY LAB")
	logf(colorDim, "  \"If it doesn't stack overflow, it wasn't cursed enough.\"")
	fmt.Println()

	// =========================================================
	// SCENARIO 1: THE HAPPY PATH — A PERFECTLY BREWED ELIXIR
	// =========================================================

	banner("SCENARIO 1: Brewing the Elixir of Dubious Sentience")

	goodIngredients := []*IngredientOp{
		NewIngredient("Powdered Moonsnail",          "())", colorBlue,    false),
		NewIngredient("Tears of a Confused Wizard",  "T_T", colorCyan,    false),
		NewIngredient("3x Dried Bat Opinions",       "^^^", colorMagenta, false),
		NewIngredient("Cursed Oat Milk (Organic)",   "~~~", colorGreen,   false),
		NewIngredient("One (1) Politician's Promise","...", colorWhite,   false),
	}

	brew1 := NewBrewOp("Elixir of Dubious Sentience", goodIngredients...)

	phase("PHASE 1 — PREPARE: Root ritual + concurrent ingredient sourcing")
	logf(colorDim, "  (The Transactor prepares the root Op first, then all children concurrently.)")
	fmt.Println()

	t1 := transactor.NewTransactor(brew1)
	ctx := context.Background()

	err := t1.Transact(ctx)
	if err != nil {
		// Should not happen in scenario 1
		failBox("UNEXPECTED FAILURE: " + err.Error())
		return
	}

	phase("PHASE 2 — COMMIT: Sealing the brew")

	successBox("ELIXIR OF DUBIOUS SENTIENCE: SUCCESSFULLY BREWED")
	logf(colorGreen, "  Potion effects: may cause recipient to suddenly have opinions")
	logf(colorGreen, "  about jazz. Side effects include mild omniscience and dry mouth.")
	fmt.Println()

	// =========================================================
	// SCENARIO 2: THE FAILURE PATH — A SPOILED INGREDIENT
	// =========================================================

	banner("SCENARIO 2: Brewing the Draught of Infinite Regret (DOOMED)")

	// This batch has a spoiled ingredient that will cause Prepare to fail.
	badIngredients := []*IngredientOp{
		NewIngredient("Crystallized Existential Dread", "***", colorBlue,    false),
		NewIngredient("Rotten Void Mushroom",           "OOO", colorRed,     true),  // <-- SPOILED
		NewIngredient("Fine Ennui Powder",              "---", colorMagenta, false),
	}

	brew2 := NewBrewOp("Draught of Infinite Regret", badIngredients...)

	phase("PHASE 1 — PREPARE: Root ritual + concurrent ingredient sourcing")
	logf(colorDim, "  (One ingredient is spoiled. Prepare will fail. Rollback must follow.)")
	fmt.Println()

	// Manually run the prepare phase to demonstrate what happens.
	// We do NOT call Transact() here because the library's reap() has an
	// infinite recursion bug that would cause a stack overflow on rollback.
	// Instead we wire up the same lifecycle by hand.

	// Step 1: Prepare the root Op.
	ctx2 := context.Background()
	prepErr := brew2.Prepare(ctx2)
	if prepErr != nil {
		failBox("ROOT PREPARE FAILED (unexpected): " + prepErr.Error())
		return
	}

	// Step 2: Prepare children concurrently — simulate what raceUntil does.
	// We collect results and check for failures.
	type result struct {
		ing *IngredientOp
		err error
	}
	results := make(chan result, len(badIngredients))
	for _, ing := range badIngredients {
		go func(i *IngredientOp) {
			err := i.Prepare(ctx2)
			results <- result{ing: i, err: err}
		}(ing)
	}

	var prepareErr error
	for range badIngredients {
		r := <-results
		if r.err != nil && prepareErr == nil {
			prepareErr = r.err
		}
	}

	if prepareErr != nil {
		fmt.Println()
		failBox("PREPARE FAILED: " + prepareErr.Error())

		// Step 3: Rollback everything that was prepared.
		manualRollback(brew2)

		rollbackBox("ROLLBACK COMPLETE — LAB SAFELY RESTORED")
		logf(colorYellow, "  The Rotten Void Mushroom has been composted.")
		logf(colorYellow, "  Madame Hexwhistle will try again on the next blood moon.")
		fmt.Println()
	} else {
		// This branch shouldn't be reached in this scenario.
		failBox("Expected a failure but everything succeeded — the mushroom lied to us.")
	}

	// =========================================================
	// FINAL SUMMARY
	// =========================================================

	banner("LAB REPORT")
	logf(colorGreen,  "  Scenario 1 — Elixir of Dubious Sentience:  BREWED  (prepare + commit)")
	logf(colorYellow, "  Scenario 2 — Draught of Infinite Regret:   ABORTED (prepare failed + rollback)")
	fmt.Println()
	logf(colorDim, "  Op tree structure demonstrated:")
	logf(colorDim, "    BrewOp (root)")
	logf(colorDim, "    +-- IngredientOp (child, leaf)")
	logf(colorDim, "    +-- IngredientOp (child, leaf)")
	logf(colorDim, "    +-- IngredientOp (child, leaf)")
	logf(colorDim, "    ...")
	fmt.Println()
	logf(colorDim, "  Lifecycle: Initialize -> Prepare (root-first) -> Commit or Rollback")
	logf(colorDim, "  Concurrency: child Ops prepared concurrently via goroutines")
	fmt.Println()
}
