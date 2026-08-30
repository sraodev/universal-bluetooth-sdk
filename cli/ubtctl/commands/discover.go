package commands

import (
	"fmt"

	"github.com/sraodev/bluetooth-service-rfcomm-python/cli/ubtctl/client"
	"github.com/sraodev/bluetooth-service-rfcomm-python/sdk/go/pkg/protocol"
)

type discoverCmd struct{}

func (discoverCmd) Name() string     { return "discover" }
func (discoverCmd) Synopsis() string { return "scan for nearby devices and stream them as they appear" }

func (discoverCmd) Run(args []string, _ RootInfo) error {
	fs := newFlagSet("discover")
	var b baseFlags
	bindBase(fs, &b)
	transport := fs.String("transport", "", "limit scan to one transport (e.g., rfcomm, ble)")
	timeout := fs.Int("scan-timeout", 8, "scan duration, seconds")
	if err := parseFlags(fs, args); err != nil {
		return err
	}

	c, ctx, cancel, err := dial(b)
	if err != nil {
		return err
	}
	defer cancel()
	defer c.Close()

	params := protocol.DiscoverParams{Transport: *transport, TimeoutSeconds: *timeout}
	fmt.Printf("%-20s %-22s %-8s %s\n", "ADDRESS", "NAME", "RSSI", "TRANSPORT")
	return c.Stream(ctx, protocol.MethodDiscover, params, func(ev map[string]any) {
		var d protocol.Device
		if err := client.Decode(ev, &d); err != nil {
			fmt.Printf("(decode error: %v)\n", err)
			return
		}
		fmt.Printf("%-20s %-22s %-8d %s\n", d.Address, d.Name, d.RSSI, d.Transport)
	})
}

func init() { register(discoverCmd{}) }
