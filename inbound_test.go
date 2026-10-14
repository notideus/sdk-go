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

const testRoute = `{"id":"8a2b","domain_id":"dom1","recipient":"help","name":"Support",` +
	`"webhook_endpoint_id":"5d1e","store":true,"status":"active",` +
	`"created_at":"2026-10-12T09:00:00Z","updated_at":"2026-10-12T09:00:00Z"}`

const testMessage = `{"id":"em9","route_id":"8a2b","from_addr":"a@example.com","from_name":"Ann",` +
	`"to_addr":"help@corp.example","subject":"Hi","text":"hello","html":"<p>hello</p>",` +
	`"headers":{"Message-ID":"<m1@example.com>"},` +
	`"attachments":[{"name":"report.csv","content_type":"text/csv","size_kb":3,"key":"k1"}],` +
	`"spam_verdict":"pass","size_kb":12,"status":"received","error":null,` +
	`"received_at":"2026-10-12T09:00:00Z"}`

func TestInboundRoutesListIncludesMeta(t *testing.T) {
	var method, path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.Path
		_, _ = w.Write([]byte(`{"data":[` + testRoute + `],"meta":{"simulate_enabled":true,"retention_days":30}}`))
	}))
	defer srv.Close()

	c := New("nt_live_k", WithBaseURL(srv.URL))
	page, err := c.Inbound.Routes.List(context.Background())
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if method != "GET" || path != "/v1/inbound/routes" {
		t.Fatalf("%s %s", method, path)
	}
	if len(page.Data) != 1 || page.Data[0].ID != "8a2b" || page.Data[0].Recipient != "help" {
		t.Fatalf("data = %+v", page.Data)
	}
	if page.Data[0].WebhookEndpointID == nil || *page.Data[0].WebhookEndpointID != "5d1e" {
		t.Fatalf("webhook_endpoint_id = %v", page.Data[0].WebhookEndpointID)
	}
	if !page.Meta.SimulateEnabled || page.Meta.RetentionDays != 30 {
		t.Fatalf("meta = %+v", page.Meta)
	}
}

func TestInboundRoutesCreate(t *testing.T) {
	var method, path string
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.Path
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &body)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"route":` + testRoute + `}`))
	}))
	defer srv.Close()

	c := New("nt_live_k", WithBaseURL(srv.URL))
	route, err := c.Inbound.Routes.Create(context.Background(), CreateInboundRouteParams{
		DomainID:          "dom1",
		Recipient:         "help",
		Name:              Ptr("Support"),
		Store:             true,
		WebhookEndpointID: Ptr("5d1e"),
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if method != "POST" || path != "/v1/inbound/routes" {
		t.Fatalf("%s %s", method, path)
	}
	if route.ID != "8a2b" || route.DomainID != "dom1" || !route.Store {
		t.Fatalf("route = %+v", route)
	}
	if body["domain_id"] != "dom1" || body["recipient"] != "help" || body["name"] != "Support" ||
		body["store"] != true || body["webhook_endpoint_id"] != "5d1e" {
		t.Fatalf("body = %v", body)
	}
}

func TestInboundRoutesUpdateHasNoDomainID(t *testing.T) {
	const id = "8a2b"
	var method, path string
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.Path
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &body)
		_, _ = w.Write([]byte(`{"route":` + testRoute + `}`))
	}))
	defer srv.Close()

	c := New("nt_live_k", WithBaseURL(srv.URL))
	route, err := c.Inbound.Routes.Update(context.Background(), id, UpdateInboundRouteParams{
		Recipient:         "support",
		Name:              Ptr("Support v2"),
		Store:             false,
		WebhookEndpointID: Ptr("5d1e"),
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if route.ID != "8a2b" {
		t.Fatalf("route = %+v", route)
	}
	if method != "PATCH" || path != "/v1/inbound/routes/"+id {
		t.Fatalf("%s %s", method, path)
	}
	if _, hasDomain := body["domain_id"]; hasDomain {
		t.Fatalf("update body must not include domain_id: %v", body)
	}
	if _, hasStatus := body["status"]; hasStatus {
		t.Fatalf("update body must not include status: %v", body)
	}
	if body["recipient"] != "support" || body["name"] != "Support v2" || body["store"] != false ||
		body["webhook_endpoint_id"] != "5d1e" {
		t.Fatalf("body = %v", body)
	}
}

func TestInboundRoutesGetAndDelete(t *testing.T) {
	const id = "8a2b"
	var calls []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		if r.Method == "DELETE" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		_, _ = w.Write([]byte(`{"route":` + testRoute + `}`))
	}))
	defer srv.Close()

	c := New("nt_live_k", WithBaseURL(srv.URL))
	route, err := c.Inbound.Routes.Get(context.Background(), id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if route.ID != "8a2b" || route.Name != "Support" {
		t.Fatalf("route = %+v", route)
	}
	if err := c.Inbound.Routes.Delete(context.Background(), id); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if len(calls) != 2 || calls[0] != "GET /v1/inbound/routes/"+id || calls[1] != "DELETE /v1/inbound/routes/"+id {
		t.Fatalf("calls = %v", calls)
	}
}

func TestInboundMessagesListSerializesFiltersAndMapsPage(t *testing.T) {
	var gotQuery url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		_, _ = w.Write([]byte(`{"data":[` + testMessage + `],"next_cursor":"60"}`))
	}))
	defer srv.Close()

	c := New("nt_live_k", WithBaseURL(srv.URL))
	page, err := c.Inbound.Messages.List(context.Background(), InboundMessageListParams{
		Q:       Ptr("hello"),
		RouteID: Ptr("8a2b"),
		Cursor:  Ptr("40"),
		Limit:   Ptr(int32(20)),
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if gotQuery.Get("q") != "hello" || gotQuery.Get("route_id") != "8a2b" ||
		gotQuery.Get("cursor") != "40" || gotQuery.Get("limit") != "20" {
		t.Fatalf("query = %v", gotQuery)
	}
	if len(page.Data) != 1 || page.Data[0].ID != "em9" {
		t.Fatalf("data = %+v", page.Data)
	}
	if page.NextCursor == nil || *page.NextCursor != "60" {
		t.Fatalf("next_cursor = %v", page.NextCursor)
	}
}

func TestInboundMessagesGetUnwrapsMessage(t *testing.T) {
	var method, path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.Path
		_, _ = w.Write([]byte(`{"message":` + testMessage + `}`))
	}))
	defer srv.Close()

	c := New("nt_live_k", WithBaseURL(srv.URL))
	msg, err := c.Inbound.Messages.Get(context.Background(), "em9")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if method != "GET" || path != "/v1/inbound/messages/em9" {
		t.Fatalf("%s %s", method, path)
	}
	if msg.ID != "em9" || msg.Subject != "Hi" || msg.FromAddr != "a@example.com" {
		t.Fatalf("message = %+v", msg)
	}
	if msg.RouteID == nil || *msg.RouteID != "8a2b" {
		t.Fatalf("route_id = %v", msg.RouteID)
	}
	if msg.Headers["Message-ID"] != "<m1@example.com>" {
		t.Fatalf("headers = %v", msg.Headers)
	}
	if len(msg.Attachments) != 1 || msg.Attachments[0].Name != "report.csv" ||
		msg.Attachments[0].ContentType != "text/csv" || msg.Attachments[0].SizeKb != 3 {
		t.Fatalf("attachments = %+v", msg.Attachments)
	}
	if msg.SpamVerdict == nil || *msg.SpamVerdict != "pass" {
		t.Fatalf("spam_verdict = %v", msg.SpamVerdict)
	}
	if msg.Error != nil || msg.Status != "received" || msg.ReceivedAt != "2026-10-12T09:00:00Z" {
		t.Fatalf("message = %+v", msg)
	}
}

func TestInboundMessagesRawDownloadsMIME(t *testing.T) {
	var method, path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.Path
		w.Header().Set("Content-Type", "message/rfc822")
		_, _ = w.Write([]byte("From: a@example.com\r\n\r\nhello"))
	}))
	defer srv.Close()

	c := New("nt_live_k", WithBaseURL(srv.URL))
	data, filename, contentType, err := c.Inbound.Messages.Raw(context.Background(), "em9")
	if err != nil {
		t.Fatalf("raw: %v", err)
	}
	if method != "GET" || path != "/v1/inbound/messages/em9/raw" {
		t.Fatalf("%s %s", method, path)
	}
	if string(data) != "From: a@example.com\r\n\r\nhello" {
		t.Fatalf("data = %q", data)
	}
	if filename != "" {
		t.Fatalf("raw has no Content-Disposition, filename = %q", filename)
	}
	if contentType != "message/rfc822" {
		t.Fatalf("contentType = %q", contentType)
	}
}

func TestInboundMessagesAttachmentDownloadsByIndex(t *testing.T) {
	var method, path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.Path
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", `attachment; filename="report.csv"`)
		_, _ = w.Write([]byte("a,b\n1,2\n"))
	}))
	defer srv.Close()

	c := New("nt_live_k", WithBaseURL(srv.URL))
	data, filename, contentType, err := c.Inbound.Messages.Attachment(context.Background(), "em9", 0)
	if err != nil {
		t.Fatalf("attachment: %v", err)
	}
	if method != "GET" || path != "/v1/inbound/messages/em9/attachments/0" {
		t.Fatalf("%s %s", method, path)
	}
	if string(data) != "a,b\n1,2\n" {
		t.Fatalf("data = %q", data)
	}
	if filename != "report.csv" {
		t.Fatalf("filename = %q", filename)
	}
	if contentType != "text/csv" {
		t.Fatalf("contentType = %q", contentType)
	}
}

func TestInboundBinaryEnvelopeErrorOnNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"code":"inbound_message_not_found","message":"no such message"}}`))
	}))
	defer srv.Close()

	c := New("nt_live_k", WithBaseURL(srv.URL))
	_, _, _, err := c.Inbound.Messages.Raw(context.Background(), "nope")
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
