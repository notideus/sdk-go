package notideus

import "context"

// SendWhatsAppMessageParams is the single-send (and batch item) payload.
// Field-for-field parity with POST /v1/whatsapp/messages in
// notideus-api/docs/public-api.md. Parameters keys are scope:position slots
// (body:1, header:1, url:1, …).
type SendWhatsAppMessageParams struct {
	TemplateID     string            `json:"template_id"`
	To             []string          `json:"to"`
	Parameters     map[string]string `json:"parameters,omitempty"`
	IdempotencyKey *string           `json:"idempotency_key,omitempty"`
}

// WhatsAppMessage is one queued message row; a send returns one entry per
// recipient, in To order.
type WhatsAppMessage struct {
	ID     string `json:"id"`
	To     string `json:"to"`
	Status string `json:"status"`
}

// WhatsAppService groups the WhatsApp product surface.
type WhatsAppService struct {
	client   *Client
	Messages *WhatsAppMessagesService
}

// WhatsAppMessagesService sends single WhatsApp template messages and batches.
type WhatsAppMessagesService struct{ client *Client }

// Send enqueues one template message per recipient (POST /v1/whatsapp/messages)
// → 201, 200 on an idempotent replay. The response carries one entry per
// recipient.
func (s *WhatsAppMessagesService) Send(ctx context.Context, params SendWhatsAppMessageParams) ([]WhatsAppMessage, error) {
	var res struct {
		Data []WhatsAppMessage `json:"data"`
	}
	err := s.client.do(ctx, apiRequest{method: "POST", path: "/v1/whatsapp/messages", body: params}, &res)
	if err != nil {
		return nil, err
	}
	return res.Data, nil
}

// SendBatch enqueues up to 100 messages, one row per recipient; per-item
// results are returned, never thrown. Idempotency is matched per key +
// recipient phone. Only whole-batch failures (400 batch_too_large/invalid_
// params, 401, 402) return an error.
func (s *WhatsAppMessagesService) SendBatch(ctx context.Context, messages []SendWhatsAppMessageParams) ([]BatchItemResult, error) {
	var res struct {
		Data []BatchItemResult `json:"data"`
	}
	err := s.client.do(ctx, apiRequest{method: "POST", path: "/v1/whatsapp/messages/batch", body: map[string]any{"messages": messages}}, &res)
	if err != nil {
		return nil, err
	}
	return res.Data, nil
}
