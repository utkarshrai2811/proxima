// Package export converts proxy log entries into common interchange formats:
// Burp Suite XML, curl commands, and a minimal OpenAPI 3.0 document.
package export

import (
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

// Entry is a proxy log entry to be exported (decoupled from the reqlog types).
type Entry struct {
	Method   string
	URL      *url.URL
	Proto    string
	Header   http.Header
	Body     []byte
	Response *Response
}

// Response is the response side of an Entry.
type Response struct {
	Proto      string
	StatusCode int
	Status     string
	Header     http.Header
	Body       []byte
}

func sortedHeaderLines(h http.Header) []string {
	keys := make([]string, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	var lines []string

	for _, k := range keys {
		for _, v := range h[k] {
			lines = append(lines, k+": "+v)
		}
	}

	return lines
}

func protoOrDefault(p string) string {
	if p == "" {
		return "HTTP/1.1"
	}

	return p
}

// rawRequestBytes reconstructs the raw HTTP request (origin-form request line).
func (e Entry) rawRequestBytes() []byte {
	var b strings.Builder

	path := "/"
	if e.URL != nil && e.URL.RequestURI() != "" {
		path = e.URL.RequestURI()
	}

	fmt.Fprintf(&b, "%s %s %s\r\n", e.Method, path, protoOrDefault(e.Proto))

	if e.URL != nil && e.Header.Get("Host") == "" {
		fmt.Fprintf(&b, "Host: %s\r\n", e.URL.Host)
	}

	for _, line := range sortedHeaderLines(e.Header) {
		b.WriteString(line)
		b.WriteString("\r\n")
	}

	b.WriteString("\r\n")
	b.Write(e.Body)

	return []byte(b.String())
}

// rawResponseBytes reconstructs the raw HTTP response.
func (e Entry) rawResponseBytes() []byte {
	if e.Response == nil {
		return nil
	}

	var b strings.Builder

	status := e.Response.Status
	if status == "" {
		status = fmt.Sprintf("%d", e.Response.StatusCode)
	}

	fmt.Fprintf(&b, "%s %s\r\n", protoOrDefault(e.Response.Proto), status)

	for _, line := range sortedHeaderLines(e.Response.Header) {
		b.WriteString(line)
		b.WriteString("\r\n")
	}

	b.WriteString("\r\n")
	b.Write(e.Response.Body)

	return []byte(b.String())
}

func hasJSONContentType(h http.Header) bool {
	return strings.Contains(strings.ToLower(h.Get("Content-Type")), "json")
}
