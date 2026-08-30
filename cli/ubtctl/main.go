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
		commands.PrintRootUsage(version)
		return 2
	}

	name, subArgs := args[0], args[1:]
	if name == "help" || name == "-h" || name == "--help" || name == "-help" {
		if len(subArgs) > 0 {
			subName := subArgs[0]
			cmd, ok := commands.Lookup(subName)
			if !ok {
				fmt.Fprintf(os.Stderr, "ubtctl: unknown command %q\n\n", subName)
				commands.PrintRootUsage(version)
				return 2
			}
			_ = cmd.Run([]string{"-h"}, commands.RootInfo{CLIVersion: version})
			return 0
		}
		commands.PrintRootUsage(version)
		return 0
	}

	cmd, ok := commands.Lookup(name)
	if !ok {
		fmt.Fprintf(os.Stderr, "ubtctl: unknown command %q\n\n", name)
		commands.PrintRootUsage(version)
		return 2
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
