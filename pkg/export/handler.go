package export

import (
	"encoding/json"
	"net/http"
)

// Result is the response of an export request.
type Result struct {
	Content  string `json:"content"`
	Filename string `json:"filename"`
	MimeType string `json:"mimeType"`
}

// Handler returns an http.Handler for POST /api/export. The resolve function
// maps the requested IDs to entries (keeping this package free of storage
// dependencies).
func Handler(resolve func(ids []string) ([]Entry, error)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

			return
		}

		var body struct {
			IDs          []string `json:"ids"`
			Format       string   `json:"format"`
			IncludeProxy bool     `json:"includeProxy"`
		}

		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)

			return
		}

		entries, err := resolve(body.IDs)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)

			return
		}

		var (
			content   []byte
			result    Result
			exportErr error
		)

		switch body.Format {
		case "BURP_XML":
			content, exportErr = ExportBurpXML(entries)
			result.Filename, result.MimeType = "proxima_export.xml", "text/xml"
		case "CURL":
			content = []byte(ExportCurlAll(entries, body.IncludeProxy))
			result.Filename, result.MimeType = "proxima_export.sh", "text/plain"
		case "OPENAPI":
			content, exportErr = ExportOpenAPI(entries)
			result.Filename, result.MimeType = "openapi.yaml", "text/yaml"
		default:
			http.Error(w, "unknown export format", http.StatusBadRequest)

			return
		}

		if exportErr != nil {
			http.Error(w, exportErr.Error(), http.StatusInternalServerError)

			return
		}

		result.Content = string(content)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(result)
	})
}
