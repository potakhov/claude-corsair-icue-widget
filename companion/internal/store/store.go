package store

import (
	"sync"

	"xeneoncc/internal/protocol"
)

// sessionTTLSecs is a generous hygiene cutoff: a session with no update for
// this long is pruned on read so the map can't grow unbounded across restarts.
// It must stay strictly looser than any reachable widget inactivity timeout so
// the widget — not the bridge — owns the "is this session live" decision. The
// widget's idleMinutes slider maxes at 180 min, so 4h leaves a comfortable margin.
const sessionTTLSecs = 4 * 3600

type Store struct {
	mu            sync.Mutex
	sessions      map[string]*protocol.Usage
	sessionOrder  []string
	notifications []protocol.Notification
	now           func() int64
}

func New(now func() int64) *Store {
	return &Store{
		sessions: map[string]*protocol.Usage{},
		now:      now,
	}
}

func (s *Store) SetUsage(u protocol.Usage) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u.ReceivedAt = s.now()
	if _, ok := s.sessions[u.SessionID]; !ok {
		s.sessionOrder = append(s.sessionOrder, u.SessionID)
	}
	cp := u
	s.sessions[u.SessionID] = &cp
}

func (s *Store) AddNotification(n protocol.Notification) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n.At = s.now()
	s.notifications = append(s.notifications, n)
	if len(s.notifications) > 10 {
		s.notifications = s.notifications[len(s.notifications)-10:]
	}
}

func (s *Store) Snapshot() protocol.State {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	st := protocol.State{BridgeTime: now}

	// Emit live sessions in first-seen order; prune stale ones as hygiene.
	kept := s.sessionOrder[:0]
	var mostRecent *protocol.Usage
	for _, id := range s.sessionOrder {
		u := s.sessions[id]
		if u == nil {
			continue
		}
		if now-u.ReceivedAt > sessionTTLSecs {
			delete(s.sessions, id)
			continue
		}
		kept = append(kept, id)
		st.Sessions = append(st.Sessions, *u)
		if mostRecent == nil || u.ReceivedAt >= mostRecent.ReceivedAt {
			mostRecent = u
		}
	}
	s.sessionOrder = kept
	if mostRecent != nil {
		mr := *mostRecent
		st.Usage = &mr
	}

	if len(s.notifications) > 0 {
		st.Notifications = append([]protocol.Notification{}, s.notifications...)
	}
	return st
}
