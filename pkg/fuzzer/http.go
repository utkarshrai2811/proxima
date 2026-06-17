package fuzzer

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

type attackDTO struct {
	ID             string     `json:"id"`
	Name           string     `json:"name"`
	Type           string     `json:"type"`
	BaseRequest    string     `json:"baseRequest"`
	Status         string     `json:"status"`
	CreatedAt      time.Time  `json:"createdAt"`
	StartedAt      *time.Time `json:"startedAt"`
	FinishedAt     *time.Time `json:"finishedAt"`
	TotalRequests  int        `json:"totalRequests"`
	CompletedCount int        `json:"completedCount"`
	ErrorCount     int        `json:"errorCount"`
}

type resultDTO struct {
	ID             string            `json:"id"`
	AttackID       string            `json:"attackId"`
	RequestIndex   int               `json:"requestIndex"`
	PayloadValues  map[string]string `json:"payloadValues"`
	RawRequest     string            `json:"rawRequest"`
	RawResponse    string            `json:"rawResponse"`
	StatusCode     int               `json:"statusCode"`
	ResponseSize   int               `json:"responseSize"`
	ResponseTimeMs int64             `json:"responseTimeMs"`
	IsError        bool              `json:"isError"`
	ErrorMessage   string            `json:"errorMessage"`
}

func toAttackDTO(a Attack) attackDTO {
	return attackDTO{
		ID: a.ID, Name: a.Name, Type: string(a.Type), BaseRequest: a.BaseRequest,
		Status: string(a.Status), CreatedAt: a.CreatedAt, StartedAt: a.StartedAt,
		FinishedAt: a.FinishedAt, TotalRequests: a.TotalRequests,
		CompletedCount: a.CompletedCount, ErrorCount: a.ErrorCount,
	}
}

func toResultDTO(r FuzzResult) resultDTO {
	return resultDTO{
		ID: r.ID, AttackID: r.AttackID, RequestIndex: r.RequestIndex,
		PayloadValues: r.PayloadValues, RawRequest: r.RawRequest, RawResponse: r.RawResponse,
		StatusCode: r.StatusCode, ResponseSize: r.ResponseSize, ResponseTimeMs: r.ResponseTimeMs,
		IsError: r.IsError, ErrorMessage: r.ErrorMessage,
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

type createInput struct {
	Name           string `json:"name"`
	Type           string `json:"type"`
	BaseRequest    string `json:"baseRequest"`
	Concurrency    int    `json:"concurrency"`
	PayloadSources []struct {
		Type     string   `json:"type"`
		Values   []string `json:"values"`
		BuiltIn  string   `json:"builtIn"`
		RangeMin int      `json:"rangeMin"`
		RangeMax int      `json:"rangeMax"`
	} `json:"payloadSources"`
}

// Handler returns an http.Handler serving the fuzzer REST + SSE API under
// /api/fuzzer/. Intended to be mounted on the (auth-gated) admin router.
func Handler(m *Manager) http.Handler {
	r := mux.NewRouter()

	r.HandleFunc("/api/fuzzer/attacks", func(w http.ResponseWriter, r *http.Request) {
		var in createInput
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)

			return
		}

		sources := make([]PayloadSource, 0, len(in.PayloadSources))
		for _, s := range in.PayloadSources {
			sources = append(sources, PayloadSource{
				Type: PayloadSourceType(s.Type), Values: s.Values,
				BuiltIn: BuiltInList(s.BuiltIn), RangeMin: s.RangeMin, RangeMax: s.RangeMax,
			})
		}

		attack, err := m.CreateAttack(AttackInput{
			Name: in.Name, Type: AttackType(in.Type), BaseRequest: in.BaseRequest,
			PayloadSources: sources, Concurrency: in.Concurrency,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		}

		writeJSON(w, toAttackDTO(*attack))
	}).Methods(http.MethodPost)

	r.HandleFunc("/api/fuzzer/attacks", func(w http.ResponseWriter, _ *http.Request) {
		attacks := m.ListAttacks()
		out := make([]attackDTO, 0, len(attacks))

		for _, a := range attacks {
			out = append(out, toAttackDTO(a))
		}

		writeJSON(w, out)
	}).Methods(http.MethodGet)

	r.HandleFunc("/api/fuzzer/attacks/{id}", func(w http.ResponseWriter, r *http.Request) {
		a, ok := m.GetAttack(mux.Vars(r)["id"])
		if !ok {
			http.Error(w, "attack not found", http.StatusNotFound)

			return
		}

		writeJSON(w, toAttackDTO(a))
	}).Methods(http.MethodGet)

	action := func(fn func(string) error) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			id := mux.Vars(r)["id"]
			if err := fn(id); err != nil {
				code := http.StatusBadRequest
				if errors.Is(err, ErrNotFound) {
					code = http.StatusNotFound
				}

				http.Error(w, err.Error(), code)

				return
			}

			a, _ := m.GetAttack(id)
			writeJSON(w, toAttackDTO(a))
		}
	}

	r.HandleFunc("/api/fuzzer/attacks/{id}/start", action(m.Start)).Methods(http.MethodPost)
	r.HandleFunc("/api/fuzzer/attacks/{id}/pause", action(m.Pause)).Methods(http.MethodPost)
	r.HandleFunc("/api/fuzzer/attacks/{id}/cancel", action(m.Cancel)).Methods(http.MethodPost)

	r.HandleFunc("/api/fuzzer/attacks/{id}/results", func(w http.ResponseWriter, r *http.Request) {
		results := m.ListResults(mux.Vars(r)["id"])
		out := make([]resultDTO, 0, len(results))

		for _, res := range results {
			out = append(out, toResultDTO(res))
		}

		writeJSON(w, out)
	}).Methods(http.MethodGet)

	r.HandleFunc("/api/fuzzer/attacks/{id}/stream", func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)

			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		ch, cancel := m.Subscribe(mux.Vars(r)["id"])
		defer cancel()

		flusher.Flush()

		for {
			select {
			case <-r.Context().Done():
				return
			case res := <-ch:
				data, err := json.Marshal(toResultDTO(res))
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
