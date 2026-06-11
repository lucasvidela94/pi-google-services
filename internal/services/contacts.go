package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sombi/pi-google-services/internal/contacts"
	"github.com/sombi/pi-google-services/internal/mcp"
)

// ContactsService implements the Service interface for Google Contacts.
type ContactsService struct {
	api *contacts.Service
}

// NewContacts creates a ContactsService from the People API wrapper.
func NewContacts(api *contacts.Service) *ContactsService {
	return &ContactsService{api: api}
}

func (s *ContactsService) Name() string { return "contacts" }

func (s *ContactsService) Scopes() []string {
	return []string{
		"https://www.googleapis.com/auth/contacts",
		"https://www.googleapis.com/auth/contacts.readonly",
	}
}

func (s *ContactsService) Tools() []mcp.ToolDefinition {
	return []mcp.ToolDefinition{
		{
			Name:        "search-contacts",
			Description: "Search contacts by name, email, or phone",
			InputSchema: mcp.InputSchema{
				Type: "object",
				Properties: map[string]mcp.PropertySchema{
					"query": {Type: "string", Description: "Search query (name, email, or phone)"},
					"limit": {Type: "number", Description: "Max results (default: 10, max: 30)", Default: 10},
				},
				Required: []string{"query"},
			},
		},
		{
			Name:        "get-contact",
			Description: "Get full details of a contact by resource name",
			InputSchema: mcp.InputSchema{
				Type: "object",
				Properties: map[string]mcp.PropertySchema{
					"resourceName": {Type: "string", Description: "Contact resource name (e.g. people/c12345)"},
				},
				Required: []string{"resourceName"},
			},
		},
		{
			Name:        "create-contact",
			Description: "Create a new contact",
			InputSchema: mcp.InputSchema{
				Type: "object",
				Properties: map[string]mcp.PropertySchema{
					"name":  {Type: "string", Description: "Contact name"},
					"email": {Type: "string", Description: "Email address"},
					"phone": {Type: "string", Description: "Phone number"},
				},
				Required: []string{"name"},
			},
		},
	}
}

func (s *ContactsService) Handle(ctx context.Context, toolName string, params json.RawMessage) (interface{}, *mcp.RPCError) {
	switch toolName {
	case "search-contacts":
		return s.handleSearchContacts(ctx, params)
	case "get-contact":
		return s.handleGetContact(ctx, params)
	case "create-contact":
		return s.handleCreateContact(ctx, params)
	default:
		return nil, &mcp.RPCError{Code: -32601, Message: fmt.Sprintf("Contacts tool not found: %s", toolName)}
	}
}

func (s *ContactsService) handleSearchContacts(ctx context.Context, params json.RawMessage) (interface{}, *mcp.RPCError) {
	var args struct {
		Query string `json:"query"`
		Limit int64  `json:"limit"`
	}
	if err := json.Unmarshal(params, &args); err != nil {
		return nil, &mcp.RPCError{Code: -32602, Message: "Invalid arguments", Data: err.Error()}
	}
	if args.Query == "" {
		return nil, &mcp.RPCError{Code: -32602, Message: "query required"}
	}

	results, err := s.api.SearchContacts(ctx, args.Query, args.Limit)
	if err != nil {
		return nil, &mcp.RPCError{Code: -32603, Message: "Failed to search contacts", Data: err.Error()}
	}

	var b strings.Builder
	if len(results) == 0 {
		b.WriteString("No contacts found.")
	} else {
		b.WriteString(contacts.FormatContacts(results))
	}
	return contentResponse(b.String()), nil
}

func (s *ContactsService) handleGetContact(ctx context.Context, params json.RawMessage) (interface{}, *mcp.RPCError) {
	var args struct {
		ResourceName string `json:"resourceName"`
	}
	if err := json.Unmarshal(params, &args); err != nil {
		return nil, &mcp.RPCError{Code: -32602, Message: "Invalid arguments", Data: err.Error()}
	}
	if args.ResourceName == "" {
		return nil, &mcp.RPCError{Code: -32602, Message: "resourceName required"}
	}

	c, err := s.api.GetContact(ctx, args.ResourceName)
	if err != nil {
		return nil, &mcp.RPCError{Code: -32603, Message: "Failed to get contact", Data: err.Error()}
	}

	return contentResponse(contacts.FormatContacts([]*contacts.ContactSummary{c})), nil
}

func (s *ContactsService) handleCreateContact(ctx context.Context, params json.RawMessage) (interface{}, *mcp.RPCError) {
	var args struct {
		Name  string `json:"name"`
		Email string `json:"email"`
		Phone string `json:"phone"`
	}
	if err := json.Unmarshal(params, &args); err != nil {
		return nil, &mcp.RPCError{Code: -32602, Message: "Invalid arguments", Data: err.Error()}
	}
	if args.Name == "" {
		return nil, &mcp.RPCError{Code: -32602, Message: "name required"}
	}

	created, err := s.api.CreateContact(ctx, args.Name, args.Email, args.Phone)
	if err != nil {
		return nil, &mcp.RPCError{Code: -32603, Message: "Failed to create contact", Data: err.Error()}
	}

	return contentResponse(fmt.Sprintf("✅ Contact created: %s%s%s\n   🔖 %s",
		created.Name,
		maybe("", created.Emails),
		maybe("", created.Phones),
		created.ResourceName)), nil
}

func maybe[T any](prefix string, items []T) string {
	if len(items) == 0 {
		return ""
	}
	return fmt.Sprintf(" %s%v", prefix, items[0])
}
