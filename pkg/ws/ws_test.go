package ws

import (
	"encoding/base64"
	"errors"
	"testing"
	"time"
)

func TestDirectionString(t *testing.T) {
	t.Parallel()

	if ClientToServer.String() != "CLIENT_TO_SERVER" {
		t.Errorf("got %q", ClientToServer.String())
	}

	if ServerToClient.String() != "SERVER_TO_CLIENT" {
		t.Errorf("got %q", ServerToClient.String())
	}
}

func TestPayloadString(t *testing.T) {
	t.Parallel()

	if got := PayloadString(1, []byte("hello")); got != "hello" {
		t.Errorf("text payload = %q, want hello", got)
	}

	bin := []byte{0x00, 0x01, 0xff}
	if got := PayloadString(2, bin); got != base64.StdEncoding.EncodeToString(bin) {
		t.Errorf("binary payload should be base64, got %q", got)
	}
}

func TestStoreSessionsAndFrames(t *testing.T) {
	t.Parallel()

	s := NewStore()
	s.CreateSession(Session{ID: "s1", URL: "wss://a", Open: true})
	s.CreateSession(Session{ID: "s2", URL: "wss://b", Open: true})

	if got := s.ListSessions(false); len(got) != 2 {
		t.Fatalf("ListSessions = %d, want 2", len(got))
	}

	s.AddFrame(Frame{SessionID: "s1", Direction: ClientToServer, Opcode: 1, Payload: []byte("hi")})
	s.AddFrame(Frame{SessionID: "s1", Direction: ServerToClient, Opcode: 1, Payload: []byte("yo")})

	frames := s.ListFrames("s1")
	if len(frames) != 2 {
		t.Fatalf("ListFrames = %d, want 2", len(frames))
	}

	if frames[0].ID == "" || frames[0].Size != 2 {
		t.Errorf("frame not finalized: %+v", frames[0])
	}

	sess, ok := s.GetSession("s1")
	if !ok || sess.FrameCount != 2 {
		t.Fatalf("session frame count = %d, want 2", sess.FrameCount)
	}

	s.CloseSession("s1")

	sess, _ = s.GetSession("s1")
	if sess.Open || sess.EndTime == nil {
		t.Error("session should be closed with an end time")
	}

	if got := s.ListSessions(true); len(got) != 1 || got[0].ID != "s2" {
		t.Errorf("active-only sessions = %+v, want only s2", got)
	}
}

func TestStoreSubscribe(t *testing.T) {
	t.Parallel()

	s := NewStore()
	s.CreateSession(Session{ID: "s1", Open: true})

	ch, cancel := s.Subscribe("s1")
	defer cancel()

	s.AddFrame(Frame{SessionID: "s1", Opcode: 1, Payload: []byte("live")})

	select {
	case f := <-ch:
		if string(f.Payload) != "live" {
			t.Errorf("subscriber got %q", f.Payload)
		}
	case <-time.After(time.Second):
		t.Fatal("subscriber did not receive the frame")
	}
}

func TestStoreSend(t *testing.T) {
	t.Parallel()

	s := NewStore()
	s.CreateSession(Session{ID: "s1", Open: true})

	if _, err := s.Send("s1", 1, []byte("x")); !errors.Is(err, ErrSessionClosed) {
		t.Fatalf("Send without a registered sender should fail, got %v", err)
	}

	var sent []byte

	s.RegisterSender("s1", func(_ int, payload []byte) error {
		sent = payload

		return nil
	})

	frame, err := s.Send("s1", 1, []byte("inject"))
	if err != nil {
		t.Fatalf("Send: %v", err)
	}

	if string(sent) != "inject" {
		t.Errorf("sender received %q", sent)
	}

	if frame.Direction != ClientToServer || string(frame.Payload) != "inject" {
		t.Errorf("recorded frame wrong: %+v", frame)
	}
}
