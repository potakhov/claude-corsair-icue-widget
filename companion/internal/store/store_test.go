package store

import (
	"testing"

	"xeneoncc/internal/protocol"
)

func TestAddNotificationStampsIncrementingIDsAndClock(t *testing.T) {
	clock := int64(1000)
	s := New(func() int64 { return clock })

	s.AddNotification(protocol.Notification{Title: "a"})
	clock = 1001
	s.AddNotification(protocol.Notification{Title: "b"})

	got := s.Snapshot().Notifications
	if len(got) != 2 {
		t.Fatalf("want 2 notifications, got %d", len(got))
	}
	if got[0].ID != 1 || got[1].ID != 2 {
		t.Fatalf("want ids 1,2 got %d,%d", got[0].ID, got[1].ID)
	}
	if got[0].At != 1000 || got[1].At != 1001 {
		t.Fatalf("want At 1000,1001 got %d,%d", got[0].At, got[1].At)
	}
}

func TestNotificationBufferCapsAtTenKeepingClimbingIDs(t *testing.T) {
	s := New(func() int64 { return 5 })
	for i := 0; i < 12; i++ {
		s.AddNotification(protocol.Notification{})
	}
	got := s.Snapshot().Notifications
	if len(got) != 10 {
		t.Fatalf("want 10 kept, got %d", len(got))
	}
	// Oldest two (ids 1,2) dropped; ids keep climbing regardless of the cap.
	if got[0].ID != 3 || got[9].ID != 12 {
		t.Fatalf("want ids 3..12, got %d..%d", got[0].ID, got[9].ID)
	}
}
