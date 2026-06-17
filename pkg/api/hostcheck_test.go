package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestHostAllowlistMiddleware covers hetty PR#108: the admin interface must
// reject requests whose Host header is not in the allowlist, defeating DNS
// rebinding attacks.
func TestHostAllowlistMiddleware(t *testing.T) {
	t.Parallel()

	allowed := []string{"localhost", "127.0.0.1", "proxima.local"}
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := HostAllowlistMiddleware(allowed, next)

	tests := []struct {
		name string
		host string
		want int
	}{
		{"localhost", "localhost", http.StatusOK},
		{"loopback ip", "127.0.0.1", http.StatusOK},
		{"disallowed host", "evil.com", http.StatusForbidden},
		{"localhost with port", "localhost:8080", http.StatusOK},
		{"custom allowed host", "proxima.local", http.StatusOK},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Host = tc.host
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tc.want {
				t.Errorf("Host %q: got status %d, want %d", tc.host, rec.Code, tc.want)
			}
		})
	}
}
