package api

import (
	"net"
	"net/http"
	"strings"
)

// HostAllowlistMiddleware rejects requests whose Host header is not in the
// allowlist. This prevents DNS rebinding attacks against the local admin
// interface. The proxy traffic endpoint is never wrapped by this middleware.
func HostAllowlistMiddleware(allowed []string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}

		for _, a := range allowed {
			if strings.EqualFold(host, a) {
				next.ServeHTTP(w, r)

				return
			}
		}

		http.Error(w, "403 forbidden: host not in allowlist", http.StatusForbidden)
	})
}
