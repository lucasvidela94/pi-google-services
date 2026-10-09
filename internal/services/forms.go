package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"google.golang.org/api/forms/v1"

	formsapi "github.com/sombi/pi-google-services/internal/forms"
	"github.com/sombi/pi-google-services/internal/mcp"
)

// FormsService implements the Service interface for Google Forms.
type FormsService struct {
	api *formsapi.Service
}

// NewForms creates a FormsService from the Forms API wrapper.
func NewForms(api *formsapi.Service) *FormsService {
	return &FormsService{api: api}
}

func (s *FormsService) Name() string { return "forms" }

func (s *FormsService) Scopes() []string {
	return []string{
		"https://www.googleapis.com/auth/forms.body",
		"https://www.googleapis.com/auth/forms.responses.readonly",
	}
}

func (s *FormsService) Tools() []mcp.ToolDefinition {
	return []mcp.ToolDefinition{
		{
			Name:        "create-form",
			Description: "Create a new Google Form (empty, then add sections/questions)",
			InputSchema: mcp.InputSchema{
				Type: "object",
				Properties: map[string]mcp.PropertySchema{
					"title":       {Type: "string", Description: "Form title (visible to responders)"},
					"description": {Type: "string", Description: "Form description"},
				},
				Required: []string{"title"},
			},
		},
		{
			Name:        "add-form-section",
			Description: "Add a section header (title + description, no question) to a form",
			InputSchema: mcp.InputSchema{
				Type: "object",
				Properties: map[string]mcp.PropertySchema{
					"formId":      {Type: "string", Description: "Form ID"},
					"title":       {Type: "string", Description: "Section title"},
					"description": {Type: "string", Description: "Section description"},
				},
				Required: []string{"formId", "title"},
			},
		},
		{
			Name:        "add-form-question",
			Description: "Add a question to a form. Without options it is free text; with options it is a RADIO/CHECKBOX/DROP_DOWN choice",
			InputSchema: mcp.InputSchema{
				Type: "object",
				Properties: map[string]mcp.PropertySchema{
					"formId":      {Type: "string", Description: "Form ID"},
					"title":       {Type: "string", Description: "Question text"},
					"description": {Type: "string", Description: "Help text shown under the question"},
					"paragraph":   {Type: "boolean", Description: "Long (paragraph) text answer instead of short text (default: false)"},
					"required":    {Type: "boolean", Description: "Whether an answer is required (default: false)"},
					"options":     {Type: "string", Description: "Comma-separated choice options (e.g. 'Si,No'). Omit for free-text question"},
					"choiceType":  {Type: "string", Description: "RADIO (default), CHECKBOX or DROP_DOWN. Only used with options"},
				},
				Required: []string{"formId", "title"},
			},
		},
		{
			Name:        "get-form",
			Description: "Get a form structure (title, items, responder link)",
			InputSchema: mcp.InputSchema{
				Type: "object",
				Properties: map[string]mcp.PropertySchema{
					"formId": {Type: "string", Description: "Form ID"},
				},
				Required: []string{"formId"},
			},
		},
		{
			Name:        "list-responses",
			Description: "List responses submitted to a form",
			InputSchema: mcp.InputSchema{
				Type: "object",
				Properties: map[string]mcp.PropertySchema{
					"formId": {Type: "string", Description: "Form ID"},
					"limit":  {Type: "number", Description: "Max responses (default: 100)", Default: 100},
				},
				Required: []string{"formId"},
			},
		},
	}
}

func (s *FormsService) Handle(ctx context.Context, toolName string, params json.RawMessage) (interface{}, *mcp.RPCError) {
	switch toolName {
	case "create-form":
		return s.handleCreateForm(ctx, params)
	case "add-form-section":
		return s.handleAddSection(ctx, params)
	case "add-form-question":
		return s.handleAddQuestion(ctx, params)
	case "get-form":
		return s.handleGetForm(ctx, params)
	case "list-responses":
		return s.handleListResponses(ctx, params)
	default:
		return nil, &mcp.RPCError{Code: -32601, Message: fmt.Sprintf("Forms tool not found: %s", toolName)}
	}
}

func (s *FormsService) handleCreateForm(ctx context.Context, params json.RawMessage) (interface{}, *mcp.RPCError) {
	var args struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(params, &args); err != nil {
		return nil, &mcp.RPCError{Code: -32602, Message: "Invalid arguments", Data: err.Error()}
	}
	if args.Title == "" {
		return nil, &mcp.RPCError{Code: -32602, Message: "title required"}
	}

	created, err := s.api.CreateForm(ctx, args.Title, args.Description)
	if err != nil {
		return nil, &mcp.RPCError{Code: -32603, Message: "Failed to create form", Data: err.Error()}
	}

	return contentResponse(fmt.Sprintf("✅ Form created: %s\n   ID: %s\n   🔗 Responder link: %s",
		created.Title, created.FormID, created.ResponderURI)), nil
}

func (s *FormsService) handleAddSection(ctx context.Context, params json.RawMessage) (interface{}, *mcp.RPCError) {
	var args struct {
		FormID      string `json:"formId"`
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(params, &args); err != nil {
		return nil, &mcp.RPCError{Code: -32602, Message: "Invalid arguments", Data: err.Error()}
	}
	if args.FormID == "" || args.Title == "" {
		return nil, &mcp.RPCError{Code: -32602, Message: "formId and title required"}
	}

	updated, err := s.api.AppendItems(ctx, args.FormID, []*forms.Item{
		formsapi.NewSectionItem(args.Title, args.Description),
	})
	if err != nil {
		return nil, &mcp.RPCError{Code: -32603, Message: "Failed to add section", Data: err.Error()}
	}

	return contentResponse(fmt.Sprintf("✅ Section added to %s\n   Total items: %d", args.FormID, len(updated.Items))), nil
}

func (s *FormsService) handleAddQuestion(ctx context.Context, params json.RawMessage) (interface{}, *mcp.RPCError) {
	var args struct {
		FormID      string `json:"formId"`
		Title       string `json:"title"`
		Description string `json:"description"`
		Paragraph   bool   `json:"paragraph"`
		Required    bool   `json:"required"`
		Options     string `json:"options"`
		ChoiceType  string `json:"choiceType"`
	}
	if err := json.Unmarshal(params, &args); err != nil {
		return nil, &mcp.RPCError{Code: -32602, Message: "Invalid arguments", Data: err.Error()}
	}
	if args.FormID == "" || args.Title == "" {
		return nil, &mcp.RPCError{Code: -32602, Message: "formId and title required"}
	}

	var item *forms.Item
	if args.Options != "" {
		var options []string
		for _, o := range strings.Split(args.Options, ",") {
			if trimmed := strings.TrimSpace(o); trimmed != "" {
				options = append(options, trimmed)
			}
		}
		if len(options) == 0 {
			return nil, &mcp.RPCError{Code: -32602, Message: "options has no valid entries"}
		}
		choiceType := args.ChoiceType
		if choiceType == "" {
			choiceType = "RADIO"
		}
		switch choiceType {
		case "RADIO", "CHECKBOX", "DROP_DOWN":
		default:
			return nil, &mcp.RPCError{Code: -32602, Message: "choiceType must be RADIO, CHECKBOX or DROP_DOWN"}
		}
		item = formsapi.NewChoiceQuestion(args.Title, args.Description, args.Required, choiceType, options)
	} else {
		item = formsapi.NewTextQuestion(args.Title, args.Description, args.Paragraph, args.Required)
	}

	updated, err := s.api.AppendItems(ctx, args.FormID, []*forms.Item{item})
	if err != nil {
		return nil, &mcp.RPCError{Code: -32603, Message: "Failed to add question", Data: err.Error()}
	}

	return contentResponse(fmt.Sprintf("✅ Question added to %s\n   Total items: %d", args.FormID, len(updated.Items))), nil
}

func (s *FormsService) handleGetForm(ctx context.Context, params json.RawMessage) (interface{}, *mcp.RPCError) {
	var args struct {
		FormID string `json:"formId"`
	}
	if err := json.Unmarshal(params, &args); err != nil {
		return nil, &mcp.RPCError{Code: -32602, Message: "Invalid arguments", Data: err.Error()}
	}
	if args.FormID == "" {
		return nil, &mcp.RPCError{Code: -32602, Message: "formId required"}
	}

	f, err := s.api.GetForm(ctx, args.FormID)
	if err != nil {
		return nil, &mcp.RPCError{Code: -32603, Message: "Failed to get form", Data: err.Error()}
	}

	return contentResponse(formatForm(f)), nil
}

func (s *FormsService) handleListResponses(ctx context.Context, params json.RawMessage) (interface{}, *mcp.RPCError) {
	var args struct {
		FormID string `json:"formId"`
		Limit  int64  `json:"limit"`
	}
	if err := json.Unmarshal(params, &args); err != nil {
		return nil, &mcp.RPCError{Code: -32602, Message: "Invalid arguments", Data: err.Error()}
	}
	if args.FormID == "" {
		return nil, &mcp.RPCError{Code: -32602, Message: "formId required"}
	}

	responses, err := s.api.ListResponses(ctx, args.FormID, args.Limit)
	if err != nil {
		return nil, &mcp.RPCError{Code: -32603, Message: "Failed to list responses", Data: err.Error()}
	}

	var b strings.Builder
	if len(responses) == 0 {
		b.WriteString("No responses yet.")
	} else {
		for i, r := range responses {
			fmt.Fprintf(&b, "%d. 📝 %s", i+1, r.ResponseID)
			if r.Responder != "" {
				fmt.Fprintf(&b, " — %s", r.Responder)
			}
			if r.Submitted != "" {
				fmt.Fprintf(&b, " (%s)", r.Submitted)
			}
			b.WriteString("\n")
		}
	}
	return contentResponse(b.String()), nil
}

// formatForm renders a form structure for display.
func formatForm(f *forms.Form) string {
	var b strings.Builder
	title := f.FormId
	if f.Info != nil && f.Info.Title != "" {
		title = f.Info.Title
	}
	fmt.Fprintf(&b, "📋 %s\n", title)
	fmt.Fprintf(&b, "   ID: %s\n", f.FormId)
	if f.ResponderUri != "" {
		fmt.Fprintf(&b, "   🔗 %s\n", f.ResponderUri)
	}
	if len(f.Items) == 0 {
		b.WriteString("No items yet.")
		return b.String()
	}
	for i, item := range f.Items {
		kind := "📝"
		detail := ""
		if item.QuestionItem != nil && item.QuestionItem.Question != nil {
			q := item.QuestionItem.Question
			switch {
			case q.TextQuestion != nil:
				if q.TextQuestion.Paragraph {
					detail = " (paragraph)"
				} else {
					detail = " (short text)"
				}
			case q.ChoiceQuestion != nil:
				detail = fmt.Sprintf(" (%s: %s)", q.ChoiceQuestion.Type, choiceOptions(q.ChoiceQuestion.Options))
			}
		} else if item.TextItem != nil {
			kind = "📌"
			detail = " (section)"
		}
		fmt.Fprintf(&b, "%d. %s %s%s\n", i+1, kind, item.Title, detail)
	}
	return b.String()
}

func choiceOptions(options []*forms.Option) string {
	values := make([]string, 0, len(options))
	for _, o := range options {
		if o != nil && o.Value != "" {
			values = append(values, o.Value)
		}
	}
	return strings.Join(values, " / ")
}
