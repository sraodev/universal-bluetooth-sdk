package commands

import (
	"context"
	"fmt"

	"github.com/sraodev/bluetooth-service-rfcomm-python/cli/ubtctl/client"
	"github.com/sraodev/bluetooth-service-rfcomm-python/sdk/go/pkg/protocol"
)

type versionCmd struct{}

func (versionCmd) Name() string     { return "version" }
func (versionCmd) Synopsis() string { return "print CLI + daemon + protocol versions" }

func (versionCmd) Run(ctx context.Context, args []string, invocation Invocation) error {
	fs := newFlagSet(invocation, "version")
	var b baseFlags
	bindBase(fs, &b)
	clientOnly := fs.Bool("client-only", false, "skip the daemon round-trip")
	if err := parseFlags(fs, args, invocation.Out); err != nil {
		return err
	}

	fmt.Fprintf(invocation.Out, "%s   %s\n", invocation.ProgramName, invocation.CLIVersion)
	fmt.Fprintf(invocation.Out, "protocol %s (wire v1)\n", protocol.Version)
	if *clientOnly {
		return nil
	}

	c, ctx, cancel, err := dial(ctx, b)
	if err != nil {
		fmt.Fprintf(invocation.Out, "ubtd     not reachable: %v\n", err)
		return nil
	}
	defer cancel()
	defer c.Close()

	res, err := c.Call(ctx, protocol.MethodVersion, nil)
	if err != nil {
		return err
	}
	var v protocol.VersionResult
	if err := client.Decode(res, &v); err != nil {
		return err
	}
	fmt.Fprintf(invocation.Out, "ubtd     %s (protocol %s)\n", v.DaemonVersion, v.ProtocolVersion)
	return nil
}
