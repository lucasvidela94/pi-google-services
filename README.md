# pi-google-services

> **Google Calendar & Gmail MCP server for Pi**

Un MCP server en Go que le da a Pi herramientas para gestionar tu calendario y correo.
Binario único, sin runtime, sin dependencias. Login con Google una vez y funciona.

## ✨ Features

| Calendar | Gmail |
|----------|-------|
| ✅ Listar eventos | ✅ Listar inbox |
| ✅ Crear con invitados | ✅ Leer emails |
| ✅ Modificar / borrar | ✅ Buscar |
| ✅ Buscar por texto | ✅ Enviar |
| ✅ Ver disponibilidad | ✅ Responder |
| ✅ Multi-calendario | |

## 🚀 Instalación (recomendada)

```bash
pi install npm:pi-google-services
pi-google-services login
# Reiniciá sesión y ya podés gestionar tu calendario y correo
```

## 🔨 Build manual

```bash
git clone https://github.com/timolabs/pi-google-services
cd pi-google-services
go build -o pi-google-services .
cp pi-google-services ~/.local/bin/
```

Y agregá a `~/.pi/agent/mcp.json`:
```json
{
  "mcpServers": {
    "google-services": {
      "command": "/home/tu/.local/bin/pi-google-services",
      "args": ["serve"]
    }
  }
}
```

## 💻 Primer uso

```bash
pi-google-services login
# → se abre el navegador → autorizás con Google → listo
pi-google-services serve    # para el MCP server
```

## 🛠 Tools (12)

### 📅 Calendar

| Tool | Descripción |
|------|-------------|
| `list-events` | Ver eventos de un día/rango |
| `create-event` | Crear eventos con invitados |
| `update-event` | Modificar evento existente |
| `delete-event` | Borrar evento |
| `search-events` | Buscar eventos por texto |
| `list-calendars` | Ver todos tus calendarios |
| `get-freebusy` | Ver disponibilidad horaria |

### 📧 Gmail

| Tool | Descripción |
|------|-------------|
| `list-inbox` | Ver bandeja de entrada |
| `get-email` | Leer un email completo |
| `search-emails` | Buscar emails |
| `send-email` | Enviar email |
| `reply-to-email` | Responder un thread |

## 🏗️ Arquitectura

```
pi-google-services/
├── main.go                        # CLI (login/serve/status)
├── package.json                   # Pi package ✦ npm
├── SKILL.md                       # Pi skill: guía de uso
├── install.js                     # Postinstall: download binary + MCP config
├── uninstall.js                   # Cleanup
├── internal/
│   ├── mcp/server.go              # Core MCP protocol (JSON-RPC)
│   ├── services/interface.go      # Service interface
│   ├── services/calendar.go       # 7 Calendar tools
│   ├── services/gmail.go          # 5 Gmail tools
│   ├── calendar/api.go            # Google Calendar API wrapper
│   ├── gmail/api.go               # Gmail API wrapper
│   ├── auth/auth.go               # OAuth2 PKCE login
│   └── config/config.go           # Token storage
├── credentials.json               # ⚡ Embed en binary
├── .github/workflows/release.yml  # CI: build + npm publish
└── CHANGELOG.md
```

## 🧪 Tests

```bash
go test ./... -v
```

15 tests (MCP protocol, config, service metadata).

## 🔮 Próximos pasos

- [ ] Google Meet — crear links de Meet en eventos
- [ ] Google Tasks — gestionar tareas
- [ ] Google Drive — buscar archivos
- [ ] Eventos recurrentes avanzados
- [ ] Windows support

## 📝 Licencia

MIT
# pi-google-services
