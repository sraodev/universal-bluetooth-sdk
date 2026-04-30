// Package sockaddr resolves the default Unix socket path for the daemon.
//
// Centralised so ubtd and ubtctl can never disagree.
package sockaddr

import (
	"os"
	"path/filepath"
)

// Default returns the path ubtd listens on (and ubtctl dials) by default.
//
//   - Honour UBTD_SOCKET if set.
//   - Otherwise prefer $XDG_RUNTIME_DIR/ubtd.sock (per-user, tmpfs).
//   - Fall back to /tmp/ubtd.sock for portability.
func Default() string {
	if s := os.Getenv("UBTD_SOCKET"); s != "" {
		return s
	}
	if r := os.Getenv("XDG_RUNTIME_DIR"); r != "" {
		return filepath.Join(r, "ubtd.sock")
	}
	return "/tmp/ubtd.sock"
}
