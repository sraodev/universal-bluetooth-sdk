// Command ubtctl is the Universal Bluetooth control CLI.
//
// It is a thin presentation layer: every command compiles to one or more
// requests against ubtd over the wire format documented in
// common/protocol/framing.md. The same command surface is what the AI
// planner targets, so adding a verb here automatically extends the AI
// tool registry.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/sraodev/bluetooth-service-rfcomm-python/cli/ubtctl/commands"
)

// cliVersion can be overridden at link time via -ldflags "-X main.cliVersion=...".
var cliVersion = "0.1.0-dev"

func main() {
	os.Exit(run(os.Args[1:], cliVersion))
}

func run(args []string, version string) int {
	if len(args) == 0 {
		commands.PrintRootUsage(os.Stderr, version)
		return 2
	}

	name, subArgs := args[0], args[1:]
	if isHelpToken(name) {
		return runHelp(subArgs, version)
	}

	cmd, ok := commands.Lookup(name)
	if !ok {
		return unknownCommand(name, version)
	}
	if err := cmd.Run(subArgs, commands.RootInfo{CLIVersion: version}); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		fmt.Fprintf(os.Stderr, "ubtctl %s: %v\n", name, err)
		return 1
	}
	return 0
}

func isHelpToken(s string) bool {
	switch s {
	case "help", "-h", "-help", "--help":
		return true
	}
	return false
}

// runHelp serves `ubtctl help [command]`. Help was asked for, so it goes to
// stdout and exits 0; only misuse of `help` itself reports on stderr.
func runHelp(args []string, version string) int {
	if len(args) > 1 {
		fmt.Fprintf(os.Stderr, "ubtctl help: expected one command, got %d\n\n", len(args))
		commands.PrintRootUsage(os.Stderr, version)
		return 2
	}
	// `help help` and `help --help` are still just a request for the root usage.
	if len(args) == 0 || isHelpToken(args[0]) {
		commands.PrintRootUsage(os.Stdout, version)
		return 0
	}

	cmd, ok := commands.Lookup(args[0])
	if !ok {
		return unknownCommand(args[0], version)
	}
	// Commands signal "usage printed" with flag.ErrHelp, except plan, whose
	// hand-rolled dispatch returns nil. Anything else is a real failure and
	// must not be reported to the user as a successful help request.
	err := cmd.Run([]string{"-h"}, commands.RootInfo{CLIVersion: version})
	if err != nil && !errors.Is(err, flag.ErrHelp) {
		fmt.Fprintf(os.Stderr, "ubtctl help %s: %v\n", args[0], err)
		return 1
	}
	return 0
}

func unknownCommand(name, version string) int {
	fmt.Fprintf(os.Stderr, "ubtctl: unknown command %q\n\n", name)
	commands.PrintRootUsage(os.Stderr, version)
	return 2
}
