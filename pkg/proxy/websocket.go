package proxy

import (
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/utkarshrai2811/proxima/pkg/ws"
)

//nolint:gochecknoglobals
var wsUpgrader = websocket.Upgrader{
	CheckOrigin:     func(*http.Request) bool { return true },
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
}

// isWebSocketUpgrade reports whether r is a WebSocket upgrade request.
func isWebSocketUpgrade(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("Upgrade"), "websocket") &&
		strings.Contains(strings.ToLower(r.Header.Get("Connection")), "upgrade")
}

// handleWebSocket intercepts a WebSocket upgrade: it dials the upstream, upgrades
// the client connection, then relays messages in both directions while logging
// every frame to the session store. Injected frames (from the UI) are written to
// the upstream connection under the same write lock.
func (p *Proxy) handleWebSocket(w http.ResponseWriter, r *http.Request, reqID string) {
	scheme := "ws"
	if r.TLS != nil {
		scheme = "wss"
	}

	target := url.URL{Scheme: scheme, Host: r.Host, Path: r.URL.Path, RawQuery: r.URL.RawQuery}

	// Forward request headers to the upstream, minus the handshake headers that
	// the dialer sets itself.
	reqHeader := http.Header{}

	for k, vv := range r.Header {
		switch strings.ToLower(k) {
		case "upgrade", "connection", "sec-websocket-key", "sec-websocket-version",
			"sec-websocket-extensions", "sec-websocket-protocol":
		default:
			reqHeader[k] = vv
		}
	}

	dialer := *websocket.DefaultDialer
	dialer.Subprotocols = websocket.Subprotocols(r)

	serverConn, resp, err := dialer.Dial(target.String(), reqHeader)
	if err != nil {
		p.logger.Errorw("WebSocket upstream dial failed.", "error", err, "url", target.String())

		if resp != nil {
			copyHeader(w.Header(), resp.Header)
			w.WriteHeader(resp.StatusCode)
		} else {
			writeError(w, http.StatusBadGateway)
		}

		return
	}
	defer serverConn.Close()

	respHeader := http.Header{}
	if proto := serverConn.Subprotocol(); proto != "" {
		respHeader.Set("Sec-WebSocket-Protocol", proto)
	}

	clientConn, err := wsUpgrader.Upgrade(w, r, respHeader)
	if err != nil {
		p.logger.Errorw("WebSocket client upgrade failed.", "error", err)

		return
	}
	defer clientConn.Close()

	sessID := ws.NewID()
	p.wsStore.CreateSession(ws.Session{
		ID:         sessID,
		RequestID:  reqID,
		URL:        target.String(),
		RemoteAddr: r.RemoteAddr,
		StartTime:  time.Now(),
		Open:       true,
	})
	defer p.wsStore.CloseSession(sessID)

	// All writes to the upstream connection (relayed client frames + injected
	// frames) are serialized: gorilla connections do not allow concurrent writes.
	var serverWriteMu sync.Mutex

	p.wsStore.RegisterSender(sessID, func(opcode int, payload []byte) error {
		serverWriteMu.Lock()
		defer serverWriteMu.Unlock()

		return serverConn.WriteMessage(opcode, payload)
	})

	errc := make(chan error, 2)

	// client -> server
	go func() {
		for {
			mt, payload, err := clientConn.ReadMessage()
			if err != nil {
				errc <- err

				return
			}

			p.wsStore.AddFrame(ws.Frame{
				SessionID: sessID, Direction: ws.ClientToServer, Opcode: mt,
				Payload: payload, Timestamp: time.Now(),
			})

			serverWriteMu.Lock()
			err = serverConn.WriteMessage(mt, payload)
			serverWriteMu.Unlock()

			if err != nil {
				errc <- err

				return
			}
		}
	}()

	// server -> client
	go func() {
		for {
			mt, payload, err := serverConn.ReadMessage()
			if err != nil {
				errc <- err

				return
			}

			p.wsStore.AddFrame(ws.Frame{
				SessionID: sessID, Direction: ws.ServerToClient, Opcode: mt,
				Payload: payload, Timestamp: time.Now(),
			})

			if err := clientConn.WriteMessage(mt, payload); err != nil {
				errc <- err

				return
			}
		}
	}()

	<-errc
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}
