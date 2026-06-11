package services

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/sombi/pi-google-services/internal/calendar"
)

// mockCalendarAPI implements calendar operations without real API calls.
type mockCalendarAPI struct {
	events []*calendar.EventSummary
}

func (m *mockCalendarAPI) ListEvents(ctx context.Context, calendarID string, timeMin, timeMax time.Time, maxResults int64) ([]*calendar.EventSummary, error) {
	return m.events, nil
}
func (m *mockCalendarAPI) CreateEvent(ctx context.Context, calendarID string, event interface{}) (interface{}, error) {
	return map[string]interface{}{
		"id":       "mock-event-1",
		"summary":  "test",
		"htmlLink": "https://calendar.google.com/event?eid=mock",
	}, nil
}
func (m *mockCalendarAPI) DeleteEvent(ctx context.Context, calendarID, eventID string) error {
	return nil
}
func (m *mockCalendarAPI) SearchEvents(ctx context.Context, query string, maxResults int64) ([]*calendar.EventSummary, error) {
	return m.events, nil
}
func (m *mockCalendarAPI) ListCalendars(ctx context.Context) ([]interface{}, error) {
	return []interface{}{}, nil
}
func (m *mockCalendarAPI) GetFreeBusy(ctx context.Context, calendarIDs []string, timeMin, timeMax time.Time) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

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
	_, err := cs.Handle(nil, "nonexistent", nil)
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
	_, err := gs.Handle(nil, "nonexistent", nil)
	if err == nil || err.Code != -32601 {
		t.Errorf("expected -32601 error, got %+v", err)
	}
}

// Test that the service tools count is correct.
func TestServiceToolsCount(t *testing.T) {
	if len((&CalendarService{}).Tools()) != 7 {
		t.Errorf("Calendar: expected 7 tools")
	}
	if len((&GmailService{}).Tools()) != 5 {
		t.Errorf("Gmail: expected 5 tools")
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
