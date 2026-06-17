package ws

import (
	"errors"
	"sync"
	"time"
)

// ErrSessionClosed is returned when injecting a frame into a session that is no
// longer live (no registered sender).
var ErrSessionClosed = errors.New("ws: session is closed")

// Store is a concurrency-safe, in-memory store of WebSocket sessions and frames
// with a pub/sub mechanism for streaming new frames to subscribers (SSE).
//
// Frames are kept in memory only: WebSocket inspection is a live activity, so
// sessions reset when the proxy restarts.
type Store struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	order    []string
	frames   map[string][]Frame
	subs     map[string]map[chan Frame]struct{}
	senders  map[string]func(opcode int, payload []byte) error
}

func NewStore() *Store {
	return &Store{
		sessions: make(map[string]*Session),
		frames:   make(map[string][]Frame),
		subs:     make(map[string]map[chan Frame]struct{}),
		senders:  make(map[string]func(opcode int, payload []byte) error),
	}
}

// CreateSession records a new session.
func (s *Store) CreateSession(sess Session) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cp := sess
	s.sessions[sess.ID] = &cp
	s.order = append(s.order, sess.ID)
}

// GetSession returns a copy of the session and whether it exists.
func (s *Store) GetSession(id string) (Session, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if sp, ok := s.sessions[id]; ok {
		return *sp, true
	}

	return Session{}, false
}

// ListSessions returns all sessions in creation order, optionally only open ones.
func (s *Store) ListSessions(activeOnly bool) []Session {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]Session, 0, len(s.order))

	for _, id := range s.order {
		sp := s.sessions[id]
		if activeOnly && !sp.Open {
			continue
		}

		out = append(out, *sp)
	}

	return out
}

// CloseSession marks a session closed and removes its live sender.
func (s *Store) CloseSession(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if sp, ok := s.sessions[id]; ok && sp.Open {
		now := time.Now()
		sp.Open = false
		sp.EndTime = &now
	}

	delete(s.senders, id)
}

// AddFrame records a frame, increments the session counter, and publishes it to
// any subscribers. It returns the stored frame (with ID/timestamp/size filled).
func (s *Store) AddFrame(f Frame) Frame {
	s.mu.Lock()

	if f.ID == "" {
		f.ID = NewID()
	}

	if f.Timestamp.IsZero() {
		f.Timestamp = time.Now()
	}

	f.Size = len(f.Payload)
	s.frames[f.SessionID] = append(s.frames[f.SessionID], f)

	if sp, ok := s.sessions[f.SessionID]; ok {
		sp.FrameCount++
	}

	chans := make([]chan Frame, 0, len(s.subs[f.SessionID]))
	for ch := range s.subs[f.SessionID] {
		chans = append(chans, ch)
	}

	s.mu.Unlock()

	for _, ch := range chans {
		select {
		case ch <- f:
		default: // drop for slow subscribers; they also re-fetch the full list
		}
	}

	return f
}

// ListFrames returns a copy of the frames for a session.
func (s *Store) ListFrames(sessionID string) []Frame {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return append([]Frame(nil), s.frames[sessionID]...)
}

// Subscribe returns a channel that receives new frames for a session and a
// cancel function to stop receiving.
func (s *Store) Subscribe(sessionID string) (<-chan Frame, func()) {
	ch := make(chan Frame, 64)

	s.mu.Lock()
	if s.subs[sessionID] == nil {
		s.subs[sessionID] = make(map[chan Frame]struct{})
	}
	s.subs[sessionID][ch] = struct{}{}
	s.mu.Unlock()

	cancel := func() {
		s.mu.Lock()
		delete(s.subs[sessionID], ch)
		s.mu.Unlock()
	}

	return ch, cancel
}

// RegisterSender exposes a way to inject a frame into a live session. The proxy
// registers this when relaying; it is removed when the session closes.
func (s *Store) RegisterSender(sessionID string, send func(opcode int, payload []byte) error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.senders[sessionID] = send
}

// Send injects a frame into a live session (client → server) and records it.
func (s *Store) Send(sessionID string, opcode int, payload []byte) (Frame, error) {
	s.mu.RLock()
	send := s.senders[sessionID]
	s.mu.RUnlock()

	if send == nil {
		return Frame{}, ErrSessionClosed
	}

	if err := send(opcode, payload); err != nil {
		return Frame{}, err
	}

	return s.AddFrame(Frame{
		SessionID: sessionID,
		Direction: ClientToServer,
		Opcode:    opcode,
		Payload:   payload,
	}), nil
}
