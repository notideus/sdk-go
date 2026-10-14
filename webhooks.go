package notideus

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
)

// WebhookEndpointStatus is a webhook endpoint's enablement state; `active`
// endpoints receive deliveries, `disabled` ones do not.
type WebhookEndpointStatus string

const (
	WebhookEndpointActive   WebhookEndpointStatus = "active"
	WebhookEndpointDisabled WebhookEndpointStatus = "disabled"
)

// WebhookEndpoint is the webhook endpoint row returned by the webhooks
// endpoints — field-for-field parity with the endpoint response shape in
// notideus-api/docs/public-api.md (## Webhooks). SecretPrefix is a display
// prefix only; the full whsec_… secret is shown once at Create.
type WebhookEndpoint struct {
	ID           string                `json:"id"`
	Name         string                `json:"name"`
	URL          string                `json:"url"`
	SecretPrefix string                `json:"secret_prefix"`
	Events       []string              `json:"events"`
	Status       WebhookEndpointStatus `json:"status"`
	CreatedAt    string                `json:"created_at"`
	UpdatedAt    string                `json:"updated_at"`
}

// WebhookCreated is the POST /v1/webhooks response: the new endpoint plus its
// signing secret. The whsec_… secret is shown once — store it; only a
// display prefix is kept server-side.
type WebhookCreated struct {
	Endpoint WebhookEndpoint `json:"endpoint"`
	Secret   string          `json:"secret"`
}

// WebhookEvent is one entry of the event catalog (GET
// /v1/webhooks/meta/events): an event type, its description, and a sample
// payload (drives the dashboard picker).
type WebhookEvent struct {
	Type        string          `json:"type"`
	Description string          `json:"description"`
	Sample      json.RawMessage `json:"sample"`
}

// WebhookListParams filters the webhooks collection; nil fields are skipped.
type WebhookListParams struct {
	Cursor *string
	Limit  *int32
}

// WebhooksPage is one page of the webhooks collection.
type WebhooksPage struct {
	Data       []WebhookEndpoint `json:"data"`
	NextCursor *string           `json:"next_cursor"`
}

// CreateWebhookParams is the POST /v1/webhooks payload. URL must be https.
type CreateWebhookParams struct {
	Name   string   `json:"name"`
	URL    string   `json:"url"`
	Events []string `json:"events"`
}

// UpdateWebhookParams is the PATCH /v1/webhooks/:id payload; all fields are
// required (the API rejects partial updates).
type UpdateWebhookParams struct {
	Name   string                `json:"name"`
	URL    string                `json:"url"`
	Events []string              `json:"events"`
	Status WebhookEndpointStatus `json:"status"`
}

// WebhooksService manages webhook endpoints: list, create, get, update,
// delete, test delivery, and the event catalog. Deliveries is the
// per-endpoint delivery log sub-service.
type WebhooksService struct {
	client     *Client
	Deliveries *WebhookDeliveriesService
}

// List lists webhook endpoints (GET /v1/webhooks, scope webhooks:read),
// newest first, filtered by the non-nil params.
func (s *WebhooksService) List(ctx context.Context, params WebhookListParams) (*WebhooksPage, error) {
	q := url.Values{}
	if params.Cursor != nil {
		q.Set("cursor", *params.Cursor)
	}
	if params.Limit != nil {
		q.Set("limit", strconv.Itoa(int(*params.Limit)))
	}
	var page WebhooksPage
	err := s.client.do(ctx, apiRequest{method: "GET", path: "/v1/webhooks", query: q}, &page)
	if err != nil {
		return nil, err
	}
	return &page, nil
}

// Create registers one webhook endpoint (POST /v1/webhooks, scope
// webhooks:write) → 201 {endpoint, secret}. The secret is shown once.
func (s *WebhooksService) Create(ctx context.Context, params CreateWebhookParams) (*WebhookCreated, error) {
	var created WebhookCreated
	err := s.client.do(ctx, apiRequest{method: "POST", path: "/v1/webhooks", body: params}, &created)
	if err != nil {
		return nil, err
	}
	return &created, nil
}

// Get fetches one webhook endpoint by id (GET /v1/webhooks/:id, scope
// webhooks:read; 404 webhook endpoint not found).
func (s *WebhooksService) Get(ctx context.Context, id string) (*WebhookEndpoint, error) {
	var res struct {
		Endpoint WebhookEndpoint `json:"endpoint"`
	}
	err := s.client.do(ctx, apiRequest{method: "GET", path: webhookPath(id)}, &res)
	if err != nil {
		return nil, err
	}
	return &res.Endpoint, nil
}

// Update replaces a webhook endpoint's name, URL, events, and status (PATCH
// /v1/webhooks/:id, scope webhooks:write) — all fields are required.
func (s *WebhooksService) Update(ctx context.Context, id string, params UpdateWebhookParams) (*WebhookEndpoint, error) {
	var res struct {
		Endpoint WebhookEndpoint `json:"endpoint"`
	}
	err := s.client.do(ctx, apiRequest{method: "PATCH", path: webhookPath(id), body: params}, &res)
	if err != nil {
		return nil, err
	}
	return &res.Endpoint, nil
}

// Delete removes one webhook endpoint by id (DELETE /v1/webhooks/:id, scope
// webhooks:write) → 204.
func (s *WebhooksService) Delete(ctx context.Context, id string) error {
	return s.client.do(ctx, apiRequest{method: "DELETE", path: webhookPath(id)}, nil)
}

// Test posts a sample payload to the endpoint right now (POST
// /v1/webhooks/:id/test, scope webhooks:write). eventID selects the event
// type; nil sends the endpoint's first subscribed event. Returns the
// delivery created by the test.
func (s *WebhooksService) Test(ctx context.Context, id string, eventID *string) (*WebhookDelivery, error) {
	var body any
	if eventID != nil {
		body = map[string]any{"event": *eventID}
	}
	var res struct {
		Delivery WebhookDelivery `json:"delivery"`
	}
	err := s.client.do(ctx, apiRequest{method: "POST", path: webhookPath(id) + "/test", body: body}, &res)
	if err != nil {
		return nil, err
	}
	return &res.Delivery, nil
}

// Events returns the event catalog with sample payloads (GET
// /v1/webhooks/meta/events, scope webhooks:read) — drives the dashboard
// picker. The documented event types include email.sent, email.delivered,
// email.bounced, email.complained, email.opened, email.clicked,
// broadcast.started, broadcast.completed, broadcast.failed, and
// inbound.received.
func (s *WebhooksService) Events(ctx context.Context) ([]WebhookEvent, error) {
	var res struct {
		Data []WebhookEvent `json:"data"`
	}
	err := s.client.do(ctx, apiRequest{method: "GET", path: "/v1/webhooks/meta/events"}, &res)
	if err != nil {
		return nil, err
	}
	return res.Data, nil
}

func webhookPath(id string) string {
	return "/v1/webhooks/" + url.PathEscape(id)
}
