# pi-google-services

Google Calendar, Gmail, and Google Meet MCP server for Pi.
Single binary, zero runtime deps. Login once, manage everything from your agent.

## Requirements

- **pi-mcp-adapter** — provides the `/mcp` command that Pi's core doesn't ship.
  Install it first:

```bash
pi install npm:pi-mcp-adapter
```

## Quick Install

```bash
pi install npm:pi-google-services
pi-google-services setup
# Restart Pi session, then:
# "show my events", "read my inbox", "create a meeting with Meet"
```

If npm's `allowScripts` blocked the postinstall (Pi's default), the binary is
installed automatically on your next Pi session start — the extension
self-heals. You can also run it manually at any time:

```bash
node ~/.pi/agent/npm/node_modules/pi-google-services/install.js
```

## Updates

```bash
# Via Pi (recommended)
pi update pi-google-services

# Via binary
pi-google-services update
```

After updating, restart your Pi session.

## Tools

### Calendar (7)

| Tool | Description |
|------|-------------|
| `list-events` | List events in a date range |
| `create-event` | Create event with attendees + Meet link |
| `update-event` | Modify existing event |
| `delete-event` | Remove event |
| `search-events` | Search by text |
| `list-calendars` | Show all calendars |
| `get-freebusy` | Check availability |

### Gmail (5)

| Tool | Description |
|------|-------------|
| `list-inbox` | Show recent emails |
| `get-email` | Read full email by ID |
| `search-emails` | Search with Gmail syntax |
| `send-email` | Send new email with optional file attachments |
| `reply-to-email` | Reply to thread with optional file attachments |

### Email Attachments

Both `send-email` and `reply-to-email` accept an optional `attachments` array.
Each attachment can reference a local file or a Google Drive file:

```json
{
  "to": "user@example.com",
  "subject": "Report",
  "body": "See attached",
  "attachments": [
    { "localPath": "/home/user/report.pdf" },
    { "driveFileId": "1a2b3c4d5e6f" }
  ]
}
```

### Tasks (5)

| Tool | Description |
|------|-------------|
| `list-tasklists` | Show all task lists |
| `list-tasks` | List tasks (pending/completed) |
| `create-task` | Create a new task |
| `complete-task` | Mark task as done |
| `delete-task` | Remove a task |

### Meet

Pass `"withMeet": true` to `create-event` to auto-generate a Google Meet link.

## Architecture

```
pi-google-services/          npm package (pi-package)
├── main.go                  CLI entry point
├── package.json             Pi manifest + npm
├── SKILL.md                 Pi skill
├── install.js               postinstall: install binary + wire Pi MCP config
├── internal/
│   ├── mcp/                 MCP protocol (JSON-RPC 2.0 / stdio)
│   ├── services/            Service interface + tool implementations
│   │   ├── calendar.go      7 tools
│   │   └── gmail.go         5 tools
│   ├── calendar/api.go      Google Calendar API wrapper
│   ├── gmail/api.go         Gmail API wrapper
│   ├── auth/                OAuth2 PKCE (browser login) + baked-in client ID
│   └── config/              Token storage
└── .github/workflows/
    └── release.yml          CI: build + bake OAuth client + npm publish (OIDC)
```

No secrets live in this repository. The OAuth client ID is baked into the
release binary by CI (via `ldflags`, see `internal/auth/embedded.go`) from
the `GOOGLE_OAUTH_CREDENTIALS_JSON` secret. `install.js` only installs the
binary and wires the Pi MCP config — it downloads no credentials.

## Transparency & Security

### Open Source, Auditable Code

This entire project is open source. Every line of code can be reviewed,
audited, and verified. The Go binary is built from this source in
GitHub Actions with provenance attestation — you can verify the build
matches the published source.

### How OAuth Works

pi-google-services uses **OAuth 2.0 with PKCE** (Proof Key for Code
Exchange), the industry standard for desktop applications:

1. You run `login` or `setup`
2. Your browser opens to Google's consent screen
3. You see exactly what permissions are being requested (calendar,
   email, tasks, drive, contacts)
4. You authorize with your Google account
5. A token is saved **locally** on your machine (`~/.config/pi-google-services/`)
6. The token never leaves your machine — all API calls go directly
   from your binary to Google

### About the Client ID

The release binary ships with a pre-registered Google Cloud OAuth client ID
baked in at build time by CI. This is **not a secret** — it's the same
mechanism used by every app that offers "Sign in with Google" (Todoist,
Notion, Fantastical, `gcalcli`, `rclone`, `gh`, etc.).

The client ID is publicly visible in the authorization URL and only
serves to identify which app is requesting access. The actual security
is in the OAuth consent screen where **you** decide what to share.

Why shared instead of "create your own project"? Registering a Google Cloud
project means enabling 5 APIs, configuring the consent screen, and creating
a Desktop OAuth client — ~10 minutes of friction before the first login.
The shared client removes that entirely: install, `setup`, authorize, done.
No Google Cloud account, no billing, no JSON files to juggle.

Prefer your own project? Set `GOOGLE_OAUTH_CREDENTIALS` to the path of your
`credentials.json` (or place it at `~/.config/pi-google-services/`). Your
own credentials always take precedence over the baked-in client.

Known limits of the shared client (by design, not bugs):

- The app is **unverified**, so Google shows a one-time warning screen.
  Click **"Continue"** to authorize.
- Unverified apps are capped at **100 users** total.
- API quota is shared across all users of the client.
- If Google ever suspends the project, auth breaks for everyone at once —
  with your own client ID that risk is yours alone.

### Why Google Shows "This app is not verified"

When you run `login` or `setup`, Google shows a warning screen saying
the app is not verified. **This is normal and safe.**

Google's verification process requires a registered domain, a formal
brand review, and proof of ownership — it's designed for public web
apps with a business behind them, not open-source CLI tools.

The warning appears once per user. Click **"Continue"** to authorize.
Your data goes directly from your machine to Google — no intermediate
servers, no tracking, no telemetry.

This is an open-source project built for utility, not monetization.
Domain registration and Google verification are not a priority.
Many popular CLI tools for Google services (like `gcalcli`) are also
unverified. This does not affect security.

### Credential Storage

| What | Where |
|------|-------|
| OAuth client ID | Baked into the release binary by CI (public by design); overridable via `GOOGLE_OAUTH_CREDENTIALS` or local `credentials.json` |
| Access/Refresh tokens | `~/.config/pi-google-services/tokens.json` (0600 permissions) |
| No data leaves your machine | All Google API calls are direct from your binary |

The binary never phones home, tracks usage, or sends telemetry.

## Headless / WSL Login (`--no-browser`)

On machines without a browser (SSH, VPS, containers) or inside WSL where
localhost forwarding breaks, use the manual paste flow:

```bash
pi-google-services login --no-browser
```

1. Open the printed authorization URL on **any device** (your phone works).
2. Approve the consent screen. Google redirects to a `localhost` page that
   fails to load — that is expected.
3. Copy the full URL from the address bar and paste it into the terminal.

The plain `login`/`setup` flow automatically falls back to this mode when
no browser can be launched.

## Development

```bash
cp /path/to/credentials.json .
go build -o pi-google-services .
./pi-google-services login
./pi-google-services serve
```

Local builds without `credentials.json` in the repo root (and without the
release `ldflags`) require your own client: set `GOOGLE_OAUTH_CREDENTIALS`
to your `credentials.json` path. Release CI bakes the shared public client
in via `-X github.com/sombi/pi-google-services/internal/auth.embeddedClientID=...`
so end users never touch this.

## Tests

```bash
go test ./... -v
```

26 unit tests (MCP protocol, config, service metadata, services, MIME multipart attachments).

## License

MIT — see [LICENSE](LICENSE).
