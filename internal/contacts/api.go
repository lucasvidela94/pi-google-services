// Package contacts wraps the Google People API v1.
package contacts

import (
	"context"
	"fmt"
	"strings"

	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	"google.golang.org/api/people/v1"
)

// Service wraps the Google People API client.
type Service struct {
	svc *people.Service
}

// ContactSummary is a lightweight contact representation.
type ContactSummary struct {
	ResourceName string   `json:"resourceName"`
	Name         string   `json:"name"`
	Emails       []string `json:"emails,omitempty"`
	Phones       []string `json:"phones,omitempty"`
	Photo        string   `json:"photo,omitempty"`
}

// New creates a Service from an OAuth2 token source.
func New(ctx context.Context, ts oauth2.TokenSource) (*Service, error) {
	svc, err := people.NewService(ctx, option.WithTokenSource(ts))
	if err != nil {
		return nil, fmt.Errorf("create people service: %w", err)
	}
	return &Service{svc: svc}, nil
}

// SearchContacts searches contacts by query string.
func (s *Service) SearchContacts(ctx context.Context, query string, pageSize int64) ([]*ContactSummary, error) {
	if pageSize <= 0 || pageSize > 30 {
		pageSize = 10
	}

	res, err := s.svc.People.SearchContacts().Query(query).PageSize(pageSize).
		ReadMask("names,emailAddresses,phoneNumbers,photos").
		Do()
	if err != nil {
		return nil, fmt.Errorf("search contacts: %w", err)
	}

	summaries := make([]*ContactSummary, 0, len(res.Results))
	for _, r := range res.Results {
		p := r.Person
		s := &ContactSummary{ResourceName: p.ResourceName}
		if len(p.Names) > 0 {
			s.Name = p.Names[0].DisplayName
		}
		for _, e := range p.EmailAddresses {
			s.Emails = append(s.Emails, e.Value)
		}
		for _, ph := range p.PhoneNumbers {
			s.Phones = append(s.Phones, fmt.Sprintf("%s (%s)", ph.Value, ph.Type))
		}
		if len(p.Photos) > 0 {
			s.Photo = p.Photos[0].Url
		}
		summaries = append(summaries, s)
	}
	return summaries, nil
}

// GetContact retrieves full details for a contact by resource name.
func (s *Service) GetContact(ctx context.Context, resourceName string) (*ContactSummary, error) {
	p, err := s.svc.People.Get(resourceName).
		PersonFields("names,emailAddresses,phoneNumbers,photos,addresses,organizations,biographies,birthdays").
		Do()
	if err != nil {
		return nil, fmt.Errorf("get contact: %w", err)
	}

	cs := &ContactSummary{ResourceName: p.ResourceName}
	if len(p.Names) > 0 {
		cs.Name = p.Names[0].DisplayName
	}
	for _, e := range p.EmailAddresses {
		cs.Emails = append(cs.Emails, e.Value)
	}
	for _, ph := range p.PhoneNumbers {
		cs.Phones = append(cs.Phones, fmt.Sprintf("%s (%s)", ph.Value, ph.Type))
	}
	if len(p.Photos) > 0 {
		cs.Photo = p.Photos[0].Url
	}
	return cs, nil
}

// CreateContact creates a new contact with name, email, and phone.
func (s *Service) CreateContact(ctx context.Context, name, email, phone string) (*ContactSummary, error) {
	person := &people.Person{}

	if name != "" {
		person.Names = []*people.Name{{GivenName: name}}
	}
	if email != "" {
		person.EmailAddresses = []*people.EmailAddress{{Value: email}}
	}
	if phone != "" {
		person.PhoneNumbers = []*people.PhoneNumber{{Value: phone}}
	}

	created, err := s.svc.People.CreateContact(person).Do()
	if err != nil {
		return nil, fmt.Errorf("create contact: %w", err)
	}

	cs := &ContactSummary{ResourceName: created.ResourceName}
	if len(created.Names) > 0 {
		cs.Name = created.Names[0].DisplayName
	}
	for _, e := range created.EmailAddresses {
		cs.Emails = append(cs.Emails, e.Value)
	}
	for _, ph := range created.PhoneNumbers {
		cs.Phones = append(cs.Phones, fmt.Sprintf("%s (%s)", ph.Value, ph.Type))
	}
	return cs, nil
}

// ListConnections returns recent contacts (first 20).
func (s *Service) ListConnections(ctx context.Context, pageSize int64) ([]*ContactSummary, error) {
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	res, err := s.svc.People.Connections.List("people/me").
		PageSize(pageSize).
		PersonFields("names,emailAddresses,phoneNumbers,photos").
		SortOrder("LAST_MODIFIED_DESCENDING").
		Do()
	if err != nil {
		return nil, fmt.Errorf("list connections: %w", err)
	}

	summaries := make([]*ContactSummary, 0, len(res.Connections))
	for _, p := range res.Connections {
		cs := &ContactSummary{ResourceName: p.ResourceName}
		if len(p.Names) > 0 {
			cs.Name = p.Names[0].DisplayName
		}
		for _, e := range p.EmailAddresses {
			cs.Emails = append(cs.Emails, e.Value)
		}
		for _, ph := range p.PhoneNumbers {
			t := ph.Type
			if t == "" {
				t = "other"
			}
			cs.Phones = append(cs.Phones, fmt.Sprintf("%s (%s)", ph.Value, t))
		}
		if len(p.Photos) > 0 {
			cs.Photo = p.Photos[0].Url
		}
		summaries = append(summaries, cs)
	}
	return summaries, nil
}

// FormatContacts formats contacts for display.
func FormatContacts(contacts []*ContactSummary) string {
	var b strings.Builder
	for i, c := range contacts {
		b.WriteString(fmt.Sprintf("%d. 👤 %s\n", i+1, c.Name))
		for _, e := range c.Emails {
			b.WriteString(fmt.Sprintf("   📧 %s\n", e))
		}
		for _, ph := range c.Phones {
			b.WriteString(fmt.Sprintf("   📞 %s\n", ph))
		}
		if c.ResourceName != "" {
			b.WriteString(fmt.Sprintf("   🔖 %s\n", c.ResourceName))
		}
	}
	return b.String()
}
