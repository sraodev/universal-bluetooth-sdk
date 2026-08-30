package server

import (
	"context"
	"io"
	"log/slog"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/sraodev/bluetooth-service-rfcomm-python/sdk/go/pkg/transport"
)

func TestShutdownClosesPartialFrameClient(t *testing.T) {
	// A short path also works on macOS, where Unix socket paths are limited.
	ln, err := net.Listen("unix", filepath.Join(t.TempDir(), "s"))
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	srv := New(slog.New(slog.NewTextHandler(io.Discard, nil)), transport.NewRegistry(), "test")
	done := make(chan error, 1)
	go func() { done <- srv.Serve(ctx, ln) }()
	conn, err := net.Dial("unix", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err := conn.Write([]byte{0}); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	for srv.sessions.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if srv.sessions.Load() == 0 {
		t.Fatal("client was not accepted")
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("shutdown blocked on partial frame")
	}
}
