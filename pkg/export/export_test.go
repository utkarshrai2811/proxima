package export

import (
	"encoding/base64"
	"encoding/xml"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func mkEntry(method, rawurl, body string) Entry {
	u, _ := url.Parse(rawurl)

	return Entry{
		Method: method,
		URL:    u,
		Proto:  "HTTP/1.1",
		Header: http.Header{"Content-Type": {"application/json"}},
		Body:   []byte(body),
		Response: &Response{
			Proto: "HTTP/1.1", StatusCode: 200, Status: "200 OK",
			Header: http.Header{"Content-Type": {"application/json"}},
			Body:   []byte(`{"ok":true}`),
		},
	}
}

func TestExportCurl(t *testing.T) {
	t.Parallel()

	got := ExportCurl(mkEntry("POST", "https://example.com/api/login", `{"u":"a"}`), true)

	for _, want := range []string{
		"curl -X POST", "'https://example.com/api/login'",
		"-H 'Content-Type: application/json'", "--data-raw", "--proxy",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("curl output missing %q:\n%s", want, got)
		}
	}
}

func TestExportBurpXML(t *testing.T) {
	t.Parallel()

	data, err := ExportBurpXML([]Entry{mkEntry("GET", "https://example.com/x?a=1", "")})
	if err != nil {
		t.Fatal(err)
	}

	var items burpItems
	if err := xml.Unmarshal(data, &items); err != nil {
		t.Fatalf("burp xml not parseable: %v", err)
	}

	if len(items.Items) != 1 {
		t.Fatalf("items = %d, want 1", len(items.Items))
	}

	it := items.Items[0]
	if it.Method != "GET" || it.Protocol != "https" || it.Port != "443" {
		t.Errorf("item metadata wrong: %+v", it)
	}

	raw, err := base64.StdEncoding.DecodeString(it.Request.Value)
	if err != nil {
		t.Fatalf("request not base64: %v", err)
	}

	if !strings.HasPrefix(string(raw), "GET /x?a=1 HTTP/1.1") {
		t.Errorf("raw request line wrong: %q", string(raw))
	}
}

func TestNormalizePath(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"/users/123": "/users/{id}",
		"/users/550e8400-e29b-41d4-a716-446655440000": "/users/{id}",
		"/a/1/b/2":       "/a/{id}/b/{id2}",
		"/static/app.js": "/static/app.js",
	}

	for in, want := range cases {
		if got, _ := normalizePath(in); got != want {
			t.Errorf("normalizePath(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestExportOpenAPI(t *testing.T) {
	t.Parallel()

	data, err := ExportOpenAPI([]Entry{mkEntry("GET", "https://example.com/users/123", "")})
	if err != nil {
		t.Fatal(err)
	}

	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("openapi output is not valid YAML: %v", err)
	}

	if doc["openapi"] != "3.0.0" {
		t.Errorf("openapi version = %v", doc["openapi"])
	}

	paths, ok := doc["paths"].(map[string]any)
	if !ok {
		t.Fatalf("paths missing or wrong type: %T", doc["paths"])
	}

	if _, ok := paths["/users/{id}"]; !ok {
		t.Errorf("expected templated path /users/{id}, got keys %v", paths)
	}
}
