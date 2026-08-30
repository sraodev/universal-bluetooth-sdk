package main

import (
	"testing"
)

func TestRun(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantCode int
	}{
		{
			name:     "no args returns 2",
			args:     []string{},
			wantCode: 2,
		},
		{
			name:     "root --help returns 0",
			args:     []string{"--help"},
			wantCode: 0,
		},
		{
			name:     "root -h returns 0",
			args:     []string{"-h"},
			wantCode: 0,
		},
		{
			name:     "root help returns 0",
			args:     []string{"help"},
			wantCode: 0,
		},
		{
			name:     "subcommand help via help verb returns 0",
			args:     []string{"help", "ping"},
			wantCode: 0,
		},
		{
			name:     "subcommand -h flag returns 0",
			args:     []string{"ping", "-h"},
			wantCode: 0,
		},
		{
			name:     "subcommand --help flag returns 0",
			args:     []string{"version", "--help"},
			wantCode: 0,
		},
		{
			name:     "unknown command returns 2",
			args:     []string{"nonexistent-cmd"},
			wantCode: 2,
		},
		{
			name:     "help for unknown command returns 2",
			args:     []string{"help", "nonexistent-cmd"},
			wantCode: 2,
		},
		{
			name:     "invalid flag for known command returns 1",
			args:     []string{"version", "--invalid-flag-that-does-not-exist"},
			wantCode: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := run(tc.args, "0.1.0-test")
			if got != tc.wantCode {
				t.Errorf("run(%v) = %d; want %d", tc.args, got, tc.wantCode)
			}
		})
	}
}
