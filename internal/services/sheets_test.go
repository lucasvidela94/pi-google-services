package services

import (
	"encoding/json"
	"testing"

	sheetsapi "github.com/sombi/pi-google-services/internal/sheets"
)

func TestSheetsServiceName(t *testing.T) {
	ss := &SheetsService{}
	if ss.Name() != "sheets" {
		t.Errorf("Name() = %q, want %q", ss.Name(), "sheets")
	}
}

func TestSheetsServiceScopes(t *testing.T) {
	ss := &SheetsService{}
	scopes := ss.Scopes()
	if len(scopes) != 1 || scopes[0] != "https://www.googleapis.com/auth/spreadsheets.readonly" {
		t.Errorf("unexpected scopes: %v", scopes)
	}
}

func TestSheetsServiceTools(t *testing.T) {
	ss := &SheetsService{}
	tools := ss.Tools()
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}
	names := make(map[string]bool)
	for _, tool := range tools {
		names[tool.Name] = true
		if tool.InputSchema.Type != "object" {
			t.Errorf("tool %s: InputSchema.Type = %q, want object", tool.Name, tool.InputSchema.Type)
		}
	}
	for _, name := range []string{"list-sheets", "read-sheet"} {
		if !names[name] {
			t.Errorf("missing tool: %s", name)
		}
	}
}

func TestSheetsServiceHandle_UnknownTool(t *testing.T) {
	ss := &SheetsService{}
	_, err := ss.Handle(nil, "nonexistent", nil)
	if err == nil || err.Code != -32601 {
		t.Errorf("expected -32601 error, got %+v", err)
	}
}

func TestSheetsServiceHandle_Validation(t *testing.T) {
	ss := &SheetsService{} // api is nil; validation must fail before any API call
	cases := []struct {
		tool    string
		params  string
		wantMsg string
	}{
		{"list-sheets", `{}`, "spreadsheetId required"},
		{"read-sheet", `{}`, "spreadsheetId required"},
		{"read-sheet", `{bad json`, "Invalid arguments"},
	}
	for _, tc := range cases {
		_, rpcErr := ss.Handle(nil, tc.tool, json.RawMessage(tc.params))
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

func TestFormatValue(t *testing.T) {
	cases := []struct {
		in   interface{}
		want string
	}{
		{nil, ""},
		{"hola", "hola"},
		{true, "true"},
		{float64(42), "42"},
		{float64(4.5), "4.5"},
	}
	for _, tc := range cases {
		if got := sheetsapi.FormatValue(tc.in); got != tc.want {
			t.Errorf("FormatValue(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestFormatRows(t *testing.T) {
	rows := [][]string{{"a", "b"}, {"1", "2"}}
	if got := sheetsapi.FormatRows(rows); got != "a | b\n1 | 2" {
		t.Errorf("FormatRows = %q", got)
	}
	if got := sheetsapi.FormatRows(nil); got != "No values in range." {
		t.Errorf("FormatRows(nil) = %q", got)
	}
}
