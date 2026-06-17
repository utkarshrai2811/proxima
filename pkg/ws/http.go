package ws

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

type sessionDTO struct {
	ID         string     `json:"id"`
	RequestID  string     `json:"requestId"`
	URL        string     `json:"url"`
	RemoteAddr string     `json:"remoteAddr"`
	StartTime  time.Time  `json:"startTime"`
	EndTime    *time.Time `json:"endTime"`
	Open       bool       `json:"open"`
	FrameCount int        `json:"frameCount"`
}

type frameDTO struct {
	ID         string    `json:"id"`
	SessionID  string    `json:"sessionId"`
	Direction  string    `json:"direction"`
	Timestamp  time.Time `json:"timestamp"`
	Opcode     string    `json:"opcode"`
	Payload    string    `json:"payload"`
	PayloadHex string    `json:"payloadHex"`
	Size       int       `json:"size"`
}

func toSessionDTO(s Session) sessionDTO {
	return sessionDTO{
		ID: s.ID, RequestID: s.RequestID, URL: s.URL, RemoteAddr: s.RemoteAddr,
		StartTime: s.StartTime, EndTime: s.EndTime, Open: s.Open, FrameCount: s.FrameCount,
	}
}

func toFrameDTO(f Frame) frameDTO {
	return frameDTO{
		ID: f.ID, SessionID: f.SessionID, Direction: f.Direction.String(),
		Timestamp: f.Timestamp, Opcode: OpcodeString(f.Opcode),
		Payload: PayloadString(f.Opcode, f.Payload), PayloadHex: PayloadHex(f.Payload), Size: f.Size,
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// Handler returns an http.Handler serving the WebSocket REST + SSE API under
// /api/ws/. It is intended to be mounted on the (auth-gated) admin router.
func Handler(store *Store) http.Handler {
	r := mux.NewRouter()

	r.HandleFunc("/api/ws/sessions", func(w http.ResponseWriter, r *http.Request) {
		activeOnly := r.URL.Query().Get("active") == "true"
		sessions := store.ListSessions(activeOnly)
		out := make([]sessionDTO, 0, len(sessions))

		for _, s := range sessions {
			out = append(out, toSessionDTO(s))
		}

		writeJSON(w, out)
	}).Methods(http.MethodGet)

	r.HandleFunc("/api/ws/sessions/{id}/frames", func(w http.ResponseWriter, r *http.Request) {
		frames := store.ListFrames(mux.Vars(r)["id"])
		out := make([]frameDTO, 0, len(frames))

		for _, f := range frames {
			out = append(out, toFrameDTO(f))
		}

		writeJSON(w, out)
	}).Methods(http.MethodGet)

	r.HandleFunc("/api/ws/sessions/{id}/frames", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Payload string `json:"payload"`
			Opcode  string `json:"opcode"`
		}

		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)

			return
		}

		opcode := 1 // TEXT
		payload := []byte(body.Payload)

		if body.Opcode == "BINARY" {
			opcode = 2

			decoded, err := base64.StdEncoding.DecodeString(body.Payload)
			if err != nil {
				http.Error(w, "binary payload must be base64", http.StatusBadRequest)

				return
			}

			payload = decoded
		}

		frame, err := store.Send(mux.Vars(r)["id"], opcode, payload)
		if err != nil {
			http.Error(w, err.Error(), http.StatusConflict)

			return
		}

		writeJSON(w, toFrameDTO(frame))
	}).Methods(http.MethodPost)

	r.HandleFunc("/api/ws/sessions/{id}/stream", func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)

			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		ch, cancel := store.Subscribe(mux.Vars(r)["id"])
		defer cancel()

		flusher.Flush()

		for {
			select {
			case <-r.Context().Done():
				return
			case f := <-ch:
				data, err := json.Marshal(toFrameDTO(f))
				if err != nil {
					continue
				}

				fmt.Fprintf(w, "data: %s\n\n", data)
				flusher.Flush()
			}
		}
	}).Methods(http.MethodGet)

	return r
}
