package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/sraodev/bluetooth-service-rfcomm-python/cli/ubtctl/client"
	"github.com/sraodev/bluetooth-service-rfcomm-python/sdk/go/pkg/protocol"
)

type pingCmd struct{}

func (pingCmd) Name() string     { return "ping" }
func (pingCmd) Synopsis() string { return "round-trip a Ping request to ubtd (clock skew + liveness)" }

func (pingCmd) Run(ctx context.Context, args []string, invocation Invocation) error {
	fs := newFlagSet(invocation, "ping")
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
	fmt.Fprintf(invocation.Out, "%s  rtt=%s  clock-skew=%d ms\n", p.Pong, rtt, skew)
	return nil
}
