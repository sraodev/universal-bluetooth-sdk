package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/sraodev/bluetooth-service-rfcomm-python/cli/ubtctl/ai"
	"github.com/sraodev/bluetooth-service-rfcomm-python/cli/ubtctl/client"
)

type askCmd struct{}

func (askCmd) Name() string { return "ask" }
func (askCmd) Synopsis() string {
	return "natural-language goal → AI planner → tool calls against ubtd"
}

func (askCmd) Run(ctx context.Context, args []string, invocation Invocation) error {
	fs := newFlagSet(invocation, "ask")
	socket := fs.String("socket", defaultSocket(), "ubtd socket path")
	dryRun := fs.Bool("dry-run", false, "skip mutating tool calls; the model still plans and reads, but Send is stubbed")
	model := fs.String("model", "", "override the Claude model (default claude-opus-4-7)")
	save := fs.String("save", "", fmt.Sprintf("write the recorded tool-call plan to FILE (replay later with: %s plan run FILE)", invocation.ProgramName))
	if err := parseFlags(fs, args, invocation.Out); err != nil {
		return err
	}
	goal := strings.TrimSpace(strings.Join(fs.Args(), " "))
	if goal == "" {
		return usageError(fmt.Errorf("provide a goal, e.g.: %s ask 'is the daemon healthy?'", invocation.ProgramName))
	}

	c, err := client.Dial(ctx, *socket)
	if err != nil {
		return err
	}
	defer c.Close()

	return ai.Run(ctx, ai.RunConfig{
		ProgramName: invocation.ProgramName,
		Goal:        goal,
		Model:       *model,
		DryRun:      *dryRun,
		SocketPath:  *socket,
		Daemon:      c,
		Out:         invocation.Out,
		SavePath:    *save,
	})
}
