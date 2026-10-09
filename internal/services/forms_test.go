package services

import (
	"encoding/json"
	"testing"

	formsapi "github.com/sombi/pi-google-services/internal/forms"
)

func TestFormsServiceName(t *testing.T) {
	fs := &FormsService{}
	if fs.Name() != "forms" {
		t.Errorf("Name() = %q, want %q", fs.Name(), "forms")
	}
}

func TestFormsServiceScopes(t *testing.T) {
	fs := &FormsService{}
	scopes := fs.Scopes()
	if len(scopes) != 2 {
		t.Fatalf("expected 2 scopes, got %d: %v", len(scopes), scopes)
	}
	want := map[string]bool{
		"https://www.googleapis.com/auth/forms.body":               true,
		"https://www.googleapis.com/auth/forms.responses.readonly": true,
	}
	for _, s := range scopes {
		if !want[s] {
			t.Errorf("unexpected scope: %s", s)
		}
	}
}

func TestFormsServiceTools(t *testing.T) {
	fs := &FormsService{}
	tools := fs.Tools()
	if len(tools) != 5 {
		t.Fatalf("expected 5 tools, got %d", len(tools))
	}
	names := make(map[string]bool)
	for _, tool := range tools {
		names[tool.Name] = true
		if tool.InputSchema.Type != "object" {
			t.Errorf("tool %s: InputSchema.Type = %q, want object", tool.Name, tool.InputSchema.Type)
		}
	}
	for _, name := range []string{
		"create-form", "add-form-section", "add-form-question",
		"get-form", "list-responses",
	} {
		if !names[name] {
			t.Errorf("missing tool: %s", name)
		}
	}
}

func TestFormsServiceHandle_UnknownTool(t *testing.T) {
	fs := &FormsService{}
	_, err := fs.Handle(nil, "nonexistent", nil)
	if err == nil || err.Code != -32601 {
		t.Errorf("expected -32601 error, got %+v", err)
	}
}

func TestFormsServiceHandle_Validation(t *testing.T) {
	fs := &FormsService{} // api is nil; validation must fail before any API call
	cases := []struct {
		tool    string
		params  string
		wantMsg string
	}{
		{"create-form", `{}`, "title required"},
		{"create-form", `{"title":""}`, "title required"},
		{"add-form-section", `{}`, "formId and title required"},
		{"add-form-section", `{"formId":"x"}`, "formId and title required"},
		{"add-form-question", `{}`, "formId and title required"},
		{"add-form-question", `{"formId":"x","title":"q","options":" , "}`, "options has no valid entries"},
		{"add-form-question", `{"formId":"x","title":"q","options":"Si,No","choiceType":"BAD"}`, "choiceType must be RADIO, CHECKBOX or DROP_DOWN"},
		{"get-form", `{}`, "formId required"},
		{"list-responses", `{}`, "formId required"},
		{"create-form", `{bad json`, "Invalid arguments"},
	}
	for _, tc := range cases {
		_, rpcErr := fs.Handle(nil, tc.tool, json.RawMessage(tc.params))
		if rpcErr == nil {
			t.Errorf("%s %s: expected error, got nil", tc.tool, tc.params)
			continue
		}
		if rpcErr.Code != -32602 {
			t.Errorf("%s %s: code = %d, want -32602", tc.tool, tc.params, rpcErr.Code)
		}
		if rpcErr.Message != tc.wantMsg {
			t.Errorf("%s %s: message = %q, want %q", tc.tool, tc.params, rpcErr.Message, tc.wantMsg)
		}
	}
}

func TestNewSectionItem(t *testing.T) {
	item := formsapi.NewSectionItem("Urgencias", "Detalle")
	if item.Title != "Urgencias" {
		t.Errorf("Title = %q", item.Title)
	}
	if item.TextItem == nil {
		t.Error("TextItem should not be nil")
	}
	if item.QuestionItem != nil {
		t.Error("QuestionItem should be nil for a section")
	}
}

func TestNewTextQuestion(t *testing.T) {
	item := formsapi.NewTextQuestion("1.1 ¿Correcto?", "Detalle:", true, false)
	if item.QuestionItem == nil || item.QuestionItem.Question == nil {
		t.Fatal("Question should not be nil")
	}
	q := item.QuestionItem.Question
	if q.TextQuestion == nil || !q.TextQuestion.Paragraph {
		t.Error("expected paragraph text question")
	}
	if q.Required {
		t.Error("expected not required")
	}
}

func TestNewChoiceQuestion(t *testing.T) {
	item := formsapi.NewChoiceQuestion("1.1 ¿Correcto?", "", false, "", []string{"Sí", "No"})
	q := item.QuestionItem.Question
	if q.ChoiceQuestion == nil {
		t.Fatal("ChoiceQuestion should not be nil")
	}
	if q.ChoiceQuestion.Type != "RADIO" {
		t.Errorf("Type = %q, want RADIO", q.ChoiceQuestion.Type)
	}
	if len(q.ChoiceQuestion.Options) != 2 {
		t.Fatalf("expected 2 options, got %d", len(q.ChoiceQuestion.Options))
	}
	if q.ChoiceQuestion.Options[0].Value != "Sí" {
		t.Errorf("Options[0] = %q", q.ChoiceQuestion.Options[0].Value)
	}
}
