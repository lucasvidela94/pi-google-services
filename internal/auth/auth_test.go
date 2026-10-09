package auth

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/oauth2"

	"github.com/sombi/pi-google-services/internal/config"
)

func TestParseAuthCode(t *testing.T) {
	tests := []struct {
		name    string
		pasted  string
		want    string
		wantErr bool
	}{
		{
			name:   "full callback URL with code",
			pasted: "http://localhost:49152/oauth/callback?state=state&code=4/0AXXXyyy&scope=email",
			want:   "4/0AXXXyyy",
		},
		{
			name:   "https callback URL",
			pasted: "http://localhost:8080/oauth/callback?code=abc.def-ghi",
			want:   "abc.def-ghi",
		},
		{
			name:   "raw code only",
			pasted: "4/0AaSdBcDeFgHiJkLmNoPqRsTuVwXyZ0123456789",
			want:   "4/0AaSdBcDeFgHiJkLmNoPqRsTuVwXyZ0123456789",
		},
		{
			name:   "URL-encoded code gets decoded",
			pasted: "http://localhost:8080/oauth/callback?code=4%2F0AQSTVwxYz&scope=calendar",
			want:   "4/0AQSTVwxYz",
		},
		{
			name:   "surrounding whitespace and newline trimmed",
			pasted: "  http://localhost:8080/oauth/callback?code=xyz123\n",
			want:   "xyz123",
		},
		{
			name:    "error param access_denied reports denial",
			pasted:  "http://localhost:8080/oauth/callback?error=access_denied&error_description=User+denied+access",
			wantErr: true,
		},
		{
			name:    "empty input",
			pasted:  "",
			wantErr: true,
		},
		{
			name:    "whitespace only",
			pasted:  "   \n\t ",
			wantErr: true,
		},
		{
			name:    "URL without code",
			pasted:  "http://localhost:8080/oauth/callback?state=state",
			wantErr: true,
		},
		{
			name:    "multi-word garbage is not accepted as raw code",
			pasted:  "oops let me find it",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseAuthCode(tt.pasted)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ParseAuthCode(%q) = %q, want error", tt.pasted, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseAuthCode(%q): %v", tt.pasted, err)
			}
			if got != tt.want {
				t.Errorf("ParseAuthCode(%q) = %q, want %q", tt.pasted, got, tt.want)
			}
		})
	}
}

func TestPromptForAuthCode(t *testing.T) {
	const testURL = "https://accounts.google.com/o/oauth2/auth?client_id=x"

	tests := []struct {
		name      string
		input     string
		wantCode  string
		wantErr   bool
		errDenied bool
		wantURLIn int // how many times authURL must appear in output
	}{
		{
			name:      "valid URL pasted first try",
			input:     "http://localhost:8080/oauth/callback?code=goodcode\n",
			wantCode:  "goodcode",
			wantURLIn: 1,
		},
		{
			name:      "garbage then valid code",
			input:     "oops let me find it\n\nhttp://localhost:8080/oauth/callback?code=second\n",
			wantCode:  "second",
			wantURLIn: 2,
		},
		{
			name:      "access_denied fails immediately",
			input:     "http://localhost:8080/oauth/callback?error=access_denied\nmore input that must not be read\n",
			wantErr:   true,
			errDenied: true,
			wantURLIn: 1,
		},
		{
			name:      "EOF without valid code",
			input:     "",
			wantErr:   true,
			wantURLIn: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out strings.Builder
			in := strings.NewReader(tt.input)

			code, err := promptForAuthCode(testURL, in, &out)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("promptForAuthCode() = %q, want error", code)
				}
				if tt.errDenied && !strings.Contains(err.Error(), "denied") {
					t.Errorf("error = %q, want it to mention denial", err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("promptForAuthCode(): %v", err)
			}
			if code != tt.wantCode {
				t.Errorf("promptForAuthCode() = %q, want %q", code, tt.wantCode)
			}
			if got := strings.Count(out.String(), testURL); got < tt.wantURLIn {
				t.Errorf("authURL shown %d times, want at least %d\noutput:\n%s", got, tt.wantURLIn, out.String())
			}
		})
	}
}

// newTestAuthenticator builds an Authenticator whose token endpoint points
// at a fake Google server returning a fixed token payload.
func newTestAuthenticator(t *testing.T, tokenJSON string) (*Authenticator, *httptest.Server) {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("fake token endpoint: bad form: %v", err)
		}
		if r.Form.Get("code_verifier") == "" && r.Form.Get("grant_type") != "refresh_token" {
			t.Errorf("token request missing code_verifier (PKCE): %v", r.Form)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, tokenJSON)
	}))
	t.Cleanup(ts.Close)

	a := &Authenticator{
		OAuthConfig: &oauth2.Config{
			ClientID:     "test-client",
			ClientSecret: "test-secret",
			Scopes:       []string{"email"},
			Endpoint: oauth2.Endpoint{
				AuthURL:  ts.URL + "/auth",
				TokenURL: ts.URL + "/token",
			},
			RedirectURL: "http://localhost:1/oauth/callback",
		},
	}
	return a, ts
}

// withTempConfigDir points config.Dir at a temp dir for the duration of the
// test and returns a cleanup restoring the original. Use as:
//
//	defer withTempConfigDir(t)()
//
// (Calling it WITHOUT the trailing () — or via bare `defer f(t)` —
// installs the override too late, and fake login tokens would overwrite
// the user's real tokens.json. See issue: suite clobbered real tokens.)
func withTempConfigDir(t *testing.T) func() {
	t.Helper()
	dir := t.TempDir()
	old := config.Dir
	config.Dir = func() (string, error) { return dir, nil }
	return func() { config.Dir = old }
}

const fakeTokenJSON = `{"access_token":"fake-at","refresh_token":"fake-rt","token_type":"Bearer","expires_in":3600}`

func TestLoginManualMode(t *testing.T) {
	defer withTempConfigDir(t)()
	a, _ := newTestAuthenticator(t, fakeTokenJSON)

	var out strings.Builder
	in := strings.NewReader("http://localhost:9999/oauth/callback?state=state&code=pasted-code\n")

	token, err := a.LoginWithOptions(context.Background(), LoginOptions{
		NoBrowser: true,
		Input:     in,
		Output:    &out,
	})
	if err != nil {
		t.Fatalf("LoginWithOptions(manual): %v", err)
	}
	if token.AccessToken != "fake-at" || !token.Valid() {
		t.Errorf("token = %+v, want access_token=fake-at and valid", token)
	}
	if !strings.Contains(out.String(), "/auth?") {
		t.Error("manual output must show the auth URL")
	}
}

func TestLoginBrowserFailureFallsBackToManual(t *testing.T) {
	defer withTempConfigDir(t)()
	a, _ := newTestAuthenticator(t, fakeTokenJSON)

	var out strings.Builder
	in := strings.NewReader("raw-pasted-code-123\n")

	token, err := a.LoginWithOptions(context.Background(), LoginOptions{
		Input:       in,
		Output:      &out,
		OpenBrowser: func(_ string) error { return fmt.Errorf("no browser here") },
	})
	if err != nil {
		t.Fatalf("LoginWithOptions(fallback): %v", err)
	}
	if token.AccessToken != "fake-at" {
		t.Errorf("access_token = %q, want fake-at", token.AccessToken)
	}
	if s := out.String(); !strings.Contains(s, "no-browser") && !strings.Contains(s, "No browser") {
		t.Errorf("output should explain manual mode, got:\n%s", s)
	}
}

func TestLoginBrowserFlowEndToEnd(t *testing.T) {
	defer withTempConfigDir(t)()
	a, _ := newTestAuthenticator(t, fakeTokenJSON)

	// Simulate Google: capture the auth URL from the browser stub, resolve
	// its loopback redirect target, and hit the callback with a code.
	openStub := func(authURL string) error {
		go func() {
			u, err := url.Parse(authURL)
			if err != nil {
				return
			}
			redirect := u.Query().Get("redirect_uri")
			ru, err := url.Parse(redirect)
			if err != nil {
				return
			}
			callback := fmt.Sprintf("http://localhost:%s%s?state=state&code=e2e-code", ru.Port(), ru.Path)
			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, callback, nil)
			if err != nil {
				return
			}
			if resp, err := http.DefaultClient.Do(req); err == nil {
				_ = resp.Body.Close()
			}
		}()
		return nil
	}

	var out strings.Builder

	token, err := a.LoginWithOptions(context.Background(), LoginOptions{
		Input:       strings.NewReader(""),
		Output:      &out,
		OpenBrowser: openStub,
	})
	if err != nil {
		t.Fatalf("LoginWithOptions(browser): %v", err)
	}
	if token.AccessToken != "fake-at" {
		t.Errorf("access_token = %q, want fake-at", token.AccessToken)
	}
}

func TestLoginManualModeUsesRealLoopbackPort(t *testing.T) {
	defer withTempConfigDir(t)()
	a, _ := newTestAuthenticator(t, fakeTokenJSON)

	var out strings.Builder
	in := strings.NewReader("http://localhost:9999/oauth/callback?code=c\n")

	if _, err := a.LoginWithOptions(context.Background(), LoginOptions{
		NoBrowser: true,
		Input:     in,
		Output:    &out,
	}); err != nil {
		t.Fatalf("LoginWithOptions(manual): %v", err)
	}

	// Regression: the redirect URL must carry a real allocated port,
	// not the placeholder port 0 (Google rejects invalid loopback ports).
	re := regexp.MustCompile(`http://localhost:(\d+)/oauth/callback`)
	m := re.FindStringSubmatch(a.OAuthConfig.RedirectURL)
	if m == nil {
		t.Fatalf("RedirectURL = %q, want localhost:<port>/oauth/callback", a.OAuthConfig.RedirectURL)
	}
	port, err := strconv.Atoi(m[1])
	if err != nil || port <= 0 || port > 65535 {
		t.Errorf("RedirectURL port = %q, want a valid TCP port", m[1])
	}
}
