package gmail

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestCreateMessage_NoAttachments(t *testing.T) {
	msg := createMessage("user@example.com", "Hello", "Body text", nil)
	if msg == nil || msg.Raw == "" {
		t.Fatal("expected non-empty message")
	}

	decoded, err := base64.URLEncoding.DecodeString(msg.Raw)
	if err != nil {
		t.Fatalf("decode raw: %v", err)
	}
	str := string(decoded)

	if !strings.Contains(str, "Content-Type: text/plain") {
		t.Errorf("expected text/plain content type, got:\n%s", str)
	}
	if !strings.Contains(str, "To: user@example.com") {
		t.Errorf("missing To header")
	}
	if strings.Contains(str, "multipart/mixed") {
		t.Errorf("should not be multipart when no attachments")
	}
}

func TestCreateMessage_WithAttachments(t *testing.T) {
	attachments := []Attachment{
		{Filename: "doc.pdf", MimeType: "application/pdf", Data: []byte("fake-pdf-content")},
		{Filename: "note.txt", MimeType: "text/plain", Data: []byte("hello")},
	}

	msg := createMessage("user@example.com", "With file", "See attached", attachments)
	if msg == nil || msg.Raw == "" {
		t.Fatal("expected non-empty message")
	}

	decoded, err := base64.URLEncoding.DecodeString(msg.Raw)
	if err != nil {
		t.Fatalf("decode raw: %v", err)
	}
	str := string(decoded)

	if !strings.Contains(str, "multipart/mixed") {
		t.Errorf("expected multipart/mixed, got:\n%s", str)
	}

	if !strings.Contains(str, `filename="doc.pdf"`) {
		t.Errorf("missing first attachment filename")
	}
	if !strings.Contains(str, `filename="note.txt"`) {
		t.Errorf("missing second attachment filename")
	}

	// Verify base64-encoded attachment data is present
	encPDF := base64.StdEncoding.EncodeToString([]byte("fake-pdf-content"))
	if !strings.Contains(str, encPDF) {
		t.Errorf("missing base64-encoded PDF data")
	}

	// Verify the body text is in a text/plain part
	if !strings.Contains(str, "See attached") {
		t.Errorf("missing body text in multipart message")
	}

	// Verify closing boundary
	if !strings.Contains(str, "--pi-google-") {
		t.Errorf("missing boundary markers")
	}
}

func TestCreateMessage_AttachmentDefaultMimeType(t *testing.T) {
	attachments := []Attachment{
		{Filename: "unknown.bin", Data: []byte("data")},
	}

	msg := createMessage("a@b.com", "S", "B", attachments)
	decoded, _ := base64.URLEncoding.DecodeString(msg.Raw)
	str := string(decoded)

	if !strings.Contains(str, "application/octet-stream") {
		t.Errorf("expected default mime type for attachment without MimeType")
	}
}

func TestEscapeFilename(t *testing.T) {
	tests := []struct {
		input  string
		expect string
	}{
		{"normal.pdf", "normal.pdf"},
		{`file"name.pdf`, "file'name.pdf"},
		{"line\r\nbreak", "linebreak"},
	}
	for _, tt := range tests {
		got := escapeFilename(tt.input)
		if got != tt.expect {
			t.Errorf("escapeFilename(%q) = %q, want %q", tt.input, got, tt.expect)
		}
	}
}
