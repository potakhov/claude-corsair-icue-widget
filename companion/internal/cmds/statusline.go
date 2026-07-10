package cmds

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"

	"xeneoncc/internal/claudecode"
	"xeneoncc/internal/client"
	"xeneoncc/internal/config"
	"xeneoncc/internal/protocol"
)

func Statusline(stdin io.Reader, stdout io.Writer) error {
	raw, err := io.ReadAll(stdin)
	if err != nil {
		return err
	}

	// Best-effort POST; never block the statusline on bridge issues.
	var in claudecode.StatuslineInput
	if json.Unmarshal(raw, &in) == nil {
		if cfg, err := config.Load(); err == nil {
			_ = client.New(baseURL(cfg.Port), cfg.Token).PostUsage(mapUsage(in))
		}
	}

	// Passthrough: wrap an existing statusline if configured, else emit our own.
	if wrap := os.Getenv("XENEON_WRAP_CMD"); wrap != "" {
		cmd := exec.Command("cmd", "/C", wrap)
		cmd.Stdin = bytes.NewReader(raw)
		cmd.Stdout = stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	fmt.Fprint(stdout, renderLine(in))
	return nil
}

func mapUsage(in claudecode.StatuslineInput) protocol.Usage {
	u := protocol.Usage{Model: in.Model.DisplayName, SessionID: in.SessionID}
	u.Folder = baseName(in.Workspace.ProjectDir)
	cost := in.Cost.TotalCostUSD
	u.CostUSD = &cost
	dur := in.Cost.TotalDurationMs
	u.DurationMs = &dur
	ctx := in.ContextWindow.UsedPercentage
	u.ContextPct = &ctx
	if in.RateLimits != nil {
		if w := in.RateLimits.FiveHour; w != nil {
			p, r := w.UsedPercentage, w.ResetsAt
			u.FiveHourPct, u.FiveHourResetsAt = &p, &r
		}
		if w := in.RateLimits.SevenDay; w != nil {
			p, r := w.UsedPercentage, w.ResetsAt
			u.WeeklyPct, u.WeeklyResetsAt = &p, &r
		}
	}
	return u
}

func renderLine(in claudecode.StatuslineInput) string {
	s := in.Model.DisplayName
	if in.RateLimits != nil && in.RateLimits.SevenDay != nil {
		s += fmt.Sprintf(" | wk %.0f%%", in.RateLimits.SevenDay.UsedPercentage)
	}
	return s + fmt.Sprintf(" | $%.3f", in.Cost.TotalCostUSD)
}

// baseName returns the last path segment of p, handling both / and \
// separators (the statusline runs through bash, so either can arrive) and
// ignoring trailing separators. Empty input yields "".
func baseName(p string) string {
	end := len(p)
	for end > 0 && (p[end-1] == '/' || p[end-1] == '\\') {
		end--
	}
	start := end
	for start > 0 && p[start-1] != '/' && p[start-1] != '\\' {
		start--
	}
	return p[start:end]
}
