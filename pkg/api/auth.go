package api

import (
	"crypto/rand"
	"crypto/subtle"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
)

// AuthCookieName is the cookie that carries the API key for browser sessions.
const AuthCookieName = "proxima_key"

//go:embed login.html
var loginPage []byte

// GenerateAPIKey returns a cryptographically random 32-byte hex string.
func GenerateAPIKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return hex.EncodeToString(b), nil
}

// AuthMiddleware returns an http.Handler that enforces API key auth when apiKey
// is non-empty. If apiKey is empty, all requests pass through.
//
// The key may be presented as (checked in order):
//  1. Header: X-Proxima-Api-Key: <key>
//  2. Header: Authorization: Bearer <key>
//  3. Cookie: proxima_key=<key>
//
// Exempt paths (never gated): /login, /api/auth/login, /health.
func AuthMiddleware(apiKey string, next http.Handler) http.Handler {
	exemptPaths := map[string]bool{
		"/login":          true,
		"/api/auth/login": true,
		"/health":         true,
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if apiKey == "" {
			next.ServeHTTP(w, r)

			return
		}

		if exemptPaths[r.URL.Path] {
			next.ServeHTTP(w, r)

			return
		}

		var presented string

		switch {
		case r.Header.Get("X-Proxima-Api-Key") != "":
			presented = r.Header.Get("X-Proxima-Api-Key")
		case strings.HasPrefix(r.Header.Get("Authorization"), "Bearer "):
			presented = strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		default:
			if c, err := r.Cookie(AuthCookieName); err == nil {
				presented = c.Value
			}
		}

		// Constant-time comparison prevents timing attacks.
		if subtle.ConstantTimeCompare([]byte(presented), []byte(apiKey)) != 1 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})

			return
		}

		next.ServeHTTP(w, r)
	})
}

// LoginHandler validates the API key and sets the session cookie.
//
//	POST /api/auth/login
//	Body: {"key": "<api-key>"}
//
// On success it sets the proxima_key cookie and returns {"status":"ok"};
// on failure it returns 401 {"error":"invalid key"}.
func LoginHandler(apiKey string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

			return
		}

		var body struct {
			Key string `json:"key"`
		}

		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)

			return
		}

		if subtle.ConstantTimeCompare([]byte(body.Key), []byte(apiKey)) != 1 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid key"})

			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     AuthCookieName,
			Value:    body.Key,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
		})
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}
}

// LoginPageHandler serves the embedded HTML login page.
func LoginPageHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(loginPage)
	}
}
