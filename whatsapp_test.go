package notideus

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWhatsAppSendReturnsOneEntryPerRecipient(t *testing.T) {
	var path, method string
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path, method = r.URL.Path, r.Method
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &body)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"data":[{"id":"9c2e","to":"+33612345678","status":"queued"},` +
			`{"id":"9c2f","to":"+1555123456","status":"queued"}]}`))
	}))
	defer srv.Close()

	c := New("nt_live_k", WithBaseURL(srv.URL))
	messages, err := c.WhatsApp.Messages.Send(context.Background(), SendWhatsAppMessageParams{
		TemplateID:     "8f1c",
		To:             []string{"+33612345678", "+1555123456"},
		Parameters:     map[string]string{"body:1": "Jane"},
		IdempotencyKey: Ptr("req-123"),
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("len = %d", len(messages))
	}
	if messages[0].ID != "9c2e" || messages[0].To != "+33612345678" || messages[0].Status != "queued" {
		t.Fatalf("messages[0] = %+v", messages[0])
	}
	if path != "/v1/whatsapp/messages" || method != "POST" {
		t.Fatalf("%s %s", method, path)
	}
	if body["template_id"] != "8f1c" || body["idempotency_key"] != "req-123" {
		t.Fatalf("body = %v", body)
	}
	params, ok := body["parameters"].(map[string]any)
	if !ok || params["body:1"] != "Jane" {
		t.Fatalf("parameters = %v", body["parameters"])
	}
}

func TestWhatsAppSendBatchReturnsPerItemResults(t *testing.T) {
	var path string
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &body)
		_, _ = w.Write([]byte(`{"data":[{"index":0,"id":"9c2e"},{"index":1,"error":{"code":"invalid_phone",` +
			`"message":"to[0]: invalid_phone: +abc is not a valid E.164 phone number"}}]}`))
	}))
	defer srv.Close()

	c := New("nt_live_k", WithBaseURL(srv.URL))
	results, err := c.WhatsApp.Messages.SendBatch(context.Background(), []SendWhatsAppMessageParams{
		{TemplateID: "8f1c", To: []string{"+33612345678"}, Parameters: map[string]string{"body:1": "Jane"}},
		{TemplateID: "8f1c", To: []string{"+abc"}},
	})
	if err != nil {
		t.Fatalf("batch: %v", err)
	}
	if len(results) != 2 || results[0].ID != "9c2e" || results[0].Error != nil {
		t.Fatalf("results[0] = %+v", results[0])
	}
	if results[1].Error == nil || results[1].Error.Code != "invalid_phone" {
		t.Fatalf("results[1] = %+v", results[1])
	}
	if path != "/v1/whatsapp/messages/batch" {
		t.Fatalf("path = %s", path)
	}
	if msgs, ok := body["messages"].([]any); !ok || len(msgs) != 2 {
		t.Fatalf("messages = %v", body["messages"])
	}
}
