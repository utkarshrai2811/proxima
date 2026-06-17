package proxy

import (
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
