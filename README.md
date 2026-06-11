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
| `send-email` | Send new email |
| `reply-to-email` | Reply to thread |

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

15 unit tests (MCP protocol, config, service metadata, services).

## License

MIT
