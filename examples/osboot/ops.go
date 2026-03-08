package main

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"time"

	"github.com/sean9999/transactor"
)

// ─── KernelBootstrapOp ────────────────────────────────────────────────────────
//
// Root Op. Simulates early kernel initialisation: allocating memory maps,
// setting up interrupt tables, and mounting the root filesystem.

var _ transactor.Op = (*KernelBootstrapOp)(nil)

type KernelBootstrapOp struct {
	child     *KernelModulesOp
	prepared  bool
	committed bool
}

func NewKernelBootstrapOp(child *KernelModulesOp) *KernelBootstrapOp {
	return &KernelBootstrapOp{child: child}
}

func (k *KernelBootstrapOp) Initialize(_ ...any) error { return nil }

func (k *KernelBootstrapOp) Prepare(ctx context.Context) error {
	return transactor.RunOrCancel(ctx, func() error {
		section("KERNEL BOOTSTRAP  [prepare]")
		steps := []string{
			"Initializing memory subsystem (SLUB allocator)",
			"Setting up interrupt descriptor table",
			"Calibrating CPU clock: 3.6 GHz detected",
			"Mounting rootfs (ext4) on /",
			"Enabling kernel preemption",
		}
		for _, s := range steps {
			time.Sleep(40 * time.Millisecond)
			ok(s)
		}
		k.prepared = true
		return nil
	})
}

func (k *KernelBootstrapOp) Commit(ctx context.Context) error {
	return transactor.RunOrCancel(ctx, func() error {
		section("KERNEL BOOTSTRAP  [commit]")
		ok("Kernel bootstrap committed — memory maps locked")
		k.committed = true
		return nil
	})
}

func (k *KernelBootstrapOp) Rollback() error {
	return k.freeMemory()
}

// freeMemory is the private cleanup method called by Rollback.
// In a real kernel this would release allocated memory regions and
// tear down the interrupt tables set up during Prepare.
func (k *KernelBootstrapOp) freeMemory() error {
	if !k.prepared {
		// Nothing was set up; nothing to tear down.
		return nil
	}
	section("KERNEL BOOTSTRAP  [rollback]")
	steps := []string{
		"Releasing interrupt descriptor table",
		"Unmounting rootfs",
		"Freeing SLUB memory allocator regions",
		"Zeroing kernel page tables",
	}
	for _, s := range steps {
		time.Sleep(30 * time.Millisecond)
		rolledBack(s)
	}
	k.prepared = false
	k.committed = false
	return nil
}

func (k *KernelBootstrapOp) Children() iter.Seq[transactor.Op] {
	return transactor.AsOps([]*KernelModulesOp{k.child})
}

// ─── KernelModulesOp ─────────────────────────────────────────────────────────
//
// Child of KernelBootstrapOp. Loads kernel modules and hardware drivers.

var _ transactor.Op = (*KernelModulesOp)(nil)

type KernelModulesOp struct {
	child          *InitSystemOp
	loadedModules  []string
	prepared       bool
	committed      bool
}

func NewKernelModulesOp(child *InitSystemOp) *KernelModulesOp {
	return &KernelModulesOp{child: child}
}

func (k *KernelModulesOp) Initialize(_ ...any) error { return nil }

func (k *KernelModulesOp) Prepare(ctx context.Context) error {
	return transactor.RunOrCancel(ctx, func() error {
		section("KERNEL MODULES  [prepare]")

		// Each module load is simulated with a short delay.
		modules := []string{
			"usbcore",
			"xhci_hcd     (USB 3.0 host controller)",
			"nvme         (NVMe storage driver)",
			"e1000e       (Intel Gigabit Ethernet)",
			"i915         (Intel HD Graphics)",
			"snd_hda_intel(HD Audio)",
		}

		for _, mod := range modules {
			time.Sleep(35 * time.Millisecond)
			ok(fmt.Sprintf("Loading module: %s", mod))
			// Record each module immediately so unloadModules() can undo it
			// even if a later step (e.g. InitSystem) fails.
			k.loadedModules = append(k.loadedModules, mod)
		}

		// Mark modules as loaded before descending so that Rollback will
		// always unload whatever was staged, regardless of child failures.
		k.prepared = true

		// After modules, walk into InitSystem and GUI ourselves.
		// walkFromRoot only traverses two levels; we recurse manually
		// for deeper nodes so that the full boot chain is exercised.
		if err := k.child.Prepare(ctx); err != nil {
			return err
		}

		return nil
	})
}

func (k *KernelModulesOp) Commit(ctx context.Context) error {
	return transactor.RunOrCancel(ctx, func() error {
		section("KERNEL MODULES  [commit]")
		ok("Kernel module table committed — drivers active")

		// Propagate commit down the chain.
		if err := k.child.Commit(ctx); err != nil {
			return err
		}

		k.committed = true
		return nil
	})
}

func (k *KernelModulesOp) Rollback() error {
	return k.unloadModules()
}

// unloadModules is the private cleanup method called by Rollback.
// Modules must be unloaded in reverse dependency order.
func (k *KernelModulesOp) unloadModules() error {
	if !k.prepared {
		return nil
	}
	section("KERNEL MODULES  [rollback]")

	// Unload in reverse order — later modules may depend on earlier ones.
	for i := len(k.loadedModules) - 1; i >= 0; i-- {
		time.Sleep(25 * time.Millisecond)
		rolledBack(fmt.Sprintf("Unloading module: %s", k.loadedModules[i]))
	}

	k.loadedModules = nil
	k.prepared = false
	k.committed = false
	return nil
}

func (k *KernelModulesOp) Children() iter.Seq[transactor.Op] {
	return transactor.AsOps([]*InitSystemOp{k.child})
}

// ─── InitSystemOp ─────────────────────────────────────────────────────────────
//
// Child of KernelModulesOp. Models systemd: starts units in dependency order
// and stops them in reverse order on rollback.

var _ transactor.Op = (*InitSystemOp)(nil)

// unit represents a single systemd-style service unit.
type unit struct {
	name     string
	requires string // dependency (empty == no dependency)
}

type InitSystemOp struct {
	child          *GuiOp
	startedUnits   []unit // tracks which units actually started (for ordered teardown)
	prepared       bool
	committed      bool
}

func NewInitSystemOp(child *GuiOp) *InitSystemOp {
	return &InitSystemOp{child: child}
}

func (s *InitSystemOp) Initialize(_ ...any) error { return nil }

var systemdUnits = []unit{
	{name: "network.target", requires: ""},
	{name: "dbus.service", requires: "network.target"},
	{name: "systemd-udevd.service", requires: ""},
	{name: "systemd-logind.service", requires: "dbus.service"},
	{name: "NetworkManager.service", requires: "network.target"},
	{name: "sshd.service", requires: "network.target"},
	{name: "avahi-daemon.service", requires: "dbus.service"},
	{name: "cups.service", requires: "dbus.service"},
}

func (s *InitSystemOp) Prepare(ctx context.Context) error {
	section("INIT SYSTEM (systemd)  [prepare]")

	for _, u := range systemdUnits {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		time.Sleep(45 * time.Millisecond)
		var msg string
		if u.requires != "" {
			msg = fmt.Sprintf("Starting %-30s (requires %s)", u.name, u.requires)
		} else {
			msg = fmt.Sprintf("Starting %-30s", u.name)
		}
		ok(msg)
		// Append immediately so stopServices() sees every started unit
		// even if a later step (e.g. GuiOp) fails.
		s.startedUnits = append(s.startedUnits, u)
	}

	// Mark services as running before descending so that Rollback will
	// always stop whatever was staged, regardless of child failures.
	s.prepared = true

	// Propagate down to GUI.
	if err := s.child.Prepare(ctx); err != nil {
		return err
	}

	return nil
}

func (s *InitSystemOp) Commit(ctx context.Context) error {
	section("INIT SYSTEM (systemd)  [commit]")
	ok("All systemd units committed — PID 1 running")

	// Propagate commit to GUI.
	if err := s.child.Commit(ctx); err != nil {
		return err
	}

	s.committed = true
	return nil
}

func (s *InitSystemOp) Rollback() error {
	return s.stopServices()
}

// stopServices is the private cleanup method called by Rollback.
// Units are stopped in reverse start order, mirroring systemd shutdown.
func (s *InitSystemOp) stopServices() error {
	if !s.prepared {
		return nil
	}
	section("INIT SYSTEM (systemd)  [rollback]")

	for i := len(s.startedUnits) - 1; i >= 0; i-- {
		time.Sleep(30 * time.Millisecond)
		rolledBack(fmt.Sprintf("Stopping %s", s.startedUnits[i].name))
	}

	s.startedUnits = nil
	s.prepared = false
	s.committed = false
	return nil
}

func (s *InitSystemOp) Children() iter.Seq[transactor.Op] {
	return transactor.AsOps([]*GuiOp{s.child})
}

// ─── GuiOp ───────────────────────────────────────────────────────────────────
//
// Child of InitSystemOp. Starts the display manager and desktop environment.
// In the failure scenario, Prepare returns an error ("no display adapter found").

var _ transactor.Op = (*GuiOp)(nil)

type GuiOp struct {
	failMode  bool // when true, Prepare returns an error
	prepared  bool
	committed bool
}

func NewGuiOp(failMode bool) *GuiOp {
	return &GuiOp{failMode: failMode}
}

func (g *GuiOp) Initialize(_ ...any) error { return nil }

func (g *GuiOp) Prepare(ctx context.Context) error {
	section("DISPLAY MANAGER / GUI  [prepare]")

	steps := []string{
		"Probing display adapters",
	}

	for _, s := range steps {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		time.Sleep(50 * time.Millisecond)
		info(s + " ...")
	}

	if g.failMode {
		time.Sleep(60 * time.Millisecond)
		fail("No display adapter found — cannot start GUI")
		return errors.New("no display adapter found")
	}

	moreSteps := []string{
		"Display adapter detected: Intel HD Graphics 620",
		"Starting Xorg display server on :0",
		"Starting GDM (GNOME Display Manager)",
		"Loading GNOME Shell",
		"Desktop environment ready",
	}
	for _, s := range moreSteps {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		time.Sleep(40 * time.Millisecond)
		ok(s)
	}

	g.prepared = true
	return nil
}

func (g *GuiOp) Commit(ctx context.Context) error {
	return transactor.RunOrCancel(ctx, func() error {
		section("DISPLAY MANAGER / GUI  [commit]")
		ok("GUI session committed — display :0 active")
		g.committed = true
		return nil
	})
}

func (g *GuiOp) Rollback() error {
	return g.teardownGui()
}

// teardownGui is the private cleanup method called by Rollback.
// Tears down the display server and frees GPU resources.
func (g *GuiOp) teardownGui() error {
	if !g.prepared {
		// GUI never came up; nothing to tear down.
		info("GUI was not started — no teardown needed")
		return nil
	}
	section("DISPLAY MANAGER / GUI  [rollback]")
	steps := []string{
		"Terminating GNOME Shell",
		"Stopping GDM (GNOME Display Manager)",
		"Killing Xorg display server on :0",
		"Releasing GPU memory",
	}
	for _, s := range steps {
		time.Sleep(30 * time.Millisecond)
		rolledBack(s)
	}
	g.prepared = false
	g.committed = false
	return nil
}

func (g *GuiOp) Children() iter.Seq[transactor.Op] {
	return nil // leaf node
}
