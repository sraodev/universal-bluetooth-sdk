package commands

import (
	"flag"
	"fmt"
	"time"

	"github.com/sraodev/bluetooth-service-rfcomm-python/cli/ubtctl/client"
	"github.com/sraodev/bluetooth-service-rfcomm-python/sdk/go/pkg/protocol"
)

type pingCmd struct{}

func (pingCmd) Name() string     { return "ping" }
func (pingCmd) Synopsis() string { return "round-trip a Ping request to ubtd (clock skew + liveness)" }

func (pingCmd) Run(args []string, _ RootInfo) error {
	fs := flag.NewFlagSet("ping", flag.ContinueOnError)
	var b baseFlags
	bindBase(fs, &b)
	if err := fs.Parse(args); err != nil {
		return err
	}

	c, ctx, cancel, err := dial(b)
	if err != nil {
		return err
	}
	defer cancel()
	defer c.Close()

	start := time.Now()
	res, err := c.Call(ctx, protocol.MethodPing, nil)
	if err != nil {
		return err
	}
	rtt := time.Since(start)

	var p protocol.PingResult
	if err := client.Decode(res, &p); err != nil {
		return err
	}
	skew := time.Now().UnixMilli() - p.ServerTimeUnixMs
	fmt.Printf("%s  rtt=%s  clock-skew=%d ms\n", p.Pong, rtt, skew)
	return nil
}

func init() { register(pingCmd{}) }
