// Package auth — embedded OAuth client credentials.
//
// The release CI bakes the public desktop OAuth client ID (and its
// non-secret client secret) into the binary via ldflags:
//
//	go build -ldflags="-X github.com/sombi/pi-google-services/internal/auth.embeddedClientID=... -X ...embeddedClientSecret=..."
//
// The client ID is public by design: it appears in every authorization URL
// the user opens, exactly like rclone, gcalcli or gh. No secret ever lives
// in the git repo and nothing is downloaded at install time. Users who want
// their own Google Cloud project can still override via GOOGLE_OAUTH_CREDENTIALS
// or credentials.json (see loadCredentials in main.go).
package auth

import (
	"github.com/sombi/pi-google-services/internal/config"
)

// Set at release build time with -ldflags -X. Empty in dev checkouts.
var (
	embeddedClientID     string
	embeddedClientSecret string
)

// EmbeddedCredentials returns the baked-in OAuth client, if present.
func EmbeddedCredentials() (*config.Credentials, bool) {
	if embeddedClientID == "" {
		return nil, false
	}
	return &config.Credentials{
		Installed: config.InstalledConfig{
			ClientID:     embeddedClientID,
			ClientSecret: embeddedClientSecret,
			AuthURI:      "https://accounts.google.com/o/oauth2/auth",
			TokenURI:     "https://oauth2.googleapis.com/token",
		},
	}, true
}
