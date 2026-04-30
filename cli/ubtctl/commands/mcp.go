package commands

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/sraodev/bluetooth-service-rfcomm-python/cli/ubtctl/ai"
	"github.com/sraodev/bluetooth-service-rfcomm-python/cli/ubtctl/client"
	"github.com/sraodev/bluetooth-service-rfcomm-python/cli/ubtctl/mcp"
)

type mcpCmd struct{}

func (mcpCmd) Name() string { return "mcp" }
func (mcpCmd) Synopsis() string {
	return "serve the ubtctl tool registry over MCP on stdio (for editors and agents)"
}

func (mcpCmd) Run(args []string, info RootInfo) error {
	fs := flag.NewFlagSet("mcp", flag.ContinueOnError)
	socket := fs.String("socket", defaultSocket(), "ubtd socket path")
	if err := fs.Parse(args); err != nil {
		return err
	}

	// Stdio is reserved for the JSON-RPC stream; logs go to stderr only.
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	c, err := client.Dial(*socket)
	if err != nil {
		return err
	}
	defer c.Close()

	registry, err := ai.BuildSpecs(c, ai.ExecModeNormal)
	if err != nil {
		return err
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	srv := mcp.New(registry, "ubtctl", info.CLIVersion, log)
	log.Info("ubtctl mcp serving on stdio",
		"socket", *socket,
		"tools", len(registry.All()),
		"protocol", mcp.ProtocolVersion,
	)
	return srv.Serve(ctx, os.Stdin, os.Stdout)
}

func init() { register(mcpCmd{}) }
