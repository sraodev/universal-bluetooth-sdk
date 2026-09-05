package commands

import (
	"context"
	"fmt"

	"github.com/sraodev/bluetooth-service-rfcomm-python/cli/ubtctl/client"
	"github.com/sraodev/bluetooth-service-rfcomm-python/sdk/go/pkg/protocol"
)

type capabilitiesCmd struct{}

func (capabilitiesCmd) Name() string     { return "capabilities" }
func (capabilitiesCmd) Synopsis() string { return "list per-transport capabilities advertised by ubtd" }

func (capabilitiesCmd) Run(ctx context.Context, args []string, invocation Invocation) error {
	fs := newFlagSet(invocation, "capabilities")
	var b baseFlags
	bindBase(fs, &b)
	if err := parseFlags(fs, args, invocation.Out); err != nil {
		return err
	}

	c, ctx, cancel, err := dial(ctx, b)
	if err != nil {
		return err
	}
	defer cancel()
	defer c.Close()

	res, err := c.Call(ctx, protocol.MethodCapabilities, nil)
	if err != nil {
		return err
	}
	var r protocol.CapabilitiesResult
	if err := client.Decode(res, &r); err != nil {
		return err
	}
	fmt.Fprintf(invocation.Out, "%-12s %-9s %-5s %-7s %s\n", "TRANSPORT", "DISCOVER", "PAIR", "STREAM", "MTU")
	for _, c := range r.Capabilities {
		fmt.Fprintf(invocation.Out, "%-12s %-9v %-5v %-7v %d\n", c.Transport, c.Discover, c.Pair, c.Stream, c.MaxMTU)
	}
	return nil
}
