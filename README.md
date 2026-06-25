# pi-google-services

Google Calendar, Gmail, and Google Meet MCP server for Pi.
Single binary, zero runtime deps. Login once, manage everything from your agent.

## Quick Install

```bash
pi install npm:pi-google-services
pi-google-services setup
# Restart Pi session, then:
# "show my events", "read my inbox", "create a meeting with Meet"
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
├── install.js               postinstall: download binary + credentials
├── internal/
│   ├── mcp/                 MCP protocol (JSON-RPC 2.0 / stdio)
│   ├── services/            Service interface + tool implementations
│   │   ├── calendar.go      7 tools
│   │   └── gmail.go         5 tools
│   ├── calendar/api.go      Google Calendar API wrapper
│   ├── gmail/api.go         Gmail API wrapper
│   ├── auth/                OAuth2 PKCE (browser login)
│   └── config/              Token storage
└── .github/workflows/
    └── release.yml          CI: build + npm publish (OIDC)
```

Credentials are stored as a GitHub secret (GOOGLE_OAUTH_CREDENTIALS_JSON),
NOT in the repository. install.js downloads them during npm postinstall.

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

The package ships with a pre-registered Google Cloud OAuth client ID.
This is **not a secret** — it's the same mechanism used by every app
that offers "Sign in with Google" (Todoist, Notion, Fantastical, etc.).

The client ID is publicly visible in the authorization URL and only
serves to identify which app is requesting access. The actual security
is in the OAuth consent screen where **you** decide what to share.

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
| OAuth client ID | Embedded in the binary (public by design) |
| Access/Refresh tokens | `~/.config/pi-google-services/tokens.json` (0600 permissions) |
| No data leaves your machine | All Google API calls are direct from your binary |

The binary never phones home, tracks usage, or sends telemetry.

## Development

```bash
cp /path/to/credentials.json .
go build -o pi-google-services .
./pi-google-services login
./pi-google-services serve
```

## Tests

```bash
go test ./... -v
```

26 unit tests (MCP protocol, config, service metadata, services, MIME multipart attachments).

## License

MIT
