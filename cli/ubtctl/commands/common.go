package commands

import (
	"context"
	"flag"
	"time"

	"github.com/sraodev/bluetooth-service-rfcomm-python/cli/ubtctl/client"
	"github.com/sraodev/bluetooth-service-rfcomm-python/sdk/go/pkg/sockaddr"
)

func defaultSocket() string { return sockaddr.Default() }

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
