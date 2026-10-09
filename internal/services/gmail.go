package services

import (
	"context"
	"encoding/json"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/sombi/pi-google-services/internal/drive"
	"github.com/sombi/pi-google-services/internal/gmail"
	"github.com/sombi/pi-google-services/internal/mcp"
)

// GmailService implements the Service interface for Gmail.
type GmailService struct {
	api      *gmail.Service
	driveAPI *drive.Service
}

// NewGmail creates a GmailService from the Gmail API wrapper.
// driveAPI is optional — when provided, email attachments can reference
// Google Drive file IDs. Pass nil to support local-file attachments only.
func NewGmail(api *gmail.Service, driveAPI *drive.Service) *GmailService {
	return &GmailService{api: api, driveAPI: driveAPI}
}

func (s *GmailService) Name() string { return "gmail" }

func (s *GmailService) Scopes() []string {
	return []string{
		"https://www.googleapis.com/auth/gmail.readonly",
		"https://www.googleapis.com/auth/gmail.send",
		"https://www.googleapis.com/auth/gmail.modify",
	}
}

func (s *GmailService) Tools() []mcp.ToolDefinition {
	return []mcp.ToolDefinition{
		{
			Name:        "list-inbox",
			Description: "List recent emails from inbox",
			InputSchema: mcp.InputSchema{
				Type: "object",
				Properties: map[string]mcp.PropertySchema{
					"maxResults": {Type: "number", Description: "Max emails (default: 20)", Default: 20},
					"query":      {Type: "string", Description: "Optional search filter"},
					"pageToken":  {Type: "string", Description: "Pagination token to get the next page (from a previous result)"},
				},
			},
		},
		{
			Name:        "get-email",
			Description: "Read a full email by ID",
			InputSchema: mcp.InputSchema{
				Type: "object",
				Properties: map[string]mcp.PropertySchema{
					"id": {Type: "string", Description: "Email message ID"},
				},
				Required: []string{"id"},
			},
		},
		{
			Name:        "search-emails",
			Description: "Search emails by query across the whole mailbox (sent, inbox, etc.)",
			InputSchema: mcp.InputSchema{
				Type: "object",
				Properties: map[string]mcp.PropertySchema{
					"query":      {Type: "string", Description: "Search query (Gmail syntax)"},
					"maxResults": {Type: "number", Description: "Max results (default: 20)", Default: 20},
					"pageToken":  {Type: "string", Description: "Pagination token to get the next page (from a previous result)"},
				},
				Required: []string{"query"},
			},
		},
		{
			Name:        "send-email",
			Description: "Send an email with optional file attachments",
			InputSchema: mcp.InputSchema{
				Type: "object",
				Properties: map[string]mcp.PropertySchema{
					"to":      {Type: "string", Description: "Recipient email"},
					"subject": {Type: "string", Description: "Email subject"},
					"body":    {Type: "string", Description: "Email body text"},
					"attachments": {
						Type:        "array",
						Description: "Files to attach. Each item can have 'localPath' (local file) or 'driveFileId' (Google Drive file ID).",
						Items: &mcp.PropertySchema{
							Type: "object",
							Properties: map[string]mcp.PropertySchema{
								"localPath":   {Type: "string", Description: "Local file path to attach"},
								"driveFileId": {Type: "string", Description: "Google Drive file ID to attach"},
							},
						},
					},
				},
				Required: []string{"to", "subject", "body"},
			},
		},
		{
			Name:        "reply-to-email",
			Description: "Reply in-thread to an email, with optional file attachments. Recipient and subject come from the thread so the reply actually threads; normally pass only threadId and body.",
			InputSchema: mcp.InputSchema{
				Type: "object",
				Properties: map[string]mcp.PropertySchema{
					"threadId": {Type: "string", Description: "Thread ID to reply to (thread_id from list-inbox, get-email, or search-emails)"},
					"to":       {Type: "string", Description: "Optional. Recipient email. Defaults to the thread's Reply-To, else its From."},
					"subject":  {Type: "string", Description: "Optional and normally omitted. Ignored when the thread has a subject: a reply whose subject differs reads as a separate email in most mail clients."},
					"body":     {Type: "string", Description: "Reply body text"},
					"attachments": {
						Type:        "array",
						Description: "Files to attach. Each item can have 'localPath' (local file) or 'driveFileId' (Google Drive file ID).",
						Items: &mcp.PropertySchema{
							Type: "object",
							Properties: map[string]mcp.PropertySchema{
								"localPath":   {Type: "string", Description: "Local file path to attach"},
								"driveFileId": {Type: "string", Description: "Google Drive file ID to attach"},
							},
						},
					},
				},
				Required: []string{"threadId", "body"},
			},
		},
	}
}

func (s *GmailService) Handle(ctx context.Context, toolName string, params json.RawMessage) (interface{}, *mcp.RPCError) {
	switch toolName {
	case "list-inbox":
		return s.handleListInbox(ctx, params)
	case "get-email":
		return s.handleGetEmail(ctx, params)
	case "search-emails":
		return s.handleSearchEmails(ctx, params)
	case "send-email":
		return s.handleSendEmail(ctx, params)
	case "reply-to-email":
		return s.handleReplyEmail(ctx, params)
	default:
		return nil, &mcp.RPCError{Code: -32601, Message: fmt.Sprintf("Gmail tool not found: %s", toolName)}
	}
}

// --- handlers ---

func (s *GmailService) handleListInbox(ctx context.Context, params json.RawMessage) (interface{}, *mcp.RPCError) {
	var args struct {
		MaxResults int64  `json:"maxResults"`
		Query      string `json:"query"`
		PageToken  string `json:"pageToken"`
	}
	if err := json.Unmarshal(params, &args); err != nil {
		return nil, &mcp.RPCError{Code: -32602, Message: "Invalid arguments", Data: err.Error()}
	}

	res, err := s.api.ListInbox(ctx, args.MaxResults, args.Query, args.PageToken)
	if err != nil {
		return nil, &mcp.RPCError{Code: -32603, Message: "Failed to list inbox", Data: err.Error()}
	}

	var b strings.Builder
	if len(res.Messages) == 0 {
		b.WriteString("📭 Inbox vacío.")
	} else {
		for i, m := range res.Messages {
			date := gmail.HumanDate(m.Date)
			fmt.Fprintf(&b, "%d. %s\n   📧 %s\n   👤 %s  🕐 %s\n   💬 %s\n",
				i+1, m.Subject, m.ID, m.From, date, m.Snippet)
		}
	}

	if res.NextPageToken != "" {
		fmt.Fprintf(&b, "\nHay más resultados. Para la siguiente página, usá pageToken=%q.", res.NextPageToken)
	}

	return contentResponse(b.String()), nil
}

func (s *GmailService) handleGetEmail(ctx context.Context, params json.RawMessage) (interface{}, *mcp.RPCError) {
	var args struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(params, &args); err != nil {
		return nil, &mcp.RPCError{Code: -32602, Message: "Invalid arguments", Data: err.Error()}
	}
	if args.ID == "" {
		return nil, &mcp.RPCError{Code: -32602, Message: "id required"}
	}

	detail, err := s.api.GetEmail(ctx, args.ID)
	if err != nil {
		return nil, &mcp.RPCError{Code: -32603, Message: "Failed to get email", Data: err.Error()}
	}

	body := detail.Body
	if detail.HTML {
		body = stripHTML(body)
	}
	if len(body) > 5000 {
		body = body[:5000] + "\n\n[...truncated at 5000 chars]"
	}

	result := fmt.Sprintf("📧 %s\nFrom: %s\nTo: %s\nDate: %s\n\n%s",
		detail.Subject, detail.From, detail.To, detail.Date, body)

	return contentResponse(result), nil
}

func (s *GmailService) handleSearchEmails(ctx context.Context, params json.RawMessage) (interface{}, *mcp.RPCError) {
	var args struct {
		Query      string `json:"query"`
		MaxResults int64  `json:"maxResults"`
		PageToken  string `json:"pageToken"`
	}
	if err := json.Unmarshal(params, &args); err != nil {
		return nil, &mcp.RPCError{Code: -32602, Message: "Invalid arguments", Data: err.Error()}
	}
	if args.Query == "" {
		return nil, &mcp.RPCError{Code: -32602, Message: "query required"}
	}

	res, err := s.api.SearchEmails(ctx, args.Query, args.MaxResults, args.PageToken)
	if err != nil {
		return nil, &mcp.RPCError{Code: -32603, Message: "Failed to search", Data: err.Error()}
	}

	var b strings.Builder
	if len(res.Messages) == 0 {
		b.WriteString("No results.")
	} else {
		for i, m := range res.Messages {
			date := gmail.HumanDate(m.Date)
			fmt.Fprintf(&b, "%d. [%s] %s\n   From: %s  %s\n   %s\n",
				i+1, m.ID, m.Subject, m.From, date, m.Snippet)
		}
	}

	if res.NextPageToken != "" {
		fmt.Fprintf(&b, "\nHay más resultados. Usá pageToken=%q para la siguiente página.", res.NextPageToken)
	}

	return contentResponse(b.String()), nil
}

func (s *GmailService) handleSendEmail(ctx context.Context, params json.RawMessage) (interface{}, *mcp.RPCError) {
	var args struct {
		To          string            `json:"to"`
		Subject     string            `json:"subject"`
		Body        string            `json:"body"`
		Attachments []attachmentInput `json:"attachments"`
	}
	if err := json.Unmarshal(params, &args); err != nil {
		return nil, &mcp.RPCError{Code: -32602, Message: "Invalid arguments", Data: err.Error()}
	}
	if args.To == "" || args.Subject == "" {
		return nil, &mcp.RPCError{Code: -32602, Message: "to and subject required"}
	}

	attachments, attErr := s.resolveAttachments(ctx, args.Attachments)
	if attErr != nil {
		return nil, attErr
	}

	sent, err := s.api.SendEmail(ctx, args.To, args.Subject, args.Body, attachments)
	if err != nil {
		return nil, &mcp.RPCError{Code: -32603, Message: "Failed to send", Data: err.Error()}
	}

	result := fmt.Sprintf("✅ Email sent to %s\nID: %s", args.To, sent.Id)
	if len(attachments) > 0 {
		result += fmt.Sprintf("\n📎 Attachments: %d", len(attachments))
	}
	return contentResponse(result), nil
}

func (s *GmailService) handleReplyEmail(ctx context.Context, params json.RawMessage) (interface{}, *mcp.RPCError) {
	var args struct {
		ThreadID    string            `json:"threadId"`
		To          string            `json:"to"`
		Subject     string            `json:"subject"`
		Body        string            `json:"body"`
		Attachments []attachmentInput `json:"attachments"`
	}
	if err := json.Unmarshal(params, &args); err != nil {
		return nil, &mcp.RPCError{Code: -32602, Message: "Invalid arguments", Data: err.Error()}
	}
	if args.ThreadID == "" || args.Body == "" {
		return nil, &mcp.RPCError{Code: -32602, Message: "threadId and body required"}
	}

	attachments, attErr := s.resolveAttachments(ctx, args.Attachments)
	if attErr != nil {
		return nil, attErr
	}

	reply, err := s.api.ReplyToEmail(ctx, args.ThreadID, args.To, args.Subject, args.Body, attachments)
	if err != nil {
		return nil, &mcp.RPCError{Code: -32603, Message: "Failed to reply", Data: err.Error()}
	}

	result := fmt.Sprintf("✅ Reply sent to %s\nSubject: %s\nID: %s\nThread: %s",
		reply.To, reply.Subject, reply.Message.Id, reply.Message.ThreadId)
	if !reply.Threaded {
		// Gmail accepted the message but did not attach it to the conversation, so
		// the recipient would see a new email. Report that instead of plain success.
		result += "\nWarning: Gmail did not add this to the requested thread; it was delivered as a new conversation."
	}
	if len(attachments) > 0 {
		result += fmt.Sprintf("\n📎 Attachments: %d", len(attachments))
	}
	return contentResponse(result), nil
}

// attachmentInput is the JSON shape accepted by the tool schema.
type attachmentInput struct {
	LocalPath   string `json:"localPath"`
	DriveFileID string `json:"driveFileId"`
}

// resolveAttachments converts attachment input params into gmail.Attachment
// structs, reading local files or downloading from Drive as needed.
func (s *GmailService) resolveAttachments(ctx context.Context, inputs []attachmentInput) ([]gmail.Attachment, *mcp.RPCError) {
	if len(inputs) == 0 {
		return nil, nil
	}

	attachments := make([]gmail.Attachment, 0, len(inputs))
	for i, input := range inputs {
		switch {
		case input.LocalPath != "":
			data, err := os.ReadFile(input.LocalPath)
			if err != nil {
				return nil, &mcp.RPCError{
					Code:    -32603,
					Message: fmt.Sprintf("Failed to read attachment %d: %s", i+1, input.LocalPath),
					Data:    err.Error(),
				}
			}
			mimeType := mime.TypeByExtension(filepath.Ext(input.LocalPath))
			if mimeType == "" {
				mimeType = "application/octet-stream"
			}
			attachments = append(attachments, gmail.Attachment{
				Filename: filepath.Base(input.LocalPath),
				MimeType: mimeType,
				Data:     data,
			})

		case input.DriveFileID != "":
			if s.driveAPI == nil {
				return nil, &mcp.RPCError{
					Code:    -32603,
					Message: "Drive attachments not available (Drive API not configured)",
				}
			}
			content, err := s.driveAPI.DownloadContent(ctx, input.DriveFileID)
			if err != nil {
				return nil, &mcp.RPCError{
					Code:    -32603,
					Message: fmt.Sprintf("Failed to download Drive file %s", input.DriveFileID),
					Data:    err.Error(),
				}
			}
			attachments = append(attachments, gmail.Attachment{
				Filename: content.Name,
				MimeType: content.MimeType,
				Data:     content.Data,
			})

		default:
			return nil, &mcp.RPCError{
				Code:    -32602,
				Message: fmt.Sprintf("Attachment %d must have 'localPath' or 'driveFileId'", i+1),
			}
		}
	}
	return attachments, nil
}

// stripHTML removes HTML tags for plain-text display.
func stripHTML(s string) string {
	var b strings.Builder
	inTag := false
	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}
