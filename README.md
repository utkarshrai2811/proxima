# Proxima

**The open-source HTTP toolkit for security research. Actively maintained.**

Proxima is a machine-in-the-middle (MITM) HTTP proxy and toolkit for security
research, penetration testing, and bug bounty work. It is a maintained fork of
[Hetty](https://github.com/dstotijn/hetty), modernized with new features and
ongoing development.

> ⚠️ Proxima is under active development. This README is a placeholder and will
> be expanded with full installation, usage, and feature documentation.

## Features

- Machine-in-the-middle (MITM) HTTP/HTTPS proxy, with logs and advanced search
- HTTP client for manually creating/editing requests, and replaying proxied requests
- Intercept requests and responses for manual review (edit, forward, drop)
- Scope support, to help keep work organized
- Web-based admin interface
- Project-based database storage

## Installation

Build from source (requires Go 1.23+ and Node 20+):

```sh
git clone https://github.com/utkarshrai2811/proxima.git
cd proxima
make build
./proxima
```

Packaged installers (Homebrew, Scoop, `go install`, and pre-built binaries)
ship with the first tagged release.

## Usage

```sh
proxima --help
```

The data directory (certificate, key, and database) is platform-specific:

| OS      | Location                                            |
| ------- | --------------------------------------------------- |
| macOS   | `~/Library/Application Support/proxima`             |
| Linux   | `$XDG_CONFIG_HOME/proxima` (or `~/.config/proxima`) |
| Windows | `%APPDATA%\proxima`                                 |

## Credits

Proxima is a fork of [Hetty](https://github.com/dstotijn/hetty) by
[@dstotijn](https://github.com/dstotijn). Huge thanks to the original author and
contributors for building an excellent foundation.

The font used in the logo and admin interface is
[JetBrains Mono](https://www.jetbrains.com/lp/mono/).
