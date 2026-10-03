package notideus

import (
	"context"
	"net/url"
)

// UnsubscribeInfo is the hosted-preferences context behind one unsubscribe
// token — field-for-field parity with GET /v1/unsubscribe/info in
// notideus-api/docs/public-api.md. Status is the effective contact state.
type UnsubscribeInfo struct {
	Email    string             `json:"email"`
	Status   string             `json:"status"`
	TeamName string             `json:"team_name"`
	Topics   []UnsubscribeTopic `json:"topics"`
}

// UnsubscribeTopic is one preference topic on the hosted unsubscribe page.
// Subscribed is the effective state (false when the contact is fully
// unsubscribed).
type UnsubscribeTopic struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Subscribed  bool   `json:"subscribed"`
}

// UnsubscribeService backs the hosted unsubscribe/preferences page. All
// routes are unauthenticated — the per-message token is the credential — so
// every call suppresses the Authorization header.
type UnsubscribeService struct{ client *Client }

// Info fetches the preference context for one token
// (GET /v1/unsubscribe/info?t=<token>) → 200; 404 on unknown token.
func (s *UnsubscribeService) Info(ctx context.Context, token string) (*UnsubscribeInfo, error) {
	q := url.Values{"t": []string{token}}
	var info UnsubscribeInfo
	err := s.client.do(ctx, apiRequest{method: "GET", path: "/v1/unsubscribe/info", query: q, noAuth: true}, &info)
	if err != nil {
		return nil, err
	}
	return &info, nil
}

// Unsubscribe opts the contact out (POST /v1/unsubscribe) → 204, idempotent.
// TopicIDs, when given, unsubscribe just those topics; with none the whole
// contact is unsubscribed and topic_ids is omitted from the body.
func (s *UnsubscribeService) Unsubscribe(ctx context.Context, token string, topicIDs ...string) error {
	body := map[string]any{"token": token}
	if len(topicIDs) > 0 {
		body["topic_ids"] = topicIDs
	}
	return s.client.do(ctx, apiRequest{method: "POST", path: "/v1/unsubscribe", body: body, noAuth: true}, nil)
}

// Preferences sets the desired subscribed topic set
// (POST /v1/unsubscribe/preferences) → 204, idempotent; 400 when a topic does
// not belong to the contact's team. The topic list is always sent — an empty
// slice unsubscribes the whole contact.
func (s *UnsubscribeService) Preferences(ctx context.Context, token string, topicIDs []string) error {
	if topicIDs == nil {
		topicIDs = []string{}
	}
	body := map[string]any{"token": token, "topic_ids": topicIDs}
	return s.client.do(ctx, apiRequest{method: "POST", path: "/v1/unsubscribe/preferences", body: body, noAuth: true}, nil)
}
