package config

import (
	"os"
	"path/filepath"
	"runtime"
)

// DataDir returns the platform-appropriate data directory for Proxima.
//
// Platform conventions:
//   - macOS:   ~/Library/Application Support/proxima
//   - Linux:   $XDG_CONFIG_HOME/proxima  (falls back to ~/.config/proxima)
//   - Windows: %APPDATA%\proxima  (e.g. C:\Users\you\AppData\Roaming\proxima)
//
// Falls back to ~/.proxima on any platform if all else fails.
func DataDir() string {
	switch runtime.GOOS {
	case "windows":
		if appdata := os.Getenv("APPDATA"); appdata != "" {
			return filepath.Join(appdata, "proxima")
		}
	case "darwin":
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, "Library", "Application Support", "proxima")
		}
	default: // linux and other unix
		if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
			return filepath.Join(xdg, "proxima")
		}
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, ".config", "proxima")
		}
	}
	// universal fallback
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".proxima")
	}
	return ".proxima"
}

// PluginDir returns the directory where Proxima looks for .js plugin files.
func PluginDir() string {
	return filepath.Join(DataDir(), "plugins")
}

// CertPath returns the default path for the CA certificate file.
func CertPath() string {
	return filepath.Join(DataDir(), "proxima_cert.pem")
}

// KeyPath returns the default path for the CA private key file.
func KeyPath() string {
	return filepath.Join(DataDir(), "proxima_key.pem")
}

// DBPath returns the default path for the database file.
func DBPath() string {
	return filepath.Join(DataDir(), "proxima.db")
}
