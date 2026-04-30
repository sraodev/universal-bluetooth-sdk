// Package stub provides an in-memory transport driver used for tests and
// for running the daemon on hosts without Bluetooth hardware (CI, dev
// containers, the AI planner's dry-run mode).
package stub

import (
	"context"
	"errors"
	"time"

	"github.com/sraodev/bluetooth-service-rfcomm-python/sdk/go/pkg/protocol"
)

type Driver struct {
	devices []protocol.Device
}

func New() *Driver {
	return &Driver{
		devices: []protocol.Device{
			{Address: "AA:BB:CC:DD:EE:01", Name: "stub-pi", Transport: "rfcomm", RSSI: -42},
			{Address: "AA:BB:CC:DD:EE:02", Name: "stub-esp32", Transport: "rfcomm", RSSI: -67},
		},
	}
}

func (d *Driver) Name() string { return "stub" }

func (d *Driver) Capability() protocol.Capability {
	return protocol.Capability{
		Transport: "rfcomm",
		Discover:  true,
		Pair:      false,
		Stream:    true,
		MaxMTU:    1024,
	}
}

func (d *Driver) Discover(ctx context.Context, _ protocol.DiscoverParams, out chan<- protocol.Device) error {
	for _, dev := range d.devices {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case out <- dev:
		}
		// Simulate scan latency without making tests slow.
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
	return nil
}

func (d *Driver) Send(_ context.Context, params protocol.SendParams) (protocol.SendResult, error) {
	if params.Address == "" {
		return protocol.SendResult{}, errors.New("address required")
	}
	return protocol.SendResult{
		BytesSent:     int64(len(params.Payload)),
		LatencyMicros: 1234,
	}, nil
}

func (d *Driver) Close() error { return nil }
