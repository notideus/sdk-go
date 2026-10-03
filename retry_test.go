package notideus

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestRetries429HonoringRetryAfterZero(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":{"code":"rate_limited","message":"slow"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := New("k", WithBaseURL(srv.URL))
	var out map[string]any
	if err := c.do(context.Background(), apiRequest{method: "GET", path: "/v1/contacts"}, &out); err != nil {
		t.Fatalf("do: %v", err)
	}
	if calls.Load() != 2 {
		t.Fatalf("calls = %d", calls.Load())
	}
}

func TestRetries5xxWithBackoff(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n <= 2 {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":{"code":"internal","message":"boom"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := New("k", WithBaseURL(srv.URL))
	var out map[string]any
	if err := c.do(context.Background(), apiRequest{method: "GET", path: "/v1/contacts"}, &out); err != nil {
		t.Fatalf("do: %v", err)
	}
	if calls.Load() != 3 {
		t.Fatalf("calls = %d", calls.Load())
	}
}

func TestDoesNotRetryKeylessPost(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"code":"rate_limited","message":"slow"}}`))
	}))
	defer srv.Close()

	c := New("k", WithBaseURL(srv.URL))
	var out map[string]any
	err := c.do(context.Background(), apiRequest{method: "POST", path: "/v1/emails", body: map[string]any{"subject": "x"}}, &out)
	var ne *NotideusError
	if !errors.As(err, &ne) || ne.Code != "rate_limited" {
		t.Fatalf("err = %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("calls = %d", calls.Load())
	}
}

func TestRetriesPostWithIdempotencyKey(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":{"code":"internal","message":"boom"}}`))
			return
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"e1"}`))
	}))
	defer srv.Close()

	c := New("k", WithBaseURL(srv.URL))
	var out map[string]any
	err := c.do(context.Background(), apiRequest{method: "POST", path: "/v1/emails", body: map[string]any{"subject": "x", "idempotency_key": "req-1"}}, &out)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	if calls.Load() != 2 || out["id"] != "e1" {
		t.Fatalf("calls=%d out=%v", calls.Load(), out)
	}
}

func TestDoesNotRetryContactWrites(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":{"code":"internal","message":"boom"}}`))
	}))
	defer srv.Close()

	c := New("k", WithBaseURL(srv.URL))
	var out map[string]any
	err := c.do(context.Background(), apiRequest{method: "POST", path: "/v1/contacts", body: map[string]any{"email": "j@x.co"}}, &out)
	var ne *NotideusError
	if !errors.As(err, &ne) || ne.Code != "internal" {
		t.Fatalf("err = %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("calls = %d", calls.Load())
	}
}

func TestStopsAfterMaxRetries(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"code":"rate_limited","message":"slow"}}`))
	}))
	defer srv.Close()

	c := New("k", WithBaseURL(srv.URL), WithMaxRetries(1))
	var out map[string]any
	err := c.do(context.Background(), apiRequest{method: "GET", path: "/v1/contacts"}, &out)
	var ne *NotideusError
	if !errors.As(err, &ne) || ne.Code != "rate_limited" {
		t.Fatalf("err = %v", err)
	}
	if calls.Load() != 2 { // initial + 1 retry
		t.Fatalf("calls = %d", calls.Load())
	}
}
