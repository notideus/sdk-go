package notideus

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestPostsJSONWithBearerAuthAndUserAgent(t *testing.T) {
	var gotMethod, gotPath, gotAuth, gotUA, gotCT, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		gotAuth, gotUA, gotCT = r.Header.Get("Authorization"), r.Header.Get("User-Agent"), r.Header.Get("Content-Type")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"abc"}`))
	}))
	defer srv.Close()

	c := New("nt_live_test", WithBaseURL(srv.URL))
	var out struct {
		ID string `json:"id"`
	}
	err := c.do(context.Background(), apiRequest{method: "POST", path: "/v1/emails", body: map[string]any{"subject": "Hi"}}, &out)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	if out.ID != "abc" {
		t.Fatalf("id = %q", out.ID)
	}
	if gotMethod != "POST" || gotPath != "/v1/emails" {
		t.Fatalf("method/path = %s %s", gotMethod, gotPath)
	}
	if gotAuth != "Bearer nt_live_test" {
		t.Fatalf("auth = %q", gotAuth)
	}
	if !strings.HasPrefix(gotUA, "notideus-go/") {
		t.Fatalf("ua = %q", gotUA)
	}
	if gotCT != "application/json" {
		t.Fatalf("ct = %q", gotCT)
	}
	if gotBody != `{"subject":"Hi"}` {
		t.Fatalf("body = %q", gotBody)
	}
}

func TestOmitsAuthorizationForKeylessClient(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := New("", WithBaseURL(srv.URL)) // keyless
	var out map[string]any
	if err := c.do(context.Background(), apiRequest{method: "GET", path: "/v1/plans"}, &out); err != nil {
		t.Fatalf("do: %v", err)
	}
	if gotAuth != "" {
		t.Fatalf("auth = %q", gotAuth)
	}
}

func TestOmitsAuthorizationOnNoAuthRequestWithKeyedClient(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := New("nt_live_secret", WithBaseURL(srv.URL))
	var out map[string]any
	if err := c.do(context.Background(), apiRequest{method: "GET", path: "/v1/plans", noAuth: true}, &out); err != nil {
		t.Fatalf("do: %v", err)
	}
	if gotAuth != "" {
		t.Fatalf("auth = %q", gotAuth)
	}
}

func TestDropsEmptyQueryParams(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := New("k", WithBaseURL(srv.URL))
	q := make(url.Values)
	q.Set("limit", "50")
	q.Set("status", "subscribed")
	var out map[string]any
	err := c.do(context.Background(), apiRequest{method: "GET", path: "/v1/contacts", query: q}, &out)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	if gotQuery != "limit=50&status=subscribed" {
		t.Fatalf("query = %q", gotQuery)
	}
}

func TestEnvelopeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"error":{"code":"from_domain_not_verified","message":"no verified domain matches the from address"}}`))
	}))
	defer srv.Close()

	c := New("k", WithBaseURL(srv.URL), WithMaxRetries(0))
	var out map[string]any
	err := c.do(context.Background(), apiRequest{method: "POST", path: "/v1/emails", body: map[string]any{}}, &out)
	var ne *NotideusError
	if !errors.As(err, &ne) {
		t.Fatalf("err = %v", err)
	}
	if ne.Code != "from_domain_not_verified" || ne.Status != 422 || ne.Message != "no verified domain matches the from address" {
		t.Fatalf("ne = %+v", ne)
	}
}

func TestParsesRetryAfterOn429(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "7")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"code":"rate_limited","message":"slow down"}}`))
	}))
	defer srv.Close()

	c := New("k", WithBaseURL(srv.URL), WithMaxRetries(0))
	var out map[string]any
	err := c.do(context.Background(), apiRequest{method: "GET", path: "/v1/contacts"}, &out)
	var ne *NotideusError
	if !errors.As(err, &ne) {
		t.Fatalf("err = %v", err)
	}
	if ne.RetryAfter != 7 {
		t.Fatalf("retryAfter = %d", ne.RetryAfter)
	}
}

func TestNoContentReturnsNil(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := New("k", WithBaseURL(srv.URL))
	if err := c.do(context.Background(), apiRequest{method: "DELETE", path: "/v1/contacts/a%40b.c"}, nil); err != nil {
		t.Fatalf("do: %v", err)
	}
}

func TestWrapsTransportFailures(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close() // all requests fail

	c := New("k", WithBaseURL(srv.URL), WithMaxRetries(0))
	var out map[string]any
	err := c.do(context.Background(), apiRequest{method: "GET", path: "/v1/contacts"}, &out)
	var ne *NotideusError
	if !errors.As(err, &ne) || ne.Code != "network_error" || ne.Status != 0 {
		t.Fatalf("err = %v", err)
	}
}
