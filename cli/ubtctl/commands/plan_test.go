package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestPlanDryRunWithoutDaemon(t *testing.T) {
	dir := t.TempDir()
	plan := filepath.Join(dir, "plan.json")
	if err := os.WriteFile(plan, []byte(`{"steps":[{"tool":"send_payload","arguments":{"address":"AA:BB:CC:DD:EE:01","payload":"hi"}}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := planRun([]string{"--socket", filepath.Join(dir, "missing.sock"), "--dry-run", plan}); err != nil {
		t.Fatal(err)
	}
	if err := planRun([]string{plan, "--yes"}); err == nil {
		t.Fatal("flags after path must not be silently ignored")
	}
}

func TestTrimPreservesUTF8(t *testing.T) {
	tests := []struct {
		name  string
		input string
		limit int
		want  string
	}{
		{"empty", "", 0, ""},
		{"ASCII under limit", "hello", 10, "hello"},
		{"ASCII at limit", "hello", 5, "hello"},
		{"ASCII truncated", "hello", 3, "hel…"},
		{"Unicode under limit", "日本", 7, "日本"},
		{"Unicode at limit", "日本", 6, "日本"},
		{"rune boundary", "日本語", 6, "日本…"},
		{"two-byte rune split", "aébc", 2, "a…"},
		{"three-byte rune split", "a日本", 3, "a…"},
		{"four-byte rune split", "a😀b", 4, "a…"},
		{"first rune exceeds budget", "😀", 2, "…"},
		{"zero budget", "hello", 0, "…"},
		{"plan result budget", "a" + strings.Repeat("日", 200), 240, "a" + strings.Repeat("日", 79) + "…"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := trim(tc.input, tc.limit)
			if !utf8.ValidString(got) {
				t.Errorf("trim produced invalid UTF-8: %q", got)
			}
			if got != tc.want {
				t.Errorf("trim(%q, %d) = %q; want %q", tc.input, tc.limit, got, tc.want)
			}
		})
	}
}
