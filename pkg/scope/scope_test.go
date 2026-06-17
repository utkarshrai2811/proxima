package scope_test

import (
	"net/http"
	"regexp"
	"testing"

	"github.com/utkarshrai2811/proxima/pkg/scope"
)

func reqWithHeader(key, value string) *http.Request {
	req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
	if key != "" {
		req.Header.Set(key, value)
	}

	return req
}

// TestRuleMatchHeaderValue covers hetty#142: when both header key and value
// patterns are set, both must match for the rule to apply.
func TestRuleMatchHeaderValue(t *testing.T) {
	t.Parallel()

	rule := scope.Rule{
		Header: scope.Header{
			Key:   regexp.MustCompile("X-Custom"),
			Value: regexp.MustCompile("secret"),
		},
	}

	tests := []struct {
		name      string
		headerKey string
		headerVal string
		want      bool
	}{
		{"matching key and value", "X-Custom", "secret", true},
		{"matching key, wrong value", "X-Custom", "public", false},
		{"non-matching key", "X-Other", "secret", false},
		{"no custom header", "", "", false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := rule.Match(reqWithHeader(tc.headerKey, tc.headerVal), nil); got != tc.want {
				t.Errorf("Match() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestRuleMatchHeaderKeyOnly verifies that an empty value pattern matches any
// value for the given header key.
func TestRuleMatchHeaderKeyOnly(t *testing.T) {
	t.Parallel()

	rule := scope.Rule{
		Header: scope.Header{
			Key: regexp.MustCompile("X-Custom"),
		},
	}

	if !rule.Match(reqWithHeader("X-Custom", "anything"), nil) {
		t.Error("expected a match for any value when the value pattern is empty")
	}

	if rule.Match(reqWithHeader("X-Other", "anything"), nil) {
		t.Error("expected no match when the key does not match")
	}
}
