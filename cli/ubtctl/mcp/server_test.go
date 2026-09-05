package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/sraodev/bluetooth-service-rfcomm-python/cli/ubtctl/tools"
)

func TestServeKeepsStdoutMachineReadable(t *testing.T) {
	input := strings.NewReader("{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"initialize\"}\n")
	var protocolOut, logs bytes.Buffer
	server := New(
		tools.NewRegistry(),
		"ubtctl",
		"test",
		slog.New(slog.NewTextHandler(&logs, nil)),
	)
	if err := server.Serve(context.Background(), input, &protocolOut); err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(strings.TrimSpace(protocolOut.String()), "\n") {
		if !json.Valid([]byte(line)) {
			t.Fatalf("MCP stdout contains non-JSON output: %q", line)
		}
	}
	if strings.Contains(protocolOut.String(), "level=") {
		t.Fatalf("MCP stdout contains log output: %q", protocolOut.String())
	}
}
