package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sombi/pi-google-services/internal/mcp"
	sheetsapi "github.com/sombi/pi-google-services/internal/sheets"
)

// SheetsService implements the Service interface for Google Sheets (read-only).
type SheetsService struct {
	api *sheetsapi.Service
}

// NewSheets creates a SheetsService from the Sheets API wrapper.
func NewSheets(api *sheetsapi.Service) *SheetsService {
	return &SheetsService{api: api}
}

func (s *SheetsService) Name() string { return "sheets" }

func (s *SheetsService) Scopes() []string {
	return []string{"https://www.googleapis.com/auth/spreadsheets.readonly"}
}

func (s *SheetsService) Tools() []mcp.ToolDefinition {
	return []mcp.ToolDefinition{
		{
			Name:        "list-sheets",
			Description: "List tab names of a spreadsheet",
			InputSchema: mcp.InputSchema{
				Type: "object",
				Properties: map[string]mcp.PropertySchema{
					"spreadsheetId": {Type: "string", Description: "Spreadsheet ID (from Drive URL or search-drive)"},
				},
				Required: []string{"spreadsheetId"},
			},
		},
		{
			Name:        "read-sheet",
			Description: "Read cell values from a spreadsheet range",
			InputSchema: mcp.InputSchema{
				Type: "object",
				Properties: map[string]mcp.PropertySchema{
					"spreadsheetId": {Type: "string", Description: "Spreadsheet ID"},
					"range":         {Type: "string", Description: "A1 notation (e.g. 'A1:C10', \"'Hoja 1'!A1:B5\"). Default: A1:Z100 on the first tab"},
				},
				Required: []string{"spreadsheetId"},
			},
		},
	}
}

func (s *SheetsService) Handle(ctx context.Context, toolName string, params json.RawMessage) (interface{}, *mcp.RPCError) {
	switch toolName {
	case "list-sheets":
		return s.handleListSheets(ctx, params)
	case "read-sheet":
		return s.handleReadSheet(ctx, params)
	default:
		return nil, &mcp.RPCError{Code: -32601, Message: fmt.Sprintf("Sheets tool not found: %s", toolName)}
	}
}

func (s *SheetsService) handleListSheets(ctx context.Context, params json.RawMessage) (interface{}, *mcp.RPCError) {
	var args struct {
		SpreadsheetID string `json:"spreadsheetId"`
	}
	if err := json.Unmarshal(params, &args); err != nil {
		return nil, &mcp.RPCError{Code: -32602, Message: "Invalid arguments", Data: err.Error()}
	}
	if args.SpreadsheetID == "" {
		return nil, &mcp.RPCError{Code: -32602, Message: "spreadsheetId required"}
	}

	titles, err := s.api.ListSheets(ctx, args.SpreadsheetID)
	if err != nil {
		return nil, &mcp.RPCError{Code: -32603, Message: "Failed to list sheets", Data: err.Error()}
	}

	var b strings.Builder
	if len(titles) == 0 {
		b.WriteString("No tabs found.")
	} else {
		for i, title := range titles {
			b.WriteString(fmt.Sprintf("%d. 📊 %s\n", i+1, title))
		}
	}
	return contentResponse(b.String()), nil
}

func (s *SheetsService) handleReadSheet(ctx context.Context, params json.RawMessage) (interface{}, *mcp.RPCError) {
	var args struct {
		SpreadsheetID string `json:"spreadsheetId"`
		Range         string `json:"range"`
	}
	if err := json.Unmarshal(params, &args); err != nil {
		return nil, &mcp.RPCError{Code: -32602, Message: "Invalid arguments", Data: err.Error()}
	}
	if args.SpreadsheetID == "" {
		return nil, &mcp.RPCError{Code: -32602, Message: "spreadsheetId required"}
	}

	rows, err := s.api.ReadRange(ctx, args.SpreadsheetID, args.Range)
	if err != nil {
		return nil, &mcp.RPCError{Code: -32603, Message: "Failed to read range", Data: err.Error()}
	}

	cellRange := args.Range
	if cellRange == "" {
		cellRange = "A1:Z100"
	}
	return contentResponse(fmt.Sprintf("📊 %s (%s)\n\n%s",
		args.SpreadsheetID, cellRange, sheetsapi.FormatRows(rows))), nil
}
