// Package gmail wraps the Gmail API v1.
package gmail

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"mime"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// Service wraps the Gmail API client.
type Service struct {
	svc    *gmail.UsersService
	lister messagesLister
}

// messagesLister abstracts the messages.list + messages.get(metadata) calls so
// pagination can be unit tested without hitting the network.
type messagesLister interface {
	listMessages(ctx context.Context, query string, maxResults int64, pageToken string, labelIDs []string) (*gmail.ListMessagesResponse, error)
	getMessageMetadata(ctx context.Context, id string) (*gmail.Message, error)
}

// gmailAPIBackend is the real messagesLister implementation against the Gmail
// HTTP API.
type gmailAPIBackend struct {
	svc *gmail.UsersService
}

func (b *gmailAPIBackend) listMessages(ctx context.Context, query string, maxResults int64, pageToken string, labelIDs []string) (*gmail.ListMessagesResponse, error) {
	call := b.svc.Messages.List("me").
		MaxResults(maxResults)
	if query != "" {
		call.Q(query)
	}
	if len(labelIDs) > 0 {
		call.LabelIds(labelIDs...)
	}
	if pageToken != "" {
		call.PageToken(pageToken)
	}
	msgs, err := call.Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}
	return msgs, nil
}

func (b *gmailAPIBackend) getMessageMetadata(ctx context.Context, id string) (*gmail.Message, error) {
	msg, err := b.svc.Messages.Get("me", id).
		Format("metadata").
		MetadataHeaders("Subject", "From", "Date").
		Context(ctx).
		Do()
	if err != nil {
		return nil, fmt.Errorf("get message metadata: %w", err)
	}
	return msg, nil
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
	return &Service{svc: svc.Users, lister: &gmailAPIBackend{svc: svc.Users}}, nil
}

// ListResult is a page of messages plus pagination metadata.
type ListResult struct {
	Messages       []*EmailSummary
	NextPageToken  string
	ResultEstimate int64
}

// ListInbox returns recent messages from the inbox. pageToken resumes
// pagination from a previous ListResult.NextPageToken; the result is
// auto-paginated until maxResults messages are collected or the mailbox is
// exhausted.
func (s *Service) ListInbox(ctx context.Context, maxResults int64, query, pageToken string) (*ListResult, error) {
	return s.searchMessages(ctx, query, maxResults, pageToken, "INBOX")
}

// GetEmail retrieves the full content of a message by ID.
func (s *Service) GetEmail(ctx context.Context, id string) (*EmailDetail, error) {
	msg, err := s.svc.Messages.Get("me", id).Format("full").Context(ctx).Do()
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

// Attachment represents a file to attach to an email.
type Attachment struct {
	Filename string
	MimeType string
	Data     []byte
}

// SendEmail sends a new email with optional attachments.
func (s *Service) SendEmail(ctx context.Context, to, subject, body string, attachments []Attachment) (*gmail.Message, error) {
	msg := createMessage(to, subject, body, attachments)
	sent, err := s.svc.Messages.Send("me", msg).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("send message: %w", err)
	}
	return sent, nil
}

// SearchEmails searches messages by query across the whole mailbox. Unlike
// ListInbox it does NOT restrict to the INBOX label, so queries like
// "in:sent", "to:someone@example.com" or "from:a OR from:b" match messages in
// any label (Sent, Archive, etc.). pageToken resumes pagination from a
// previous ListResult.NextPageToken.
func (s *Service) SearchEmails(ctx context.Context, query string, maxResults int64, pageToken string) (*ListResult, error) {
	return s.searchMessages(ctx, query, maxResults, pageToken)
}

// searchMessages lists messages matching query and/or labels, auto-paginating
// with nextPageToken until maxResults are collected or the mailbox is
// exhausted. labelIDs, when non-empty, are ANDed with the query (Gmail
// labelIds semantics); leave empty to search every label.
func (s *Service) searchMessages(ctx context.Context, query string, maxResults int64, pageToken string, labelIDs ...string) (*ListResult, error) {
	if maxResults <= 0 {
		maxResults = 20
	}
	if maxResults > 500 {
		maxResults = 500
	}
	if s.lister == nil {
		return nil, fmt.Errorf("list messages: lister not configured")
	}

	result := &ListResult{}

	for int64(len(result.Messages)) < maxResults {
		remaining := maxResults - int64(len(result.Messages))

		res, err := s.lister.listMessages(ctx, query, remaining, pageToken, labelIDs)
		if err != nil {
			return nil, fmt.Errorf("list messages: %w", err)
		}
		// ResultSizeEstimate is the estimated total across pages, so keep the
		// largest value rather than overwriting with a later page's estimate.
		if res.ResultSizeEstimate > result.ResultEstimate {
			result.ResultEstimate = res.ResultSizeEstimate
		}
		result.NextPageToken = res.NextPageToken

		for _, m := range res.Messages {
			msg, err := s.lister.getMessageMetadata(ctx, m.Id)
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
			result.Messages = append(result.Messages, summary)
			if int64(len(result.Messages)) >= maxResults {
				break
			}
		}

		if res.NextPageToken == "" {
			break
		}
		pageToken = res.NextPageToken
	}

	return result, nil
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

func createMessage(to, subject, body string, attachments []Attachment) *gmail.Message {
	return buildMessage(to, subject, body, attachments, "")
}

// buildMessage assembles the RFC 5322 message. extraHeaders is inserted verbatim
// after Subject and must already be CRLF-terminated; replies use it to carry
// In-Reply-To and References.
func buildMessage(to, subject, body string, attachments []Attachment, extraHeaders string) *gmail.Message {
	encSubject := mime.BEncoding.Encode("UTF-8", subject)

	if len(attachments) == 0 {
		msg := fmt.Sprintf("From: me\r\nTo: %s\r\nSubject: %s\r\n%sMIME-Version: 1.0\r\nContent-Type: text/plain; charset=\"UTF-8\"\r\n\r\n%s", to, encSubject, extraHeaders, body)
		encoded := base64.URLEncoding.EncodeToString([]byte(msg))
		return &gmail.Message{Raw: encoded}
	}

	boundary := fmt.Sprintf("pi-google-%d", time.Now().UnixNano())

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "From: me\r\nTo: %s\r\nSubject: %s\r\n%sMIME-Version: 1.0\r\nContent-Type: multipart/mixed; boundary=\"%s\"\r\n\r\n", to, encSubject, extraHeaders, boundary)

	fmt.Fprintf(&buf, "--%s\r\nContent-Type: text/plain; charset=\"UTF-8\"\r\n\r\n%s\r\n", boundary, body)

	for _, att := range attachments {
		mt := att.MimeType
		if mt == "" {
			mt = "application/octet-stream"
		}
		fmt.Fprintf(&buf, "--%s\r\n", boundary)
		fmt.Fprintf(&buf, "Content-Type: %s\r\n", mt)
		fmt.Fprintf(&buf, "Content-Transfer-Encoding: base64\r\n")
		fmt.Fprintf(&buf, "Content-Disposition: attachment; filename=\"%s\"\r\n\r\n", escapeFilename(att.Filename))

		encoded := base64.StdEncoding.EncodeToString(att.Data)
		for i := 0; i < len(encoded); i += 76 {
			end := i + 76
			if end > len(encoded) {
				end = len(encoded)
			}
			buf.WriteString(encoded[i:end])
			buf.WriteString("\r\n")
		}
	}

	fmt.Fprintf(&buf, "--%s--\r\n", boundary)

	encoded := base64.URLEncoding.EncodeToString(buf.Bytes())
	return &gmail.Message{Raw: encoded}
}

// escapeFilename removes characters that would break the Content-Disposition header.
func escapeFilename(name string) string {
	name = strings.ReplaceAll(name, "\"", "'")
	name = strings.ReplaceAll(name, "\r", "")
	name = strings.ReplaceAll(name, "\n", "")
	return name
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
