// Package main demonstrates the Transactor library using a simulated OS boot
// sequence. The Op tree forms a linear chain:
//
//	KernelBootstrapOp  (root)
//	└── KernelModulesOp
//	    └── InitSystemOp  (systemd-style services)
//	        └── GuiOp     (display manager / desktop environment)
//
// Two scenarios are run:
//  1. Happy path — full boot succeeds and the system becomes ready.
//  2. Failure path — the GUI cannot start ("no display adapter found"),
//     triggering a full rollback that unwinds the chain in reverse.
package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/sean9999/transactor"
)

func main() {
	runScenario("SCENARIO 1: HAPPY PATH — SUCCESSFUL BOOT", false)
	fmt.Println()
	fmt.Println(strings.Repeat("╌", 70))
	fmt.Println()
	runScenario("SCENARIO 2: FAILURE PATH — NO DISPLAY ADAPTER", true)
}

// runScenario builds the Op tree, runs Transact(), and reports the outcome.
// When failGui is true the GuiOp.Prepare() returns an error, exercising the
// full rollback path.
func runScenario(title string, failGui bool) {
	banner(title)

	// Build the chain bottom-up: leaf first, then each parent wraps its child.
	//
	// walkFromRoot in the Transactor traverses the root and its direct children.
	// For the deeper levels of this chain (InitSystemOp, GuiOp) we cascade
	// Prepare and Commit manually inside each Op's own methods, which is the
	// correct pattern for chains deeper than two levels.
	// Rollback uses reap(), which recurses through all Children() automatically.
	guiOp := NewGuiOp(failGui)
	initOp := NewInitSystemOp(guiOp)
	modulesOp := NewKernelModulesOp(initOp)
	rootOp := NewKernelBootstrapOp(modulesOp)

	ctx := context.Background()

	err := transactor.NewTransactor(rootOp).Transact(ctx)
	if err != nil {
		systemFailed("no display adapter found")
		fmt.Printf("\n  %sError detail:%s %v\n\n", bold+red, reset, err)
	} else {
		systemReady()
	}
}
