// Command ubtd is the Universal Bluetooth control-plane daemon.
//
// It listens on a Unix domain socket, multiplexes typed requests onto
// platform-specific transport drivers, and is the only process that
// touches the Bluetooth stack directly. Both the typed CLI (ubtctl) and
// the AI planner speak to it through the wire format documented in
// common/protocol/framing.md.
package main

import (
	"context"
	"flag"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/sraodev/bluetooth-service-rfcomm-python/cli/ubtd/server"
	"github.com/sraodev/bluetooth-service-rfcomm-python/sdk/go/pkg/sockaddr"
	"github.com/sraodev/bluetooth-service-rfcomm-python/sdk/go/pkg/transport"
	"github.com/sraodev/bluetooth-service-rfcomm-python/sdk/go/pkg/transport/linuxrfcomm"
	"github.com/sraodev/bluetooth-service-rfcomm-python/sdk/go/pkg/transport/stub"
)

// daemonVersion can be overridden at link time via -ldflags "-X main.daemonVersion=...".
var daemonVersion = "0.1.0-dev"

func main() {
	socket := flag.String("socket", sockaddr.Default(), "Unix domain socket path")
	driver := flag.String("driver", "stub", "transport driver: stub | linuxrfcomm")
	bluetoothctl := flag.String("bluetoothctl", "", "override path to bluetoothctl (linuxrfcomm only; default = PATH lookup)")
	logJSON := flag.Bool("log-json", false, "emit JSON-structured logs")
	flag.Parse()

	log := newLogger(*logJSON)

	registry := transport.NewRegistry()
	switch *driver {
	case "stub":
		registry.Register(stub.New())
	case "linuxrfcomm":
		registry.Register(linuxrfcomm.New(*bluetoothctl))
	default:
		log.Error("unknown driver", "name", *driver, "supported", []string{"stub", "linuxrfcomm"})
		os.Exit(2)
	}
	defer registry.Close()

	if err := os.MkdirAll(filepath.Dir(*socket), 0o755); err != nil {
		log.Error("create socket dir", "err", err)
		os.Exit(1)
	}
	// Remove stale socket from a previous crash.
	_ = os.Remove(*socket)

	ln, err := net.Listen("unix", *socket)
	if err != nil {
		log.Error("listen", "socket", *socket, "err", err)
		os.Exit(1)
	}
	if err := os.Chmod(*socket, 0o600); err != nil {
		log.Warn("chmod socket", "err", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	log.Info("ubtd listening",
		"socket", *socket,
		"version", daemonVersion,
		"drivers", registry.Names(),
	)

	srv := server.New(log, registry, daemonVersion)
	if err := srv.Serve(ctx, ln); err != nil {
		log.Error("serve", "err", err)
		os.Exit(1)
	}
	log.Info("ubtd shutdown")
}

func newLogger(asJSON bool) *slog.Logger {
	level := slog.LevelInfo
	if v := os.Getenv("UBTD_LOG_LEVEL"); v != "" {
		_ = level.UnmarshalText([]byte(v))
	}
	opts := &slog.HandlerOptions{Level: level}
	if asJSON {
		return slog.New(slog.NewJSONHandler(os.Stderr, opts))
	}
	return slog.New(slog.NewTextHandler(os.Stderr, opts))
}
