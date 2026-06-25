package services

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveAttachments_Empty(t *testing.T) {
	gs := &GmailService{}
	atts, err := gs.resolveAttachments(context.Background(), nil)
	if err != nil {
		t.Fatalf("expected no error, got %+v", err)
	}
	if atts != nil {
		t.Errorf("expected nil attachments, got %d items", len(atts))
	}
}

func TestResolveAttachments_LocalPath(t *testing.T) {
	tmp := t.TempDir()
	filePath := filepath.Join(tmp, "test.txt")
	if err := os.WriteFile(filePath, []byte("hello world"), 0644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	gs := &GmailService{}
	atts, err := gs.resolveAttachments(context.Background(), []attachmentInput{
		{LocalPath: filePath},
	})
	if err != nil {
		t.Fatalf("expected no error, got %+v", err)
	}
	if len(atts) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(atts))
	}
	if atts[0].Filename != "test.txt" {
		t.Errorf("filename = %q, want %q", atts[0].Filename, "test.txt")
	}
	if string(atts[0].Data) != "hello world" {
		t.Errorf("data = %q, want %q", string(atts[0].Data), "hello world")
	}
	if atts[0].MimeType != "text/plain; charset=utf-8" {
		t.Errorf("mimeType = %q, want %q", atts[0].MimeType, "text/plain; charset=utf-8")
	}
}

func TestResolveAttachments_DriveFileID_NoDriveAPI(t *testing.T) {
	gs := &GmailService{}
	_, err := gs.resolveAttachments(context.Background(), []attachmentInput{
		{DriveFileID: "abc123"},
	})
	if err == nil {
		t.Fatal("expected error when Drive API not configured")
	}
	if err.Code != -32603 {
		t.Errorf("error code = %d, want -32603", err.Code)
	}
}

func TestResolveAttachments_NeitherPathNorID(t *testing.T) {
	gs := &GmailService{}
	_, err := gs.resolveAttachments(context.Background(), []attachmentInput{
		{},
	})
	if err == nil {
		t.Fatal("expected error for empty attachment")
	}
	if err.Code != -32602 {
		t.Errorf("error code = %d, want -32602", err.Code)
	}
}

func TestResolveAttachments_LocalFileNotFound(t *testing.T) {
	gs := &GmailService{}
	_, err := gs.resolveAttachments(context.Background(), []attachmentInput{
		{LocalPath: "/nonexistent/path/file.txt"},
	})
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if err.Code != -32603 {
		t.Errorf("error code = %d, want -32603", err.Code)
	}
}
