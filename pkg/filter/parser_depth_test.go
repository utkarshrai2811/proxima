package filter

import (
	"strings"
	"testing"
)

// TestParseQueryDepthLimit is a regression test for hetty#153: deeply nested
// filter expressions must not overflow the stack. Beyond the depth limit the
// parser returns a descriptive error instead of panicking.
func TestParseQueryDepthLimit(t *testing.T) {
	t.Parallel()

	t.Run("within depth limit parses", func(t *testing.T) {
		t.Parallel()

		input := strings.Repeat("(", 40) + "foo" + strings.Repeat(")", 40)
		if _, err := ParseQuery(input); err != nil {
			t.Fatalf("expected a 40-level nested expression to parse, got error: %v", err)
		}
	})

	t.Run("beyond depth limit errors without panic", func(t *testing.T) {
		t.Parallel()

		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("ParseQuery panicked on deeply nested input: %v", r)
			}
		}()

		input := strings.Repeat("(", 60) + "foo" + strings.Repeat(")", 60)

		_, err := ParseQuery(input)
		if err == nil {
			t.Fatal("expected an error for a 60-level nested expression, got nil")
		}
	})
}
