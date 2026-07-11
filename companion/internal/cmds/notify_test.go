package cmds

import (
	"net/http/httptest"
	"testing"

	"xeneoncc/internal/config"
	"xeneoncc/internal/server"
	"xeneoncc/internal/store"
)

// withBridge stands up a real store+server on an httptest URL and points config
// + baseURL at it, so NotifyManual exercises the full POST path.
func withBridge(t *testing.T) *store.Store {
	t.Helper()
	t.Setenv("XENEON_BRIDGE_HOME", t.TempDir())
	if err := config.Save(config.Config{Port: 8787, Token: "tok"}); err != nil {
		t.Fatalf("save config: %v", err)
	}
	st := store.New(func() int64 { return 42 })
	srv := httptest.NewServer(server.New(st, "tok"))
	t.Cleanup(srv.Close)
	t.Setenv("XENEON_BRIDGE_URL", srv.URL)
	return st
}

func TestNotifyManualPostsToBridge(t *testing.T) {
	st := withBridge(t)
	err := NotifyManual([]string{"--title", "Permission needed", "--message", "Bash", "--session", "s1", "--type", "permission"})
	if err != nil {
		t.Fatalf("NotifyManual: %v", err)
	}
	got := st.Snapshot().Notifications
	if len(got) != 1 {
		t.Fatalf("want 1 notification, got %d", len(got))
	}
	n := got[0]
	if n.Title != "Permission needed" || n.Message != "Bash" || n.SessionID != "s1" || n.Type != "permission" {
		t.Fatalf("unexpected notification: %+v", n)
	}
	if n.ID != 1 {
		t.Fatalf("want stamped id 1, got %d", n.ID)
	}
}

func TestNotifyManualRequiresTitleOrMessage(t *testing.T) {
	if err := NotifyManual([]string{"--session", "s1"}); err == nil {
		t.Fatal("want error when neither --title nor --message is given")
	}
}

func TestNotifyManualErrorsWhenBridgeDown(t *testing.T) {
	t.Setenv("XENEON_BRIDGE_HOME", t.TempDir())
	if err := config.Save(config.Config{Port: 8787, Token: "tok"}); err != nil {
		t.Fatalf("save config: %v", err)
	}
	t.Setenv("XENEON_BRIDGE_URL", "http://127.0.0.1:1") // nothing listening
	if err := NotifyManual([]string{"--title", "x"}); err == nil {
		t.Fatal("want error when the bridge is unreachable")
	}
}

func TestNotifyManualErrorsWhenBridgeRejects(t *testing.T) {
	t.Setenv("XENEON_BRIDGE_HOME", t.TempDir())
	if err := config.Save(config.Config{Port: 8787, Token: "tok"}); err != nil {
		t.Fatalf("save config: %v", err)
	}
	// Server expects a DIFFERENT token, so the POST is rejected (401).
	srv := httptest.NewServer(server.New(store.New(func() int64 { return 42 }), "other"))
	t.Cleanup(srv.Close)
	t.Setenv("XENEON_BRIDGE_URL", srv.URL)
	if err := NotifyManual([]string{"--title", "x"}); err == nil {
		t.Fatal("want error when the bridge rejects the POST (401)")
	}
}
