// Package ai is the AI front-end of ubtctl.
//
// It auto-derives Claude tools from the daemon's typed RPC surface, then
// runs Claude Opus with an adaptive thinking budget against a user-supplied
// natural-language goal. Every tool call funnels through the same daemon
// connection a typed CLI command would use, so there is exactly one
// execution path no matter who is driving.
package ai

// ExecMode controls how mutating tools (Send, in v1) behave during a run.
//
//   - ExecModeNormal   → run the call against the daemon
//   - ExecModeDryRun   → never touch the daemon for mutating tools; return
//                        a synthetic "would have called X" result so the
//                        model can finish planning without side effects
//   - ExecModeAutoYes  → like Normal, but skip interactive confirmation
type ExecMode int

const (
	ExecModeNormal ExecMode = iota
	ExecModeAutoYes
	ExecModeDryRun
)

func (m ExecMode) String() string {
	switch m {
	case ExecModeAutoYes:
		return "auto-yes"
	case ExecModeDryRun:
		return "dry-run"
	default:
		return "normal"
	}
}
