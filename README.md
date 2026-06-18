# Proxima

**The open-source HTTP toolkit for security research. Actively maintained.**

[![CI](https://github.com/utkarshrai2811/proxima/actions/workflows/ci.yml/badge.svg)](https://github.com/utkarshrai2811/proxima/actions/workflows/ci.yml)
[![Latest Release](https://img.shields.io/github/v/release/utkarshrai2811/proxima)](https://github.com/utkarshrai2811/proxima/releases/latest)

Proxima is a machine-in-the-middle (MITM) HTTP proxy and security testing toolkit
for penetration testers and bug bounty hunters. Single binary, runs locally, with
a web-based UI.

---

## Features

| Feature | Hetty | Proxima |
|---|---|---|
| MITM HTTP/HTTPS proxy | ✅ | ✅ |
| Proxy log with search | ✅ | ✅ |
| Request/response intercept | ✅ | ✅ |
| Scope management | ✅ | ✅ |
| HTTP client / replay | ✅ | ✅ |
| Project-based storage | ✅ | ✅ |
| **Authentication (API key)** | ❌ | ✅ |
| **DNS rebinding protection** | ❌ | ✅ |
| **Body read limits** | ❌ | ✅ |
| **WebSocket proxying + replay** | ❌ | ✅ |
| **Intruder-style fuzzer** | ❌ | ✅ |
| **Export (Burp XML / curl / OpenAPI)** | ❌ | ✅ |
| **JavaScript plugin system** | ❌ | ✅ |
| **Modern React + Vite UI** | ❌ | ✅ |
| **Cross-platform data directory** | ❌ | ✅ |

---

## Installation

### macOS — Homebrew

```bash
brew tap utkarshrai2811/proxima
brew install --cask proxima
```

Upgrade: `brew update && brew upgrade --cask proxima`

### Windows — Scoop

```bash
scoop bucket add proxima https://github.com/utkarshrai2811/scoop-proxima
scoop install proxima
```

Upgrade: `scoop update && scoop update proxima`

### Linux — package or tarball

Download the package for your distribution from
[Releases](https://github.com/utkarshrai2811/proxima/releases/latest):

```bash
# Debian / Ubuntu
sudo dpkg -i proxima_<version>_linux_amd64.deb

# Fedora / RHEL / openSUSE
sudo rpm -i proxima_<version>_linux_amd64.rpm

# Alpine
sudo apk add --allow-untrusted proxima_<version>_linux_amd64.apk
```

Or extract the tarball and place the binary on your `PATH`:

```bash
tar xzf proxima_<version>_linux_amd64.tar.gz
sudo mv proxima /usr/local/bin/
```

### All platforms — Go install

```bash
go install github.com/utkarshrai2811/proxima/cmd/proxima@latest
```

### Build from source

Requires Go 1.25+ and Node 20+:

```bash
git clone https://github.com/utkarshrai2811/proxima.git
cd proxima
make build
./proxima
```

---

## Quick start

```bash
# Start with default settings (no auth, port 8080)
proxima

# Auto-generate an API key (printed once on startup)
proxima --api-key=auto

# Custom listen address
proxima --addr=:9090
```

Open `http://localhost:8080`. Configure your browser or OS to use
`http://127.0.0.1:8080` as an HTTP proxy. Download the CA certificate from the
Settings page and install it to intercept HTTPS.

---

## Data directory

Proxima stores its certificate, database, and plugins in a platform-appropriate
directory:

| OS | Data directory |
|---|---|
| macOS | `~/Library/Application Support/proxima/` |
| Linux | `~/.config/proxima/` (or `$XDG_CONFIG_HOME/proxima/`) |
| Windows | `%APPDATA%\proxima\` |

Override any path with the `--cert`, `--key`, and `--db` flags.

---

## CLI reference

```
proxima [flags]

  --addr           Listen address (default: ":8080")
  --cert           CA certificate path (default: platform data dir)
  --key            CA private key path (default: platform data dir)
  --db             Database path (default: platform data dir)
  --api-key        API key for admin UI auth. "auto" = generate a random key.
                   Leave empty to disable auth (default).
  --allowed-hosts  Comma-separated allowed Host header values for the admin UI
                   (default: "localhost,127.0.0.1")
  --max-body-size  Maximum captured body size (default: "10MB"). Units: B, KB, MB, GB.
  --verbose        Enable verbose logging
  --json           JSON log output
  --version, -v    Print version
  --help, -h       Print help
```

---

## Authentication

By default the admin interface has no authentication (suitable for local,
single-user use). To enable it:

```bash
proxima --api-key=auto          # generate a random key (printed once on startup)
proxima --api-key=mysecretkey   # use a specific key
```

When enabled, present the key via any of:

- Header: `X-Proxima-Api-Key: <key>`
- Header: `Authorization: Bearer <key>`
- The login page at `/login` (sets an `HttpOnly`, `SameSite=Strict` session cookie)

The `/login`, `/api/auth/login`, and `/health` endpoints are always exempt.

**The proxy traffic endpoint is never gated by authentication** — only the admin
UI and API are.

---

## WebSocket proxying

Proxima intercepts WebSocket connections automatically. View sessions and live
frames on the WebSockets page, inject frames into an open session, and stream
updates in real time.

## Fuzzer

Mark positions in a request template with `§name§`, choose an attack type
(Sniper, Battering Ram, Pitchfork, Cluster Bomb), select payload lists, and run.
Results stream in real time with status, length, and timing columns.

## Export

Select entries in the Proxy Log and export them as:

- **Burp Suite XML** — import directly into Burp Suite
- **curl** — paste into your terminal
- **OpenAPI 3.0** — an auto-generated API skeleton from observed traffic

## Plugins

Drop a `.js` file into your platform's plugin directory to install a plugin:

```javascript
export const meta = {
  name: "Header Injector",
  version: "1.0.0",
  description: "Adds a header to every request",
  author: "Your Name"
};

export function onRequest(ctx) {
  ctx.request.headers['X-Custom'] = 'value';
  return ctx;
}
```

See [PLUGINS.md](./PLUGINS.md) for the full plugin development guide.

---

## Contributing

Issues and pull requests are welcome on
[GitHub](https://github.com/utkarshrai2811/proxima). Please search existing issues
before opening a new one.

---

## Credits

Proxima is a fork of [Hetty](https://github.com/dstotijn/hetty) by
[@dstotijn](https://github.com/dstotijn). Huge thanks to the original author and
contributors for building an excellent foundation.

The font used in the admin interface is
[JetBrains Mono](https://www.jetbrains.com/lp/mono/).
