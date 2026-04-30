// Package commands hosts the ubtctl subcommand registry.
//
// Each subcommand is a small struct implementing Command. Keeping the
// registry typed (rather than hand-rolled string switches scattered
// across main) is what lets the AI planner enumerate the surface and
// generate tool schemas mechanically.
package commands

import (
	"fmt"
	"sort"
)

type RootInfo struct {
	CLIVersion string
	Commit     string
}

type Command interface {
	Name() string
	Synopsis() string
	Run(args []string, info RootInfo) error
}

var registry = map[string]Command{}

func register(c Command) {
	registry[c.Name()] = c
}

func Lookup(name string) (Command, bool) {
	c, ok := registry[name]
	return c, ok
}

func PrintRootUsage(version string) {
	fmt.Printf("ubtctl %s — Universal Bluetooth CLI\n\n", version)
	fmt.Println("Usage:")
	fmt.Println("  ubtctl <command> [flags]")
	fmt.Println()
	fmt.Println("Commands:")
	names := make([]string, 0, len(registry))
	for n := range registry {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		fmt.Printf("  %-12s %s\n", n, registry[n].Synopsis())
	}
	fmt.Println()
	fmt.Println("Flags common to all commands:")
	fmt.Println("  --socket <path>   override ubtd socket path (env UBTD_SOCKET)")
}
