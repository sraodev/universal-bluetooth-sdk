package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/sraodev/bluetooth-service-rfcomm-python/cli/ubtctl/client"
	"github.com/sraodev/bluetooth-service-rfcomm-python/sdk/go/pkg/protocol"
)

type statusCmd struct{}

func (statusCmd) Name() string     { return "status" }
func (statusCmd) Synopsis() string { return "show daemon health and registered drivers" }

func (statusCmd) Run(ctx context.Context, args []string, invocation Invocation) error {
	fs := newFlagSet(invocation, "status")
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

	res, err := c.Call(ctx, protocol.MethodStatus, nil)
	if err != nil {
		return err
	}
	var s protocol.StatusResult
	if err := client.Decode(res, &s); err != nil {
		return err
	}
	fmt.Fprintf(invocation.Out, "state:    %s\n", s.State)
	fmt.Fprintf(invocation.Out, "sessions: %d\n", s.ActiveSessions)
	fmt.Fprintf(invocation.Out, "drivers:  %s\n", strings.Join(s.Drivers, ", "))
	return nil
}
