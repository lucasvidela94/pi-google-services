package services

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

// We test the service layer by using its tool definitions and handler routing.
func TestCalendarServiceName(t *testing.T) {
	cs := &CalendarService{}
	if cs.Name() != "calendar" {
		t.Errorf("Name() = %q, want %q", cs.Name(), "calendar")
	}
}

func TestCalendarServiceScopes(t *testing.T) {
	cs := &CalendarService{}
	scopes := cs.Scopes()
	if len(scopes) < 2 {
		t.Errorf("expected at least 2 scopes, got %d", len(scopes))
	}
}

func TestCalendarServiceTools(t *testing.T) {
	cs := &CalendarService{}
	tools := cs.Tools()
	if len(tools) != 7 {
		t.Errorf("expected 7 tools, got %d", len(tools))
	}

	// Verify all expected tool names are present
	names := make(map[string]bool)
	for _, tool := range tools {
		names[tool.Name] = true
	}
	for _, name := range []string{
		"list-events", "create-event", "update-event",
		"delete-event", "search-events", "list-calendars", "get-freebusy",
	} {
		if !names[name] {
			t.Errorf("missing tool: %s", name)
		}
	}
}

func TestCalendarServiceHandle_UnknownTool(t *testing.T) {
	cs := &CalendarService{}
	_, err := cs.Handle(context.TODO(), "nonexistent", nil)
	if err == nil || err.Code != -32601 {
		t.Errorf("expected -32601 error, got %+v", err)
	}
}

func TestGmailServiceName(t *testing.T) {
	gs := &GmailService{}
	if gs.Name() != "gmail" {
		t.Errorf("Name() = %q, want %q", gs.Name(), "gmail")
	}
}

func TestGmailServiceScopes(t *testing.T) {
	gs := &GmailService{}
	scopes := gs.Scopes()
	if len(scopes) < 2 {
		t.Errorf("expected at least 2 scopes, got %d", len(scopes))
	}
}

func TestGmailServiceTools(t *testing.T) {
	gs := &GmailService{}
	tools := gs.Tools()
	if len(tools) != 5 {
		t.Errorf("expected 5 tools, got %d", len(tools))
	}

	names := make(map[string]bool)
	for _, tool := range tools {
		names[tool.Name] = true
	}
	for _, name := range []string{
		"list-inbox", "get-email", "search-emails",
		"send-email", "reply-to-email",
	} {
		if !names[name] {
			t.Errorf("missing tool: %s", name)
		}
	}
}

func TestGmailServiceHandle_UnknownTool(t *testing.T) {
	gs := &GmailService{}
	_, err := gs.Handle(context.TODO(), "nonexistent", nil)
	if err == nil || err.Code != -32601 {
		t.Errorf("expected -32601 error, got %+v", err)
	}
}

// TestListEventsHandlers validates parameter parsing.
func TestListEventsArgs(t *testing.T) {
	// Test that the time parsing doesn't panic on empty
	var args struct {
		CalendarID string `json:"calendarId"`
		TimeMin    string `json:"timeMin"`
		TimeMax    string `json:"timeMax"`
		MaxResults int64  `json:"maxResults"`
	}

	// Empty JSON should produce zero values
	data := []byte(`{}`)
	if err := json.Unmarshal(data, &args); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Valid time should parse
	args.TimeMin = "2026-06-16T10:00:00-03:00"
	tm, err := time.Parse(time.RFC3339, args.TimeMin)
	if err != nil {
		t.Fatalf("time parse: %v", err)
	}
	if tm.Year() != 2026 || tm.Month() != 6 || tm.Day() != 16 {
		t.Errorf("unexpected date: %v", tm)
	}
}

// --- Drive service tests ---

func TestDriveServiceName(t *testing.T) {
	ds := &DriveService{}
	if ds.Name() != "drive" {
		t.Errorf("Name() = %q, want %q", ds.Name(), "drive")
	}
}

func TestDriveServiceScopes(t *testing.T) {
	ds := &DriveService{}
	scopes := ds.Scopes()
	if len(scopes) < 1 {
		t.Errorf("expected at least 1 scope, got %d", len(scopes))
	}
}

func TestDriveServiceTools(t *testing.T) {
	ds := &DriveService{}
	tools := ds.Tools()
	if len(tools) != 6 {
		t.Errorf("expected 6 tools, got %d", len(tools))
	}
	names := make(map[string]bool)
	for _, tool := range tools {
		names[tool.Name] = true
	}
	for _, name := range []string{
		"list-files", "search-drive", "upload-file",
		"download-file", "create-folder", "delete-file",
	} {
		if !names[name] {
			t.Errorf("missing tool: %s", name)
		}
	}
}

func TestDriveServiceHandle_UnknownTool(t *testing.T) {
	ds := &DriveService{}
	_, err := ds.Handle(context.TODO(), "nonexistent", nil)
	if err == nil || err.Code != -32601 {
		t.Errorf("expected -32601 error, got %+v", err)
	}
}

func TestServiceToolsCount(t *testing.T) {
	calLen := len((&CalendarService{}).Tools())
	gmailLen := len((&GmailService{}).Tools())
	tasksLen := len((&TasksService{}).Tools())
	driveLen := len((&DriveService{}).Tools())
	contactsLen := len((&ContactsService{}).Tools())
	formsLen := len((&FormsService{}).Tools())
	docsLen := len((&DocsService{}).Tools())
	sheetsLen := len((&SheetsService{}).Tools())

	if calLen != 7 {
		t.Errorf("Calendar: expected 7, got %d", calLen)
	}
	if gmailLen != 5 {
		t.Errorf("Gmail: expected 5, got %d", gmailLen)
	}
	if tasksLen != 5 {
		t.Errorf("Tasks: expected 5, got %d", tasksLen)
	}
	if driveLen != 6 {
		t.Errorf("Drive: expected 6, got %d", driveLen)
	}
	if contactsLen != 3 {
		t.Errorf("Contacts: expected 3, got %d", contactsLen)
	}
	if formsLen != 5 {
		t.Errorf("Forms: expected 5, got %d", formsLen)
	}
	if docsLen != 3 {
		t.Errorf("Docs: expected 3, got %d", docsLen)
	}
	if sheetsLen != 2 {
		t.Errorf("Sheets: expected 2, got %d", sheetsLen)
	}
	total := calLen + gmailLen + tasksLen + driveLen + contactsLen + formsLen + docsLen + sheetsLen
	if total != 36 {
		t.Errorf("Total tools: expected 36 (7+5+5+6+3+5+3+2), got %d", total)
	}
}

func TestContactsServiceName(t *testing.T) {
	ds := &ContactsService{}
	if ds.Name() != "contacts" {
		t.Errorf("Name() = %q, want %q", ds.Name(), "contacts")
	}
}

func TestContactsServiceScopes(t *testing.T) {
	ds := &ContactsService{}
	scopes := ds.Scopes()
	if len(scopes) < 1 {
		t.Errorf("expected at least 1 scope, got %d", len(scopes))
	}
}

func TestContactsServiceTools(t *testing.T) {
	ds := &ContactsService{}
	tools := ds.Tools()
	if len(tools) != 3 {
		t.Errorf("expected 3 tools, got %d", len(tools))
	}
	names := make(map[string]bool)
	for _, tool := range tools {
		names[tool.Name] = true
	}
	for _, name := range []string{"search-contacts", "get-contact", "create-contact"} {
		if !names[name] {
			t.Errorf("missing tool: %s", name)
		}
	}
}
