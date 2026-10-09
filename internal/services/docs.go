package services

import (
	"context"
	"encoding/json"
	"fmt"

	docsapi "github.com/sombi/pi-google-services/internal/docs"
	"github.com/sombi/pi-google-services/internal/mcp"
)

// DocsService implements the Service interface for Google Docs.
type DocsService struct {
	api *docsapi.Service
}

// NewDocs creates a DocsService from the Docs API wrapper.
func NewDocs(api *docsapi.Service) *DocsService {
	return &DocsService{api: api}
}

func (s *DocsService) Name() string { return "docs" }

func (s *DocsService) Scopes() []string {
	return []string{"https://www.googleapis.com/auth/documents"}
}

func (s *DocsService) Tools() []mcp.ToolDefinition {
	return []mcp.ToolDefinition{
		{
			Name:        "get-doc",
			Description: "Read a Google Doc as plain text (title + body)",
			InputSchema: mcp.InputSchema{
				Type: "object",
				Properties: map[string]mcp.PropertySchema{
					"docId": {Type: "string", Description: "Document ID (from Drive URL or search-drive)"},
				},
				Required: []string{"docId"},
			},
		},
		{
			Name:        "create-doc",
			Description: "Create an empty Google Doc",
			InputSchema: mcp.InputSchema{
				Type: "object",
				Properties: map[string]mcp.PropertySchema{
					"title": {Type: "string", Description: "Document title"},
				},
				Required: []string{"title"},
			},
		},
		{
			Name:        "append-to-doc",
			Description: "Append text at the end of a Google Doc",
			InputSchema: mcp.InputSchema{
				Type: "object",
				Properties: map[string]mcp.PropertySchema{
					"docId": {Type: "string", Description: "Document ID"},
					"text":  {Type: "string", Description: "Text to append (newlines create new paragraphs)"},
				},
				Required: []string{"docId", "text"},
			},
		},
	}
}

func (s *DocsService) Handle(ctx context.Context, toolName string, params json.RawMessage) (interface{}, *mcp.RPCError) {
	switch toolName {
	case "get-doc":
		return s.handleGetDoc(ctx, params)
	case "create-doc":
		return s.handleCreateDoc(ctx, params)
	case "append-to-doc":
		return s.handleAppendToDoc(ctx, params)
	default:
		return nil, &mcp.RPCError{Code: -32601, Message: fmt.Sprintf("Docs tool not found: %s", toolName)}
	}
}

func (s *DocsService) handleGetDoc(ctx context.Context, params json.RawMessage) (interface{}, *mcp.RPCError) {
	var args struct {
		DocID string `json:"docId"`
	}
	if err := json.Unmarshal(params, &args); err != nil {
		return nil, &mcp.RPCError{Code: -32602, Message: "Invalid arguments", Data: err.Error()}
	}
	if args.DocID == "" {
		return nil, &mcp.RPCError{Code: -32602, Message: "docId required"}
	}

	title, text, err := s.api.GetText(ctx, args.DocID)
	if err != nil {
		return nil, &mcp.RPCError{Code: -32603, Message: "Failed to get document", Data: err.Error()}
	}
	if text == "" {
		text = "(empty document)"
	}
	return contentResponse(fmt.Sprintf("📄 %s\n   ID: %s\n\n%s", title, args.DocID, text)), nil
}

func (s *DocsService) handleCreateDoc(ctx context.Context, params json.RawMessage) (interface{}, *mcp.RPCError) {
	var args struct {
		Title string `json:"title"`
	}
	if err := json.Unmarshal(params, &args); err != nil {
		return nil, &mcp.RPCError{Code: -32602, Message: "Invalid arguments", Data: err.Error()}
	}
	if args.Title == "" {
		return nil, &mcp.RPCError{Code: -32602, Message: "title required"}
	}

	created, err := s.api.CreateDocument(ctx, args.Title)
	if err != nil {
		return nil, &mcp.RPCError{Code: -32603, Message: "Failed to create document", Data: err.Error()}
	}

	return contentResponse(fmt.Sprintf("✅ Document created: %s\n   ID: %s\n   🔗 https://docs.google.com/document/d/%s/edit",
		created.Title, created.DocID, created.DocID)), nil
}

func (s *DocsService) handleAppendToDoc(ctx context.Context, params json.RawMessage) (interface{}, *mcp.RPCError) {
	var args struct {
		DocID string `json:"docId"`
		Text  string `json:"text"`
	}
	if err := json.Unmarshal(params, &args); err != nil {
		return nil, &mcp.RPCError{Code: -32602, Message: "Invalid arguments", Data: err.Error()}
	}
	if args.DocID == "" || args.Text == "" {
		return nil, &mcp.RPCError{Code: -32602, Message: "docId and text required"}
	}

	if err := s.api.AppendText(ctx, args.DocID, args.Text); err != nil {
		return nil, &mcp.RPCError{Code: -32603, Message: "Failed to append", Data: err.Error()}
	}

	return contentResponse(fmt.Sprintf("✅ Appended to %s", args.DocID)), nil
}
