package proxy

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/utkarshrai2811/proxima/pkg/ws"
)

func TestIsWebSocketUpgrade(t *testing.T) {
	t.Parallel()

	up := httptest.NewRequest(http.MethodGet, "/", nil)
	up.Header.Set("Upgrade", "websocket")
	up.Header.Set("Connection", "Upgrade")

	if !isWebSocketUpgrade(up) {
		t.Error("expected upgrade request to be detected")
	}

	if isWebSocketUpgrade(httptest.NewRequest(http.MethodGet, "/", nil)) {
		t.Error("plain request should not be detected as an upgrade")
	}
}

func TestWebSocketRelay(t *testing.T) {
	t.Parallel()

	echoUpgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	echo := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := echoUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()

		for {
			mt, msg, err := c.ReadMessage()
			if err != nil {
				return
			}

			if err := c.WriteMessage(mt, msg); err != nil {
				return
			}
		}
	}))
	defer echo.Close()

	echoHost := strings.TrimPrefix(echo.URL, "http://")

	caCert, caKey, err := NewCA("Proxima Test", "Proxima Test CA", time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	store := ws.NewStore()

	p, err := NewProxy(Config{CACert: caCert, CAKey: caKey, WSStore: store})
	if err != nil {
		t.Fatal(err)
	}

	// A test server that points the proxy's WebSocket handler at the echo server.
	proxySrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Host = echoHost
		p.handleWebSocket(w, r, "req-1")
	}))
	defer proxySrv.Close()

	client, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(proxySrv.URL, "http"), nil)
	if err != nil {
		t.Fatalf("client dial: %v", err)
	}
	defer client.Close()

	if err := client.WriteMessage(websocket.TextMessage, []byte("ping")); err != nil {
		t.Fatal(err)
	}

	_, msg, err := client.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}

	if string(msg) != "ping" {
		t.Fatalf("echo = %q, want ping", msg)
	}

	// Frames are logged asynchronously; poll until both directions are recorded.
	deadline := time.After(2 * time.Second)

	for {
		sessions := store.ListSessions(false)
		if len(sessions) == 1 && sessions[0].FrameCount >= 2 {
			break
		}

		select {
		case <-deadline:
			t.Fatalf("expected one session with >=2 frames, got %+v", store.ListSessions(false))
		case <-time.After(10 * time.Millisecond):
		}
	}

	frames := store.ListFrames(store.ListSessions(false)[0].ID)

	var sawC2S, sawS2C bool

	for _, f := range frames {
		if f.Direction == ws.ClientToServer && string(f.Payload) == "ping" {
			sawC2S = true
		}

		if f.Direction == ws.ServerToClient && string(f.Payload) == "ping" {
			sawS2C = true
		}
	}

	if !sawC2S || !sawS2C {
		t.Errorf("expected both directions logged; frames=%+v", frames)
	}
}
