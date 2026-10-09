// Package sheets wraps the Google Sheets API v4 (read-only).
package sheets

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

// Service wraps the Google Sheets API client.
type Service struct {
	sheets *sheets.SpreadsheetsService
	values *sheets.SpreadsheetsValuesService
}

// New creates a Service from an OAuth2 token source.
func New(ctx context.Context, ts oauth2.TokenSource) (*Service, error) {
	svc, err := sheets.NewService(ctx, option.WithTokenSource(ts))
	if err != nil {
		return nil, fmt.Errorf("create sheets service: %w", err)
	}
	return &Service{
		sheets: sheets.NewSpreadsheetsService(svc),
		values: sheets.NewSpreadsheetsValuesService(svc),
	}, nil
}

// ListSheets returns the tab titles of a spreadsheet.
func (s *Service) ListSheets(ctx context.Context, spreadsheetID string) ([]string, error) {
	if spreadsheetID == "" {
		return nil, fmt.Errorf("spreadsheetId required")
	}
	spreadsheet, err := s.sheets.Get(spreadsheetID).
		Fields("properties.title,sheets.properties.title").
		Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("get spreadsheet: %w", err)
	}
	titles := make([]string, 0, len(spreadsheet.Sheets))
	for _, sh := range spreadsheet.Sheets {
		if sh.Properties != nil {
			titles = append(titles, sh.Properties.Title)
		}
	}
	return titles, nil
}

// ReadRange returns a range as rows of strings.
func (s *Service) ReadRange(ctx context.Context, spreadsheetID, cellRange string) ([][]string, error) {
	if spreadsheetID == "" {
		return nil, fmt.Errorf("spreadsheetId required")
	}
	if cellRange == "" {
		cellRange = "A1:Z100"
	}
	res, err := s.values.Get(spreadsheetID, cellRange).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("read range: %w", err)
	}
	rows := make([][]string, 0, len(res.Values))
	for _, row := range res.Values {
		cells := make([]string, 0, len(row))
		for _, v := range row {
			cells = append(cells, FormatValue(v))
		}
		rows = append(rows, cells)
	}
	return rows, nil
}

// FormatValue renders a Sheets cell value as a string.
func FormatValue(v interface{}) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case bool:
		return strconv.FormatBool(t)
	case float64:
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", t))
	}
}

// FormatRows renders rows for display, one pipe-separated row per line.
func FormatRows(rows [][]string) string {
	if len(rows) == 0 {
		return "No values in range."
	}
	var b strings.Builder
	for _, row := range rows {
		b.WriteString(strings.Join(row, " | "))
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}
