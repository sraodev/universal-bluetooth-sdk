package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/sraodev/bluetooth-service-rfcomm-python/cli/ubtctl/commands"
)

// App owns the process-level CLI dependencies and command dispatch policy.
type App struct {
	registry   *commands.Registry
	invocation commands.Invocation
}

func NewApp(programName, version string, in io.Reader, out, errOut io.Writer) (*App, error) {
	registry, err := commands.NewDefaultRegistry()
	if err != nil {
		return nil, fmt.Errorf("build command registry: %w", err)
	}
	return &App{
		registry: registry,
		invocation: commands.Invocation{
			ProgramName: programName,
			CLIVersion:  version,
			In:          in,
			Out:         out,
			ErrOut:      errOut,
		},
	}, nil
}

// Run executes one CLI invocation and returns its process exit code.
func (a *App) Run(ctx context.Context, args []string) int {
	if len(args) == 0 {
		a.registry.PrintUsage(a.invocation.ErrOut, a.invocation, nil)
		return 2
	}
	if isHelpToken(args[0]) {
		return a.help(ctx, a.registry, nil, args[1:])
	}
	return a.dispatch(ctx, a.registry, nil, args)
}

func (a *App) dispatch(ctx context.Context, registry *commands.Registry, path, args []string) int {
	name := args[0]
	command, ok := registry.Lookup(name)
	if !ok {
		return a.unknownCommand(registry, path, name)
	}
	commandPath := appendPath(path, name)
	if group, ok := command.(commands.Group); ok {
		rest := args[1:]
		if len(rest) == 0 {
			fmt.Fprintf(a.invocation.ErrOut, "%s: missing command\n\n", a.commandName(commandPath))
			group.Subcommands().PrintUsage(a.invocation.ErrOut, a.invocation, commandPath)
			return 2
		}
		if isHelpToken(rest[0]) {
			return a.help(ctx, group.Subcommands(), commandPath, rest[1:])
		}
		return a.dispatch(ctx, group.Subcommands(), commandPath, rest)
	}

	err := command.Run(ctx, args[1:], a.invocation)
	return a.commandResult(commandPath, err)
}

func (a *App) help(ctx context.Context, registry *commands.Registry, path, args []string) int {
	if len(args) == 0 || isHelpToken(args[0]) {
		registry.PrintUsage(a.invocation.Out, a.invocation, path)
		return 0
	}
	name := args[0]
	command, ok := registry.Lookup(name)
	if !ok {
		return a.unknownCommand(registry, path, name)
	}
	commandPath := appendPath(path, name)
	if group, ok := command.(commands.Group); ok {
		return a.help(ctx, group.Subcommands(), commandPath, args[1:])
	}
	if len(args) > 1 {
		fmt.Fprintf(a.invocation.ErrOut, "%s help: %s has no subcommands\n", a.invocation.ProgramName, strings.Join(commandPath, " "))
		return 2
	}
	err := command.Run(ctx, []string{"-h"}, a.invocation)
	return a.commandResult(commandPath, err)
}

func (a *App) commandResult(path []string, err error) int {
	if err == nil || errors.Is(err, flag.ErrHelp) {
		return 0
	}
	fmt.Fprintf(a.invocation.ErrOut, "%s: %v\n", a.commandName(path), err)
	var usageErr *commands.UsageError
	if errors.As(err, &usageErr) {
		return 2
	}
	return 1
}

func (a *App) unknownCommand(registry *commands.Registry, path []string, name string) int {
	fmt.Fprintf(a.invocation.ErrOut, "%s: unknown command %q\n\n", a.commandName(path), name)
	registry.PrintUsage(a.invocation.ErrOut, a.invocation, path)
	return 2
}

func (a *App) commandName(path []string) string {
	if len(path) == 0 {
		return a.invocation.ProgramName
	}
	return a.invocation.ProgramName + " " + strings.Join(path, " ")
}

func appendPath(path []string, name string) []string {
	result := make([]string, len(path), len(path)+1)
	copy(result, path)
	return append(result, name)
}

func isHelpToken(s string) bool {
	switch s {
	case "help", "-h", "-help", "--help":
		return true
	}
	return false
}
