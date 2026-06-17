package proxy

import (
	"crypto/tls"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// TestLoadOrCreateCACreatesCertDir is a regression test for hetty#147:
// LoadOrCreateCA must create the parent directory of the cert file using the
// keyDir variable, not a literal "keyDir" path. With the bug, a directory
// literally named "keyDir" in the working directory makes the os.Stat("keyDir")
// guard succeed, so the cert's parent directory is never created and os.Create
// fails.
func TestLoadOrCreateCACreatesCertDir(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)

	// Decoy entry that the buggy os.Stat("keyDir") literal would match.
	if err := os.Mkdir("keyDir", 0o755); err != nil {
		t.Fatalf("failed to create decoy dir: %v", err)
	}

	// Cert and key live under distinct, not-yet-existing subdirectories.
	certFile := filepath.Join(tmp, "certs", "proxima_cert.pem")
	keyFile := filepath.Join(tmp, "keys", "proxima_key.pem")

	caCert, caKey, err := LoadOrCreateCA(keyFile, certFile)
	if err != nil {
		t.Fatalf("LoadOrCreateCA returned error: %v", err)
	}

	if caCert == nil || caKey == nil {
		t.Fatal("expected non-nil CA certificate and key")
	}

	for _, f := range []string{certFile, keyFile} {
		if _, err := os.Stat(f); err != nil {
			t.Fatalf("expected %s to exist: %v", f, err)
		}
	}

	// A second call must load the existing pair from disk without error.
	if _, _, err := LoadOrCreateCA(keyFile, certFile); err != nil {
		t.Fatalf("LoadOrCreateCA (reload) returned error: %v", err)
	}
}

// TestCertCacheBounded is a regression test for hetty PR#129: the generated
// certificate cache must stay bounded to prevent unbounded memory growth.
func TestCertCacheBounded(t *testing.T) {
	t.Parallel()

	c := newCertCache()

	const inserts = maxCertCacheSize + certCacheEvictCount + 50
	for i := 0; i < inserts; i++ {
		c.set(fmt.Sprintf("host-%d.example.com", i), &tls.Certificate{})

		if got := len(c.cache); got > maxCertCacheSize {
			t.Fatalf("cache exceeded max size after %d inserts: %d > %d", i+1, got, maxCertCacheSize)
		}
	}

	if got := len(c.cache); got > maxCertCacheSize {
		t.Fatalf("final cache size %d exceeds max %d", got, maxCertCacheSize)
	}

	// A freshly stored entry must be retrievable.
	c.set("lookup.example.com", &tls.Certificate{})

	if _, ok := c.get("lookup.example.com"); !ok {
		t.Fatal("expected to find a just-stored host in the cache")
	}

	// The oldest entries must have been evicted.
	if _, ok := c.get("host-0.example.com"); ok {
		t.Fatal("expected oldest entry to be evicted from the cache")
	}
}
