package plugin

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// Handler returns an http.Handler serving the plugin REST API under /api/plugins.
// Intended to be mounted on the (auth-gated) admin router.
func Handler(m *Manager) http.Handler {
	r := mux.NewRouter()

	r.HandleFunc("/api/plugins", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, m.List())
	}).Methods(http.MethodGet)

	r.HandleFunc("/api/plugins/open-folder", func(w http.ResponseWriter, _ *http.Request) {
		if err := OpenPluginDir(m.PluginDir()); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		writeJSON(w, map[string]bool{"ok": true})
	}).Methods(http.MethodPost)

	op := func(fn func(string) error) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if err := fn(mux.Vars(r)["name"]); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)

				return
			}

			// Return the updated plugin info.
			name := mux.Vars(r)["name"]
			for _, info := range m.List() {
				if info.Name == name {
					writeJSON(w, info)

					return
				}
			}

			writeJSON(w, map[string]bool{"ok": true})
		}
	}

	r.HandleFunc("/api/plugins/{name}/enable", op(m.Enable)).Methods(http.MethodPost)
	r.HandleFunc("/api/plugins/{name}/disable", op(m.Disable)).Methods(http.MethodPost)
	r.HandleFunc("/api/plugins/{name}/reload", op(m.Reload)).Methods(http.MethodPost)

	return r
}
