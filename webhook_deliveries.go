package notideus

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
)

// WebhookDeliveryStatus is the delivery lifecycle state of a webhook
// delivery, as documented in notideus-api/docs/public-api.md (## Webhooks).
type WebhookDeliveryStatus string

const (
	WebhookDeliveryPending WebhookDeliveryStatus = "pending"
	WebhookDeliverySuccess WebhookDeliveryStatus = "success"
	WebhookDeliveryFailed  WebhookDeliveryStatus = "failed"
)

// WebhookDelivery is one webhook delivery row — field-for-field parity with
// the delivery response shape in notideus-api/docs/public-api.md.
type WebhookDelivery struct {
	ID             string                `json:"id"`
	EndpointID     string                `json:"endpoint_id"`
	EventType      string                `json:"event_type"`
	Payload        json.RawMessage       `json:"payload"`
	Status         WebhookDeliveryStatus `json:"status"`
	Attempts       int32                 `json:"attempts"`
	LastStatusCode *int32                `json:"last_status_code"`
	LastError      *string               `json:"last_error"`
	DeliveredAt    *string               `json:"delivered_at"`
	CreatedAt      string                `json:"created_at"`
}

// WebhookAttempt is one HTTP try of a webhook delivery (one row per try).
type WebhookAttempt struct {
	Attempt    int32   `json:"attempt"`
	StatusCode *int32  `json:"status_code"`
	Error      *string `json:"error"`
	DurationMs int32   `json:"duration_ms"`
	CreatedAt  string  `json:"created_at"`
}

// WebhookDeliveryResult is the GET /v1/webhook-deliveries/:id response: the
// delivery plus its per-try attempt log.
type WebhookDeliveryResult struct {
	Delivery WebhookDelivery  `json:"delivery"`
	Attempts []WebhookAttempt `json:"attempts"`
}

// WebhookDeliveryListParams filters the webhook-deliveries collection; nil
// fields are skipped.
type WebhookDeliveryListParams struct {
	EndpointID *string
	EventType  *string
	Status     *WebhookDeliveryStatus
	Cursor     *string
	Limit      *int32
}

// WebhookDeliveriesPage is one page of the webhook-deliveries collection.
type WebhookDeliveriesPage struct {
	Data       []WebhookDelivery `json:"data"`
	NextCursor *string           `json:"next_cursor"`
}

// WebhookDeliveriesService is the per-endpoint webhook delivery log: list,
// get (with attempts), and redeliver.
type WebhookDeliveriesService struct {
	client *Client
}

// List lists webhook deliveries, newest first, filtered by the non-nil
// params (GET /v1/webhook-deliveries, scope webhooks:read).
func (s *WebhookDeliveriesService) List(ctx context.Context, params WebhookDeliveryListParams) (*WebhookDeliveriesPage, error) {
	q := url.Values{}
	if params.EndpointID != nil {
		q.Set("endpoint_id", *params.EndpointID)
	}
	if params.EventType != nil {
		q.Set("event_type", *params.EventType)
	}
	if params.Status != nil {
		q.Set("status", string(*params.Status))
	}
	if params.Cursor != nil {
		q.Set("cursor", *params.Cursor)
	}
	if params.Limit != nil {
		q.Set("limit", strconv.Itoa(int(*params.Limit)))
	}
	var page WebhookDeliveriesPage
	err := s.client.do(ctx, apiRequest{method: "GET", path: "/v1/webhook-deliveries", query: q}, &page)
	if err != nil {
		return nil, err
	}
	return &page, nil
}

// Get fetches one webhook delivery by id with its attempt log (404
// webhook_delivery_not_found).
func (s *WebhookDeliveriesService) Get(ctx context.Context, id string) (*WebhookDeliveryResult, error) {
	var res WebhookDeliveryResult
	err := s.client.do(ctx, apiRequest{method: "GET", path: webhookDeliveryPath(id)}, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// Redeliver replays a failed delivery (POST …/redeliver, scope
// webhooks:write).
func (s *WebhookDeliveriesService) Redeliver(ctx context.Context, id string) (*WebhookDelivery, error) {
	var res struct {
		Delivery WebhookDelivery `json:"delivery"`
	}
	err := s.client.do(ctx, apiRequest{method: "POST", path: webhookDeliveryPath(id) + "/redeliver"}, &res)
	if err != nil {
		return nil, err
	}
	return &res.Delivery, nil
}

func webhookDeliveryPath(id string) string {
	return "/v1/webhook-deliveries/" + url.PathEscape(id)
}
