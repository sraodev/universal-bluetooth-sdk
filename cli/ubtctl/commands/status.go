package commands

import (
	"fmt"
	"strings"

	"github.com/sraodev/bluetooth-service-rfcomm-python/cli/ubtctl/client"
	"github.com/sraodev/bluetooth-service-rfcomm-python/sdk/go/pkg/protocol"
)

type statusCmd struct{}

func (statusCmd) Name() string     { return "status" }
func (statusCmd) Synopsis() string { return "show daemon health and registered drivers" }

func (statusCmd) Run(args []string, _ RootInfo) error {
	fs := newFlagSet("status")
	var b baseFlags
	bindBase(fs, &b)
	if err := parseFlags(fs, args); err != nil {
		return err
	}

	c, ctx, cancel, err := dial(b)
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
	fmt.Printf("state:    %s\n", s.State)
	fmt.Printf("sessions: %d\n", s.ActiveSessions)
	fmt.Printf("drivers:  %s\n", strings.Join(s.Drivers, ", "))
	return nil
}

func init() { register(statusCmd{}) }
