---
description: "Gmail, Calendar, Tasks, Drive, Contacts, Forms, Docs & Sheets MCP tools for Pi (OAuth, single binary)"
---

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
- `pageToken` — pagination token to get the next page (from a previous result)

### `get-email`
Read a full email by ID.

Examples:
- "leé el primer email de la lista"
- "mostrame el contenido completo del mail de belo"

Arguments:
- `id` (required) — email message ID

### `search-emails`
Search emails with Gmail syntax across the whole mailbox (sent, inbox, etc.).

Examples:
- "buscá emails de belo"
- "encontrá mails sobre 'Uber' de esta semana"
- "mostrame los no leídos"
- "mostrame los emails que envié en agosto" (`query: in:sent after:2026/08/01`)

Arguments:
- `query` (required) — Gmail search query
- `maxResults` — max results (default: 20)
- `pageToken` — pagination token to get the next page (from a previous result)

### `send-email`
Send a new email with optional file attachments.

Examples:
- "enviále un mail a lucsk94@gmail.com con asunto 'Prueba' y cuerpo 'Hola, esto es una prueba'"
- "mandále un email a juan@mail.com diciendo que la reunión se pasó al viernes"
- "enviále un email a maria@work.com con el PDF /home/user/reporte.pdf adjunto"

Arguments:
- `to` (required) — recipient email
- `subject` (required) — email subject
- `body` — email body text
- `attachments` — array of files to attach. Each item can have `localPath` (local file) or `driveFileId` (Google Drive file ID)

### `reply-to-email`
Reply to an existing thread with optional file attachments.

Examples:
- "respondé el mail de Natalia aceptando la invitación"
- "contestále al de belo que ya lo vi"
- "respondé al mail de ventas adjuntando /home/user/cotizacion.pdf"

Arguments:
- `threadId` (required) — thread to reply to
- `to` (required) — recipient email
- `subject` (required) — reply subject
- `body` — reply body text
- `attachments` — array of files to attach. Each item can have `localPath` (local file) or `driveFileId` (Google Drive file ID)

## Tasks tools

### `list-tasklists`
Show all task lists.

Examples:
- "mostrame mis listas de tareas"

### `list-tasks`
List tasks (pending, completed, or all).

Examples:
- "mostrame mis tareas pendientes"
- "qué tareas tengo para hacer?"
- "mostrame las tareas completadas"

Arguments:
- `taskListId` — task list ID (default: @default)
- `status` — filter: pending, completed, or '' for all
- `maxResults` — max results (default: 50)

### `create-task`
Create a new task.

Examples:
- "creá una tarea para comprar leche"
- "agendá 'llamar al dentista' para mañana"

Arguments:
- `title` (required) — task title
- `taskListId` — target list (default: @default)
- `notes` — optional description
- `dueDate` — due date (YYYY-MM-DD)

### `complete-task`
Mark a task as done.

Examples:
- "marcá como hecha la tarea de la leche"
- "completá la tarea del dentista"

Arguments:
- `taskId` (required) — task to complete
- `taskListId` — target list (default: @default)

### `delete-task`
Delete a task.

Examples:
- "borrá la tarea de prueba"

Arguments:
- `taskId` (required) — task to delete
- `taskListId` — target list (default: @default)

## Forms tools

> Requires re-authorization after updating (`pi-google-services login`), since
> Forms adds new OAuth scopes. The Forms API creates/reads forms and reads
> responses — it cannot submit responses as a user.

### `create-form`
Create an empty form, then add sections/questions.

Examples:
- "creá un formulario 'Cuestionario de política de la clínica'"

Arguments:
- `title` (required) — form title
- `description` — form description

### `add-form-section`
Add a section header (title + description, no question).

Arguments:
- `formId` (required) — form ID
- `title` (required) — section title
- `description` — section description

### `add-form-question`
Add a question. Without `options` it is free text; with `options`
(comma-separated, e.g. `Sí,No`) it is a choice question.

Arguments:
- `formId` (required) — form ID
- `title` (required) — question text
- `description` — help text under the question
- `paragraph` — long-text answer instead of short text (default: false)
- `required` — whether an answer is required (default: false)
- `options` — comma-separated choices (omit for free text)
- `choiceType` — RADIO (default), CHECKBOX or DROP_DOWN

### `get-form`
Show a form's structure and responder link.

Arguments:
- `formId` (required) — form ID

### `list-responses`
List responses submitted to a form.

Arguments:
- `formId` (required) — form ID
- `limit` — max responses (default: 100)

## Docs tools

> Requires re-authorization after updating (`pi-google-services login`), since
> Docs adds a new OAuth scope.

### `get-doc`
Read a Google Doc as plain text (title + body, including tables).

Examples:
- "leé este doc y resumilo"
- "qué dice el documento de la reunión?"

Arguments:
- `docId` (required) — document ID (from Drive URL or `search-drive`)

### `create-doc`
Create an empty Google Doc.

Arguments:
- `title` (required) — document title

### `append-to-doc`
Append text at the end of a Google Doc (newlines create new paragraphs).

Arguments:
- `docId` (required) — document ID
- `text` (required) — text to append

## Sheets tools (read-only)

> Requires re-authorization after updating (`pi-google-services login`), since
> Sheets adds a new OAuth scope. Write access is intentionally excluded.

### `list-sheets`
List tab names of a spreadsheet.

Arguments:
- `spreadsheetId` (required) — spreadsheet ID (from Drive URL or `search-drive`)

### `read-sheet`
Read cell values from a spreadsheet range.

Arguments:
- `spreadsheetId` (required) — spreadsheet ID
- `range` — A1 notation (e.g. `A1:C10`, `'Hoja 1'!A1:B5`). Default: `A1:Z100` on the first tab
