package notideus

import (
	"context"
	"net/url"
	"strconv"
)

// InboundRoute is the inbound route row — field-for-field parity with the
// route response shape in notideus-api/docs/public-api.md (## Inbound email).
// A route selects which mail enters the pipeline: Store keeps the message in
// the team inbox, WebhookEndpointID pushes inbound.received to that endpoint
// (the route is the opt-in — no subscription needed).
type InboundRoute struct {
	ID                string  `json:"id"`
	DomainID          string  `json:"domain_id"`
	Recipient         string  `json:"recipient"`
	Name              string  `json:"name"`
	WebhookEndpointID *string `json:"webhook_endpoint_id"`
	Store             bool    `json:"store"`
	Status            string  `json:"status"`
	CreatedAt         string  `json:"created_at"`
	UpdatedAt         string  `json:"updated_at"`
}

// InboundRoutesPage is the GET /v1/inbound/routes response: the team's routes
// plus deployment meta (simulate_enabled, retention_days — plan-dependent).
type InboundRoutesPage struct {
	Data []InboundRoute `json:"data"`
	Meta struct {
		SimulateEnabled bool  `json:"simulate_enabled"`
		RetentionDays   int32 `json:"retention_days"`
	} `json:"meta"`
}

// CreateInboundRouteParams is the POST /v1/inbound/routes payload. Recipient
// is the local part or "*" for a domain catch-all. At least one of Store /
// WebhookEndpointID is required (400 otherwise). Creating the first route on
// a domain provisions the MX record and the SES receipt rule.
type CreateInboundRouteParams struct {
	DomainID          string  `json:"domain_id"`
	Recipient         string  `json:"recipient"`
	Name              *string `json:"name,omitempty"`
	Store             bool    `json:"store"`
	WebhookEndpointID *string `json:"webhook_endpoint_id,omitempty"`
}

// UpdateInboundRouteParams is the PATCH /v1/inbound/routes/:id payload — it
// has NO DomainID and NO Status (the domain is fixed at create; status is
// managed elsewhere).
type UpdateInboundRouteParams struct {
	Recipient         string  `json:"recipient"`
	Name              *string `json:"name,omitempty"`
	Store             bool    `json:"store"`
	WebhookEndpointID *string `json:"webhook_endpoint_id,omitempty"`
}

// InboundRoutesService manages inbound routes on the team's verified domains:
// list, create, get, update, delete.
type InboundRoutesService struct {
	client *Client
}

// List lists the team's inbound routes (GET /v1/inbound/routes, scope
// inbound:read) with deployment meta.
func (s *InboundRoutesService) List(ctx context.Context) (*InboundRoutesPage, error) {
	var page InboundRoutesPage
	err := s.client.do(ctx, apiRequest{method: "GET", path: "/v1/inbound/routes"}, &page)
	if err != nil {
		return nil, err
	}
	return &page, nil
}

// Create adds one inbound route (POST /v1/inbound/routes, scope inbound:write)
// → 201.
func (s *InboundRoutesService) Create(ctx context.Context, params CreateInboundRouteParams) (*InboundRoute, error) {
	var res struct {
		Route InboundRoute `json:"route"`
	}
	err := s.client.do(ctx, apiRequest{method: "POST", path: "/v1/inbound/routes", body: params}, &res)
	if err != nil {
		return nil, err
	}
	return &res.Route, nil
}

// Get fetches one inbound route by id (GET /v1/inbound/routes/:id, scope
// inbound:read; 404 inbound route not found).
func (s *InboundRoutesService) Get(ctx context.Context, id string) (*InboundRoute, error) {
	var res struct {
		Route InboundRoute `json:"route"`
	}
	err := s.client.do(ctx, apiRequest{method: "GET", path: inboundRoutePath(id)}, &res)
	if err != nil {
		return nil, err
	}
	return &res.Route, nil
}

// Update replaces an inbound route's recipient, name, store flag, and webhook
// endpoint (PATCH /v1/inbound/routes/:id, scope inbound:write).
func (s *InboundRoutesService) Update(ctx context.Context, id string, params UpdateInboundRouteParams) (*InboundRoute, error) {
	var res struct {
		Route InboundRoute `json:"route"`
	}
	err := s.client.do(ctx, apiRequest{method: "PATCH", path: inboundRoutePath(id), body: params}, &res)
	if err != nil {
		return nil, err
	}
	return &res.Route, nil
}

// Delete removes one inbound route by id (DELETE /v1/inbound/routes/:id,
// scope inbound:write) → 204.
func (s *InboundRoutesService) Delete(ctx context.Context, id string) error {
	return s.client.do(ctx, apiRequest{method: "DELETE", path: inboundRoutePath(id)}, nil)
}

// InboundAttachment is one entry of an inbound message's attachments list;
// the 0-based index in this list is the :n of the attachment download
// endpoint. Key is the backing object key (informational).
type InboundAttachment struct {
	Name        string `json:"name"`
	ContentType string `json:"content_type"`
	SizeKb      int32  `json:"size_kb"`
	Key         string `json:"key"`
}

// InboundMessage is the inbound message row — field-for-field parity with the
// message response shape in notideus-api/docs/public-api.md (headers and the
// attachments list included).
type InboundMessage struct {
	ID          string              `json:"id"`
	RouteID     *string             `json:"route_id"`
	FromAddr    string              `json:"from_addr"`
	FromName    string              `json:"from_name"`
	ToAddr      string              `json:"to_addr"`
	Subject     string              `json:"subject"`
	Text        string              `json:"text"`
	HTML        string              `json:"html"`
	Headers     map[string]string   `json:"headers"` // keys are lowercased server-side
	Attachments []InboundAttachment `json:"attachments"`
	SpamVerdict *string             `json:"spam_verdict"`
	SizeKb      int32               `json:"size_kb"`
	Status      string              `json:"status"`
	Error       *string             `json:"error"`
	ReceivedAt  string              `json:"received_at"`
}

// InboundMessageListParams filters the team inbox; nil fields are skipped.
type InboundMessageListParams struct {
	Q       *string
	RouteID *string
	Cursor  *string
	Limit   *int32
}

// InboundMessagesPage is one page of the team inbox.
type InboundMessagesPage struct {
	Data       []InboundMessage `json:"data"`
	NextCursor *string          `json:"next_cursor"`
}

// InboundMessagesService is the team inbox: list, get, raw MIME download, and
// per-attachment downloads.
type InboundMessagesService struct {
	client *Client
}

// List lists inbound messages, newest first, filtered by the non-nil params
// (GET /v1/inbound/messages, scope inbound:read).
func (s *InboundMessagesService) List(ctx context.Context, params InboundMessageListParams) (*InboundMessagesPage, error) {
	q := url.Values{}
	if params.Q != nil {
		q.Set("q", *params.Q)
	}
	if params.RouteID != nil {
		q.Set("route_id", *params.RouteID)
	}
	if params.Cursor != nil {
		q.Set("cursor", *params.Cursor)
	}
	if params.Limit != nil {
		q.Set("limit", strconv.Itoa(int(*params.Limit)))
	}
	var page InboundMessagesPage
	err := s.client.do(ctx, apiRequest{method: "GET", path: "/v1/inbound/messages", query: q}, &page)
	if err != nil {
		return nil, err
	}
	return &page, nil
}

// Get fetches one inbound message by id with headers and the attachments list
// (GET /v1/inbound/messages/:id, scope inbound:read; 404 inbound message not
// found).
func (s *InboundMessagesService) Get(ctx context.Context, id string) (*InboundMessage, error) {
	var res struct {
		Message InboundMessage `json:"message"`
	}
	err := s.client.do(ctx, apiRequest{method: "GET", path: inboundMessagePath(id)}, &res)
	if err != nil {
		return nil, err
	}
	return &res.Message, nil
}

// Raw downloads the raw MIME of one message (GET /v1/inbound/messages/:id/raw,
// scope inbound:read; message/rfc822). Returns the bytes, the
// Content-Disposition filename ("" when absent), and the Content-Type. No
// retries — callers can re-invoke.
func (s *InboundMessagesService) Raw(ctx context.Context, id string) ([]byte, string, string, error) {
	return s.client.doBinary(ctx, apiRequest{method: "GET", path: inboundMessagePath(id) + "/raw"})
}

// Attachment downloads one attachment of a message by its 0-based index (GET
// /v1/inbound/messages/:id/attachments/:n, scope inbound:read). Returns the
// bytes, the Content-Disposition filename, and the Content-Type. No retries —
// callers can re-invoke.
func (s *InboundMessagesService) Attachment(ctx context.Context, messageID string, index int) ([]byte, string, string, error) {
	return s.client.doBinary(ctx, apiRequest{method: "GET", path: inboundMessagePath(messageID) + "/attachments/" + strconv.Itoa(index)})
}

// InboundService groups the inbound email surface: Routes manages receiving
// routes on the team's verified domains; Messages is the team inbox.
type InboundService struct {
	client   *Client
	Routes   *InboundRoutesService
	Messages *InboundMessagesService
}

func inboundRoutePath(id string) string {
	return "/v1/inbound/routes/" + url.PathEscape(id)
}

func inboundMessagePath(id string) string {
	return "/v1/inbound/messages/" + url.PathEscape(id)
}
