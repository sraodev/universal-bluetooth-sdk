package commands

import (
	"context"
	"fmt"

	"github.com/sraodev/bluetooth-service-rfcomm-python/cli/ubtctl/client"
	"github.com/sraodev/bluetooth-service-rfcomm-python/sdk/go/pkg/protocol"
)

type discoverCmd struct{}

func (discoverCmd) Name() string     { return "discover" }
func (discoverCmd) Synopsis() string { return "scan for nearby devices and stream them as they appear" }

func (discoverCmd) Run(ctx context.Context, args []string, invocation Invocation) error {
	fs := newFlagSet(invocation, "discover")
	var b baseFlags
	bindBase(fs, &b)
	transport := fs.String("transport", "", "limit scan to one transport (e.g., rfcomm, ble)")
	timeout := fs.Int("scan-timeout", 8, "scan duration, seconds")
	if err := parseFlags(fs, args, invocation.Out); err != nil {
		return err
	}

	c, ctx, cancel, err := dial(ctx, b)
	if err != nil {
		return err
	}
	defer cancel()
	defer c.Close()

	params := protocol.DiscoverParams{Transport: *transport, TimeoutSeconds: *timeout}
	fmt.Fprintf(invocation.Out, "%-20s %-22s %-8s %s\n", "ADDRESS", "NAME", "RSSI", "TRANSPORT")
	return c.Stream(ctx, protocol.MethodDiscover, params, func(ev map[string]any) {
		var d protocol.Device
		if err := client.Decode(ev, &d); err != nil {
			fmt.Fprintf(invocation.Out, "(decode error: %v)\n", err)
			return
		}
		fmt.Fprintf(invocation.Out, "%-20s %-22s %-8d %s\n", d.Address, d.Name, d.RSSI, d.Transport)
	})
}
