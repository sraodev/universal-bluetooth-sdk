package commands

import (
	"context"
	"errors"
	"flag"
	"io"
	"os"
	"time"

	"github.com/sraodev/bluetooth-service-rfcomm-python/cli/ubtctl/client"
	"github.com/sraodev/bluetooth-service-rfcomm-python/sdk/go/pkg/sockaddr"
)

func defaultSocket() string { return sockaddr.Default() }

func newFlagSet(name string) *flag.FlagSet {
	return flag.NewFlagSet(name, flag.ContinueOnError)
}

// parseFlags parses args and prints usage only when the user asked for it.
//
// flag writes help text and parse diagnostics to the same writer, so it gets
// none: requested help is output and belongs on stdout, while a bad flag is
// reported once by main on stderr rather than twice on two streams.
func parseFlags(fs *flag.FlagSet, args []string) error {
	fs.SetOutput(io.Discard)
	err := fs.Parse(args)
	if errors.Is(err, flag.ErrHelp) {
		fs.SetOutput(os.Stdout)
		fs.Usage()
	}
	return err
}

type baseFlags struct {
	socket  string
	timeout time.Duration
}

func bindBase(fs *flag.FlagSet, b *baseFlags) {
	fs.StringVar(&b.socket, "socket", defaultSocket(), "ubtd socket path")
	fs.DurationVar(&b.timeout, "timeout", 10*time.Second, "request timeout")
}

func dial(b baseFlags) (*client.Client, context.Context, context.CancelFunc, error) {
	c, err := client.Dial(b.socket)
	if err != nil {
		return nil, nil, nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), b.timeout)
	return c, ctx, cancel, nil
}
