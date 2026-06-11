# pi-google-services

Google Calendar & Gmail MCP server for Pi. Lets your Pi agent manage your calendar and emails through natural language.

## Setup

```bash
pi-google-services login    # Opens browser → authorize with Google
pi-google-services serve    # Start MCP server (done automatically by Pi)
```

## Calendar tools

### `list-events`
Show events in a date range.

Examples:
- "mostrame los eventos de mañana"
- "qué tengo esta semana?"
- "eventos del 20 al 25 de junio"

Arguments:
- `timeMin` (ISO 8601) — start of range (default: today 00:00)
- `timeMax` (ISO 8601) — end of range (default: today 23:59)
- `calendarId` — calendar to query (default: primary)
- `maxResults` — max events (default: 50)

### `create-event`
Create a new calendar event with optional attendees and Google Meet.

Examples:
- "creá una reunión mañana a las 15"
- "agendá una llamada con juan@gmail.com el jueves a las 10"
- "creá un evento 'Cumpleaños' el 25/12 todo el día"
- "creá un meet virtual mañana a las 16 con Meet"

Arguments:
- `summary` (required) — event title
- `startTime` (required) — ISO 8601 start
- `endTime` (required) — ISO 8601 end
- `attendees` — comma-separated emails to invite
- `withMeet` (boolean) — add Google Meet link
- `description` — event description
- `location` — event location
- `calendarId` — target calendar (default: primary)

### `update-event`
Modify an existing event.

Examples:
- "cambiá la reunión de mañana a las 16"
- "renombrá el evento de la cena a 'Cena con amigos'"

Arguments:
- `eventId` (required) — event to modify
- `summary`, `startTime`, `endTime` — fields to update
- `calendarId` — target calendar (default: primary)

### `delete-event`
Remove an event.

Examples:
- "borrá el evento de prueba del viernes"
- "eliminá la reunión de las 15"

Arguments:
- `eventId` (required) — event to delete
- `calendarId` — target calendar (default: primary)

### `search-events`
Search events by text.

Examples:
- "buscá eventos de 'reunión'"
- "encontrá cuando hablé de 'presentación'"

Arguments:
- `query` (required) — text to search
- `maxResults` — max results (default: 50)

### `list-calendars`
List all available calendars.

Examples:
- "mostrame mis calendarios"
- "qué calendarios tengo?"

### `get-freebusy`
Check availability across calendars.

Examples:
- "estoy libre mañana a las 15?"
- "qué horarios tengo ocupados esta semana?"

Arguments:
- `timeMin`, `timeMax` — range to check
- `calendarIds` — comma-separated calendar IDs

## Gmail tools

### `list-inbox`
List recent inbox messages.

Examples:
- "mostrame mis emails"
- "qué hay en mi bandeja de entrada?"
- "mostrame los últimos 5 emails de LinkedIn"

Arguments:
- `maxResults` — max emails (default: 20)
- `query` — optional Gmail search filter

### `get-email`
Read a full email by ID.

Examples:
- "leé el primer email de la lista"
- "mostrame el contenido completo del mail de belo"

Arguments:
- `id` (required) — email message ID

### `search-emails`
Search emails with Gmail syntax.

Examples:
- "buscá emails de belo"
- "encontrá mails sobre 'Uber' de esta semana"
- "mostrame los no leídos"

Arguments:
- `query` (required) — Gmail search query
- `maxResults` — max results (default: 20)

### `send-email`
Send a new email.

Examples:
- "enviále un mail a lucsk94@gmail.com con asunto 'Prueba' y cuerpo 'Hola, esto es una prueba'"
- "mandále un email a juan@mail.com diciendo que la reunión se pasó al viernes"

Arguments:
- `to` (required) — recipient email
- `subject` (required) — email subject
- `body` — email body text

### `reply-to-email`
Reply to an existing thread.

Examples:
- "respondé el mail de Natalia aceptando la invitación"
- "contestále al de belo que ya lo vi"

Arguments:
- `threadId` (required) — thread to reply to
- `to` (required) — recipient email
- `subject` (required) — reply subject
- `body` — reply body text
