package client

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

func TestDialHonorsCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	client, err := Dial(ctx, filepath.Join(t.TempDir(), "absent.sock"))
	if client != nil {
		client.Close()
		t.Fatal("Dial returned a client for a cancelled context")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Dial error = %v; want context.Canceled", err)
	}
}
