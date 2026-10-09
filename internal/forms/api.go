// Package forms wraps the Google Forms API v1.
package forms

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"
	"google.golang.org/api/forms/v1"
	"google.golang.org/api/option"
)

// Service wraps the Google Forms API client.
type Service struct {
	forms     *forms.FormsService
	responses *forms.FormsResponsesService
}

// FormSummary is a lightweight form representation.
type FormSummary struct {
	FormID       string `json:"formId"`
	Title        string `json:"title"`
	Description  string `json:"description,omitempty"`
	ResponderURI string `json:"responderUri,omitempty"`
	ItemCount    int    `json:"itemCount,omitempty"`
}

// ResponseSummary is a lightweight form response representation.
type ResponseSummary struct {
	ResponseID string `json:"responseId"`
	Responder  string `json:"responder,omitempty"`
	Submitted  string `json:"submitted,omitempty"`
}

// New creates a Service from an OAuth2 token source.
func New(ctx context.Context, ts oauth2.TokenSource) (*Service, error) {
	svc, err := forms.NewService(ctx, option.WithTokenSource(ts))
	if err != nil {
		return nil, fmt.Errorf("create forms service: %w", err)
	}
	return &Service{
		forms:     forms.NewFormsService(svc),
		responses: forms.NewFormsResponsesService(svc),
	}, nil
}

// CreateForm creates an empty form with title and optional description.
// The Forms API only accepts info.title on create, so the description is
// applied afterwards with an UpdateFormInfo batch request.
func (s *Service) CreateForm(ctx context.Context, title, description string) (*FormSummary, error) {
	if title == "" {
		return nil, fmt.Errorf("title required")
	}
	created, err := s.forms.Create(&forms.Form{
		Info: &forms.Info{Title: title},
	}).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("create form: %w", err)
	}
	if description != "" {
		updated, err := s.forms.BatchUpdate(created.FormId, &forms.BatchUpdateFormRequest{
			Requests: []*forms.Request{
				{
					UpdateFormInfo: &forms.UpdateFormInfoRequest{
						Info:       &forms.Info{Description: description},
						UpdateMask: "description",
					},
				},
			},
			IncludeFormInResponse: true,
		}).Context(ctx).Do()
		if err != nil {
			return nil, fmt.Errorf("set form description: %w", err)
		}
		if updated.Form != nil {
			return summarize(updated.Form), nil
		}
	}
	return summarize(created), nil
}

// GetForm returns a form with its items.
func (s *Service) GetForm(ctx context.Context, formID string) (*forms.Form, error) {
	if formID == "" {
		return nil, fmt.Errorf("formId required")
	}
	f, err := s.forms.Get(formID).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("get form: %w", err)
	}
	return f, nil
}

// AppendItems adds items at the end of the form in a single BatchUpdate call.
func (s *Service) AppendItems(ctx context.Context, formID string, items []*forms.Item) (*forms.Form, error) {
	if formID == "" {
		return nil, fmt.Errorf("formId required")
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("no items to add")
	}

	current, err := s.GetForm(ctx, formID)
	if err != nil {
		return nil, err
	}
	base := int64(len(current.Items))

	requests := make([]*forms.Request, 0, len(items))
	for i, item := range items {
		requests = append(requests, &forms.Request{
			CreateItem: &forms.CreateItemRequest{
				Item: item,
				// Index 0 must be force-sent: omitempty would drop it and
				// the API rejects a missing location index.
				Location: &forms.Location{
					Index:           base + int64(i),
					ForceSendFields: []string{"Index"},
				},
			},
		})
	}

	updated, err := s.forms.BatchUpdate(formID, &forms.BatchUpdateFormRequest{
		Requests:              requests,
		IncludeFormInResponse: true,
	}).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("batch update form: %w", err)
	}
	if updated.Form != nil {
		return updated.Form, nil
	}
	return s.GetForm(ctx, formID)
}

// ListResponses returns response metadata for a form.
func (s *Service) ListResponses(ctx context.Context, formID string, pageSize int64) ([]*ResponseSummary, error) {
	if formID == "" {
		return nil, fmt.Errorf("formId required")
	}
	if pageSize <= 0 || pageSize > 5000 {
		pageSize = 100
	}

	res, err := s.responses.List(formID).PageSize(pageSize).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("list responses: %w", err)
	}

	summaries := make([]*ResponseSummary, 0, len(res.Responses))
	for _, r := range res.Responses {
		summaries = append(summaries, &ResponseSummary{
			ResponseID: r.ResponseId,
			Responder:  r.RespondentEmail,
			Submitted:  r.LastSubmittedTime,
		})
	}
	return summaries, nil
}

func summarize(f *forms.Form) *FormSummary {
	s := &FormSummary{FormID: f.FormId, ResponderURI: f.ResponderUri}
	if f.Info != nil {
		s.Title = f.Info.Title
		s.Description = f.Info.Description
	}
	s.ItemCount = len(f.Items)
	return s
}

// NewSectionItem builds a section header (title + description, no question).
func NewSectionItem(title, description string) *forms.Item {
	return &forms.Item{
		Title:       title,
		Description: description,
		TextItem:    &forms.TextItem{},
	}
}

// NewTextQuestion builds a free-text question item.
func NewTextQuestion(title, description string, paragraph, required bool) *forms.Item {
	return &forms.Item{
		Title:       title,
		Description: description,
		QuestionItem: &forms.QuestionItem{
			Question: &forms.Question{
				Required:     required,
				TextQuestion: &forms.TextQuestion{Paragraph: paragraph},
			},
		},
	}
}

// NewChoiceQuestion builds a RADIO/CHECKBOX/DROP_DOWN question item.
func NewChoiceQuestion(title, description string, required bool, choiceType string, options []string) *forms.Item {
	if choiceType == "" {
		choiceType = "RADIO"
	}
	opts := make([]*forms.Option, 0, len(options))
	for _, o := range options {
		if o == "" {
			continue
		}
		opts = append(opts, &forms.Option{Value: o})
	}
	return &forms.Item{
		Title:       title,
		Description: description,
		QuestionItem: &forms.QuestionItem{
			Question: &forms.Question{
				Required: required,
				ChoiceQuestion: &forms.ChoiceQuestion{
					Type:    choiceType,
					Options: opts,
				},
			},
		},
	}
}
