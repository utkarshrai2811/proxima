# Changelog

All notable changes to Proxima are documented here.
Format: [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).
Versioning: [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [0.8.0] — 2026-06-18

Proxima is a fork of [Hetty](https://github.com/dstotijn/hetty) v0.7.0 by
[@dstotijn](https://github.com/dstotijn). This release applies all outstanding
bug fixes, hardens the security posture, modernizes the UI, and adds four major
new features.

### Bug fixes

- **Intercept nil panic** — guard nil `modReq` in `ModifyRequest`/`CancelRequest`
  to prevent a proxy crash when intercept is toggled mid-flight (upstream hetty#145)
- **LoadOrCreateCA hardcoded path** — replace the string literal with the `keyDir`
  variable so custom cert paths are respected (upstream hetty#147)
- **Proxy cert cache memory leak** — bound the TLS certificate cache at 1,000 entries
  with LRU-style eviction (upstream hetty PR#129)
- **File descriptor leak** — add `defer f.Close()` after all cert/key file opens
  (upstream hetty PR#130)
- **Filter parser DoS** — add a recursion depth limit (50) to the filter expression
  parser; deeply nested expressions now return an error instead of overflowing
  the stack (upstream hetty#153)
- **Scope header value rule ignored** — compile and enforce the Header Value regexp
  in scope rules; previously only the Header Name regexp was matched (upstream hetty#142)

### Security hardening

- **Authentication** — the `--api-key` flag gates the admin UI and GraphQL API. Supports
  `--api-key=auto` for random key generation on startup. Proxy traffic is never gated.
  (upstream hetty#141)
- **DNS rebinding protection** — a Host header allowlist on all admin routes, configurable
  via `--allowed-hosts` (upstream hetty PR#108)
- **Body read limits** — the `--max-body-size` flag (default 10 MB) prevents memory
  exhaustion from large response bodies (upstream hetty#143)

### New features

- **WebSocket proxying** — intercept and log WebSocket frames in both directions,
  inject frames from the UI, and stream session/frame updates to the WebSockets page
  in real time via Server-Sent Events.
- **Intruder-style fuzzer** — four attack types (Sniper, Battering Ram, Pitchfork,
  Cluster Bomb), built-in payload lists (SQLi, XSS, passwords, dirs), concurrency
  control, pause/resume/cancel, real-time result streaming, and difference highlighting.
- **Export** — export selected proxy log entries as Burp Suite XML (importable by
  Burp), curl commands, or an OpenAPI 3.0 skeleton (with path-parameter detection).
- **Plugin system** — JavaScript plugins via the Goja engine. Drop `.js` files into
  your platform's plugin directory. Hooks: `onRequest`, `onResponse`, `onIntercept`,
  `onFuzzResult`. Sandbox globals: `proxima.log`, `proxima.alert`, `proxima.http`
  (with loopback/SSRF protection), `proxima.store`. Per-plugin timeout and error
  isolation — a faulty plugin never crashes the proxy. Hot-reload from the UI and a
  cross-platform "open plugins folder" action.

### UI

- Replaced Next.js + Material UI with React + Vite + TypeScript + Tailwind CSS
- Dark monospace design (JetBrains Mono throughout)
- Persistent sidebar navigation with a status bar
- New features (WebSockets, Fuzzer, Plugins) are served over REST + Server-Sent
  Events on the admin router; the existing GraphQL API is unchanged.

### Cross-platform

- **Data directory follows platform conventions** (new):
  - macOS: `~/Library/Application Support/proxima/`
  - Linux: `~/.config/proxima/` (or `$XDG_CONFIG_HOME/proxima/`)
  - Windows: `%APPDATA%\proxima\`
- Builds for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64
- Distribution: Homebrew Cask (macOS), Scoop bucket (Windows), and `.deb`/`.rpm`/`.apk`
  packages plus tarballs (Linux)

### Breaking changes

- Binary renamed: `hetty` → `proxima`
- Default data directory changed (see Cross-platform above)
- Old paths: `~/.hetty/hetty_cert.pem`, `~/.hetty/hetty_key.pem`, `~/.hetty/hetty.db`

### Migration from Hetty

```bash
# macOS
mkdir -p ~/Library/Application\ Support/proxima
cp ~/.hetty/hetty_cert.pem ~/Library/Application\ Support/proxima/proxima_cert.pem
cp ~/.hetty/hetty_key.pem  ~/Library/Application\ Support/proxima/proxima_key.pem
cp ~/.hetty/hetty.db       ~/Library/Application\ Support/proxima/proxima.db

# Linux
mkdir -p ~/.config/proxima
cp ~/.hetty/hetty_cert.pem ~/.config/proxima/proxima_cert.pem
cp ~/.hetty/hetty_key.pem  ~/.config/proxima/proxima_key.pem
cp ~/.hetty/hetty.db       ~/.config/proxima/proxima.db

# Windows (PowerShell)
$dest = "$env:APPDATA\proxima"
New-Item -ItemType Directory -Force -Path $dest
Copy-Item "$env:USERPROFILE\.hetty\hetty_cert.pem" "$dest\proxima_cert.pem"
Copy-Item "$env:USERPROFILE\.hetty\hetty_key.pem"  "$dest\proxima_key.pem"
Copy-Item "$env:USERPROFILE\.hetty\hetty.db"        "$dest\proxima.db"
```

---

## [upstream] Hetty v0.7.0 — 2022-03-29

Proxima is forked from this release. See the
[Hetty releases](https://github.com/dstotijn/hetty/releases) for earlier history.
