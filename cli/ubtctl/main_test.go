package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sraodev/bluetooth-service-rfcomm-python/cli/ubtctl/commands"
)

func TestExecutableArguments(t *testing.T) {
	goTool, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("Go tool unavailable: %v", err)
	}

	dir := t.TempDir()
	canonical := filepath.Join(dir, "ubt")
	build := exec.Command(goTool, "build", "-o", canonical, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Skipf("cannot build ubt: %v\n%s", err, output)
	}

	t.Setenv("UBTD_SOCKET", filepath.Join(t.TempDir(), "absent.sock"))
	tests := []struct {
		binary         string
		wantName       string
		wantLegacyNote bool
	}{
		{binary: "ubt", wantName: "ubt"},
		{binary: "ubtctl", wantName: "ubtctl", wantLegacyNote: true},
		{binary: "temporary-test-binary", wantName: "ubt"},
	}

	for _, tc := range tests {
		t.Run(tc.binary, func(t *testing.T) {
			binary := filepath.Join(dir, tc.binary)
			if binary != canonical {
				if err := os.Link(canonical, binary); err != nil {
					t.Fatal(err)
				}
			}

			code, stdout, stderr := runExecutable(t, binary, "--help")
			if code != 0 || stderr != "" {
				t.Fatalf("%s --help: code=%d stdout=%q stderr=%q", tc.binary, code, stdout, stderr)
			}
			if !strings.Contains(stdout, "  "+tc.wantName+" <command> [flags]") {
				t.Fatalf("%s --help does not present as %q:\n%s", tc.binary, tc.wantName, stdout)
			}
			if got := strings.Contains(stdout, "legacy alias supported through 0.x"); got != tc.wantLegacyNote {
				t.Fatalf("%s legacy note = %v; want %v", tc.binary, got, tc.wantLegacyNote)
			}

			code, _, stderr = runExecutable(t, binary, "nonexistent-cmd")
			if code != 2 || !strings.Contains(stderr, tc.wantName+`: unknown command "nonexistent-cmd"`) {
				t.Fatalf("%s unknown command: code=%d stderr=%q", tc.binary, code, stderr)
			}

			code, stdout, stderr = runExecutable(t, binary, "version", "--client-only")
			if code != 0 || stderr != "" || !strings.HasPrefix(stdout, tc.wantName+"   ") {
				t.Fatalf("%s version: code=%d stdout=%q stderr=%q", tc.binary, code, stdout, stderr)
			}
		})
	}
}

func runExecutable(t *testing.T, binary string, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errOut bytes.Buffer
	command := exec.Command(binary, args...)
	command.Stdout = &out
	command.Stderr = &errOut
	err := command.Run()
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("run %s: %v", binary, err)
		}
		code = exitErr.ExitCode()
	}
	return code, out.String(), errOut.String()
}

func TestProgramName(t *testing.T) {
	tests := map[string]string{
		"/usr/local/bin/ubt":       "ubt",
		"/usr/local/bin/ubtctl":    "ubtctl",
		"/tmp/copied-test-program": "ubt",
		"ubtctl.exe":               "ubtctl",
		"UBTCTL.EXE":               "ubtctl",
	}
	for input, want := range tests {
		if got := programName(input); got != want {
			t.Errorf("programName(%q) = %q; want %q", input, got, want)
		}
	}
}

func capture(t *testing.T, args []string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errOut bytes.Buffer
	app, err := NewApp("ubtctl", "0.1.0-test", strings.NewReader(""), &out, &errOut)
	if err != nil {
		t.Fatal(err)
	}
	code = app.Run(context.Background(), args)
	return code, out.String(), errOut.String()
}

func TestRun(t *testing.T) {
	t.Setenv("UBTD_SOCKET", filepath.Join(t.TempDir(), "absent.sock"))

	tests := []struct {
		name       string
		args       []string
		wantCode   int
		wantStdout string
		wantStderr string
	}{
		{
			name:       "no args prints usage to stderr",
			args:       []string{},
			wantCode:   2,
			wantStderr: "Usage:\n  ubtctl <command> [flags]",
		},
		{
			name:       "root help",
			args:       []string{"--help"},
			wantCode:   0,
			wantStdout: "Usage:\n  ubtctl <command> [flags]",
		},
		{
			name:       "help help returns root usage",
			args:       []string{"help", "help"},
			wantCode:   0,
			wantStdout: "Universal Bluetooth CLI",
		},
		{
			name:       "leaf help through help command",
			args:       []string{"help", "ping"},
			wantCode:   0,
			wantStdout: "Usage of ubtctl ping:",
		},
		{
			name:       "leaf help through flag",
			args:       []string{"status", "--help"},
			wantCode:   0,
			wantStdout: "Usage of ubtctl status:",
		},
		{
			name:       "group help",
			args:       []string{"help", "plan"},
			wantCode:   0,
			wantStdout: "Usage:\n  ubtctl plan <command> [flags]",
		},
		{
			name:       "nested help through help command",
			args:       []string{"help", "plan", "show"},
			wantCode:   0,
			wantStdout: "Usage:\n  ubtctl plan show <file>",
		},
		{
			name:       "nested help through flag",
			args:       []string{"plan", "run", "-h"},
			wantCode:   0,
			wantStdout: "Usage:\n  ubtctl plan run [flags] <file>",
		},
		{
			name:       "missing nested command",
			args:       []string{"plan"},
			wantCode:   2,
			wantStderr: "ubtctl plan: missing command",
		},
		{
			name:       "unknown root command",
			args:       []string{"nonexistent-cmd"},
			wantCode:   2,
			wantStderr: `ubtctl: unknown command "nonexistent-cmd"`,
		},
		{
			name:       "unknown nested command",
			args:       []string{"plan", "nonexistent-cmd"},
			wantCode:   2,
			wantStderr: `ubtctl plan: unknown command "nonexistent-cmd"`,
		},
		{
			name:       "extra help path",
			args:       []string{"help", "ping", "status"},
			wantCode:   2,
			wantStderr: "ubtctl help: ping has no subcommands",
		},
		{
			name:       "invalid flag is usage error",
			args:       []string{"version", "--invalid-flag-that-does-not-exist"},
			wantCode:   2,
			wantStderr: "ubtctl version: flag provided but not defined",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			code, stdout, stderr := capture(t, tc.args)
			if code != tc.wantCode {
				t.Errorf("Run(%v) = %d; want %d", tc.args, code, tc.wantCode)
			}
			if tc.wantStdout != "" && !strings.Contains(stdout, tc.wantStdout) {
				t.Errorf("stdout missing %q; got:\n%s", tc.wantStdout, stdout)
			}
			if tc.wantStderr != "" && !strings.Contains(stderr, tc.wantStderr) {
				t.Errorf("stderr missing %q; got:\n%s", tc.wantStderr, stderr)
			}
			if tc.wantStdout != "" && stderr != "" {
				t.Errorf("help path wrote to stderr:\n%s", stderr)
			}
			if tc.wantCode != 0 && stdout != "" {
				t.Errorf("error path wrote to stdout:\n%s", stdout)
			}
		})
	}
}

func TestHelpForEveryCommand(t *testing.T) {
	t.Setenv("UBTD_SOCKET", filepath.Join(t.TempDir(), "absent.sock"))
	app, err := NewApp("ubtctl", "0.1.0-test", strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}

	var paths [][]string
	var visit func(*commands.Registry, []string)
	visit = func(registry *commands.Registry, prefix []string) {
		for _, name := range registry.Names() {
			path := append(append([]string{}, prefix...), name)
			paths = append(paths, path)
			if group, ok := lookupGroup(t, registry, name); ok {
				visit(group.Subcommands(), path)
			}
		}
	}
	visit(app.registry, nil)
	if len(paths) == 0 {
		t.Fatal("no commands registered")
	}

	for _, path := range paths {
		path := path
		t.Run(strings.Join(path, " "), func(t *testing.T) {
			for _, args := range [][]string{
				append(append([]string{}, path...), "-h"),
				append([]string{"help"}, path...),
			} {
				code, stdout, stderr := capture(t, args)
				if code != 0 || stdout == "" || stderr != "" {
					t.Errorf("Run(%v): code=%d stdout=%q stderr=%q", args, code, stdout, stderr)
				}
			}
		})
	}
}

func lookupGroup(t *testing.T, registry *commands.Registry, name string) (commands.Group, bool) {
	t.Helper()
	command, ok := registry.Lookup(name)
	if !ok {
		t.Fatalf("command %q disappeared", name)
	}
	group, ok := command.(commands.Group)
	return group, ok
}

type contextCommand struct {
	called *bool
}

func (contextCommand) Name() string     { return "context" }
func (contextCommand) Synopsis() string { return "test context forwarding" }
func (c contextCommand) Run(ctx context.Context, _ []string, _ commands.Invocation) error {
	*c.called = true
	return ctx.Err()
}

func TestAppForwardsCancellation(t *testing.T) {
	called := false
	registry, err := commands.NewRegistry(contextCommand{called: &called})
	if err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	app := &App{
		registry: registry,
		invocation: commands.Invocation{
			ProgramName: "ubtctl",
			CLIVersion:  "test",
			In:          strings.NewReader(""),
			Out:         &out,
			ErrOut:      &errOut,
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if code := app.Run(ctx, []string{"context"}); code != 1 {
		t.Fatalf("Run(cancelled) = %d; want 1", code)
	}
	if !called {
		t.Fatal("command did not receive the cancelled context")
	}
}
