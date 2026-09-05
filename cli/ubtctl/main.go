// Command ubt is the Universal Bluetooth control CLI.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
)

// cliVersion can be overridden at link time via -ldflags "-X main.cliVersion=...".
var cliVersion = "0.1.0-dev"

func main() {
	name := programName(os.Args[0])
	app, err := NewApp(name, cliVersion, os.Stdin, os.Stdout, os.Stderr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", name, err)
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	os.Exit(app.Run(ctx, os.Args[1:]))
}

func programName(argv0 string) string {
	base := strings.TrimSuffix(strings.ToLower(filepath.Base(argv0)), ".exe")
	if base == "ubtctl" {
		return "ubtctl"
	}
	return "ubt"
}
