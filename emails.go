package notideus

import "context"

// SendEmailParams is the single-send (and batch item) payload. Field-for-
// field parity with POST /v1/emails in notideus-api/docs/public-api.md.
type SendEmailParams struct {
	From           string            `json:"from"`
	To             []string          `json:"to"`
	Subject        string            `json:"subject"`
	HTML           string            `json:"html,omitempty"`
	Text           string            `json:"text,omitempty"`
	TemplateID     *string           `json:"template_id,omitempty"`
	Variables      map[string]string `json:"variables,omitempty"`
	Tags           []string          `json:"tags,omitempty"`
	Headers        map[string]string `json:"headers,omitempty"`
	ReplyTo        string            `json:"reply_to,omitempty"`
	IdempotencyKey *string           `json:"idempotency_key,omitempty"`
}

// Email is the stored email row returned by sends.
type Email struct {
	ID               string            `json:"id"`
	TeamID           string            `json:"team_id"`
	DomainID         string            `json:"domain_id"`
	TemplateID       *string           `json:"template_id"`
	FromAddress      string            `json:"from_address"`
	ToEmails         []string          `json:"to_emails"`
	Subject          string            `json:"subject"`
	HTML             string            `json:"html"`
	SizeKb           float64           `json:"size_kb"`
	Tags             []string          `json:"tags"`
	Status           string            `json:"status"`
	IdempotencyKey   *string           `json:"idempotency_key"`
	CreatedAt        string            `json:"created_at"`
	RequeuedAt       *string           `json:"requeued_at"`
	SesMessageID     *string           `json:"ses_message_id"`
	BroadcastID      *string           `json:"broadcast_id"`
	ContactID        *string           `json:"contact_id"`
	UnsubscribeToken *string           `json:"unsubscribe_token"`
	BodyText         string            `json:"body_text"`
	Headers          map[string]string `json:"headers"`
	ReplyTo          string            `json:"reply_to"`
}

// BatchItemError is the per-item failure of a batch call.
type BatchItemError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// BatchItemResult is one item's outcome: ID on success, Error on failure.
type BatchItemResult struct {
	Index int
	ID    string
	Error *BatchItemError
}

// EmailsService sends single emails and batches.
type EmailsService struct{ client *Client }

// Send enqueues one email (POST /v1/emails).
func (s *EmailsService) Send(ctx context.Context, params SendEmailParams) (*Email, error) {
	var email Email
	err := s.client.do(ctx, apiRequest{method: "POST", path: "/v1/emails", body: params}, &email)
	if err != nil {
		return nil, err
	}
	return &email, nil
}

// SendBatch enqueues up to 100 emails; per-item results are returned, never
// thrown. Only whole-batch failures (401/402/400) return an error.
func (s *EmailsService) SendBatch(ctx context.Context, emails []SendEmailParams) ([]BatchItemResult, error) {
	var res struct {
		Data []BatchItemResult `json:"data"`
	}
	err := s.client.do(ctx, apiRequest{method: "POST", path: "/v1/emails/batch", body: map[string]any{"emails": emails}}, &res)
	if err != nil {
		return nil, err
	}
	return res.Data, nil
}
