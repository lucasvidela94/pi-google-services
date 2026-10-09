// Package docs wraps the Google Docs API v1.
package docs

import (
	"context"
	"fmt"
	"strings"

	"golang.org/x/oauth2"
	"google.golang.org/api/docs/v1"
	"google.golang.org/api/option"
)

// Service wraps the Google Docs API client.
type Service struct {
	docs *docs.DocumentsService
}

// DocSummary is a lightweight document representation.
type DocSummary struct {
	DocID string `json:"docId"`
	Title string `json:"title"`
}

// New creates a Service from an OAuth2 token source.
func New(ctx context.Context, ts oauth2.TokenSource) (*Service, error) {
	svc, err := docs.NewService(ctx, option.WithTokenSource(ts))
	if err != nil {
		return nil, fmt.Errorf("create docs service: %w", err)
	}
	return &Service{docs: docs.NewDocumentsService(svc)}, nil
}

// GetDocument returns the full document.
func (s *Service) GetDocument(ctx context.Context, docID string) (*docs.Document, error) {
	if docID == "" {
		return nil, fmt.Errorf("docId required")
	}
	d, err := s.docs.Get(docID).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("get document: %w", err)
	}
	return d, nil
}

// GetText returns the document title and its body as plain text.
func (s *Service) GetText(ctx context.Context, docID string) (title, text string, err error) {
	d, err := s.GetDocument(ctx, docID)
	if err != nil {
		return "", "", err
	}
	return d.Title, ExtractText(d), nil
}

// CreateDocument creates an empty document with the given title.
func (s *Service) CreateDocument(ctx context.Context, title string) (*DocSummary, error) {
	if title == "" {
		return nil, fmt.Errorf("title required")
	}
	created, err := s.docs.Create(&docs.Document{Title: title}).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("create document: %w", err)
	}
	return &DocSummary{DocID: created.DocumentId, Title: created.Title}, nil
}

// AppendText appends text at the end of the document body.
func (s *Service) AppendText(ctx context.Context, docID, text string) error {
	if docID == "" {
		return fmt.Errorf("docId required")
	}
	if text == "" {
		return fmt.Errorf("text required")
	}
	// EndOfSegmentLocation with no segment ID targets the end of the body:
	// no index arithmetic, no empty-index pitfalls.
	_, err := s.docs.BatchUpdate(docID, &docs.BatchUpdateDocumentRequest{
		Requests: []*docs.Request{
			{
				InsertText: &docs.InsertTextRequest{
					EndOfSegmentLocation: &docs.EndOfSegmentLocation{},
					Text:                 text,
				},
			},
		},
	}).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("append text: %w", err)
	}
	return nil
}

// ExtractText renders a document body as plain text (paragraphs and tables).
func ExtractText(d *docs.Document) string {
	if d == nil || d.Body == nil {
		return ""
	}
	var b strings.Builder
	writeElements(&b, d.Body.Content)
	return strings.TrimRight(b.String(), "\n")
}

func writeElements(b *strings.Builder, elements []*docs.StructuralElement) {
	for _, el := range elements {
		switch {
		case el.Paragraph != nil:
			for _, pe := range el.Paragraph.Elements {
				if pe.TextRun != nil {
					b.WriteString(pe.TextRun.Content)
				}
			}
		case el.Table != nil:
			for _, row := range el.Table.TableRows {
				cells := make([]string, 0, len(row.TableCells))
				for _, cell := range row.TableCells {
					var cb strings.Builder
					writeElements(&cb, cell.Content)
					cells = append(cells, strings.TrimSpace(cb.String()))
				}
				b.WriteString(strings.Join(cells, " | "))
				b.WriteString("\n")
			}
		}
	}
}
