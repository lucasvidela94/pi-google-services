package main

import (
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func TestWantsNoBrowser(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{name: "no args", args: nil, want: false},
		{name: "plain login", args: []string{"pi-google-services", "login"}, want: false},
		{name: "login with flag after subcommand", args: []string{"pi-google-services", "login", "--no-browser"}, want: true},
		{name: "setup with flag after subcommand", args: []string{"pi-google-services", "setup", "--no-browser"}, want: true},
		{name: "other flags do not trigger it", args: []string{"pi-google-services", "login", "--verbose"}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := wantsNoBrowser(tt.args); got != tt.want {
				t.Errorf("wantsNoBrowser(%v) = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}

func gzipBytes(t *testing.T, s string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	if _, err := w.Write([]byte(s)); err != nil {
		t.Fatalf("gzip write: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}
	return buf.Bytes()
}

func TestDecompressGzipLimited(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		limit   int64
		wantErr bool
		want    string
	}{
		{name: "under limit", input: "hola mundo", limit: 100, wantErr: false, want: "hola mundo"},
		{name: "exact limit ok", input: "1234567890", limit: 10, wantErr: false, want: "1234567890"},
		{name: "over limit aborts", input: "0123456789abcdef", limit: 10, wantErr: true},
		{name: "not gzip", input: "", limit: 10, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := gzipBytes(t, tt.input)
			if tt.name == "not gzip" {
				data = []byte("plain, not gzip")
			}
			path := filepath.Join(t.TempDir(), "out.bin")
			err := decompressGzipLimited(data, path, 0600, tt.limit)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
					t.Errorf("partial output should be removed on abort")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read output: %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("content = %q, want %q", got, tt.want)
			}
		})
	}
}
