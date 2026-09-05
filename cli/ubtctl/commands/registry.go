// Package commands defines the typed Universal Bluetooth CLI command tree.
//
// The command registry is the presentation layer used by the CLI dispatcher.
// The AI planner and MCP server use their own shared tool registry in tools.
package commands

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
)

// Invocation contains the process-level dependencies shared by commands.
type Invocation struct {
	ProgramName string
	CLIVersion  string
	In          io.Reader
	Out         io.Writer
	ErrOut      io.Writer
}

type Command interface {
	Name() string
	Synopsis() string
	Run(context.Context, []string, Invocation) error
}

// Group is a command whose children are dispatched by the root application.
type Group interface {
	Command
	Subcommands() *Registry
}

// UsageError marks invalid command-line input. The application maps it to exit 2.
type UsageError struct {
	Err error
}

func (e *UsageError) Error() string { return e.Err.Error() }
func (e *UsageError) Unwrap() error { return e.Err }

func usageError(err error) error {
	if err == nil {
		return nil
	}
	return &UsageError{Err: err}
}

// Registry is an immutable command lookup table after construction.
type Registry struct {
	commands map[string]Command
}

// NewRegistry validates and registers commands explicitly.
func NewRegistry(commands ...Command) (*Registry, error) {
	registry := &Registry{commands: make(map[string]Command, len(commands))}
	for _, command := range commands {
		if command == nil {
			return nil, errors.New("register command: nil command")
		}
		name := command.Name()
		if name == "" {
			return nil, errors.New("register command: empty name")
		}
		if _, exists := registry.commands[name]; exists {
			return nil, fmt.Errorf("register command %q: duplicate name", name)
		}
		registry.commands[name] = command
	}
	return registry, nil
}

// NewDefaultRegistry constructs the complete CLI command tree.
func NewDefaultRegistry() (*Registry, error) {
	planCommands, err := NewRegistry(planShowCmd{}, planRunCmd{})
	if err != nil {
		return nil, fmt.Errorf("register plan commands: %w", err)
	}
	return NewRegistry(
		askCmd{},
		capabilitiesCmd{},
		discoverCmd{},
		mcpCmd{},
		pingCmd{},
		planCmd{subcommands: planCommands},
		sendCmd{},
		statusCmd{},
		versionCmd{},
	)
}

func (r *Registry) Lookup(name string) (Command, bool) {
	command, ok := r.commands[name]
	return command, ok
}

// Names returns registered command names in sorted order.
func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.commands))
	for name := range r.commands {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// PrintUsage writes usage for the registry at path.
func (r *Registry) PrintUsage(w io.Writer, invocation Invocation, path []string) {
	if len(path) == 0 {
		fmt.Fprintf(w, "%s %s — Universal Bluetooth CLI\n\n", invocation.ProgramName, invocation.CLIVersion)
	}
	fullPath := invocation.ProgramName
	for _, part := range path {
		fullPath += " " + part
	}
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintf(w, "  %s <command> [flags]\n\n", fullPath)
	fmt.Fprintln(w, "Commands:")
	for _, name := range r.Names() {
		fmt.Fprintf(w, "  %-12s %s\n", name, r.commands[name].Synopsis())
	}
	if len(path) == 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Flags common to all commands:")
		fmt.Fprintln(w, "  --socket <path>   override ubtd socket path (env UBTD_SOCKET)")
		if invocation.ProgramName == "ubtctl" {
			fmt.Fprintln(w)
			fmt.Fprintln(w, "Compatibility:")
			fmt.Fprintln(w, "  ubtctl is a legacy alias supported through 0.x; use ubt for new scripts.")
		}
	}
}
