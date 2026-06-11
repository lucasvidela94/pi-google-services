# Privacy Policy

pi-google-services is a local CLI tool that connects your Pi AI agent to your Google account.

## What data is accessed

This tool uses the Google OAuth 2.0 API to access the following Google services,
only after you explicitly authorize each one:

- **Google Calendar** — read and manage your events
- **Gmail** — read, send, and manage your emails
- **Google Tasks** — read and manage your tasks
- **Google Drive** — read and upload files
- **Google People / Contacts** — read and manage your contacts

## How data is used

All API calls are made **directly from your machine** to Google's servers.
No data passes through any intermediate server. The binary never:

- Phones home
- Tracks usage
- Collects analytics
- Sends telemetry
- Stores your data externally

## Where credentials are stored

- **OAuth tokens** are saved locally in `~/.config/pi-google-services/tokens.json`
  with restricted file permissions (0600).
- **Credentials** are downloaded once during installation and never leave your machine.

## Third-party services

This tool communicates exclusively with Google's official APIs
(calendar.googleapis.com, gmail.googleapis.com, etc.).
No other third-party services are used.

## Open Source

The full source code is available at:
https://github.com/lucasvidela94/pi-google-services

You can audit every line of code that handles your data.

## Contact

lucasan.videla@gmail.com
