// Command ubtctl is the Universal Bluetooth control CLI.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

// cliVersion can be overridden at link time via -ldflags "-X main.cliVersion=...".
var cliVersion = "0.1.0-dev"

func main() {
	app, err := NewApp("ubtctl", cliVersion, os.Stdin, os.Stdout, os.Stderr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ubtctl: %v\n", err)
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	os.Exit(app.Run(ctx, os.Args[1:]))
}
