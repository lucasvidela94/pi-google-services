// Package calendar wraps the Google Calendar v3 API.
package calendar

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/oauth2"
	gcal "google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

// Service wraps the Google Calendar API client.
type Service struct {
	svc *gcal.Service
}

// EventSummary is a simplified calendar event for display.
type EventSummary struct {
	ID          string `json:"id"`
	Summary     string `json:"summary"`
	Description string `json:"description,omitempty"`
	Start       string `json:"start"`
	End         string `json:"end"`
	Location    string `json:"location,omitempty"`
	HTMLLink    string `json:"html_link,omitempty"`
	Attendees   int    `json:"attendees,omitempty"`
	Creator     string `json:"creator,omitempty"`
}

// New creates a Service from an OAuth2 token source.
func New(ctx context.Context, ts oauth2.TokenSource) (*Service, error) {
	svc, err := gcal.NewService(ctx, option.WithTokenSource(ts))
	if err != nil {
		return nil, fmt.Errorf("create calendar service: %w", err)
	}
	return &Service{svc: svc}, nil
}

// ListEvents returns events in a time range.
func (s *Service) ListEvents(ctx context.Context, calendarID string, timeMin, timeMax time.Time, maxResults int64) ([]*EventSummary, error) {
	if calendarID == "" {
		calendarID = "primary"
	}
	if maxResults <= 0 {
		maxResults = 50
	}

	events, err := s.svc.Events.List(calendarID).
		TimeMin(timeMin.Format(time.RFC3339)).
		TimeMax(timeMax.Format(time.RFC3339)).
		MaxResults(maxResults).
		OrderBy("startTime").
		SingleEvents(true).
		Do()
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}

	result := make([]*EventSummary, 0, len(events.Items))
	for _, e := range events.Items {
		se := &EventSummary{
			ID:       e.Id,
			Summary:  e.Summary,
			Start:    fmtDateTime(e.Start),
			End:      fmtDateTime(e.End),
			HTMLLink: e.HtmlLink,
		}
		if e.Description != "" {
			se.Description = truncate(e.Description, 200)
		}
		if e.Location != "" {
			se.Location = e.Location
		}
		if len(e.Attendees) > 0 {
			se.Attendees = len(e.Attendees)
		}
		if e.Creator != nil {
			se.Creator = e.Creator.Email
		}
		result = append(result, se)
	}
	return result, nil
}

// CreateEvent creates a new event. Accepts a raw gcal.Event for full control.
func (s *Service) CreateEvent(ctx context.Context, calendarID string, event *gcal.Event) (*gcal.Event, error) {
	if calendarID == "" {
		calendarID = "primary"
	}
	created, err := s.svc.Events.Insert(calendarID, event).Do()
	if err != nil {
		return nil, fmt.Errorf("create event: %w", err)
	}
	return created, nil
}

// UpdateEvent patches an existing event.
func (s *Service) UpdateEvent(ctx context.Context, calendarID, eventID string, event *gcal.Event) (*gcal.Event, error) {
	if calendarID == "" {
		calendarID = "primary"
	}
	updated, err := s.svc.Events.Update(calendarID, eventID, event).Do()
	if err != nil {
		return nil, fmt.Errorf("update event: %w", err)
	}
	return updated, nil
}

// DeleteEvent removes an event.
func (s *Service) DeleteEvent(ctx context.Context, calendarID, eventID string) error {
	if calendarID == "" {
		calendarID = "primary"
	}
	return s.svc.Events.Delete(calendarID, eventID).Do()
}

// SearchEvents queries events by text.
func (s *Service) SearchEvents(ctx context.Context, query string, maxResults int64) ([]*EventSummary, error) {
	if maxResults <= 0 {
		maxResults = 50
	}
	events, err := s.svc.Events.List("primary").
		Q(query).
		MaxResults(maxResults).
		OrderBy("startTime").
		SingleEvents(true).
		Do()
	if err != nil {
		return nil, fmt.Errorf("search events: %w", err)
	}
	result := make([]*EventSummary, 0, len(events.Items))
	for _, e := range events.Items {
		result = append(result, &EventSummary{
			ID:      e.Id,
			Summary: e.Summary,
			Start:   fmtDateTime(e.Start),
			End:     fmtDateTime(e.End),
		})
	}
	return result, nil
}

// ListCalendars returns all calendars.
func (s *Service) ListCalendars(ctx context.Context) ([]*gcal.CalendarListEntry, error) {
	calList, err := s.svc.CalendarList.List().Do()
	if err != nil {
		return nil, fmt.Errorf("list calendars: %w", err)
	}
	return calList.Items, nil
}

// GetFreeBusy checks availability.
func (s *Service) GetFreeBusy(ctx context.Context, calendarIDs []string, timeMin, timeMax time.Time) (map[string]gcal.FreeBusyCalendar, error) {
	if len(calendarIDs) == 0 {
		calendarIDs = []string{"primary"}
	}
	req := &gcal.FreeBusyRequest{
		TimeMin: timeMin.Format(time.RFC3339),
		TimeMax: timeMax.Format(time.RFC3339),
		Items:   make([]*gcal.FreeBusyRequestItem, len(calendarIDs)),
	}
	for i, id := range calendarIDs {
		req.Items[i] = &gcal.FreeBusyRequestItem{Id: id}
	}
	resp, err := s.svc.Freebusy.Query(req).Do()
	if err != nil {
		return nil, fmt.Errorf("freebusy: %w", err)
	}
	return resp.Calendars, nil
}

func fmtDateTime(dt *gcal.EventDateTime) string {
	if dt == nil {
		return ""
	}
	if dt.DateTime != "" {
		t, err := time.Parse(time.RFC3339, dt.DateTime)
		if err != nil {
			return dt.DateTime
		}
		return t.Format("Mon Jan 2 15:04 MST")
	}
	if dt.Date != "" {
		return dt.Date
	}
	return ""
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
