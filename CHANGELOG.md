# Changelog

## v0.1.26

- **Fix (updater): aborta si el gzip expande más de 64MB en vez de truncar en silencio** — `io.Copy` contra un `LimitReader` agotado devuelve `nil`, así que el cap anterior escribía un binario truncado y lo instalaba. Ahora se piden `limit+1` bytes y se compara `n` (`decompressGzipLimited`), con test de regresión (`TestDecompressGzipLimited`).

## v0.1.25

- **Gate anti-slop: `golangci-lint` estricto en verde + CI** — `.golangci.yml` (errcheck, errorlint, wrapcheck, govet, staticcheck, gosec, noctx, bodyclose, sqlclosecheck, rowserrcheck, contextcheck, gocritic, revive, unused, ineffassign, cyclop, dupl, forbidigo, depguard, misspell), workflow `ci.yml` (gofmt + vet + lint + `go test -race`), `AGENTS.md` corto y plugin opencode (gofmt nativo + gate en `session.idle`).
- **Fixes que encontró el gate (52 hallazgos, 0 pendientes)** — `ctx` propagado a todas las llamadas Google API (`.Context(ctx)`), HTTP y `exec` con contexto, `ReadHeaderTimeout` en el callback OAuth, errores envueltos con `%w`, `compareVersions` sin `Sscanf` (además arregla pánico con versiones de distinto largo), MIME detectado ahora sí enviado en uploads de Drive, cap de 64MB anti-descompresión en el updater, código muerto eliminado (`allTools`, `BaseService`, `mockCalendarAPI`), tests sin `nil` ctx ni permisos 0644.

## v0.1.24

- **Feature: Google Docs (3 tools)** — `get-doc` (plain-text extraction, paragraphs + tables), `create-doc`, `append-to-doc` (via `EndOfSegmentLocation`, no index arithmetic). Scope: `documents`. Re-login required.
- **Feature: Google Sheets read-only (2 tools)** — `list-sheets` (tab names), `read-sheet` (A1 notation, default `A1:Z100`). Scope: `spreadsheets.readonly`. Re-login required. Write access deliberately excluded (YAGNI until a real case needs it).
- **Fix (bug destructivo): la suite ya no pisa el `tokens.json` real** — los 4 tests de login en `internal/auth` usaban `defer withTempConfigDir(t)`, que instala el override del config dir *al retornar* el test, no durante: cada `go test ./...` sobrescribía el token real del usuario con fakes y obligaba a re-login. `withTempConfigDir` ahora devuelve cleanup (`defer withTempConfigDir(t)()`) y se verificó con token canario que la suite no toca el config real.

## v0.1.23

- **Feature: Google Forms (5 tools)** — `create-form`, `add-form-section`, `add-form-question` (texto libre o choice RADIO/CHECKBOX/DROP_DOWN vía `options`), `get-form`, `list-responses`. Sigue el mismo patrón que los demás servicios (`internal/forms/api.go` + `internal/services/forms.go`, builders puros `NewSectionItem`/`NewTextQuestion`/`NewChoiceQuestion` testeados sin red). Scopes nuevos: `forms.body` + `forms.responses.readonly` — hay que re-autorizar (`login` de nuevo). Nota: la API de Forms crea/lee formularios y lee respuestas; no existe endpoint para *enviar* respuestas como usuario.
- **Tests: 31 tools across 6 services** — `TestServiceToolsCount` actualizado (7+5+5+6+3+5) + `forms_test.go` (scopes, tools, validación de handlers, builders).

## v0.1.22

- **Security/trust: OAuth client baked into the binary, nothing downloaded at install** — `install.js` no longer ships or copies `credentials.json` (that postinstall copy is what scanners and reviewers flag). Release CI now bakes the public desktop client ID into the binary via `ldflags` (`internal/auth/embedded.go`) from the same `GOOGLE_OAUTH_CREDENTIALS_JSON` secret. End-user flow is unchanged: install, `setup`, authorize. Own-project override still wins via `GOOGLE_OAUTH_CREDENTIALS` or local `credentials.json`.
- **Legal: added `LICENSE` (MIT)** — the README always said MIT but the file was missing, leaving forks in a gray zone.
- **Docs: "About the Client ID" now states the tradeoff** — why shared (zero-config), plus honest limits: unverified warning, 100-user cap, shared quota, single point of suspension.
- Gracias a [@giuseppecrj](https://github.com/giuseppecrj), cuyo fork endurecido expuso estos huecos (credenciales en postinstall, `LICENSE` faltante).

## v0.1.21

- **Fix: `pi update --extensions` ya no falla con ETXTBSY** — `install.js` sobrescribía el binario en `~/.local/bin` con `fs.writeFileSync`, que en Linux no puede reemplazar un ejecutable en uso (el MCP server corriendo). Ahora escribe a un archivo temporal y hace `rename` atómico sobre el destino: el proceso viejo conserva su inode y el siguiente arranque usa el binario nuevo. Mismo patrón que `downloadUpdate` en `main.go`.

## v0.1.20

- **Fix (bug crítico): `search-emails` ya no restringe al label INBOX** — `SearchEmails` delegaba en `ListInbox`, que siempre aplicaba `labelIds=INBOX`. La API de Gmail combina `labelIds` y `q` con AND, así que toda búsqueda quedaba limitada a mensajes que además estuvieran en la bandeja de entrada. Por eso `in:sent`, `in:sent after:2026/09/01`, `to:alguien@gmail.com` y `from:a OR from:b` devolvían "No results" aunque los mensajes existieran, y `in:sent after:2026/08/01` solo devolvía los autoenviados (que están en INBOX y SENT). Ahora `search-emails` busca en todo el mailbox.
- **Feature: paginación real** — `list-inbox` y `search-emails` aceptan `pageToken` y avisan cuando hay más resultados (devolviendo el token para continuar). Internamente se pagina con `nextPageToken` hasta completar `maxResults` (máx. 500).
- **Refactor: `gmail.Service` usa una `messagesLister`** — abstrae `messages.list` + `messages.get` para poder testear la paginación sin red.

## v0.1.19

- **Feature: headless login (`--no-browser`)** — `login` and `setup` now accept `--no-browser` for machines without a browser (SSH, VPS, containers, WSL with broken localhost forwarding). The tool prints the authorization URL; you open it on any device (phone included), approve, and paste back the redirected URL. PKCE is preserved end-to-end.
- **Fix: automatic manual-mode fallback** — when no browser can be launched, the flow degrades to the manual paste prompt instead of printing a dead URL and hanging.
- **Fix: authorization URL now carries a real loopback port** — the callback server always starts before building the auth URL. Previously, in flows without a listener the URL contained `redirect_uri=http://localhost:0/...`, which Google can reject.
- **Refactor: `Authenticator.LoginWithOptions(opts)`** — browser and manual flows share one code path (single callback server, single exchange). `Login()` keeps its signature as the default-options wrapper. Input/output are injectable for testing.

## v0.1.18

- **Fix: `setup` now requests all 5 service scopes** — previously it only asked Google for Calendar permissions (hardcoded in `NewFromCredentials`), so Gmail/Tasks/Drive/Contacts failed after following the official `setup` flow.
- **Refactor: single source of truth for OAuth scopes** — `login`, `setup` and `serve` now share one `newAuthenticator()` that always derives scopes from `allScopes()` (the registered services). `serve` no longer trusts scopes persisted in `config.json`, which could drift out of sync. The redundant `config.json` persistence (`Config`/`Load`/`Save`) was removed — client credentials live only in `credentials.json`, tokens in `tokens.json`.
- **Refactor: `config.Credentials.AppConfig()`** — the Installed→Web fallback moved from inside the auth constructor to the data type that owns it.

## v0.1.17

- **Fix: silent MCP config skip in install.js** — `~/.pi/agent/mcp.json` is now created (with `mkdir -p`) when missing, instead of being skipped with a warning while install still reported success. Install now fails with a clear error if the MCP config can't be resolved.
- **Fix: install no longer depends on a fragile GitHub download** — the npm tarball already ships the platform binaries and `credentials.json`, so `install.js` unpacks from the local package. GitHub Releases is only a fallback for dev checkouts. This removes the postinstall network dependency entirely.
- **Fix: self-healing Pi extension** — if npm's `allowScripts` blocked the postinstall (Pi's default), the extension installs the missing binary on session start via `pi.exec()` instead of silently failing. The extension also now dispatches `/mcp reconnect` correctly with `expandPromptTemplates: true` (it was being sent to the model as plain text).
- **Fix: detect missing pi-mcp-adapter** — the extension checks whether `/mcp` is available and guides the user to install `pi-mcp-adapter` instead of sending a command that doesn't exist. Documented as a requirement in the README.
- **Fix: credentials.json validation in CI** — `bundle-credentials` now validates the JSON with `jq` and fails the build on corruption, instead of publishing a broken asset. Re-save the `GOOGLE_OAUTH_CREDENTIALS_JSON` secret as single-line JSON (the previous value had line-wrapping inside string literals).
- **Fix: SKILL.md missing frontmatter** — added the `description` field required by Pi's skill toolchain.

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
