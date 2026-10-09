// Package auth handles the OAuth 2.0 PKCE flow for Google Calendar access.
package auth

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/sombi/pi-google-services/internal/config"
)

const redirectPath = "/oauth/callback"

// ParseAuthCode extracts the authorization code from user-pasted input.
// It accepts either a full callback URL (copied from the browser address bar
// after Google redirects, even if the page failed to load) or a raw code.
func ParseAuthCode(pasted string) (string, error) {
	pasted = strings.TrimSpace(pasted)
	if pasted == "" {
		return "", fmt.Errorf("no input: paste the callback URL or the code")
	}

	// Raw code (no URL): accept as-is if it looks like one token.
	// Google auth codes never contain whitespace; anything with spaces is
	// user chatter, not a code.
	if !strings.Contains(pasted, "://") {
		if strings.ContainsAny(pasted, " \t") {
			return "", fmt.Errorf("not a valid code or URL: %q", pasted)
		}
		return pasted, nil
	}

	u, err := url.Parse(pasted)
	if err != nil {
		return "", fmt.Errorf("parse pasted URL: %w", err)
	}
	q := u.Query()
	if e := q.Get("error"); e != "" {
		desc := q.Get("error_description")
		if desc != "" {
			return "", fmt.Errorf("authorization denied (%s): %s", e, desc)
		}
		return "", fmt.Errorf("authorization denied (%s)", e)
	}
	code := q.Get("code")
	if code == "" {
		return "", fmt.Errorf("no code found in pasted URL")
	}
	return code, nil
}

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
}

// NewFromCredentials creates an Authenticator from a Google-provided credentials
// JSON file (the one you download from Google Cloud Console).
// Scopes are provided by the caller (e.g. allScopes() in main) so setup and
// login always request the same set of permissions.
func NewFromCredentials(creds *config.Credentials, scopes []string) *Authenticator {
	app := creds.AppConfig()
	return &Authenticator{
		OAuthConfig: &oauth2.Config{
			ClientID:     app.ClientID,
			ClientSecret: app.ClientSecret,
			Scopes:       scopes,
			Endpoint:     google.Endpoint,
			RedirectURL:  "http://localhost:0" + redirectPath,
		},
	}
}

// promptForAuthCode shows the auth URL with manual-mode instructions and
// reads user input until it can parse an authorization code. It re-prompts
// on unparseable input and fails immediately on a denied authorization.
func promptForAuthCode(authURL string, input io.Reader, output io.Writer) (string, error) {
	scanner := bufio.NewScanner(input)
	for {
		fmt.Fprintf(output, `
No browser available — authorize from any device:

  1. Open this URL (on this machine or your phone):

     %s

  2. Approve the consent screen. Google will redirect to a localhost page
     that FAILS TO LOAD — that is expected.

  3. Copy the full URL from the browser address bar and paste it here.

Paste the URL or code (empty line to retry, Ctrl+C to cancel):
> `, authURL)

		if !scanner.Scan() {
			return "", fmt.Errorf("no authorization code entered")
		}
		code, err := ParseAuthCode(scanner.Text())
		if err == nil {
			return code, nil
		}
		if strings.Contains(err.Error(), "denied") {
			return "", err
		}
		fmt.Fprintf(output, "⚠ %v\n", err)
	}
}

// LoginOptions configures the login flow. Zero values give the classic
// behavior: try the system browser and fail loudly if unavailable.
type LoginOptions struct {
	// NoBrowser skips the browser/loopback flow entirely and prompts for
	// a pasted authorization URL instead. Use on headless hosts (SSH,
	// VPS, containers) or inside WSL when localhost forwarding breaks.
	NoBrowser bool

	// Input is where the pasted URL is read from in manual mode
	// (default os.Stdin). Output receives prompts (default os.Stdout).
	Input  io.Reader
	Output io.Writer

	// OpenBrowser overrides the default browser launcher (tests).
	OpenBrowser func(string) error
}

func (o LoginOptions) input() io.Reader {
	if o.Input != nil {
		return o.Input
	}
	return os.Stdin
}

func (o LoginOptions) output() io.Writer {
	if o.Output != nil {
		return o.Output
	}
	return os.Stdout
}

func (o LoginOptions) openBrowser(ctx context.Context) func(string) error {
	if o.OpenBrowser != nil {
		return o.OpenBrowser
	}
	return func(url string) error { return openBrowser(ctx, url) }
}

// Login performs the PKCE OAuth flow with default options.
func (a *Authenticator) Login(ctx context.Context) (*oauth2.Token, error) {
	return a.LoginWithOptions(ctx, LoginOptions{})
}

// LoginWithOptions performs the PKCE OAuth flow. It tries the loopback +
// browser flow unless opts.NoBrowser is set, and falls back to manual
// paste mode when the browser cannot be launched.
func (a *Authenticator) LoginWithOptions(ctx context.Context, opts LoginOptions) (*oauth2.Token, error) {
	pkce, err := GeneratePKCE()
	if err != nil {
		return nil, fmt.Errorf("generate pkce: %w", err)
	}

	// The callback server always runs: it allocates the real loopback port
	// that goes into the authorization URL (port 0 is not a valid
	// redirect), and catches the redirect whenever a browser is involved.
	cb, cleanup, err := startCallbackServer(ctx)
	if err != nil {
		return nil, err
	}
	defer cleanup()
	a.OAuthConfig.RedirectURL = cb.url

	authURL := a.authCodeURL(pkce)

	var authCode string
	switch {
	case opts.NoBrowser:
		if authCode, err = promptForAuthCode(authURL, opts.input(), opts.output()); err != nil {
			return nil, err
		}
	default:
		fmt.Fprintln(opts.output(), "\n📎 Opening browser for Google authorization...")
		if openErr := opts.openBrowser(ctx)(authURL); openErr != nil {
			// Browser unavailable: degrade to manual paste instead of dying.
			fmt.Fprintln(opts.output(), "\n⚠ Could not launch a browser — switching to manual mode.")
			if authCode, err = promptForAuthCode(authURL, opts.input(), opts.output()); err != nil {
				return nil, err
			}
			break
		}
		fmt.Fprintln(opts.output(), "Check your browser and authorize the application.")

		select {
		case authCode = <-cb.codeCh:
			fmt.Fprintln(opts.output(), "✓ Authorization code received, exchanging for tokens...")
		case err := <-cb.errCh:
			return nil, fmt.Errorf("callback: %w", err)
		case <-ctx.Done():
			return nil, fmt.Errorf("login cancelled")
		}
	}

	token, err := a.exchange(ctx, pkce, authCode)
	if err != nil {
		return nil, err
	}

	a.Token = token

	if err := saveOAuthToken(token); err != nil {
		log.Printf("Warning: could not save token: %v", err)
	}

	fmt.Fprintln(opts.output(), "✓ Authentication successful!")
	return token, nil
}

// authCodeURL builds the Google authorization URL with PKCE parameters.
func (a *Authenticator) authCodeURL(pkce *PKCEParams) string {
	return a.OAuthConfig.AuthCodeURL("state",
		oauth2.SetAuthURLParam("code_challenge", pkce.CodeChallenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)
}

// exchange trades an authorization code for tokens using the PKCE verifier
// and persists the result to disk.
func (a *Authenticator) exchange(ctx context.Context, pkce *PKCEParams, code string) (*oauth2.Token, error) {
	token, err := a.OAuthConfig.Exchange(ctx, code,
		oauth2.SetAuthURLParam("code_verifier", pkce.CodeVerifier),
	)
	if err != nil {
		return nil, fmt.Errorf("token exchange: %w", err)
	}
	return token, nil
}

// callbackServer is the local loopback server that catches Google's
// redirect. It always runs, even in manual paste mode: it allocates the
// real port that goes into the authorization URL.
type callbackServer struct {
	server *http.Server
	url    string // full redirect URL (http://localhost:<port>/oauth/callback)
	codeCh <-chan string
	errCh  <-chan error
}

// startCallbackServer binds a loopback listener and serves the OAuth
// callback handler on a random port. Call cleanup when done.
func startCallbackServer(ctx context.Context) (*callbackServer, func(), error) {
	lc := net.ListenConfig{}
	listener, err := lc.Listen(ctx, "tcp", "127.0.0.1:0")
	if err != nil {
		return nil, nil, fmt.Errorf("listen: %w", err)
	}

	port := listener.Addr().(*net.TCPAddr).Port

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

	server := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go server.Serve(listener) //nolint:errcheck // listener already bound

	cleanup := func() {
		_ = server.Close()
		_ = listener.Close()
	}

	return &callbackServer{
		server: server,
		url:    fmt.Sprintf("http://localhost:%d%s", port, redirectPath),
		codeCh: codeCh,
		errCh:  errCh,
	}, cleanup, nil
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
		return nil, fmt.Errorf("load tokens: %w", err)
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
func openBrowser(ctx context.Context, url string) error {
	// Try xdg-open first (Linux with desktop env)
	if err := exec.CommandContext(ctx, "xdg-open", url).Start(); err == nil {
		return nil
	}
	// macOS
	if err := exec.CommandContext(ctx, "open", url).Start(); err == nil {
		return nil
	}
	// Windows
	if err := exec.CommandContext(ctx, "cmd", "/c", "start", url).Start(); err == nil {
		return nil
	}
	// Fallback: try common browsers
	for _, browser := range []string{"x-www-browser", "firefox", "google-chrome", "chromium", "brave"} {
		if err := exec.CommandContext(ctx, browser, url).Start(); err == nil {
			return nil
		}
	}
	return fmt.Errorf("no browser found; open manually: %s", url)
}
