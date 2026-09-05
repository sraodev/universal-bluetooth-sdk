package ai

import (
	"strings"
	"testing"
)

func TestSystemPromptUsesInvokedProgramName(t *testing.T) {
	for _, name := range []string{"ubt", "ubtctl"} {
		t.Run(name, func(t *testing.T) {
			prompt := SystemPromptFor(name, "/tmp/ubtd.sock", true)
			if !strings.Contains(prompt, "planner for "+name) {
				t.Fatalf("prompt does not identify %q", name)
			}
			if !strings.Contains(prompt, "corresponding "+name+" subcommand") {
				t.Fatalf("prompt does not use %q in its CLI example", name)
			}
		})
	}
}
