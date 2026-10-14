package notideus

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

const testEndpoint = `{"id":"5d1e","name":"Primary","url":"https://example.com/hook",` +
	`"secret_prefix":"whsec_5d1eab","events":["email.delivered","email.bounced"],` +
	`"status":"active","created_at":"2026-10-12T09:00:00Z","updated_at":"2026-10-12T09:00:00Z"}`

const testDelivery = `{"id":"9c1f","endpoint_id":"5d1e","event_type":"email.delivered",` +
	`"payload":{"id":"em1"},"status":"success","attempts":1,"last_status_code":200,` +
	`"last_error":null,"delivered_at":"2026-10-12T09:00:01Z","created_at":"2026-10-12T09:00:00Z"}`

func TestWebhooksListSerializesFiltersAndMapsPage(t *testing.T) {
	var gotQuery url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		_, _ = w.Write([]byte(`{"data":[` + testEndpoint + `],"next_cursor":"20"}`))
	}))
	defer srv.Close()

	c := New("nt_live_k", WithBaseURL(srv.URL))
	page, err := c.Webhooks.List(context.Background(), WebhookListParams{
		Cursor: Ptr("0"),
		Limit:  Ptr(int32(20)),
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if gotQuery.Get("cursor") != "0" || gotQuery.Get("limit") != "20" {
		t.Fatalf("query = %v", gotQuery)
	}
	if len(page.Data) != 1 || page.Data[0].ID != "5d1e" || page.Data[0].Name != "Primary" {
		t.Fatalf("data = %+v", page.Data)
	}
	if page.Data[0].SecretPrefix != "whsec_5d1eab" || page.Data[0].Status != WebhookEndpointActive {
		t.Fatalf("endpoint = %+v", page.Data[0])
	}
	if len(page.Data[0].Events) != 2 || page.Data[0].Events[1] != "email.bounced" {
		t.Fatalf("events = %v", page.Data[0].Events)
	}
	if page.NextCursor == nil || *page.NextCursor != "20" {
		t.Fatalf("next_cursor = %v", page.NextCursor)
	}
}

func TestWebhooksCreateUnwrapsEndpointAndSecret(t *testing.T) {
	var method, path string
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.Path
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &body)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"endpoint":` + testEndpoint + `,"secret":"whsec_full_secret"}`))
	}))
	defer srv.Close()

	c := New("nt_live_k", WithBaseURL(srv.URL))
	created, err := c.Webhooks.Create(context.Background(), CreateWebhookParams{
		Name:   "Primary",
		URL:    "https://example.com/hook",
		Events: []string{"email.delivered"},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if method != "POST" || path != "/v1/webhooks" {
		t.Fatalf("%s %s", method, path)
	}
	if created.Endpoint.ID != "5d1e" || created.Endpoint.URL != "https://example.com/hook" {
		t.Fatalf("endpoint = %+v", created.Endpoint)
	}
	if created.Secret != "whsec_full_secret" {
		t.Fatalf("secret = %q", created.Secret)
	}
	if body["name"] != "Primary" || body["url"] != "https://example.com/hook" {
		t.Fatalf("body = %v", body)
	}
	if events, ok := body["events"].([]any); !ok || len(events) != 1 || events[0] != "email.delivered" {
		t.Fatalf("events = %v", body["events"])
	}
}

func TestWebhooksGetUnwrapsEndpoint(t *testing.T) {
	var method, path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.Path
		_, _ = w.Write([]byte(`{"endpoint":` + testEndpoint + `}`))
	}))
	defer srv.Close()

	c := New("nt_live_k", WithBaseURL(srv.URL))
	ep, err := c.Webhooks.Get(context.Background(), "5d1e")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if method != "GET" || path != "/v1/webhooks/5d1e" {
		t.Fatalf("%s %s", method, path)
	}
	if ep.ID != "5d1e" || ep.SecretPrefix != "whsec_5d1eab" {
		t.Fatalf("endpoint = %+v", ep)
	}
}

func TestWebhooksUpdateSendsFullBody(t *testing.T) {
	const id = "5d1e"
	var method, path string
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.Path
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &body)
		_, _ = w.Write([]byte(`{"endpoint":` + testEndpoint + `}`))
	}))
	defer srv.Close()

	c := New("nt_live_k", WithBaseURL(srv.URL))
	ep, err := c.Webhooks.Update(context.Background(), id, UpdateWebhookParams{
		Name:   "Primary v2",
		URL:    "https://example.com/hook2",
		Events: []string{"email.bounced"},
		Status: WebhookEndpointDisabled,
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if ep.ID != "5d1e" {
		t.Fatalf("endpoint = %+v", ep)
	}
	if method != "PATCH" || path != "/v1/webhooks/"+id {
		t.Fatalf("%s %s", method, path)
	}
	if body["name"] != "Primary v2" || body["url"] != "https://example.com/hook2" || body["status"] != "disabled" {
		t.Fatalf("body = %v", body)
	}
	if events, ok := body["events"].([]any); !ok || len(events) != 1 || events[0] != "email.bounced" {
		t.Fatalf("events = %v", body["events"])
	}
}

func TestWebhooksDeleteNoContent(t *testing.T) {
	const id = "5d1e"
	var method, path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := New("nt_live_k", WithBaseURL(srv.URL))
	if err := c.Webhooks.Delete(context.Background(), id); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if method != "DELETE" || path != "/v1/webhooks/"+id {
		t.Fatalf("%s %s", method, path)
	}
}

func TestWebhooksTestWithAndWithoutEvent(t *testing.T) {
	const id = "5d1e"
	var paths []string
	var bodies []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		b, _ := io.ReadAll(r.Body)
		var body map[string]any
		if len(b) > 0 {
			_ = json.Unmarshal(b, &body)
		}
		bodies = append(bodies, body)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"delivery":` + testDelivery + `}`))
	}))
	defer srv.Close()

	c := New("nt_live_k", WithBaseURL(srv.URL))
	d, err := c.Webhooks.Test(context.Background(), id, nil)
	if err != nil {
		t.Fatalf("test without event: %v", err)
	}
	if d.ID != "9c1f" || d.Status != WebhookDeliverySuccess {
		t.Fatalf("delivery = %+v", d)
	}
	d, err = c.Webhooks.Test(context.Background(), id, Ptr("email.bounced"))
	if err != nil {
		t.Fatalf("test with event: %v", err)
	}
	if d.EventType != "email.delivered" {
		t.Fatalf("delivery = %+v", d)
	}
	wantPath := "/v1/webhooks/" + id + "/test"
	if len(paths) != 2 || paths[0] != wantPath || paths[1] != wantPath {
		t.Fatalf("paths = %v", paths)
	}
	if bodies[0] != nil {
		t.Fatalf("nil event should send no body, got %v", bodies[0])
	}
	if bodies[1] == nil || bodies[1]["event"] != "email.bounced" {
		t.Fatalf("event body = %v", bodies[1])
	}
}

func TestWebhooksEventsUnwrapsCatalog(t *testing.T) {
	var method, path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.Path
		_, _ = w.Write([]byte(`{"data":[{"type":"email.delivered","description":"Email delivered",` +
			`"sample":{"id":"em1","event":"email.delivered"}}]}`))
	}))
	defer srv.Close()

	c := New("nt_live_k", WithBaseURL(srv.URL))
	events, err := c.Webhooks.Events(context.Background())
	if err != nil {
		t.Fatalf("events: %v", err)
	}
	if method != "GET" || path != "/v1/webhooks/meta/events" {
		t.Fatalf("%s %s", method, path)
	}
	if len(events) != 1 || events[0].Type != "email.delivered" || events[0].Description != "Email delivered" {
		t.Fatalf("events = %+v", events)
	}
	var sample map[string]any
	if err := json.Unmarshal(events[0].Sample, &sample); err != nil || sample["id"] != "em1" {
		t.Fatalf("sample = %s", events[0].Sample)
	}
}

func TestWebhookDeliveriesListSerializesFiltersAndMapsPage(t *testing.T) {
	var gotQuery url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		_, _ = w.Write([]byte(`{"data":[` + testDelivery + `],"next_cursor":"40"}`))
	}))
	defer srv.Close()

	c := New("nt_live_k", WithBaseURL(srv.URL))
	page, err := c.Webhooks.Deliveries.List(context.Background(), WebhookDeliveryListParams{
		EndpointID: Ptr("5d1e"),
		EventType:  Ptr("email.delivered"),
		Status:     Ptr(WebhookDeliveryFailed),
		Cursor:     Ptr("20"),
		Limit:      Ptr(int32(20)),
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if gotQuery.Get("endpoint_id") != "5d1e" || gotQuery.Get("event_type") != "email.delivered" ||
		gotQuery.Get("status") != "failed" || gotQuery.Get("cursor") != "20" || gotQuery.Get("limit") != "20" {
		t.Fatalf("query = %v", gotQuery)
	}
	if len(page.Data) != 1 || page.Data[0].ID != "9c1f" {
		t.Fatalf("data = %+v", page.Data)
	}
	if page.Data[0].LastStatusCode == nil || *page.Data[0].LastStatusCode != 200 {
		t.Fatalf("last_status_code = %v", page.Data[0].LastStatusCode)
	}
	if page.Data[0].DeliveredAt == nil || *page.Data[0].DeliveredAt != "2026-10-12T09:00:01Z" {
		t.Fatalf("delivered_at = %v", page.Data[0].DeliveredAt)
	}
	if page.NextCursor == nil || *page.NextCursor != "40" {
		t.Fatalf("next_cursor = %v", page.NextCursor)
	}
}

func TestWebhookDeliveriesGetReturnsAttempts(t *testing.T) {
	var method, path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.Path
		_, _ = w.Write([]byte(`{"delivery":` + testDelivery + `,"attempts":[{"attempt":1,` +
			`"status_code":200,"error":null,"duration_ms":120,"created_at":"2026-10-12T09:00:00Z"}]}`))
	}))
	defer srv.Close()

	c := New("nt_live_k", WithBaseURL(srv.URL))
	res, err := c.Webhooks.Deliveries.Get(context.Background(), "9c1f")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if method != "GET" || path != "/v1/webhook-deliveries/9c1f" {
		t.Fatalf("%s %s", method, path)
	}
	if res.Delivery.ID != "9c1f" || res.Delivery.Attempts != 1 {
		t.Fatalf("delivery = %+v", res.Delivery)
	}
	if len(res.Attempts) != 1 || res.Attempts[0].Attempt != 1 || res.Attempts[0].StatusCode == nil ||
		*res.Attempts[0].StatusCode != 200 || res.Attempts[0].DurationMs != 120 {
		t.Fatalf("attempts = %+v", res.Attempts)
	}
}

func TestWebhookDeliveriesRedeliver(t *testing.T) {
	var method, path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.Path
		_, _ = w.Write([]byte(`{"delivery":` + testDelivery + `}`))
	}))
	defer srv.Close()

	c := New("nt_live_k", WithBaseURL(srv.URL))
	d, err := c.Webhooks.Deliveries.Redeliver(context.Background(), "9c1f")
	if err != nil {
		t.Fatalf("redeliver: %v", err)
	}
	if method != "POST" || path != "/v1/webhook-deliveries/9c1f/redeliver" {
		t.Fatalf("%s %s", method, path)
	}
	if d.ID != "9c1f" || d.EndpointID != "5d1e" {
		t.Fatalf("delivery = %+v", d)
	}
}

func TestDoBinaryReturnsBytesFilenameAndContentType(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/inbound/messages/em1/raw" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "message/rfc822")
		w.Header().Set("Content-Disposition", `attachment; filename="message.eml"`)
		_, _ = w.Write([]byte("From: a@example.com\r\n\r\nhi"))
	}))
	defer srv.Close()

	c := New("nt_live_k", WithBaseURL(srv.URL))
	data, filename, contentType, err := c.doBinary(context.Background(), apiRequest{method: "GET", path: "/v1/inbound/messages/em1/raw"})
	if err != nil {
		t.Fatalf("doBinary: %v", err)
	}
	if string(data) != "From: a@example.com\r\n\r\nhi" {
		t.Fatalf("data = %q", data)
	}
	if filename != "message.eml" {
		t.Fatalf("filename = %q", filename)
	}
	if contentType != "message/rfc822" {
		t.Fatalf("contentType = %q", contentType)
	}
}

func TestDoBinaryNoDispositionFallsBackToEmptyFilename(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("raw-bytes"))
	}))
	defer srv.Close()

	c := New("nt_live_k", WithBaseURL(srv.URL))
	data, filename, _, err := c.doBinary(context.Background(), apiRequest{method: "GET", path: "/x"})
	if err != nil {
		t.Fatalf("doBinary: %v", err)
	}
	if string(data) != "raw-bytes" || filename != "" {
		t.Fatalf("data = %q filename = %q", data, filename)
	}
}

func TestDoBinaryEnvelopeErrorOnNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"code":"inbound_message_not_found","message":"no such message"}}`))
	}))
	defer srv.Close()

	c := New("nt_live_k", WithBaseURL(srv.URL))
	_, _, _, err := c.doBinary(context.Background(), apiRequest{method: "GET", path: "/v1/inbound/messages/nope/raw"})
	if err == nil {
		t.Fatal("expected error")
	}
	var ne *NotideusError
	if !errors.As(err, &ne) {
		t.Fatalf("error type = %T", err)
	}
	if ne.Code != "inbound_message_not_found" || ne.Status != 404 {
		t.Fatalf("error = %+v", ne)
	}
}
