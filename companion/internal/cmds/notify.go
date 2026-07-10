package cmds

import (
	"encoding/json"
	"io"

	"xeneoncc/internal/claudecode"
	"xeneoncc/internal/client"
	"xeneoncc/internal/config"
	"xeneoncc/internal/protocol"
)

// Notify posts a device alert. It must never block or fail Claude Code, so it
// swallows every error and returns nil.
func Notify(stdin io.Reader) error {
	raw, err := io.ReadAll(stdin)
	if err != nil {
		return nil
	}
	var in claudecode.NotificationInput
	if json.Unmarshal(raw, &in) != nil {
		return nil
	}
	cfg, err := config.Load()
	if err != nil {
		return nil
	}
	_ = client.New(baseURL(cfg.Port), cfg.Token).PostNotify(protocol.Notification{
		Type:      firstNonEmpty(in.NotificationType, in.HookEventName),
		Title:     in.Title,
		Message:   in.Message,
		SessionID: in.SessionID,
	})
	return nil
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
