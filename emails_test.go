package notideus

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEmailsSend(t *testing.T) {
	var path, method string
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path, method = r.URL.Path, r.Method
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &body)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"9c2e","from_address":"noreply@acme.com","subject":"Welcome","status":"queued"}`))
	}))
	defer srv.Close()

	c := New("nt_live_k", WithBaseURL(srv.URL))
	email, err := c.Emails.Send(context.Background(), SendEmailParams{
		From:           "Acme Inc <noreply@acme.com>",
		To:             []string{"jane@example.com"},
		Subject:        "Welcome",
		HTML:           "<p>Hi {{name}}</p>",
		Variables:      map[string]string{"name": "Jane"},
		Tags:           []string{"welcome"},
		IdempotencyKey: Ptr("req-123"),
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if email.ID != "9c2e" {
		t.Fatalf("id = %q", email.ID)
	}
	if path != "/v1/emails" || method != "POST" {
		t.Fatalf("%s %s", method, path)
	}
	if body["from"] != "Acme Inc <noreply@acme.com>" || body["idempotency_key"] != "req-123" {
		t.Fatalf("body = %v", body)
	}
}

func TestEmailsSendBatchReturnsPerItemResults(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &body)
		_, _ = w.Write([]byte(`{"data":[{"index":0,"id":"9c2e"},{"index":1,"error":{"code":"from_domain_not_verified","message":"no verified domain matches the from address"}}]}`))
	}))
	defer srv.Close()

	c := New("nt_live_k", WithBaseURL(srv.URL))
	results, err := c.Emails.SendBatch(context.Background(), []SendEmailParams{
		{From: "a@b.co", To: []string{"j@x.co"}, Subject: "one", HTML: "<p>1</p>"},
		{From: "a@b.co", To: []string{"k@x.co"}, Subject: "two", HTML: "<p>2</p>"},
	})
	if err != nil {
		t.Fatalf("batch: %v", err)
	}
	if len(results) != 2 || results[0].ID != "9c2e" || results[0].Error != nil {
		t.Fatalf("results[0] = %+v", results[0])
	}
	if results[1].Error == nil || results[1].Error.Code != "from_domain_not_verified" {
		t.Fatalf("results[1] = %+v", results[1])
	}
	if body["emails"] == nil {
		t.Fatalf("body = %v", body)
	}
}

func TestEmailsSendBatchWholeFailureThrows(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusPaymentRequired)
		_, _ = w.Write([]byte(`{"error":{"code":"quota_exceeded","message":"allowance reached"}}`))
	}))
	defer srv.Close()

	c := New("nt_live_k", WithBaseURL(srv.URL), WithMaxRetries(0))
	_, err := c.Emails.SendBatch(context.Background(), []SendEmailParams{
		{From: "a@b.co", To: []string{"j@x.co"}, Subject: "s", HTML: "<p>h</p>"},
	})
	var ne *NotideusError
	if !errors.As(err, &ne) || ne.Code != "quota_exceeded" {
		t.Fatalf("err = %v", err)
	}
}
