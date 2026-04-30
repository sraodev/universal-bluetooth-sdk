package commands

import (
	"flag"
	"fmt"

	"github.com/sraodev/bluetooth-service-rfcomm-python/cli/ubtctl/client"
	"github.com/sraodev/bluetooth-service-rfcomm-python/sdk/go/pkg/protocol"
)

type versionCmd struct{}

func (versionCmd) Name() string     { return "version" }
func (versionCmd) Synopsis() string { return "print CLI + daemon + protocol versions" }

func (versionCmd) Run(args []string, info RootInfo) error {
	fs := flag.NewFlagSet("version", flag.ContinueOnError)
	var b baseFlags
	bindBase(fs, &b)
	clientOnly := fs.Bool("client-only", false, "skip the daemon round-trip")
	if err := fs.Parse(args); err != nil {
		return err
	}

	fmt.Printf("ubtctl   %s\n", info.CLIVersion)
	fmt.Printf("protocol %s (wire v1)\n", protocol.Version)
	if *clientOnly {
		return nil
	}

	c, ctx, cancel, err := dial(b)
	if err != nil {
		fmt.Printf("ubtd     not reachable: %v\n", err)
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
	fmt.Printf("ubtd     %s (protocol %s)\n", v.DaemonVersion, v.ProtocolVersion)
	return nil
}

func init() { register(versionCmd{}) }
