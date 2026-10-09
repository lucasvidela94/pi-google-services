package services

import (
	"context"
	"encoding/json"
	"testing"

	"google.golang.org/api/docs/v1"

	docsapi "github.com/sombi/pi-google-services/internal/docs"
)

func TestDocsServiceName(t *testing.T) {
	ds := &DocsService{}
	if ds.Name() != "docs" {
		t.Errorf("Name() = %q, want %q", ds.Name(), "docs")
	}
}

func TestDocsServiceScopes(t *testing.T) {
	ds := &DocsService{}
	scopes := ds.Scopes()
	if len(scopes) != 1 || scopes[0] != "https://www.googleapis.com/auth/documents" {
		t.Errorf("unexpected scopes: %v", scopes)
	}
}

func TestDocsServiceTools(t *testing.T) {
	ds := &DocsService{}
	tools := ds.Tools()
	if len(tools) != 3 {
		t.Fatalf("expected 3 tools, got %d", len(tools))
	}
	names := make(map[string]bool)
	for _, tool := range tools {
		names[tool.Name] = true
		if tool.InputSchema.Type != "object" {
			t.Errorf("tool %s: InputSchema.Type = %q, want object", tool.Name, tool.InputSchema.Type)
		}
	}
	for _, name := range []string{"get-doc", "create-doc", "append-to-doc"} {
		if !names[name] {
			t.Errorf("missing tool: %s", name)
		}
	}
}

func TestDocsServiceHandle_UnknownTool(t *testing.T) {
	ds := &DocsService{}
	_, err := ds.Handle(context.TODO(), "nonexistent", nil)
	if err == nil || err.Code != -32601 {
		t.Errorf("expected -32601 error, got %+v", err)
	}
}

func TestDocsServiceHandle_Validation(t *testing.T) {
	ds := &DocsService{} // api is nil; validation must fail before any API call
	cases := []struct {
		tool    string
		params  string
		wantMsg string
	}{
		{"get-doc", `{}`, "docId required"},
		{"create-doc", `{}`, "title required"},
		{"append-to-doc", `{}`, "docId and text required"},
		{"append-to-doc", `{"docId":"x"}`, "docId and text required"},
		{"get-doc", `{bad json`, "Invalid arguments"},
	}
	for _, tc := range cases {
		_, rpcErr := ds.Handle(context.TODO(), tc.tool, json.RawMessage(tc.params))
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

func TestExtractText(t *testing.T) {
	tr := func(s string) *docs.ParagraphElement {
		return &docs.ParagraphElement{TextRun: &docs.TextRun{Content: s}}
	}
	doc := &docs.Document{
		Body: &docs.Body{Content: []*docs.StructuralElement{
			{Paragraph: &docs.Paragraph{Elements: []*docs.ParagraphElement{
				tr("Hola "), tr("mundo\n"),
			}}},
			{Table: &docs.Table{TableRows: []*docs.TableRow{
				{TableCells: []*docs.TableCell{
					{Content: []*docs.StructuralElement{
						{Paragraph: &docs.Paragraph{Elements: []*docs.ParagraphElement{tr("a1\n")}}},
					}},
					{Content: []*docs.StructuralElement{
						{Paragraph: &docs.Paragraph{Elements: []*docs.ParagraphElement{tr("b1\n")}}},
					}},
				}},
			}}},
		}},
	}
	if got := docsapi.ExtractText(doc); got != "Hola mundo\na1 | b1" {
		t.Errorf("ExtractText = %q", got)
	}
	if got := docsapi.ExtractText(nil); got != "" {
		t.Errorf("ExtractText(nil) = %q", got)
	}
	if got := docsapi.ExtractText(&docs.Document{}); got != "" {
		t.Errorf("ExtractText(empty) = %q", got)
	}
}
