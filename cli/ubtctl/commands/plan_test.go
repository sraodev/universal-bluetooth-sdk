package commands

import (
	"os"
	"path/filepath"
	"testing"
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
