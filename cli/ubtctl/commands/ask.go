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
	"github.com/sraodev/bluetooth-service-rfcomm-python/sdk/go/pkg/sockaddr"
)

func defaultSocket() string { return sockaddr.Default() }

type askCmd struct{}

func (askCmd) Name() string     { return "ask" }
func (askCmd) Synopsis() string { return "natural-language goal → AI planner → tool calls against ubtd" }

func (askCmd) Run(args []string, _ RootInfo) error {
	fs := flag.NewFlagSet("ask", flag.ContinueOnError)
	socket := fs.String("socket", defaultSocket(), "ubtd socket path")
	dryRun := fs.Bool("dry-run", false, "skip mutating tool calls; the model still plans and reads, but Send is stubbed")
	yes := fs.Bool("yes", false, "auto-approve mutating tool calls (currently the only mutator is Send)")
	model := fs.String("model", "", "override the Claude model (default claude-opus-4-7)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	goal := strings.TrimSpace(strings.Join(fs.Args(), " "))
	if goal == "" {
		return errors.New("provide a goal, e.g.: ubtctl ask 'is the daemon healthy?'")
	}

	mode := ai.ExecModeNormal
	switch {
	case *dryRun:
		mode = ai.ExecModeDryRun
	case *yes:
		mode = ai.ExecModeAutoYes
	}

	// `ask` runs the agentic loop; the per-tool-call deadline is enforced
	// inside the daemon. We give the outer context only signal cancellation.
	c, err := client.Dial(*socket)
	if err != nil {
		return err
	}
	defer c.Close()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	return ai.Run(ctx, ai.Plan{
		Goal:       goal,
		Model:      *model,
		Mode:       mode,
		SocketPath: *socket,
		Daemon:     c,
		Out:        os.Stdout,
	})
}

func init() { register(askCmd{}) }
