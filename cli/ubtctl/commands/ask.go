package commands

import (
	"context"
	"errors"
	"flag"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/sraodev/bluetooth-service-rfcomm-python/cli/ubtctl/ai"
	"github.com/sraodev/bluetooth-service-rfcomm-python/cli/ubtctl/client"
)

type askCmd struct{}

func (askCmd) Name() string     { return "ask" }
func (askCmd) Synopsis() string { return "natural-language goal → AI planner → tool calls against ubtd" }

func (askCmd) Run(args []string, _ RootInfo) error {
	fs := flag.NewFlagSet("ask", flag.ContinueOnError)
	socket := fs.String("socket", defaultSocket(), "ubtd socket path")
	dryRun := fs.Bool("dry-run", false, "skip mutating tool calls; the model still plans and reads, but Send is stubbed")
	model := fs.String("model", "", "override the Claude model (default claude-opus-4-7)")
	save := fs.String("save", "", "write the recorded tool-call plan to FILE (replay later with: ubtctl plan run FILE)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	goal := strings.TrimSpace(strings.Join(fs.Args(), " "))
	if goal == "" {
		return errors.New("provide a goal, e.g.: ubtctl ask 'is the daemon healthy?'")
	}

	c, err := client.Dial(*socket)
	if err != nil {
		return err
	}
	defer c.Close()

	// `ask` runs the agentic loop; tool-call deadlines are enforced
	// inside the daemon. The outer context only carries signal cancellation.
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	return ai.Run(ctx, ai.RunConfig{
		Goal:       goal,
		Model:      *model,
		DryRun:     *dryRun,
		SocketPath: *socket,
		Daemon:     c,
		Out:        os.Stdout,
		SavePath:   *save,
	})
}

func init() { register(askCmd{}) }
