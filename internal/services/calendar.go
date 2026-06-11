package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	gcal "google.golang.org/api/calendar/v3"

	"github.com/sombi/pi-google-services/internal/calendar"
	"github.com/sombi/pi-google-services/internal/mcp"
)

// CalendarService implements the Service interface for Google Calendar.
type CalendarService struct {
	api *calendar.Service
}

// NewCalendar creates a CalendarService from the calendar API wrapper.
func NewCalendar(api *calendar.Service) *CalendarService {
	return &CalendarService{api: api}
}

func (s *CalendarService) Name() string { return "calendar" }

func (s *CalendarService) Scopes() []string {
	return []string{
		"https://www.googleapis.com/auth/calendar",
		"https://www.googleapis.com/auth/calendar.events",
	}
}

func (s *CalendarService) Tools() []mcp.ToolDefinition {
	return []mcp.ToolDefinition{
		{
			Name:        "list-events",
			Description: "List calendar events in a date range",
			InputSchema: mcp.InputSchema{
				Type: "object",
				Properties: map[string]mcp.PropertySchema{
					"calendarId": {Type: "string", Description: "Calendar ID (default: primary)"},
					"timeMin":    {Type: "string", Description: "Start time (ISO 8601, e.g. 2026-06-10T00:00:00Z)"},
					"timeMax":    {Type: "string", Description: "End time (ISO 8601)"},
					"maxResults": {Type: "number", Description: "Maximum events to return (default: 50)", Default: 50},
				},
			},
		},
		{
			Name:        "create-event",
			Description: "Create a new calendar event with optional attendees and Google Meet",
			InputSchema: mcp.InputSchema{
				Type: "object",
				Properties: map[string]mcp.PropertySchema{
					"calendarId":  {Type: "string", Description: "Calendar ID (default: primary)"},
					"summary":     {Type: "string", Description: "Event title"},
					"description": {Type: "string", Description: "Event description"},
					"startTime":   {Type: "string", Description: "Start time (ISO 8601)"},
					"endTime":     {Type: "string", Description: "End time (ISO 8601)"},
					"location":    {Type: "string", Description: "Event location"},
					"attendees":   {Type: "string", Description: "Comma-separated email addresses"},
					"withMeet":    {Type: "boolean", Description: "Add Google Meet link (default: false)"},
				},
				Required: []string{"summary", "startTime", "endTime"},
			},
		},
		{
			Name:        "update-event",
			Description: "Update an existing calendar event",
			InputSchema: mcp.InputSchema{
				Type: "object",
				Properties: map[string]mcp.PropertySchema{
					"calendarId": {Type: "string", Description: "Calendar ID (default: primary)"},
					"eventId":    {Type: "string", Description: "Event ID to update"},
					"summary":    {Type: "string", Description: "New title"},
					"startTime":  {Type: "string", Description: "New start time (ISO 8601)"},
					"endTime":    {Type: "string", Description: "New end time (ISO 8601)"},
				},
				Required: []string{"eventId"},
			},
		},
		{
			Name:        "delete-event",
			Description: "Delete a calendar event",
			InputSchema: mcp.InputSchema{
				Type: "object",
				Properties: map[string]mcp.PropertySchema{
					"calendarId": {Type: "string", Description: "Calendar ID (default: primary)"},
					"eventId":    {Type: "string", Description: "Event ID to delete"},
				},
				Required: []string{"eventId"},
			},
		},
		{
			Name:        "search-events",
			Description: "Search calendar events by text query",
			InputSchema: mcp.InputSchema{
				Type: "object",
				Properties: map[string]mcp.PropertySchema{
					"query":      {Type: "string", Description: "Search query text"},
					"maxResults": {Type: "number", Description: "Max results (default: 50)", Default: 50},
				},
				Required: []string{"query"},
			},
		},
		{
			Name:        "list-calendars",
			Description: "List all available calendars",
			InputSchema: mcp.InputSchema{Type: "object", Properties: map[string]mcp.PropertySchema{}},
		},
		{
			Name:        "get-freebusy",
			Description: "Check availability across calendars",
			InputSchema: mcp.InputSchema{
				Type: "object",
				Properties: map[string]mcp.PropertySchema{
					"calendarIds": {Type: "string", Description: "Comma-separated calendar IDs (default: primary)"},
					"timeMin":     {Type: "string", Description: "Start time (ISO 8601)"},
					"timeMax":     {Type: "string", Description: "End time (ISO 8601)"},
				},
			},
		},
	}
}

func (s *CalendarService) Handle(ctx context.Context, toolName string, params json.RawMessage) (interface{}, *mcp.RPCError) {
	switch toolName {
	case "list-events":
		return s.handleListEvents(ctx, params)
	case "create-event":
		return s.handleCreateEvent(ctx, params)
	case "update-event":
		return s.handleUpdateEvent(ctx, params)
	case "delete-event":
		return s.handleDeleteEvent(ctx, params)
	case "search-events":
		return s.handleSearchEvents(ctx, params)
	case "list-calendars":
		return s.handleListCalendars(ctx, params)
	case "get-freebusy":
		return s.handleFreeBusy(ctx, params)
	default:
		return nil, &mcp.RPCError{Code: -32601, Message: fmt.Sprintf("Calendar tool not found: %s", toolName)}
	}
}

// --- handlers ---

func (s *CalendarService) handleListEvents(ctx context.Context, params json.RawMessage) (interface{}, *mcp.RPCError) {
	var args struct {
		CalendarID string `json:"calendarId"`
		TimeMin    string `json:"timeMin"`
		TimeMax    string `json:"timeMax"`
		MaxResults int64  `json:"maxResults"`
	}
	if err := json.Unmarshal(params, &args); err != nil {
		return nil, &mcp.RPCError{Code: -32602, Message: "Invalid arguments", Data: err.Error()}
	}

	now := time.Now()
	timeMin := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	timeMax := timeMin.Add(24 * time.Hour)
	if args.TimeMin != "" {
		if t, err := time.Parse(time.RFC3339, args.TimeMin); err == nil {
			timeMin = t
		}
	}
	if args.TimeMax != "" {
		if t, err := time.Parse(time.RFC3339, args.TimeMax); err == nil {
			timeMax = t
		}
	}

	events, err := s.api.ListEvents(ctx, args.CalendarID, timeMin, timeMax, args.MaxResults)
	if err != nil {
		return nil, &mcp.RPCError{Code: -32603, Message: "Failed to list events", Data: err.Error()}
	}

	return contentResponse(formatEvents(events)), nil
}

func (s *CalendarService) handleCreateEvent(ctx context.Context, params json.RawMessage) (interface{}, *mcp.RPCError) {
	var args struct {
		CalendarID  string `json:"calendarId"`
		Summary     string `json:"summary"`
		Description string `json:"description"`
		StartTime   string `json:"startTime"`
		EndTime     string `json:"endTime"`
		Location    string `json:"location"`
		Attendees   string `json:"attendees"`
		WithMeet    bool   `json:"withMeet"`
	}
	if err := json.Unmarshal(params, &args); err != nil {
		return nil, &mcp.RPCError{Code: -32602, Message: "Invalid arguments", Data: err.Error()}
	}
	if args.Summary == "" || args.StartTime == "" || args.EndTime == "" {
		return nil, &mcp.RPCError{Code: -32602, Message: "summary, startTime, endTime required"}
	}

	event := &gcal.Event{
		Summary:     args.Summary,
		Description: args.Description,
		Location:    args.Location,
		Start:       &gcal.EventDateTime{DateTime: args.StartTime},
		End:         &gcal.EventDateTime{DateTime: args.EndTime},
	}
	if args.Attendees != "" {
		for _, email := range strings.Split(args.Attendees, ",") {
			email = strings.TrimSpace(email)
			if email != "" {
				event.Attendees = append(event.Attendees, &gcal.EventAttendee{Email: email})
			}
		}
	}

	created, err := s.api.CreateEvent(ctx, args.CalendarID, event, args.WithMeet)
	if err != nil {
		return nil, &mcp.RPCError{Code: -32603, Message: "Failed to create event", Data: err.Error()}
	}

	meetInfo := ""
	if args.WithMeet && created.ConferenceData != nil && created.ConferenceData.EntryPoints != nil {
		for _, ep := range created.ConferenceData.EntryPoints {
			if ep.EntryPointType == "video" {
				meetInfo = fmt.Sprintf("\n🎥 Meet: %s", ep.Uri)
				break
			}
		}
	}
	attendeesInfo := ""
	if args.Attendees != "" {
		attendeesInfo = fmt.Sprintf("\nAttendees: %s", args.Attendees)
	}
	return contentResponse(fmt.Sprintf("✅ Event created: %s\nID: %s\nLink: %s%s%s", created.Summary, created.Id, created.HtmlLink, meetInfo, attendeesInfo)), nil
}

func (s *CalendarService) handleUpdateEvent(ctx context.Context, params json.RawMessage) (interface{}, *mcp.RPCError) {
	var args struct {
		CalendarID string `json:"calendarId"`
		EventID    string `json:"eventId"`
		Summary    string `json:"summary"`
		StartTime  string `json:"startTime"`
		EndTime    string `json:"endTime"`
	}
	if err := json.Unmarshal(params, &args); err != nil {
		return nil, &mcp.RPCError{Code: -32602, Message: "Invalid arguments", Data: err.Error()}
	}
	if args.EventID == "" {
		return nil, &mcp.RPCError{Code: -32602, Message: "eventId required"}
	}

	event := &gcal.Event{Id: args.EventID}
	if args.Summary != "" {
		event.Summary = args.Summary
	}
	if args.StartTime != "" {
		event.Start = &gcal.EventDateTime{DateTime: args.StartTime}
	}
	if args.EndTime != "" {
		event.End = &gcal.EventDateTime{DateTime: args.EndTime}
	}

	updated, err := s.api.UpdateEvent(ctx, args.CalendarID, args.EventID, event)
	if err != nil {
		return nil, &mcp.RPCError{Code: -32603, Message: "Failed to update event", Data: err.Error()}
	}
	return contentResponse(fmt.Sprintf("✅ Event updated: %s\nLink: %s", updated.Summary, updated.HtmlLink)), nil
}

func (s *CalendarService) handleDeleteEvent(ctx context.Context, params json.RawMessage) (interface{}, *mcp.RPCError) {
	var args struct {
		CalendarID string `json:"calendarId"`
		EventID    string `json:"eventId"`
	}
	if err := json.Unmarshal(params, &args); err != nil {
		return nil, &mcp.RPCError{Code: -32602, Message: "Invalid arguments", Data: err.Error()}
	}
	if args.EventID == "" {
		return nil, &mcp.RPCError{Code: -32602, Message: "eventId required"}
	}
	if err := s.api.DeleteEvent(ctx, args.CalendarID, args.EventID); err != nil {
		return nil, &mcp.RPCError{Code: -32603, Message: "Failed to delete event", Data: err.Error()}
	}
	return contentResponse(fmt.Sprintf("✅ Event deleted (ID: %s)", args.EventID)), nil
}

func (s *CalendarService) handleSearchEvents(ctx context.Context, params json.RawMessage) (interface{}, *mcp.RPCError) {
	var args struct {
		Query      string `json:"query"`
		MaxResults int64  `json:"maxResults"`
	}
	if err := json.Unmarshal(params, &args); err != nil {
		return nil, &mcp.RPCError{Code: -32602, Message: "Invalid arguments", Data: err.Error()}
	}
	if args.Query == "" {
		return nil, &mcp.RPCError{Code: -32602, Message: "query required"}
	}
	events, err := s.api.SearchEvents(ctx, args.Query, args.MaxResults)
	if err != nil {
		return nil, &mcp.RPCError{Code: -32603, Message: "Failed to search", Data: err.Error()}
	}
	return contentResponse(formatEvents(events)), nil
}

func (s *CalendarService) handleListCalendars(ctx context.Context, _ json.RawMessage) (interface{}, *mcp.RPCError) {
	calendars, err := s.api.ListCalendars(ctx)
	if err != nil {
		return nil, &mcp.RPCError{Code: -32603, Message: "Failed to list calendars", Data: err.Error()}
	}
	var b strings.Builder
	for _, cal := range calendars {
		b.WriteString(fmt.Sprintf("📅 %s", cal.Summary))
		if cal.Primary {
			b.WriteString(" (primary)")
		}
		b.WriteString(fmt.Sprintf("\n  ID: %s\n", cal.Id))
		if cal.Description != "" {
			b.WriteString(fmt.Sprintf("  %s\n", cal.Description))
		}
	}
	return contentResponse(b.String()), nil
}

func (s *CalendarService) handleFreeBusy(ctx context.Context, params json.RawMessage) (interface{}, *mcp.RPCError) {
	var args struct {
		CalendarIDs string `json:"calendarIds"`
		TimeMin     string `json:"timeMin"`
		TimeMax     string `json:"timeMax"`
	}
	if err := json.Unmarshal(params, &args); err != nil {
		return nil, &mcp.RPCError{Code: -32602, Message: "Invalid arguments", Data: err.Error()}
	}

	now := time.Now()
	timeMin := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	timeMax := timeMin.Add(24 * time.Hour)
	if args.TimeMin != "" {
		if t, err := time.Parse(time.RFC3339, args.TimeMin); err == nil {
			timeMin = t
		}
	}
	if args.TimeMax != "" {
		if t, err := time.Parse(time.RFC3339, args.TimeMax); err == nil {
			timeMax = t
		}
	}

	var ids []string
	if args.CalendarIDs != "" {
		for _, id := range strings.Split(args.CalendarIDs, ",") {
			ids = append(ids, strings.TrimSpace(id))
		}
	}

	calendars, err := s.api.GetFreeBusy(ctx, ids, timeMin, timeMax)
	if err != nil {
		return nil, &mcp.RPCError{Code: -32603, Message: "Failed to get free/busy", Data: err.Error()}
	}

	var b strings.Builder
	for id, cal := range calendars {
		b.WriteString(fmt.Sprintf("📅 %s\n", id))
		if len(cal.Busy) == 0 {
			b.WriteString("  ✅ Free all day\n")
		} else {
			for _, busy := range cal.Busy {
				b.WriteString(fmt.Sprintf("  ❌ Busy: %s → %s\n", busy.Start, busy.End))
			}
		}
	}
	return contentResponse(b.String()), nil
}

// helpers

func contentResponse(text string) map[string]interface{} {
	return map[string]interface{}{
		"content": []map[string]interface{}{
			{"type": "text", "text": text},
		},
	}
}

func formatEvents(events []*calendar.EventSummary) string {
	if len(events) == 0 {
		return "No events found."
	}
	var b strings.Builder
	for i, e := range events {
		b.WriteString(fmt.Sprintf("%d. %s\n   📅 %s → %s", i+1, e.Summary, e.Start, e.End))
		if e.Description != "" {
			desc := strings.ReplaceAll(e.Description, "\n", "\n   ")
			b.WriteString(fmt.Sprintf("\n   📝 %s", desc))
		}
		if e.Location != "" {
			b.WriteString(fmt.Sprintf("\n   📍 %s", e.Location))
		}
		b.WriteString("\n")
	}
	return b.String()
}
