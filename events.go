package notideus

import "context"

// SendEventParams is the custom event payload for POST /v1/events/send. Set
// exactly one of Email or ContactID to identify the automation recipient.
type SendEventParams struct {
	Event          string         `json:"event"`
	Email          string         `json:"email,omitempty"`
	ContactID      string         `json:"contact_id,omitempty"`
	Payload        map[string]any `json:"payload,omitempty"`
	IdempotencyKey *string        `json:"idempotency_key,omitempty"`
}

// AutomationEvent is an accepted custom event.
type AutomationEvent struct {
	ID        string         `json:"id"`
	Event     string         `json:"event"`
	Email     string         `json:"email"`
	ContactID *string        `json:"contact_id"`
	Payload   map[string]any `json:"payload"`
	Source    string         `json:"source"`
	CreatedAt string         `json:"created_at"`
}

// SendEventResult reports whether this call created the event or replayed it.
type SendEventResult struct {
	Event   AutomationEvent `json:"event"`
	Created bool            `json:"created"`
}

// EventsService sends custom events that start matching automations.
type EventsService struct{ client *Client }

// Send accepts one event for asynchronous automation dispatch.
func (s *EventsService) Send(ctx context.Context, params SendEventParams) (*SendEventResult, error) {
	var result SendEventResult
	err := s.client.do(ctx, apiRequest{method: "POST", path: "/v1/events/send", body: params}, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
