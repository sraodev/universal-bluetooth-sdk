package commands

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/sraodev/bluetooth-service-rfcomm-python/cli/ubtctl/ai"
	"github.com/sraodev/bluetooth-service-rfcomm-python/cli/ubtctl/client"
)

type planCmd struct {
	subcommands *Registry
}

func (planCmd) Name() string     { return "plan" }
func (planCmd) Synopsis() string { return "show / replay a saved AI plan (no LLM)" }
func (c planCmd) Subcommands() *Registry {
	return c.subcommands
}
func (planCmd) Run(context.Context, []string, Invocation) error {
	return usageError(errors.New("missing command"))
}

type planShowCmd struct{}

func (planShowCmd) Name() string     { return "show" }
func (planShowCmd) Synopsis() string { return "print a saved plan as a human-readable summary" }
func (planShowCmd) Run(_ context.Context, args []string, invocation Invocation) error {
	fs := newFlagSet(invocation, "plan show")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage:\n  %s plan show <file>\n", invocation.ProgramName)
	}
	if err := parseFlags(fs, args, invocation.Out); err != nil {
		return err
	}
	if len(fs.Args()) != 1 {
		return usageError(errors.New("provide one file path"))
	}
	return planShow(fs.Args()[0], invocation)
}

func planShow(path string, invocation Invocation) error {
	p, err := ai.LoadPlan(path)
	if err != nil {
		return err
	}
	fmt.Fprintf(invocation.Out, "goal:    %s\n", p.Goal)
	fmt.Fprintf(invocation.Out, "mode:    %s\n", p.Mode)
	if p.Model != "" {
		fmt.Fprintf(invocation.Out, "model:   %s\n", p.Model)
	}
	fmt.Fprintf(invocation.Out, "created: %s\n", p.CreatedAt.Local().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(invocation.Out, "steps:   %d\n\n", len(p.Steps))
	for i, step := range p.Steps {
		mutating := ""
		if step.Mutating {
			mutating = " [mutating]"
		}
		arguments := strings.TrimSpace(string(step.Arguments))
		if arguments == "" {
			arguments = "{}"
		}
		fmt.Fprintf(invocation.Out, "[%d] %s%s\n      args: %s\n", i, step.Tool, mutating, arguments)
		if step.Result != "" {
			fmt.Fprintf(invocation.Out, "      result: %s\n", trim(step.Result, 240))
		}
	}
	return nil
}

type planRunCmd struct{}

func (planRunCmd) Name() string     { return "run" }
func (planRunCmd) Synopsis() string { return "re-execute a saved plan against ubtd (no LLM)" }
func (planRunCmd) Run(ctx context.Context, args []string, invocation Invocation) error {
	fs := newFlagSet(invocation, "plan run")
	socket := fs.String("socket", defaultSocket(), "ubtd socket path")
	dryRun := fs.Bool("dry-run", false, "print what would run; do not contact the daemon")
	yes := fs.Bool("yes", false, "allow mutating tools")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage:\n  %s plan run [flags] <file>\n\nFlags:\n", invocation.ProgramName)
		fs.PrintDefaults()
	}
	if err := parseFlags(fs, args, invocation.Out); err != nil {
		return err
	}
	rest := fs.Args()
	if len(rest) != 1 {
		return usageError(errors.New("provide one file path; flags must precede the path"))
	}
	p, err := ai.LoadPlan(rest[0])
	if err != nil {
		return err
	}

	var daemon *client.Client
	if !*dryRun {
		daemon, err = client.Dial(ctx, *socket)
		if err != nil {
			return err
		}
		defer daemon.Close()
	}

	registry := ai.BuildSpecs(daemon, false)
	return ai.Replay(ctx, p, registry, ai.ReplayOptions{
		AllowMutating: *yes,
		DryRun:        *dryRun,
		Out:           invocation.Out,
	})
}

// trim keeps at most n bytes of valid UTF-8 text, then appends an ellipsis if truncated.
func trim(s string, n int) string {
	if len(s) <= n {
		return s
	}
	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
	}
	return s[:n] + "…"
}
