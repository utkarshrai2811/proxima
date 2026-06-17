package config

import (
	"os"
	"runtime"
	"strings"
	"testing"
)

func TestDataDirNotEmpty(t *testing.T) {
	if DataDir() == "" {
		t.Fatal("DataDir() returned empty string")
	}
}

func TestDataDirNoHettyReference(t *testing.T) {
	if strings.Contains(DataDir(), "hetty") {
		t.Fatalf("DataDir() contains 'hetty': %s", DataDir())
	}
}

func TestDataDirXDGRespected(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("XDG test only relevant on Linux")
	}
	os.Setenv("XDG_CONFIG_HOME", "/tmp/testxdg")
	defer os.Unsetenv("XDG_CONFIG_HOME")
	if !strings.HasPrefix(DataDir(), "/tmp/testxdg") {
		t.Fatalf("XDG_CONFIG_HOME not respected: got %s", DataDir())
	}
}
