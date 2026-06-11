// Package auth handles the OAuth 2.0 PKCE flow for Google Calendar access.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log"
	"net"
	"net/http"
	"os/exec"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/sombi/pi-google-services/internal/config"
)

const redirectPath = "/oauth/callback"

// PKCEParams holds the PKCE code challenge data.
type PKCEParams struct {
	CodeVerifier  string
	CodeChallenge string
}

// GeneratePKCE creates a new PKCE challenge pair (S256 method).
func GeneratePKCE() (*PKCEParams, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("random: %w", err)
	}
	verifier := base64.RawURLEncoding.EncodeToString(b)

	h := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(h[:])

	return &PKCEParams{
		CodeVerifier:  verifier,
		CodeChallenge: challenge,
	}, nil
}

// Authenticator manages OAuth2 authentication with PKCE for Google APIs.
type Authenticator struct {
	OAuthConfig *oauth2.Config
	Token       *oauth2.Token
	cfg         *config.Config
}

// NewFromCredentials creates an Authenticator from a Google-provided credentials
// JSON file (the one you download from Google Cloud Console).
func NewFromCredentials(creds *config.Credentials) *Authenticator {
	installed := creds.Installed
	if installed.ClientID == "" {
		installed = creds.Web
	}

	cfg := &config.Config{
		ClientID:     installed.ClientID,
		ClientSecret: installed.ClientSecret,
		Scopes: []string{
			"https://www.googleapis.com/auth/calendar",
			"https://www.googleapis.com/auth/calendar.events",
		},
	}

	return newFromConfig(cfg)
}

// NewFromConfig creates an Authenticator from the app Config.
func NewFromConfig(cfg *config.Config) *Authenticator {
	return newFromConfig(cfg)
}

func newFromConfig(cfg *config.Config) *Authenticator {
	return &Authenticator{
		cfg: cfg,
		OAuthConfig: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			Scopes:       cfg.Scopes,
			Endpoint:     google.Endpoint,
			RedirectURL:  "http://localhost:0" + redirectPath,
		},
	}
}

// Login performs the PKCE OAuth flow:
// 1. Starts a local HTTP server on a random port
// 2. Opens the browser to Google's authorization endpoint
// 3. Catches the callback with the authorization code
// 4. Exchanges the code + code_verifier for tokens
// Returns the OAuth2 token.
func (a *Authenticator) Login(ctx context.Context) (*oauth2.Token, error) {
	pkce, err := GeneratePKCE()
	if err != nil {
		return nil, fmt.Errorf("generate pkce: %w", err)
	}

	// Local server to catch the redirect
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen: %w", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	a.OAuthConfig.RedirectURL = fmt.Sprintf("http://localhost:%d%s", port, redirectPath)

	// Build auth URL with PKCE params
	authURL := a.OAuthConfig.AuthCodeURL("state",
		oauth2.SetAuthURLParam("code_challenge", pkce.CodeChallenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc(redirectPath, func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			errCh <- fmt.Errorf("no code in callback: %s", r.URL.String())
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "Authorization failed. Close this window and try again.")
			return
		}
		codeCh <- code
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "✓ Authorized! You can close this window and return to Pi.")
	})

	server := &http.Server{Handler: mux}
	go server.Serve(listener)
	defer server.Close()

	// Open browser
	fmt.Println("\n📎 Opening browser for Google authorization...")
	if err := openBrowser(authURL); err != nil {
		fmt.Printf("Could not open browser automatically.\n")
		fmt.Printf("Open this URL manually:\n%s\n", authURL)
	} else {
		fmt.Println("Check your browser and authorize the application.")
	}

	// Wait for the auth code
	var authCode string
	select {
	case authCode = <-codeCh:
		fmt.Println("✓ Authorization code received, exchanging for tokens...")
	case err := <-errCh:
		return nil, fmt.Errorf("callback: %w", err)
	case <-ctx.Done():
		return nil, fmt.Errorf("login cancelled")
	}

	// Exchange code for token (PKCE verifier is required here)
	token, err := a.OAuthConfig.Exchange(ctx, authCode,
		oauth2.SetAuthURLParam("code_verifier", pkce.CodeVerifier),
	)
	if err != nil {
		return nil, fmt.Errorf("token exchange: %w", err)
	}

	a.Token = token

	// Save token to disk
	if err := saveOAuthToken(token); err != nil {
		log.Printf("Warning: could not save token: %v", err)
	}

	fmt.Println("✓ Authentication successful!")
	return token, nil
}

// TokenSource returns a TokenSource that auto-refreshes the OAuth token.
func (a *Authenticator) TokenSource(ctx context.Context, token *oauth2.Token) oauth2.TokenSource {
	return a.OAuthConfig.TokenSource(ctx, token)
}

// saveOAuthToken persists the token to the config directory.
func saveOAuthToken(token *oauth2.Token) error {
	t := &config.Tokens{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenType:    token.TokenType,
		Expiry:       token.Expiry.Format(time.RFC3339),
	}
	if err := config.SaveTokens(t); err != nil {
		return fmt.Errorf("save tokens: %w", err)
	}
	return nil
}

// LoadToken reads a previously stored token from disk.
func LoadToken() (*oauth2.Token, error) {
	t, err := config.LoadTokens()
	if err != nil {
		return nil, err
	}
	if t.AccessToken == "" {
		return nil, nil
	}
	expiry, _ := time.Parse(time.RFC3339, t.Expiry)
	return &oauth2.Token{
		AccessToken:  t.AccessToken,
		RefreshToken: t.RefreshToken,
		TokenType:    t.TokenType,
		Expiry:       expiry,
	}, nil
}

// HasToken returns true if a stored token exists on disk.
func HasToken() bool {
	t, err := LoadToken()
	return err == nil && t != nil
}

// openBrowser opens a URL in the default system browser.
func openBrowser(url string) error {
	// Try xdg-open first (Linux with desktop env)
	if err := exec.Command("xdg-open", url).Start(); err == nil {
		return nil
	}
	// macOS
	if err := exec.Command("open", url).Start(); err == nil {
		return nil
	}
	// Windows
	if err := exec.Command("cmd", "/c", "start", url).Start(); err == nil {
		return nil
	}
	// Fallback: try common browsers
	for _, browser := range []string{"x-www-browser", "firefox", "google-chrome", "chromium", "brave"} {
		if err := exec.Command(browser, url).Start(); err == nil {
			return nil
		}
	}
	return fmt.Errorf("no browser found; open manually: %s", url)
}
