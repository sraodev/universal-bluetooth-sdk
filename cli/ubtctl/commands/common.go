package commands

import (
	"context"
	"errors"
	"flag"
	"io"
	"time"

	"github.com/sraodev/bluetooth-service-rfcomm-python/cli/ubtctl/client"
	"github.com/sraodev/bluetooth-service-rfcomm-python/sdk/go/pkg/sockaddr"
)

func defaultSocket() string { return sockaddr.Default() }

func newFlagSet(invocation Invocation, name string) *flag.FlagSet {
	return flag.NewFlagSet(invocation.ProgramName+" "+name, flag.ContinueOnError)
}

// parseFlags prints requested help to stdout and returns invalid input as a
// UsageError so the application can report it once on stderr.
func parseFlags(fs *flag.FlagSet, args []string, out io.Writer) error {
	fs.SetOutput(io.Discard)
	err := fs.Parse(args)
	if errors.Is(err, flag.ErrHelp) {
		fs.SetOutput(out)
		fs.Usage()
		return err
	}
	return usageError(err)
}

type baseFlags struct {
	socket  string
	timeout time.Duration
}

func bindBase(fs *flag.FlagSet, b *baseFlags) {
	fs.StringVar(&b.socket, "socket", defaultSocket(), "ubtd socket path")
	fs.DurationVar(&b.timeout, "timeout", 10*time.Second, "request timeout")
}

func dial(parent context.Context, b baseFlags) (*client.Client, context.Context, context.CancelFunc, error) {
	ctx, cancel := context.WithTimeout(parent, b.timeout)
	c, err := client.Dial(ctx, b.socket)
	if err != nil {
		cancel()
		return nil, nil, nil, err
	}
	return c, ctx, cancel, nil
}
