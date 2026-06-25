# Changelog

## v0.1.16

- **Fix: install.js redirect handling** — GitHub releases return HTTP 302 redirects, but the download function used `res.location` (non-existent) instead of `res.headers.location`. Redirects were never followed, causing every install/update to fail with "HTTP 302" error.

## v0.1.15

- **Fix: install.js URL doubling** — `REPO` already contains the full GitHub URL, so prepending `https://github.com/` produced `https://github.com/https://github.com/...` (404). This was the root cause of install/update always failing — the binary was never downloaded, always falling back to whatever was in `~/.local/bin/`.

## v0.1.14

- **Fix: install.js platform mapping** — `x64` now correctly maps to `amd64` to match GitHub Release asset names (Go's `GOARCH` nomenclature). Previously, Linux x64 users couldn't download the binary during install/update.

## v0.1.13

- **Email attachments**: `send-email` and `reply-to-email` now accept an optional `attachments` array
- Attach from local file paths (`localPath`) or Google Drive file IDs (`driveFileId`)
- MIME `multipart/mixed` encoding with base64-wrapped attachment data
- 9 new unit tests (MIME multipart, attachment resolution, edge cases)

## v0.1.0

- Initial release
- Google Calendar: list, create, update, delete, search events
- Gmail: list inbox, read, send, reply, search
- OAuth2 PKCE login with embedded credentials
- Pi MCP integration via package install
