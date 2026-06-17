// Package ws provides WebSocket session/frame logging for the MITM proxy and an
// in-memory store with a pub/sub mechanism used to stream frames to the UI.
package ws

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"time"
)

// Direction is the travel direction of a WebSocket frame.
type Direction int

const (
	ClientToServer Direction = iota
	ServerToClient
)

func (d Direction) String() string {
	if d == ClientToServer {
		return "CLIENT_TO_SERVER"
	}

	return "SERVER_TO_CLIENT"
}

// Session represents a proxied WebSocket connection.
type Session struct {
	ID         string
	RequestID  string // links to the proxy log entry for the HTTP upgrade request
	URL        string
	RemoteAddr string
	StartTime  time.Time
	EndTime    *time.Time
	Open       bool
	FrameCount int
}

// Frame is a single WebSocket message observed on a session.
type Frame struct {
	ID        string
	SessionID string
	Direction Direction
	Timestamp time.Time
	Opcode    int // 1=text, 2=binary, 8=close, 9=ping, 10=pong
	Payload   []byte
	Size      int
}

// OpcodeString returns a human-readable opcode name.
func OpcodeString(opcode int) string {
	switch opcode {
	case 1:
		return "TEXT"
	case 2:
		return "BINARY"
	case 8:
		return "CLOSE"
	case 9:
		return "PING"
	case 10:
		return "PONG"
	default:
		return "UNKNOWN"
	}
}

// PayloadString returns UTF-8 text for text frames, base64 for everything else.
func PayloadString(opcode int, payload []byte) string {
	if opcode == 1 {
		return string(payload)
	}

	return base64.StdEncoding.EncodeToString(payload)
}

// PayloadHex always returns the hex-encoded payload (for binary inspection).
func PayloadHex(payload []byte) string {
	return hex.EncodeToString(payload)
}

// NewID returns a random 16-byte hex identifier.
func NewID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)

	return hex.EncodeToString(b)
}
