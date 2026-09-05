package commands

import (
	"context"
	"log/slog"

	"github.com/sraodev/bluetooth-service-rfcomm-python/cli/ubtctl/ai"
	"github.com/sraodev/bluetooth-service-rfcomm-python/cli/ubtctl/client"
	"github.com/sraodev/bluetooth-service-rfcomm-python/cli/ubtctl/mcp"
)

type mcpCmd struct{}

func (mcpCmd) Name() string { return "mcp" }
func (mcpCmd) Synopsis() string {
	return "serve the ubtctl tool registry over MCP on stdio (for editors and agents)"
}

func (mcpCmd) Run(ctx context.Context, args []string, invocation Invocation) error {
	fs := newFlagSet(invocation, "mcp")
	socket := fs.String("socket", defaultSocket(), "ubtd socket path")
	if err := parseFlags(fs, args, invocation.Out); err != nil {
		return err
	}

	// Stdio is reserved for the JSON-RPC stream; logs go to stderr only.
	log := slog.New(slog.NewTextHandler(invocation.ErrOut, &slog.HandlerOptions{Level: slog.LevelInfo}))

	c, err := client.Dial(ctx, *socket)
	if err != nil {
		return err
	}
	defer c.Close()

	registry := ai.BuildSpecs(c, false)

	srv := mcp.New(registry, invocation.ProgramName, invocation.CLIVersion, log)
	log.Info("ubtctl mcp serving on stdio",
		"socket", *socket,
		"tools", len(registry.All()),
		"protocol", mcp.ProtocolVersion,
	)
	return srv.Serve(ctx, invocation.In, invocation.Out)
}
