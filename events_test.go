package notideus

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEventsSend(t *testing.T) {
	var requestBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/events/send" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer nt_live_k" {
			t.Fatalf("authorization = %q", r.Header.Get("Authorization"))
		}
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatal(err)
		}
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"created":true,"event":{"id":"evt-1","event":"user.created","email":"ada@example.com","payload":{"first_name":"Ada"},"source":"api","created_at":"2026-11-13T10:00:00Z"}}`))
	}))
	defer srv.Close()
	client := New("nt_live_k", WithBaseURL(srv.URL))
	result, err := client.Events.Send(context.Background(), SendEventParams{
		Event: "user.created", Email: "ada@example.com", Payload: map[string]any{"first_name": "Ada"},
		IdempotencyKey: Ptr("signup-42"),
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if !result.Created || result.Event.ID != "evt-1" {
		t.Fatalf("result = %+v", result)
	}
	if requestBody["event"] != "user.created" || requestBody["email"] != "ada@example.com" || requestBody["idempotency_key"] != "signup-42" {
		t.Fatalf("body = %+v", requestBody)
	}
	payload, ok := requestBody["payload"].(map[string]any)
	if !ok || payload["first_name"] != "Ada" {
		t.Fatalf("payload = %+v", requestBody["payload"])
	}
}

func TestEventsSendToContactAndReplay(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"created":false,"event":{"id":"evt-1","event":"user.created","email":"ada@example.com","contact_id":"contact-1","source":"api","created_at":"2026-11-13T10:00:00Z"}}`))
	}))
	defer srv.Close()
	client := New("nt_live_k", WithBaseURL(srv.URL))
	result, err := client.Events.Send(context.Background(), SendEventParams{Event: "user.created", ContactID: "contact-1"})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if result.Created || result.Event.ContactID == nil || *result.Event.ContactID != "contact-1" {
		t.Fatalf("result = %+v", result)
	}
}
