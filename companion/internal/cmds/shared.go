package cmds

import (
	"fmt"
	"os"
)

// baseURL is where subcommands reach the bridge. XENEON_BRIDGE_URL overrides
// the default loopback address (used by tests and non-default setups).
func baseURL(port int) string {
	if v := os.Getenv("XENEON_BRIDGE_URL"); v != "" {
		return v
	}
	return fmt.Sprintf("http://127.0.0.1:%d", port)
}
