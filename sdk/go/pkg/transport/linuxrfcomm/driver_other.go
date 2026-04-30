//go:build !linux

package linuxrfcomm

import (
	"context"

	"github.com/sraodev/bluetooth-service-rfcomm-python/sdk/go/pkg/protocol"
)

// Driver is the non-Linux placeholder. It registers with the daemon so the
// capability matrix stays honest, but every operation returns
// CodeNotImplemented — the typed CLI and AI planner already know how to
// react to that.
type Driver struct{}

func New(_ string) *Driver { return &Driver{} }

func (*Driver) Name() string { return "linuxrfcomm" }

func (*Driver) Capability() protocol.Capability {
	return protocol.Capability{
		Transport: "rfcomm",
		Discover:  false,
		Pair:      false,
		Stream:    false,
		MaxMTU:    0,
	}
}

func (*Driver) Discover(_ context.Context, _ protocol.DiscoverParams, _ chan<- protocol.Device) error {
	return &protocol.Error{Code: protocol.CodeNotImplemented, Message: "linuxrfcomm: only available on Linux"}
}

func (*Driver) Send(_ context.Context, _ protocol.SendParams) (protocol.SendResult, error) {
	return protocol.SendResult{}, &protocol.Error{Code: protocol.CodeNotImplemented, Message: "linuxrfcomm: only available on Linux"}
}

func (*Driver) Close() error { return nil }
