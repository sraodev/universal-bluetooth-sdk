package commands

import (
	"context"
	"reflect"
	"testing"
)

type stubCommand string

func (c stubCommand) Name() string                                  { return string(c) }
func (stubCommand) Synopsis() string                                { return "test command" }
func (stubCommand) Run(context.Context, []string, Invocation) error { return nil }

func TestNewRegistry(t *testing.T) {
	tests := []struct {
		name     string
		commands []Command
		want     []string
		wantErr  bool
	}{
		{name: "sorted explicit commands", commands: []Command{stubCommand("zeta"), stubCommand("alpha")}, want: []string{"alpha", "zeta"}},
		{name: "duplicate", commands: []Command{stubCommand("same"), stubCommand("same")}, wantErr: true},
		{name: "empty name", commands: []Command{stubCommand("")}, wantErr: true},
		{name: "nil command", commands: []Command{nil}, wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			registry, err := NewRegistry(tc.commands...)
			if (err != nil) != tc.wantErr {
				t.Fatalf("NewRegistry() error = %v; want error %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			if got := registry.Names(); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("Names() = %v; want %v", got, tc.want)
			}
		})
	}
}

func TestDefaultRegistryContainsPlanCommands(t *testing.T) {
	registry, err := NewDefaultRegistry()
	if err != nil {
		t.Fatal(err)
	}
	command, ok := registry.Lookup("plan")
	if !ok {
		t.Fatal("plan command is not registered")
	}
	group, ok := command.(Group)
	if !ok {
		t.Fatal("plan command is not a command group")
	}
	if got, want := group.Subcommands().Names(), []string{"run", "show"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("plan commands = %v; want %v", got, want)
	}
}
