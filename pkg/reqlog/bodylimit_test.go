package reqlog

import (
	"bytes"
	"io"
	"testing"
)

// TestReadBodyForLogging is a regression test for hetty#143: bodies stored in
// the log must be capped at the configured size (and flagged truncated), while
// the full body is still forwarded.
func TestReadBodyForLogging(t *testing.T) {
	t.Parallel()

	t.Run("oversized body is truncated but forwarded in full", func(t *testing.T) {
		t.Parallel()

		const limit = 10

		full := bytes.Repeat([]byte("a"), 25)

		logged, truncated, fullReader, err := readBodyForLogging(io.NopCloser(bytes.NewReader(full)), limit)
		if err != nil {
			t.Fatal(err)
		}

		if !truncated {
			t.Error("expected truncated = true for an oversized body")
		}

		if int64(len(logged)) != limit {
			t.Errorf("logged length = %d, want %d", len(logged), limit)
		}

		forwarded, err := io.ReadAll(fullReader)
		if err != nil {
			t.Fatal(err)
		}

		if !bytes.Equal(forwarded, full) {
			t.Errorf("forwarded body length = %d, want %d (full body must be forwarded intact)",
				len(forwarded), len(full))
		}
	})

	t.Run("body within limit is not truncated", func(t *testing.T) {
		t.Parallel()

		body := []byte("hello world")

		logged, truncated, fullReader, err := readBodyForLogging(io.NopCloser(bytes.NewReader(body)), 100)
		if err != nil {
			t.Fatal(err)
		}

		if truncated {
			t.Error("expected truncated = false")
		}

		if !bytes.Equal(logged, body) {
			t.Errorf("logged = %q, want %q", logged, body)
		}

		forwarded, _ := io.ReadAll(fullReader)
		if !bytes.Equal(forwarded, body) {
			t.Error("forwarded body must equal the original")
		}
	})

	t.Run("no limit reads the whole body", func(t *testing.T) {
		t.Parallel()

		body := bytes.Repeat([]byte("x"), 1000)

		logged, truncated, _, err := readBodyForLogging(io.NopCloser(bytes.NewReader(body)), 0)
		if err != nil {
			t.Fatal(err)
		}

		if truncated {
			t.Error("expected truncated = false when no limit is set")
		}

		if len(logged) != 1000 {
			t.Errorf("logged length = %d, want 1000", len(logged))
		}
	})
}
