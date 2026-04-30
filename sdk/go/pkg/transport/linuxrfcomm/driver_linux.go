//go:build linux

package linuxrfcomm

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"golang.org/x/sys/unix"

	"github.com/sraodev/bluetooth-service-rfcomm-python/sdk/go/pkg/protocol"
)

// btprotoRFCOMM is the Linux Bluetooth RFCOMM protocol number.
// (Not exported by x/sys/unix; defined in <bluetooth/bluetooth.h>.)
const btprotoRFCOMM = 3

// Driver speaks RFCOMM directly to the kernel over AF_BLUETOOTH sockets.
type Driver struct {
	bluetoothctl string // optional override; "" → look up "bluetoothctl" on PATH
}

// New returns a Driver. The optional path overrides bluetoothctl discovery
// (helpful for unprivileged tests that point at a fake binary).
func New(bluetoothctlPath string) *Driver {
	return &Driver{bluetoothctl: bluetoothctlPath}
}

func (*Driver) Name() string { return "linuxrfcomm" }

func (*Driver) Capability() protocol.Capability {
	return protocol.Capability{
		Transport: "rfcomm",
		Discover:  true, // limited: lists known peers, not live scan
		Pair:      false,
		Stream:    true,
		MaxMTU:    32768,
	}
}

// Discover enumerates devices known to BlueZ via `bluetoothctl devices`.
// This is *not* a live scan — pairing/inquiry is left to the OS shell. The
// trade-off: zero D-Bus dependency in the daemon for v1, while still
// exposing useful state to the AI planner.
func (d *Driver) Discover(ctx context.Context, _ protocol.DiscoverParams, out chan<- protocol.Device) error {
	bin := d.bluetoothctl
	if bin == "" {
		var err error
		bin, err = exec.LookPath("bluetoothctl")
		if err != nil {
			return &protocol.Error{
				Code:    protocol.CodeNotImplemented,
				Message: "bluetoothctl not on PATH; install bluez-utils to enable discover on this driver",
			}
		}
	}

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(cctx, bin, "devices")
	stdout, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("bluetoothctl: %w", err)
	}
	for _, line := range strings.Split(string(stdout), "\n") {
		dev, ok := parseBluetoothctlDevice(line)
		if !ok {
			continue
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case out <- dev:
		}
	}
	return nil
}

func (*Driver) Send(ctx context.Context, params protocol.SendParams) (protocol.SendResult, error) {
	if params.Address == "" {
		return protocol.SendResult{}, &protocol.Error{Code: protocol.CodeInvalidParams, Message: "address required"}
	}
	channel := params.UUIDPort
	if channel <= 0 {
		channel = 1 // RFCOMM channel 1 is the SPP convention
	}
	if channel > 30 {
		return protocol.SendResult{}, &protocol.Error{Code: protocol.CodeInvalidParams, Message: "rfcomm channel must be 1..30"}
	}
	bdaddr, err := parseBDAddr(params.Address)
	if err != nil {
		return protocol.SendResult{}, &protocol.Error{Code: protocol.CodeInvalidParams, Message: err.Error()}
	}

	fd, err := unix.Socket(unix.AF_BLUETOOTH, unix.SOCK_STREAM|unix.SOCK_CLOEXEC, btprotoRFCOMM)
	if err != nil {
		return protocol.SendResult{}, &protocol.Error{Code: protocol.CodeTransportError, Message: "socket: " + err.Error()}
	}
	defer unix.Close(fd)

	// Honour ctx cancellation by interrupting the connect with a deadline.
	if dl, ok := ctx.Deadline(); ok {
		_ = unix.SetsockoptTimeval(fd, unix.SOL_SOCKET, unix.SO_SNDTIMEO, timevalUntil(dl))
	}

	start := time.Now()
	if err := unix.Connect(fd, &unix.SockaddrRFCOMM{Addr: bdaddr, Channel: uint8(channel)}); err != nil {
		return protocol.SendResult{}, &protocol.Error{
			Code:    protocol.CodeTransportError,
			Message: fmt.Sprintf("connect %s ch%d: %v", params.Address, channel, err),
		}
	}

	written := 0
	for written < len(params.Payload) {
		n, err := unix.Write(fd, params.Payload[written:])
		if err != nil {
			return protocol.SendResult{}, &protocol.Error{Code: protocol.CodeTransportError, Message: "write: " + err.Error()}
		}
		written += n
	}
	return protocol.SendResult{
		BytesSent:     int64(written),
		LatencyMicros: time.Since(start).Microseconds(),
	}, nil
}

func (*Driver) Close() error { return nil }

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// parseBDAddr converts "AA:BB:CC:DD:EE:FF" into the wire-order [6]byte
// expected by AF_BLUETOOTH (little-endian: least-significant byte first).
func parseBDAddr(s string) ([6]byte, error) {
	parts := strings.Split(strings.TrimSpace(s), ":")
	if len(parts) != 6 {
		return [6]byte{}, fmt.Errorf("not a Bluetooth address: %q", s)
	}
	var b [6]byte
	for i, p := range parts {
		if len(p) != 2 {
			return [6]byte{}, fmt.Errorf("not a Bluetooth address: %q", s)
		}
		var v byte
		_, err := fmt.Sscanf(p, "%02x", &v)
		if err != nil {
			return [6]byte{}, fmt.Errorf("not a Bluetooth address: %q", s)
		}
		b[5-i] = v // wire order is reversed
	}
	return b, nil
}

func parseBluetoothctlDevice(line string) (protocol.Device, bool) {
	// Lines look like: "Device AA:BB:CC:DD:EE:FF Friendly Name"
	const prefix = "Device "
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, prefix) {
		return protocol.Device{}, false
	}
	rest := strings.TrimPrefix(line, prefix)
	sp := strings.IndexByte(rest, ' ')
	if sp <= 0 {
		return protocol.Device{}, false
	}
	addr := rest[:sp]
	if _, err := parseBDAddr(addr); err != nil {
		return protocol.Device{}, false
	}
	return protocol.Device{
		Address:   addr,
		Name:      strings.TrimSpace(rest[sp+1:]),
		Transport: "rfcomm",
	}, true
}

func timevalUntil(dl time.Time) *unix.Timeval {
	d := time.Until(dl)
	if d < 0 {
		d = 0
	}
	tv := unix.NsecToTimeval(d.Nanoseconds())
	return &tv
}
