// Package transport defines the port through which ubtd talks to concrete
// Bluetooth (or Bluetooth-like) hardware adapters.
//
// Each platform/transport pair (Linux+RFCOMM, macOS+CoreBluetooth, ...)
// implements Driver. The daemon picks drivers at runtime based on what
// the host advertises through Capabilities().
package transport

import (
	"context"

	"github.com/sraodev/bluetooth-service-rfcomm-python/sdk/go/pkg/protocol"
)

// Driver is the contract every transport adapter must satisfy.
type Driver interface {
	// Name is a short identifier ("rfcomm-bluez", "ble-corebluetooth").
	Name() string

	// Capability advertises what this driver can do on this host.
	Capability() protocol.Capability

	// Discover emits devices over out until ctx is cancelled. Drivers that
	// cannot scan return CodeNotImplemented immediately.
	Discover(ctx context.Context, params protocol.DiscoverParams, out chan<- protocol.Device) error

	// Send delivers a payload to a peer. Returns bytes-written + latency.
	Send(ctx context.Context, params protocol.SendParams) (protocol.SendResult, error)

	// Close releases any background resources held by the driver.
	Close() error
}

// Registry holds drivers selected for the running daemon.
type Registry struct {
	byName      map[string]Driver
	byTransport map[string]Driver
}

func NewRegistry() *Registry {
	return &Registry{
		byName:      map[string]Driver{},
		byTransport: map[string]Driver{},
	}
}

func (r *Registry) Register(d Driver) {
	r.byName[d.Name()] = d
	r.byTransport[d.Capability().Transport] = d
}

func (r *Registry) ForTransport(t string) (Driver, bool) {
	d, ok := r.byTransport[t]
	return d, ok
}

func (r *Registry) Names() []string {
	out := make([]string, 0, len(r.byName))
	for n := range r.byName {
		out = append(out, n)
	}
	return out
}

func (r *Registry) Capabilities() []protocol.Capability {
	out := make([]protocol.Capability, 0, len(r.byName))
	for _, d := range r.byName {
		out = append(out, d.Capability())
	}
	return out
}

func (r *Registry) Close() error {
	var firstErr error
	for _, d := range r.byName {
		if err := d.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
