package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sombi/pi-google-services/internal/gmail"
	"github.com/sombi/pi-google-services/internal/mcp"
)

// GmailService implements the Service interface for Gmail.
type GmailService struct {
	api *gmail.Service
}

// NewGmail creates a GmailService from the Gmail API wrapper.
func NewGmail(api *gmail.Service) *GmailService {
	return &GmailService{api: api}
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
			Description: "Search emails by query",
			InputSchema: mcp.InputSchema{
				Type: "object",
				Properties: map[string]mcp.PropertySchema{
					"query":      {Type: "string", Description: "Search query (Gmail syntax)"},
					"maxResults": {Type: "number", Description: "Max results (default: 20)", Default: 20},
				},
				Required: []string{"query"},
			},
		},
		{
			Name:        "send-email",
			Description: "Send an email",
			InputSchema: mcp.InputSchema{
				Type: "object",
				Properties: map[string]mcp.PropertySchema{
					"to":      {Type: "string", Description: "Recipient email"},
					"subject": {Type: "string", Description: "Email subject"},
					"body":    {Type: "string", Description: "Email body text"},
				},
				Required: []string{"to", "subject", "body"},
			},
		},
		{
			Name:        "reply-to-email",
			Description: "Reply to an email thread",
			InputSchema: mcp.InputSchema{
				Type: "object",
				Properties: map[string]mcp.PropertySchema{
					"threadId": {Type: "string", Description: "Thread ID to reply to"},
					"to":       {Type: "string", Description: "Recipient email"},
					"subject":  {Type: "string", Description: "Reply subject"},
					"body":     {Type: "string", Description: "Reply body text"},
				},
				Required: []string{"threadId", "to", "subject", "body"},
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
	}
	if err := json.Unmarshal(params, &args); err != nil {
		return nil, &mcp.RPCError{Code: -32602, Message: "Invalid arguments", Data: err.Error()}
	}

	msgs, err := s.api.ListInbox(ctx, args.MaxResults, args.Query)
	if err != nil {
		return nil, &mcp.RPCError{Code: -32603, Message: "Failed to list inbox", Data: err.Error()}
	}

	var b strings.Builder
	if len(msgs) == 0 {
		b.WriteString("📭 Inbox vacío.")
	} else {
		for i, m := range msgs {
			date := gmail.HumanDate(m.Date)
			b.WriteString(fmt.Sprintf("%d. %s\n   📧 %s\n   👤 %s  🕐 %s\n   💬 %s\n",
				i+1, m.Subject, m.ID, m.From, date, m.Snippet))
		}
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
	}
	if err := json.Unmarshal(params, &args); err != nil {
		return nil, &mcp.RPCError{Code: -32602, Message: "Invalid arguments", Data: err.Error()}
	}
	if args.Query == "" {
		return nil, &mcp.RPCError{Code: -32602, Message: "query required"}
	}

	msgs, err := s.api.SearchEmails(ctx, args.Query, args.MaxResults)
	if err != nil {
		return nil, &mcp.RPCError{Code: -32603, Message: "Failed to search", Data: err.Error()}
	}

	var b strings.Builder
	if len(msgs) == 0 {
		b.WriteString("No results.")
	} else {
		for i, m := range msgs {
			date := gmail.HumanDate(m.Date)
			b.WriteString(fmt.Sprintf("%d. [%s] %s\n   From: %s  %s\n   %s\n",
				i+1, m.ID, m.Subject, m.From, date, m.Snippet))
		}
	}

	return contentResponse(b.String()), nil
}

func (s *GmailService) handleSendEmail(ctx context.Context, params json.RawMessage) (interface{}, *mcp.RPCError) {
	var args struct {
		To      string `json:"to"`
		Subject string `json:"subject"`
		Body    string `json:"body"`
	}
	if err := json.Unmarshal(params, &args); err != nil {
		return nil, &mcp.RPCError{Code: -32602, Message: "Invalid arguments", Data: err.Error()}
	}
	if args.To == "" || args.Subject == "" {
		return nil, &mcp.RPCError{Code: -32602, Message: "to and subject required"}
	}

	sent, err := s.api.SendEmail(ctx, args.To, args.Subject, args.Body)
	if err != nil {
		return nil, &mcp.RPCError{Code: -32603, Message: "Failed to send", Data: err.Error()}
	}

	return contentResponse(fmt.Sprintf("✅ Email sent to %s\nID: %s", args.To, sent.Id)), nil
}

func (s *GmailService) handleReplyEmail(ctx context.Context, params json.RawMessage) (interface{}, *mcp.RPCError) {
	var args struct {
		ThreadID string `json:"threadId"`
		To       string `json:"to"`
		Subject  string `json:"subject"`
		Body     string `json:"body"`
	}
	if err := json.Unmarshal(params, &args); err != nil {
		return nil, &mcp.RPCError{Code: -32602, Message: "Invalid arguments", Data: err.Error()}
	}
	if args.ThreadID == "" || args.To == "" || args.Subject == "" {
		return nil, &mcp.RPCError{Code: -32602, Message: "threadId, to, subject required"}
	}

	sent, err := s.api.ReplyToEmail(ctx, args.ThreadID, args.To, args.Subject, args.Body)
	if err != nil {
		return nil, &mcp.RPCError{Code: -32603, Message: "Failed to reply", Data: err.Error()}
	}

	return contentResponse(fmt.Sprintf("✅ Reply sent\nID: %s", sent.Id)), nil
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
