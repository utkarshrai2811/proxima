package export

import (
	"fmt"
	"strings"
)

// shellQuote single-quotes a value for POSIX shells, escaping embedded quotes.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// ExportCurl converts an entry into a curl command. When includeProxy is true,
// it appends --proxy http://127.0.0.1:8080.
func ExportCurl(e Entry, includeProxy bool) string {
	target := ""
	if e.URL != nil {
		target = e.URL.String()
	}

	parts := []string{fmt.Sprintf("curl -X %s %s", e.Method, shellQuote(target))}

	for _, line := range sortedHeaderLines(e.Header) {
		// curl recomputes Content-Length, so omit it.
		if strings.HasPrefix(strings.ToLower(line), "content-length:") {
			continue
		}

		parts = append(parts, "-H "+shellQuote(line))
	}

	if len(e.Body) > 0 {
		parts = append(parts, "--data-raw "+shellQuote(string(e.Body)))
	}

	if includeProxy {
		parts = append(parts, "--proxy 'http://127.0.0.1:8080'")
	}

	return strings.Join(parts, " \\\n  ")
}

// ExportCurlAll renders one curl command per entry, separated by blank lines.
func ExportCurlAll(entries []Entry, includeProxy bool) string {
	cmds := make([]string, 0, len(entries))
	for _, e := range entries {
		cmds = append(cmds, ExportCurl(e, includeProxy))
	}

	return strings.Join(cmds, "\n\n")
}
