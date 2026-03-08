package main

import (
	"fmt"
	"strings"
	"time"
)

// ANSI color codes
const (
	reset   = "\033[0m"
	bold    = "\033[1m"
	dim     = "\033[2m"
	red     = "\033[31m"
	green   = "\033[32m"
	yellow  = "\033[33m"
	blue    = "\033[34m"
	magenta = "\033[35m"
	cyan    = "\033[36m"
	white   = "\033[37m"
	bgBlue  = "\033[44m"
	bgRed   = "\033[41m"
)

func ts() string {
	return dim + time.Now().Format("15:04:05.000") + reset
}

func ok(msg string) {
	fmt.Printf("  %s  %-55s %s[  OK  ]%s\n", ts(), msg, bold+green, reset)
}

func fail(msg string) {
	fmt.Printf("  %s  %-55s %s[ FAIL ]%s\n", ts(), msg, bold+red, reset)
}

func rolledBack(msg string) {
	fmt.Printf("  %s  %-55s %s[ROLLED]%s\n", ts(), msg, bold+yellow, reset)
}

func info(msg string) {
	fmt.Printf("  %s  %s%s%s\n", ts(), dim+cyan, msg, reset)
}

func section(title string) {
	width := 70
	bar := strings.Repeat("─", width)
	fmt.Printf("\n%s%s%s\n", bold+blue, bar, reset)
	padding := (width - len(title)) / 2
	fmt.Printf("%s%s%s%s%s\n", bold+blue, strings.Repeat(" ", padding), white+bold, title, reset)
	fmt.Printf("%s%s%s\n\n", bold+blue, bar, reset)
}

func banner(title string) {
	width := 70
	bar := strings.Repeat("═", width)
	fmt.Printf("\n%s%s%s\n", bold+cyan, bar, reset)
	padding := (width - len(title)) / 2
	fmt.Printf("%s%s%s%s%s\n", bgBlue+bold, strings.Repeat(" ", padding), white+bold, title, reset)
	fmt.Printf("%s%s%s\n\n", bold+cyan, bar, reset)
}

func systemReady() {
	lines := []string{
		"",
		"  ╔══════════════════════════════════════════════════════════════╗",
		"  ║                                                              ║",
		"  ║           SYSTEM READY  —  Boot sequence complete           ║",
		"  ║                                                              ║",
		"  ╚══════════════════════════════════════════════════════════════╝",
		"",
	}
	for _, l := range lines {
		fmt.Printf("%s%s%s\n", bold+green, l, reset)
	}
}

func systemFailed(reason string) {
	lines := []string{
		"",
		"  ╔══════════════════════════════════════════════════════════════╗",
		"  ║                                                              ║",
		"  ║           BOOT FAILED  —  System rolled back                ║",
		"  ║                                                              ║",
		fmt.Sprintf("  ║  Reason: %-52s║", reason),
		"  ║                                                              ║",
		"  ╚══════════════════════════════════════════════════════════════╝",
		"",
	}
	for _, l := range lines {
		fmt.Printf("%s%s%s\n", bold+red, l, reset)
	}
}
