package api

import (
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const testAPIKey = "test-secret-key"

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
}

func doAuthReq(t *testing.T, key string, mutate func(*http.Request)) *httptest.ResponseRecorder {
	t.Helper()

	h := AuthMiddleware(key, okHandler())
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	if mutate != nil {
		mutate(req)
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	return rec
}

func TestAuthDisabled(t *testing.T) {
	t.Parallel()

	if rec := doAuthReq(t, "", nil); rec.Code != http.StatusOK {
		t.Fatalf("got %d, want 200 (auth disabled passes through)", rec.Code)
	}
}

func TestAuthEnabledCorrectHeader(t *testing.T) {
	t.Parallel()

	rec := doAuthReq(t, testAPIKey, func(r *http.Request) {
		r.Header.Set("X-Proxima-Api-Key", testAPIKey)
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", rec.Code)
	}
}

func TestAuthEnabledBearerToken(t *testing.T) {
	t.Parallel()

	rec := doAuthReq(t, testAPIKey, func(r *http.Request) {
		r.Header.Set("Authorization", "Bearer "+testAPIKey)
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", rec.Code)
	}
}

func TestAuthEnabledCookie(t *testing.T) {
	t.Parallel()

	rec := doAuthReq(t, testAPIKey, func(r *http.Request) {
		r.AddCookie(&http.Cookie{Name: AuthCookieName, Value: testAPIKey})
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", rec.Code)
	}
}

func TestAuthEnabledWrongKey(t *testing.T) {
	t.Parallel()

	rec := doAuthReq(t, testAPIKey, func(r *http.Request) {
		r.Header.Set("X-Proxima-Api-Key", "wrong")
	})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("got %d, want 401", rec.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	if body["error"] != "unauthorized" {
		t.Fatalf("error = %q, want %q", body["error"], "unauthorized")
	}
}

func TestAuthEnabledNoKey(t *testing.T) {
	t.Parallel()

	if rec := doAuthReq(t, testAPIKey, nil); rec.Code != http.StatusUnauthorized {
		t.Fatalf("got %d, want 401", rec.Code)
	}
}

func TestAuthExemptLogin(t *testing.T) {
	t.Parallel()

	h := AuthMiddleware(testAPIKey, okHandler())
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("got %d, want 200 (/login exempt)", rec.Code)
	}
}

func TestAuthExemptHealth(t *testing.T) {
	t.Parallel()

	h := AuthMiddleware(testAPIKey, okHandler())
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("got %d, want 200 (/health exempt)", rec.Code)
	}
}

func TestGenerateAPIKey(t *testing.T) {
	t.Parallel()

	key, err := GenerateAPIKey()
	if err != nil {
		t.Fatal(err)
	}

	if len(key) != 64 {
		t.Fatalf("key length = %d, want 64 (32 bytes hex)", len(key))
	}

	if _, err := hex.DecodeString(key); err != nil {
		t.Fatalf("key is not valid hex: %v", err)
	}

	if key2, _ := GenerateAPIKey(); key == key2 {
		t.Fatal("two generated keys are identical")
	}
}

func TestLoginHandlerSuccess(t *testing.T) {
	t.Parallel()

	h := LoginHandler(testAPIKey)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"key":"`+testAPIKey+`"}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", rec.Code)
	}

	var found bool

	for _, c := range rec.Result().Cookies() {
		if c.Name == AuthCookieName && c.Value == testAPIKey {
			found = true

			if !c.HttpOnly {
				t.Error("session cookie must be HttpOnly")
			}

			if c.SameSite != http.SameSiteStrictMode {
				t.Error("session cookie must be SameSite=Strict")
			}
		}
	}

	if !found {
		t.Fatal("expected the proxima_key cookie to be set")
	}
}

func TestLoginHandlerWrongKey(t *testing.T) {
	t.Parallel()

	h := LoginHandler(testAPIKey)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"key":"wrong"}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("got %d, want 401", rec.Code)
	}
}
