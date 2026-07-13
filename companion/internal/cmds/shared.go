package cmds

import (
	"fmt"
	"os"
	"strings"

	"xeneoncc/internal/config"
)

// baseURL is where client subcommands reach the bridge:
// XENEON_BRIDGE_URL env override → config.url → local loopback:port.
func baseURL(cfg config.Config) string {
	if v := os.Getenv("XENEON_BRIDGE_URL"); v != "" {
		return v
	}
	if cfg.URL != "" {
		return strings.TrimRight(cfg.URL, "/")
	}
	return fmt.Sprintf("http://127.0.0.1:%d", cfg.Port)
}
