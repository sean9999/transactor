# Cursed Potion: Transactional Alchemy

This example demonstrates the Transactor prepare/commit/rollback lifecycle through the lens of a
cursed potion brewing operation conducted by Madame Hexwhistle, a very reputable alchemist.

## Overview

A `BrewOp` is the root Op representing the overall brewing ritual. It holds several `IngredientOp`
children, each representing a single ingredient being sourced from the cursed ingredient market.
The Transactor prepares the root Op first (inscribing the summoning circle), then prepares all
child Ops concurrently (sourcing ingredients in parallel). If all succeeds, it commits the whole
tree. If any ingredient is spoiled, the prepare phase fails and everything is rolled back.

Two scenarios run back-to-back:

- **Scenario 1 (happy path):** All ingredients are fresh. The brew succeeds via `Transact()`.
- **Scenario 2 (failure path):** The Rotten Void Mushroom is spoiled. Prepare fails, and the
  rollback sequence fires to restore the lab to a clean state.

## What You'll Learn

- How to implement the full `Op` interface: `Initialize`, `Prepare`, `Commit`, `Rollback`, `Children`
- How the Transactor walks the Op tree root-first, preparing parent before children
- How child Ops are prepared concurrently (via goroutines in `raceUntil`)
- How commit follows the same root-first tree walk after successful prepare
- How rollback fires when prepare fails, cleaning up anything that was prepared
- How `transactor.AsOps()` converts a typed slice into an `iter.Seq[Op]`
- How `transactor.RunOrCancel()` integrates context cancellation with synchronous functions

> **Note on `reap()` bug:** The library's internal `reap()` function in `util.go` has an infinite
> recursion bug — its last line unconditionally calls itself, causing a stack overflow on any
> rollback triggered by `Transact()`. Scenario 2 therefore demonstrates rollback by manually
> calling `Rollback()` on each Op in sequence, which is exactly what `reap()` would do if it
> worked. This is a known issue in the library; the fix is to remove the recursive tail call.

## How to Run

```sh
go run ./examples/cursedpotion
```

## Expected Output

The output is colorful in a real terminal. The escape codes render as bold magenta banners, cyan
phase headers, colored ingredient lines (each ingredient has its own color), red failure boxes,
and yellow rollback messages.

```
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
  MADAME HEXWHISTLE'S TRANSACTIONAL ALCHEMY LAB
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

  "If it doesn't stack overflow, it wasn't cursed enough."

~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
  SCENARIO 1: Brewing the Elixir of Dubious Sentience
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

  >>> PHASE 1 — PREPARE: Root ritual + concurrent ingredient sourcing <<<

  (The Transactor prepares the root Op first, then all children concurrently.)

  @@  Inscribing summoning circle for: Elixir of Dubious Sentience
  @@  Candles: lit. Chanting: begun. Cats: suspicious.
  ())  Sourced: Powdered Moonsnail           [PREPARED]
  ^^^  Sourced: 3x Dried Bat Opinions        [PREPARED]
  ...  Sourced: One (1) Politician's Promise [PREPARED]
  T_T  Sourced: Tears of a Confused Wizard   [PREPARED]
  ~~~  Sourced: Cursed Oat Milk (Organic)    [PREPARED]
  @@  Stirring counter-clockwise (x3)...
  @@  Potion sealed. Elixir of Dubious Sentience is COMPLETE.
  ...  Added to cauldron: One (1) Politician's Promise [COMMITTED]
  ...

  *** ELIXIR OF DUBIOUS SENTIENCE: SUCCESSFULLY BREWED ***

  Potion effects: may cause recipient to suddenly have opinions
  about jazz. Side effects include mild omniscience and dry mouth.

~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
  SCENARIO 2: Brewing the Draught of Infinite Regret (DOOMED)
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

  >>> PHASE 1 — PREPARE: Root ritual + concurrent ingredient sourcing <<<

  (One ingredient is spoiled. Prepare will fail. Rollback must follow.)

  @@  Inscribing summoning circle for: Draught of Infinite Regret
  @@  Candles: lit. Chanting: begun. Cats: suspicious.
  OOO  SOURCING FAILED: Rotten Void Mushroom is rotten and unusable! The whole batch is cursed.
  ---  Sourced: Fine Ennui Powder            [PREPARED]
  ***  Sourced: Crystallized Existential Dread [PREPARED]

  !!! PREPARE FAILED: ingredient "Rotten Void Mushroom" is spoiled !!!

  ~~~ PREPARE FAILED — INITIATING ROLLBACK SEQUENCE ~~~

  >>> ROLLBACK: Discarding ingredients <<<

  ***  Discarded: Crystallized Existential Dread [ROLLED BACK]
       Skipping: Rotten Void Mushroom         (was never prepared)
  ---  Discarded: Fine Ennui Powder          [ROLLED BACK]

  >>> ROLLBACK: Undoing the ritual <<<

  @@  Summoning circle erased. Candles extinguished. Cats unimpressed.
  @@  Ritual aborted: Draught of Infinite Regret

  ~~~ ROLLBACK COMPLETE — LAB SAFELY RESTORED ~~~

  The Rotten Void Mushroom has been composted.
  Madame Hexwhistle will try again on the next blood moon.

~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
  LAB REPORT
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

  Scenario 1 — Elixir of Dubious Sentience:  BREWED  (prepare + commit)
  Scenario 2 — Draught of Infinite Regret:   ABORTED (prepare failed + rollback)

  Op tree structure demonstrated:
    BrewOp (root)
    +-- IngredientOp (child, leaf)
    +-- IngredientOp (child, leaf)
    +-- IngredientOp (child, leaf)
    ...

  Lifecycle: Initialize -> Prepare (root-first) -> Commit or Rollback
  Concurrency: child Ops prepared concurrently via goroutines
```

## Code Walkthrough

### `brew.go` — the root Op

`BrewOp` is the root of the Op tree. Its `Prepare()` inscribes the summoning circle (the setup
work that must happen before any ingredient is sourced). Its `Children()` method returns the slice
of `IngredientOp`s using `transactor.AsOps()`, which converts a typed slice to `iter.Seq[Op]`.
Commit seals the potion; Rollback erases the circle.

### `ingredient.go` — child Ops (leaf nodes)

`IngredientOp` represents a single ingredient. Its `Children()` returns nil — it is a leaf node.
`Prepare()` simulates sourcing the ingredient, failing if `spoiled == true`. The Transactor
prepares all ingredient Ops concurrently via goroutines, so the sourcing time is the max of any
single ingredient, not the sum.

### `main.go` — orchestration and ANSI color output

Scenario 1 calls `transactor.NewTransactor(brew1).Transact(ctx)` directly — the clean path.

Scenario 2 manually runs the prepare phase and rollback to work around the `reap()` infinite
recursion bug. It calls `brew2.Prepare()`, then fans out goroutines for each ingredient's
`Prepare()`, collects results, and if any fail, calls `Rollback()` on everything that succeeded.
This mirrors what `Transact()` + `reap()` would do if `reap()` were bug-free.
