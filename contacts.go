package notideus

import (
	"context"
	"net/url"
	"strconv"
)

// Contact is the contact row returned by the contacts endpoints — field-for-
// field parity with the contact response shape in notideus-api/docs/public-api.md.
type Contact struct {
	ID                 string            `json:"id"`
	Email              string            `json:"email"`
	Status             string            `json:"status"`
	Properties         map[string]string `json:"properties"`
	SubscribedTopicIDs []string          `json:"subscribed_topic_ids"`
	CreatedAt          string            `json:"created_at"`
	UpdatedAt          string            `json:"updated_at"`
}

// ListContactsParams filters the contacts collection; nil fields are skipped.
type ListContactsParams struct {
	Cursor *string
	Email  *string
	Status *string
	Limit  *int
}

// ContactsPage is one page of the contacts collection.
type ContactsPage struct {
	Data       []Contact `json:"data"`
	NextCursor *string   `json:"next_cursor"`
}

// CreateContactParams is the POST /v1/contacts payload.
type CreateContactParams struct {
	Email      string            `json:"email"`
	Properties map[string]string `json:"properties,omitempty"`
	TopicIDs   []string          `json:"topic_ids,omitempty"`
}

// UpdateContactParams is the PATCH /v1/contacts/:email payload. Properties
// replaces the existing map; TopicIDs, when present, replaces subscriptions.
type UpdateContactParams struct {
	Properties map[string]string `json:"properties,omitempty"`
	TopicIDs   []string          `json:"topic_ids,omitempty"`
}

// BulkContactInput is one upsert item for Bulk (up to 1000 per call).
type BulkContactInput struct {
	Email      string            `json:"email"`
	Properties map[string]string `json:"properties,omitempty"`
}

// ContactsService manages contacts: list, create, get, update, delete, bulk
// upsert, and subscribe-status flips.
type ContactsService struct{ client *Client }

// List lists contacts, newest first, filtered by the non-nil params.
func (s *ContactsService) List(ctx context.Context, params ListContactsParams) (*ContactsPage, error) {
	q := url.Values{}
	if params.Cursor != nil {
		q.Set("cursor", *params.Cursor)
	}
	if params.Email != nil {
		q.Set("email", *params.Email)
	}
	if params.Status != nil {
		q.Set("status", *params.Status)
	}
	if params.Limit != nil {
		q.Set("limit", strconv.Itoa(*params.Limit))
	}
	var page ContactsPage
	err := s.client.do(ctx, apiRequest{method: "GET", path: "/v1/contacts", query: q}, &page)
	if err != nil {
		return nil, err
	}
	return &page, nil
}

// Create adds one contact (POST /v1/contacts) → 201; 409 contact_exists on a
// duplicate address. The new contact starts out subscribed.
func (s *ContactsService) Create(ctx context.Context, params CreateContactParams) (*Contact, error) {
	var contact Contact
	err := s.client.do(ctx, apiRequest{method: "POST", path: "/v1/contacts", body: params}, &contact)
	if err != nil {
		return nil, err
	}
	return &contact, nil
}

// Get fetches one contact by email address (404 contact_not_found).
func (s *ContactsService) Get(ctx context.Context, email string) (*Contact, error) {
	var contact Contact
	err := s.client.do(ctx, apiRequest{method: "GET", path: contactPath(email)}, &contact)
	if err != nil {
		return nil, err
	}
	return &contact, nil
}

// Update replaces a contact's properties and, when TopicIDs is set, its topic
// subscriptions.
func (s *ContactsService) Update(ctx context.Context, email string, params UpdateContactParams) (*Contact, error) {
	var contact Contact
	err := s.client.do(ctx, apiRequest{method: "PATCH", path: contactPath(email), body: params}, &contact)
	if err != nil {
		return nil, err
	}
	return &contact, nil
}

// Delete removes one contact by email address → 204.
func (s *ContactsService) Delete(ctx context.Context, email string) error {
	return s.client.do(ctx, apiRequest{method: "DELETE", path: contactPath(email)}, nil)
}

// Bulk upserts up to 1000 contacts, matching on (team, email); per-item
// results are returned, never thrown. Only whole-batch failures (400
// batch_too_large, 401, 402 quota_exceeded) return an error.
func (s *ContactsService) Bulk(ctx context.Context, contacts []BulkContactInput) ([]BatchItemResult, error) {
	var res struct {
		Data []BatchItemResult `json:"data"`
	}
	err := s.client.do(ctx, apiRequest{method: "POST", path: "/v1/contacts/bulk", body: map[string]any{"contacts": contacts}}, &res)
	if err != nil {
		return nil, err
	}
	return res.Data, nil
}

// Unsubscribe records an unsubscribe event with source api; idempotent —
// repeated calls are no-ops.
func (s *ContactsService) Unsubscribe(ctx context.Context, email string) (*Contact, error) {
	var contact Contact
	err := s.client.do(ctx, apiRequest{method: "POST", path: contactPath(email) + "/unsubscribe"}, &contact)
	if err != nil {
		return nil, err
	}
	return &contact, nil
}

// Resubscribe flips an unsubscribed contact back to subscribed. Suppressed
// (bounced/complained) contacts fail with 409 contact_suppressed.
func (s *ContactsService) Resubscribe(ctx context.Context, email string) (*Contact, error) {
	var contact Contact
	err := s.client.do(ctx, apiRequest{method: "POST", path: contactPath(email) + "/resubscribe"}, &contact)
	if err != nil {
		return nil, err
	}
	return &contact, nil
}

func contactPath(email string) string {
	return "/v1/contacts/" + url.PathEscape(email)
}
