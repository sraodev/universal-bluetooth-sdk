package commands

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/sraodev/bluetooth-service-rfcomm-python/cli/ubtctl/ai"
	"github.com/sraodev/bluetooth-service-rfcomm-python/cli/ubtctl/client"
)

type planCmd struct{}

func (planCmd) Name() string     { return "plan" }
func (planCmd) Synopsis() string { return "show / replay a saved AI plan (no LLM)" }

func (planCmd) Run(args []string, _ RootInfo) error {
	if len(args) == 0 {
		printPlanUsage()
		return errors.New("plan: missing subcommand")
	}
	switch args[0] {
	case "-h", "--help", "help":
		printPlanUsage()
		return nil
	case "show":
		return planShow(args[1:])
	case "run":
		return planRun(args[1:])
	default:
		printPlanUsage()
		return fmt.Errorf("plan: unknown subcommand %q", args[0])
	}
}

func init() { register(planCmd{}) }

func printPlanUsage() {
	fmt.Println(`Usage:
  ubtctl plan show <file>            print a saved plan as a human-readable summary
  ubtctl plan run  <file> [flags]    re-execute a saved plan against ubtd (no LLM)

plan run flags:
  --socket <path>   override ubtd socket path (env UBTD_SOCKET)
  --dry-run         print what would run; do not contact the daemon
  --yes             allow steps tagged "mutating" (otherwise plan run aborts)`)
}

// ---------------------------------------------------------------------------
// plan show
// ---------------------------------------------------------------------------

func planShow(args []string) error {
	if len(args) == 0 {
		return errors.New("plan show: missing file path")
	}
	p, err := ai.LoadPlan(args[0])
	if err != nil {
		return err
	}
	fmt.Printf("goal:    %s\n", p.Goal)
	fmt.Printf("mode:    %s\n", p.Mode)
	if p.Model != "" {
		fmt.Printf("model:   %s\n", p.Model)
	}
	fmt.Printf("created: %s\n", p.CreatedAt.Local().Format("2006-01-02 15:04:05"))
	fmt.Printf("steps:   %d\n\n", len(p.Steps))
	for i, s := range p.Steps {
		flag := ""
		if s.Mutating {
			flag = " [mutating]"
		}
		args := strings.TrimSpace(string(s.Arguments))
		if args == "" {
			args = "{}"
		}
		fmt.Printf("[%d] %s%s\n      args: %s\n", i, s.Tool, flag, args)
		if s.Result != "" {
			fmt.Printf("      result: %s\n", trim(s.Result, 240))
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// plan run
// ---------------------------------------------------------------------------

func planRun(args []string) error {
	fs := flag.NewFlagSet("plan run", flag.ContinueOnError)
	socket := fs.String("socket", defaultSocket(), "ubtd socket path")
	dryRun := fs.Bool("dry-run", false, "print what would run; do not contact the daemon")
	yes := fs.Bool("yes", false, "allow steps tagged as mutating")
	if err := fs.Parse(args); err != nil {
		return err
	}
	rest := fs.Args()
	if len(rest) == 0 {
		return errors.New("plan run: missing file path")
	}
	p, err := ai.LoadPlan(rest[0])
	if err != nil {
		return err
	}

	c, err := client.Dial(*socket)
	if err != nil {
		return err
	}
	defer c.Close()

	registry := ai.BuildSpecs(c, false)
	return ai.Replay(context.Background(), p, registry, ai.ReplayOptions{
		AllowMutating: *yes,
		DryRun:        *dryRun,
		Out:           os.Stdout,
	})
}

func trim(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
