// Package commands hosts the ubtctl subcommand registry.
//
// Each subcommand is a small struct implementing Command. Keeping the
// registry typed (rather than hand-rolled string switches scattered
// across main) is what lets the AI planner enumerate the surface and
// generate tool schemas mechanically.
package commands

import (
	"fmt"
	"io"
	"sort"
)

type RootInfo struct {
	CLIVersion string
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

// PrintRootUsage writes the command list to w. The caller picks the stream:
// stdout when the user asked for help, stderr when usage accompanies an error.
func PrintRootUsage(w io.Writer, version string) {
	fmt.Fprintf(w, "ubtctl %s — Universal Bluetooth CLI\n\n", version)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  ubtctl <command> [flags]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	names := Names()
	for _, n := range names {
		fmt.Fprintf(w, "  %-12s %s\n", n, registry[n].Synopsis())
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Flags common to all commands:")
	fmt.Fprintln(w, "  --socket <path>   override ubtd socket path (env UBTD_SOCKET)")
}

// Names returns the registered command names in sorted order.
func Names() []string {
	names := make([]string, 0, len(registry))
	for n := range registry {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
