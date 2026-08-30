package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sraodev/bluetooth-service-rfcomm-python/cli/ubtctl/commands"
)

// capture runs the CLI with os.Stdout and os.Stderr redirected. The command
// packages print through the real files, so swapping them is what lets the
// test see subcommand usage as well as run's own output.
func capture(t *testing.T, args []string) (code int, stdout, stderr string) {
	t.Helper()

	outR, outW, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	errR, errW, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}

	origOut, origErr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = outW, errW

	// Drain concurrently so a usage page larger than the pipe buffer cannot
	// deadlock the test.
	outC, errC := make(chan string, 1), make(chan string, 1)
	go func() { b, _ := io.ReadAll(outR); outC <- string(b) }()
	go func() { b, _ := io.ReadAll(errR); errC <- string(b) }()

	func() {
		defer func() {
			os.Stdout, os.Stderr = origOut, origErr
			outW.Close()
			errW.Close()
		}()
		code = run(args, "0.1.0-test")
	}()

	return code, <-outC, <-errC
}

func TestRun(t *testing.T) {
	// Pin the socket somewhere that cannot exist, so a developer running a
	// daemon locally gets the same result CI does. Any help path that starts
	// dialing will fail here instead of silently passing.
	t.Setenv("UBTD_SOCKET", filepath.Join(t.TempDir(), "absent.sock"))

	tests := []struct {
		name       string
		args       []string
		wantCode   int
		wantStdout string
		wantStderr string
	}{
		{
			name:       "no args prints usage to stderr and returns 2",
			args:       []string{},
			wantCode:   2,
			wantStderr: "Usage:\n  ubtctl <command> [flags]",
		},
		{
			name:       "root --help returns 0",
			args:       []string{"--help"},
			wantCode:   0,
			wantStdout: "Usage:\n  ubtctl <command> [flags]",
		},
		{
			name:       "root -h returns 0",
			args:       []string{"-h"},
			wantCode:   0,
			wantStdout: "Universal Bluetooth CLI",
		},
		{
			name:       "root -help returns 0",
			args:       []string{"-help"},
			wantCode:   0,
			wantStdout: "Universal Bluetooth CLI",
		},
		{
			name:       "root help returns 0",
			args:       []string{"help"},
			wantCode:   0,
			wantStdout: "Universal Bluetooth CLI",
		},
		{
			name:       "help help returns root usage",
			args:       []string{"help", "help"},
			wantCode:   0,
			wantStdout: "Universal Bluetooth CLI",
		},
		{
			name:       "subcommand help via help verb returns 0",
			args:       []string{"help", "ping"},
			wantCode:   0,
			wantStdout: "Usage of ping:",
		},
		{
			name:       "subcommand -h flag returns 0",
			args:       []string{"ping", "-h"},
			wantCode:   0,
			wantStdout: "Usage of ping:",
		},
		{
			name:       "subcommand --help flag returns 0",
			args:       []string{"status", "--help"},
			wantCode:   0,
			wantStdout: "Usage of status:",
		},
		{
			name:       "help with extra arguments returns 2",
			args:       []string{"help", "ping", "status"},
			wantCode:   2,
			wantStderr: "expected one command, got 2",
		},
		{
			name:       "unknown command returns 2",
			args:       []string{"nonexistent-cmd"},
			wantCode:   2,
			wantStderr: `ubtctl: unknown command "nonexistent-cmd"`,
		},
		{
			name:       "help for unknown command returns 2",
			args:       []string{"help", "nonexistent-cmd"},
			wantCode:   2,
			wantStderr: `ubtctl: unknown command "nonexistent-cmd"`,
		},
		{
			name:       "invalid flag for known command returns 1",
			args:       []string{"version", "--invalid-flag-that-does-not-exist"},
			wantCode:   1,
			wantStderr: "ubtctl version: flag provided but not defined",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			code, stdout, stderr := capture(t, tc.args)
			if code != tc.wantCode {
				t.Errorf("run(%v) = %d; want %d", tc.args, code, tc.wantCode)
			}
			if tc.wantStdout != "" && !strings.Contains(stdout, tc.wantStdout) {
				t.Errorf("stdout missing %q; got:\n%s", tc.wantStdout, stdout)
			}
			if tc.wantStderr != "" && !strings.Contains(stderr, tc.wantStderr) {
				t.Errorf("stderr missing %q; got:\n%s", tc.wantStderr, stderr)
			}
			// Help is requested output; diagnostics are not. Neither should
			// end up on the other's stream.
			if tc.wantStdout != "" && stderr != "" {
				t.Errorf("help path wrote to stderr:\n%s", stderr)
			}
			if tc.wantCode != 0 && stdout != "" {
				t.Errorf("error path wrote to stdout:\n%s", stdout)
			}
		})
	}
}

// TestHelpForEveryCommand covers the acceptance criterion across the whole
// surface rather than two hand-picked names: ask and mcp dial the daemon
// immediately after parsing, so a -h that stopped short-circuiting would
// fail here against the absent socket set in TestRun.
func TestHelpForEveryCommand(t *testing.T) {
	t.Setenv("UBTD_SOCKET", filepath.Join(t.TempDir(), "absent.sock"))

	names := commands.Names()
	if len(names) == 0 {
		t.Fatal("no commands registered")
	}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			for _, args := range [][]string{{name, "-h"}, {"help", name}} {
				code, stdout, stderr := capture(t, args)
				if code != 0 {
					t.Errorf("run(%v) = %d; want 0 (stderr: %s)", args, code, stderr)
				}
				if stdout == "" {
					t.Errorf("run(%v) printed no help to stdout", args)
				}
			}
		})
	}
}
