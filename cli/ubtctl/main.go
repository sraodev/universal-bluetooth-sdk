// Command ubtctl is the Universal Bluetooth control CLI.
//
// It is a thin presentation layer: every command compiles to one or more
// requests against ubtd over the wire format documented in
// common/protocol/framing.md. The same command surface is what the AI
// planner targets, so adding a verb here automatically extends the AI
// tool registry.
package main

import (
	"fmt"
	"os"

	"github.com/sraodev/bluetooth-service-rfcomm-python/cli/ubtctl/commands"
)

// cliVersion can be overridden at link time via -ldflags "-X main.cliVersion=...".
var cliVersion = "0.1.0-dev"

func main() {
	if len(os.Args) < 2 {
		commands.PrintRootUsage(cliVersion)
		os.Exit(2)
	}

	name, args := os.Args[1], os.Args[2:]
	cmd, ok := commands.Lookup(name)
	if !ok {
		fmt.Fprintf(os.Stderr, "ubtctl: unknown command %q\n\n", name)
		commands.PrintRootUsage(cliVersion)
		os.Exit(2)
	}
	if err := cmd.Run(args, commands.RootInfo{CLIVersion: cliVersion}); err != nil {
		fmt.Fprintf(os.Stderr, "ubtctl %s: %v\n", name, err)
		os.Exit(1)
	}
}
