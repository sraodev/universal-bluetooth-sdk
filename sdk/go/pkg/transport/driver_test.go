package transport

import (
	"context"
	"slices"
	"testing"

	"github.com/sraodev/bluetooth-service-rfcomm-python/sdk/go/pkg/protocol"
)

type registryTestDriver struct {
	name       string
	capability protocol.Capability
}

func (d registryTestDriver) Name() string                    { return d.name }
func (d registryTestDriver) Capability() protocol.Capability { return d.capability }
func (d registryTestDriver) Close() error                    { return nil }
func (d registryTestDriver) Discover(context.Context, protocol.DiscoverParams, chan<- protocol.Device) error {
	return nil
}
func (d registryTestDriver) Send(context.Context, protocol.SendParams) (protocol.SendResult, error) {
	return protocol.SendResult{}, nil
}

func TestRegistryReturnsDriversInNameOrder(t *testing.T) {
	alpha := registryTestDriver{
		name:       "alpha",
		capability: protocol.Capability{Transport: "rfcomm"},
	}
	zulu := registryTestDriver{
		name:       "zulu",
		capability: protocol.Capability{Transport: "ble"},
	}

	tests := []struct {
		name    string
		drivers []Driver
	}{
		{name: "forward registration", drivers: []Driver{alpha, zulu}},
		{name: "reverse registration", drivers: []Driver{zulu, alpha}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			registry := NewRegistry()
			for _, driver := range tc.drivers {
				registry.Register(driver)
			}

			if got, want := registry.Names(), []string{"alpha", "zulu"}; !slices.Equal(got, want) {
				t.Fatalf("Names() = %v, want %v", got, want)
			}

			got := registry.Capabilities()
			want := []protocol.Capability{alpha.capability, zulu.capability}
			if !slices.Equal(got, want) {
				t.Fatalf("Capabilities() = %v, want %v", got, want)
			}

			for transport, wantName := range map[string]string{"rfcomm": "alpha", "ble": "zulu"} {
				driver, ok := registry.ForTransport(transport)
				if !ok {
					t.Fatalf("ForTransport(%q) did not find a driver", transport)
				}
				if driver.Name() != wantName {
					t.Fatalf("ForTransport(%q) name = %q, want %q", transport, driver.Name(), wantName)
				}
			}
		})
	}
}
