package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDir(t *testing.T) {
	dir, err := Dir()
	if err != nil {
		t.Fatalf("Dir(): %v", err)
	}
	if dir == "" {
		t.Fatal("Dir() returned empty")
	}
	if !filepath.IsAbs(dir) {
		t.Errorf("Dir() = %q, want absolute path", dir)
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	tmp := t.TempDir()
	origDir := Dir
	Dir = func() (string, error) { return tmp, nil }
	t.Cleanup(func() { Dir = origDir })

	cfg := &Config{
		ClientID: "test-client-id",
		Scopes:   []string{"scope1", "scope2"},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save(): %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load(): %v", err)
	}
	if loaded.ClientID != "test-client-id" {
		t.Errorf("ClientID = %q, want %q", loaded.ClientID, "test-client-id")
	}
	if len(loaded.Scopes) != 2 {
		t.Errorf("Scopes = %v, want 2", loaded.Scopes)
	}
}

func TestSaveAndLoadTokens(t *testing.T) {
	tmp := t.TempDir()
	origDir := Dir
	Dir = func() (string, error) { return tmp, nil }
	t.Cleanup(func() { Dir = origDir })

	tok := &Tokens{
		AccessToken:  "access123",
		RefreshToken: "refresh456",
		TokenType:    "Bearer",
	}
	if err := SaveTokens(tok); err != nil {
		t.Fatalf("SaveTokens(): %v", err)
	}

	loaded, err := LoadTokens()
	if err != nil {
		t.Fatalf("LoadTokens(): %v", err)
	}
	if loaded.AccessToken != "access123" {
		t.Errorf("AccessToken = %q, want %q", loaded.AccessToken, "access123")
	}
	if loaded.RefreshToken != "refresh456" {
		t.Errorf("RefreshToken = %q, want %q", loaded.RefreshToken, "refresh456")
	}
}

func TestLoadTokensEmpty(t *testing.T) {
	tmp := t.TempDir()
	origDir := Dir
	Dir = func() (string, error) { return tmp, nil }
	t.Cleanup(func() { Dir = origDir })

	tok, err := LoadTokens()
	if err != nil {
		t.Fatalf("LoadTokens(): %v", err)
	}
	if tok.AccessToken != "" {
		t.Errorf("expected empty token, got %q", tok.AccessToken)
	}
}

func TestLoadCredentialsFromBytes(t *testing.T) {
	data := []byte(`{
		"installed": {
			"client_id": "test-id.apps.googleusercontent.com",
			"client_secret": "test-secret",
			"auth_uri": "https://accounts.google.com/o/oauth2/auth",
			"token_uri": "https://oauth2.googleapis.com/token",
			"project_id": "test-project"
		}
	}`)

	creds, err := LoadCredentialsFromBytes(data)
	if err != nil {
		t.Fatalf("LoadCredentialsFromBytes(): %v", err)
	}
	if creds.Installed.ClientID != "test-id.apps.googleusercontent.com" {
		t.Errorf("ClientID = %q", creds.Installed.ClientID)
	}
	if creds.Installed.ClientSecret != "test-secret" {
		t.Errorf("ClientSecret = %q", creds.Installed.ClientSecret)
	}
}

func TestLoadCredentialsFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "credentials.json")
	data := `{"installed":{"client_id":"file-id.apps.googleusercontent.com","client_secret":"file-secret"}}`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	creds, err := LoadCredentials(path)
	if err != nil {
		t.Fatalf("LoadCredentials(): %v", err)
	}
	if creds.Installed.ClientID != "file-id.apps.googleusercontent.com" {
		t.Errorf("ClientID = %q", creds.Installed.ClientID)
	}
}
