// Package gmail wraps the Gmail API v1.
package gmail

import (
	"context"
	"encoding/base64"
	"fmt"
	"mime"
	"time"

	"golang.org/x/oauth2"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// Service wraps the Gmail API client.
type Service struct {
	svc *gmail.UsersService
}

// EmailSummary is a lightweight email representation.
type EmailSummary struct {
	ID       string   `json:"id"`
	ThreadID string   `json:"thread_id"`
	Subject  string   `json:"subject"`
	From     string   `json:"from"`
	Date     string   `json:"date"`
	Snippet  string   `json:"snippet"`
	LabelIDs []string `json:"label_ids,omitempty"`
}

// EmailDetail is a full email with body content.
type EmailDetail struct {
	EmailSummary
	To   string `json:"to"`
	Body string `json:"body"`
	HTML bool   `json:"html"`
}

// New creates a Gmail Service from an OAuth2 token source.
func New(ctx context.Context, ts oauth2.TokenSource) (*Service, error) {
	svc, err := gmail.NewService(ctx, option.WithTokenSource(ts))
	if err != nil {
		return nil, fmt.Errorf("create gmail service: %w", err)
	}
	return &Service{svc: svc.Users}, nil
}

// ListInbox returns recent messages from the inbox.
func (s *Service) ListInbox(ctx context.Context, maxResults int64, query string) ([]*EmailSummary, error) {
	if maxResults <= 0 {
		maxResults = 20
	}

	call := s.svc.Messages.List("me").
		MaxResults(maxResults).
		LabelIds("INBOX")
	if query != "" {
		call.Q(query)
	}

	res, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}

	summaries := make([]*EmailSummary, 0, len(res.Messages))
	for _, m := range res.Messages {
		msg, err := s.svc.Messages.Get("me", m.Id).
			Format("metadata").
			MetadataHeaders("Subject", "From", "Date").
			Do()
		if err != nil {
			continue // skip unreadable messages
		}

		summary := &EmailSummary{
			ID:       msg.Id,
			ThreadID: msg.ThreadId,
			Snippet:  msg.Snippet,
			LabelIDs: msg.LabelIds,
		}
		for _, h := range msg.Payload.Headers {
			switch h.Name {
			case "Subject":
				summary.Subject = h.Value
			case "From":
				summary.From = h.Value
			case "Date":
				summary.Date = h.Value
			}
		}
		summaries = append(summaries, summary)
	}
	return summaries, nil
}

// GetEmail retrieves the full content of a message by ID.
func (s *Service) GetEmail(ctx context.Context, id string) (*EmailDetail, error) {
	msg, err := s.svc.Messages.Get("me", id).Format("full").Do()
	if err != nil {
		return nil, fmt.Errorf("get message: %w", err)
	}

	detail := &EmailDetail{
		EmailSummary: EmailSummary{
			ID:       msg.Id,
			ThreadID: msg.ThreadId,
			Snippet:  msg.Snippet,
			LabelIDs: msg.LabelIds,
		},
	}

	for _, h := range msg.Payload.Headers {
		switch h.Name {
		case "Subject":
			detail.Subject = h.Value
		case "From":
			detail.From = h.Value
		case "Date":
			detail.Date = h.Value
		case "To":
			detail.To = h.Value
		}
	}

	// Extract body from the payload (prefer plain text)
	body, html := extractBody(msg.Payload, 0)
	detail.Body = body
	detail.HTML = html

	return detail, nil
}

// SendEmail sends a new email.
func (s *Service) SendEmail(ctx context.Context, to, subject, body string) (*gmail.Message, error) {
	msg := createMessage(to, subject, body)
	sent, err := s.svc.Messages.Send("me", msg).Do()
	if err != nil {
		return nil, fmt.Errorf("send message: %w", err)
	}
	return sent, nil
}

// ReplyToEmail replies to an existing thread.
func (s *Service) ReplyToEmail(ctx context.Context, threadID, to, subject, body string) (*gmail.Message, error) {
	msg := createMessage(to, subject, body)
	msg.ThreadId = threadID
	sent, err := s.svc.Messages.Send("me", msg).Do()
	if err != nil {
		return nil, fmt.Errorf("reply: %w", err)
	}
	return sent, nil
}

// SearchEmails searches messages by query.
func (s *Service) SearchEmails(ctx context.Context, query string, maxResults int64) ([]*EmailSummary, error) {
	return s.ListInbox(ctx, maxResults, query)
}

// --- helpers ---

func extractBody(part *gmail.MessagePart, depth int) (body string, html bool) {
	if part == nil || depth > 5 {
		return "", false
	}

	// Check this part's body
	if part.MimeType == "text/plain" && part.Body != nil && part.Body.Data != "" {
		data, _ := base64.URLEncoding.DecodeString(part.Body.Data)
		return string(data), false
	}
	if part.MimeType == "text/html" && part.Body != nil && part.Body.Data != "" {
		data, _ := base64.URLEncoding.DecodeString(part.Body.Data)
		return string(data), true
	}

	// Recurse into child parts
	for _, child := range part.Parts {
		b, h := extractBody(child, depth+1)
		if b != "" {
			return b, h
		}
	}
	return "", false
}

func createMessage(to, subject, body string) *gmail.Message {
	// Use Go's standard mime.BEncoding for proper MIME encoded-word format.
	// This handles emojis and non-ASCII characters correctly.
	encSubject := mime.BEncoding.Encode("UTF-8", subject)
	msg := fmt.Sprintf("From: me\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=\"UTF-8\"\r\n\r\n%s", to, encSubject, body)
	encoded := base64.URLEncoding.EncodeToString([]byte(msg))
	return &gmail.Message{Raw: encoded}
}

// HumanDate parses and reformats RFC1123 dates for display.
func HumanDate(raw string) string {
	t, err := time.Parse(time.RFC1123Z, raw)
	if err != nil {
		t, err = time.Parse("Mon, 2 Jan 2006 15:04:05 -0700", raw)
		if err != nil {
			return raw
		}
	}
	if t.After(time.Now().Add(-24 * time.Hour)) {
		return t.Format("15:04")
	}
	return t.Format("Jan 2")
}
