package cmds

import (
	"encoding/json"
	"errors"
	"flag"
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
	_ = client.New(baseURL(cfg), cfg.Token).PostNotify(protocol.Notification{
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

// NotifyManual posts a notification from CLI flags (for testing/demos). Unlike
// the stdin hook path it reports errors so a down bridge is visible.
func NotifyManual(args []string) error {
	fs := flag.NewFlagSet("notify", flag.ContinueOnError)
	title := fs.String("title", "", "notification title")
	message := fs.String("message", "", "notification message")
	session := fs.String("session", "", "session id the toast belongs to")
	typ := fs.String("type", "manual", "notification type")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *title == "" && *message == "" {
		return errors.New("notify: at least one of --title or --message is required")
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	return client.New(baseURL(cfg), cfg.Token).PostNotify(protocol.Notification{
		Type:      *typ,
		Title:     *title,
		Message:   *message,
		SessionID: *session,
	})
}
