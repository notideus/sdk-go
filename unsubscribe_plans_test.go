package notideus

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUnsubscribeInfoUsesQueryParamWithoutAuthHeader(t *testing.T) {
	var path, query, auth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path, query = r.URL.Path, r.URL.RawQuery
		auth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{"email":"jane@example.com","status":"subscribed","team_name":"Acme Inc",` +
			`"topics":[{"id":"7b1c","name":"newsletter","display_name":"Newsletter","subscribed":true}]}`))
	}))
	defer srv.Close()

	c := New("nt_live_k", WithBaseURL(srv.URL))
	info, err := c.Unsubscribe.Info(context.Background(), "550e8400-e29b-41d4-a716-446655440000")
	if err != nil {
		t.Fatalf("info: %v", err)
	}
	if path != "/v1/unsubscribe/info" {
		t.Fatalf("path = %s", path)
	}
	if query != "t=550e8400-e29b-41d4-a716-446655440000" {
		t.Fatalf("query = %s", query)
	}
	if auth != "" {
		t.Fatalf("Authorization = %q, want none", auth)
	}
	if info.Email != "jane@example.com" || info.Status != "subscribed" || info.TeamName != "Acme Inc" {
		t.Fatalf("info = %+v", info)
	}
	if len(info.Topics) != 1 || info.Topics[0].ID != "7b1c" || info.Topics[0].DisplayName != "Newsletter" || !info.Topics[0].Subscribed {
		t.Fatalf("topics = %+v", info.Topics)
	}
}

func TestUnsubscribeOmitsTopicIDsWhenNotGiven(t *testing.T) {
	var path string
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := New("nt_live_k", WithBaseURL(srv.URL))
	err := c.Unsubscribe.Unsubscribe(context.Background(), "tok")
	if err != nil {
		t.Fatalf("unsubscribe: %v", err)
	}
	if path != "/v1/unsubscribe" {
		t.Fatalf("path = %s", path)
	}
	if len(body) != 1 || body["token"] != "tok" {
		t.Fatalf("body = %v", body)
	}
}

func TestUnsubscribeSendsTopicIDsWhenGiven(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := New("nt_live_k", WithBaseURL(srv.URL))
	err := c.Unsubscribe.Unsubscribe(context.Background(), "tok", "7b1c")
	if err != nil {
		t.Fatalf("unsubscribe: %v", err)
	}
	if body["token"] != "tok" {
		t.Fatalf("body = %v", body)
	}
	if ids, ok := body["topic_ids"].([]any); !ok || len(ids) != 1 || ids[0] != "7b1c" {
		t.Fatalf("topic_ids = %v", body["topic_ids"])
	}
}

func TestUnsubscribePreferencesAlwaysSendsTopicIDs(t *testing.T) {
	var path string
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := New("nt_live_k", WithBaseURL(srv.URL))
	err := c.Unsubscribe.Preferences(context.Background(), "tok", []string{})
	if err != nil {
		t.Fatalf("preferences: %v", err)
	}
	if path != "/v1/unsubscribe/preferences" {
		t.Fatalf("path = %s", path)
	}
	if body["token"] != "tok" {
		t.Fatalf("body = %v", body)
	}
	if ids, ok := body["topic_ids"].([]any); !ok || len(ids) != 0 {
		t.Fatalf("topic_ids = %v, want empty array", body["topic_ids"])
	}
}

func TestPlansListWithoutAuthAndWithLocale(t *testing.T) {
	var path, query, auth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path, query = r.URL.Path, r.URL.RawQuery
		auth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{"free_allowances":{"emails":5000,"contacts":1000},` +
			`"overage_per_1000_cents":75,"plans":[],"contacts_tiers":[]}`))
	}))
	defer srv.Close()

	c := New("nt_live_k", WithBaseURL(srv.URL))
	catalog, err := c.Plans.List(context.Background(), Ptr("fr"))
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if path != "/v1/plans" {
		t.Fatalf("path = %s", path)
	}
	if query != "locale=fr" {
		t.Fatalf("query = %s", query)
	}
	if auth != "" {
		t.Fatalf("Authorization = %q, want none", auth)
	}
	if catalog.FreeAllowances.Emails != 5000 || catalog.FreeAllowances.Contacts != 1000 {
		t.Fatalf("free_allowances = %+v", catalog.FreeAllowances)
	}
	if catalog.OveragePer1000Cents == nil || *catalog.OveragePer1000Cents != 75 {
		t.Fatalf("overage = %v", catalog.OveragePer1000Cents)
	}
	if len(catalog.Plans) != 0 || len(catalog.ContactsTiers) != 0 {
		t.Fatalf("plans = %d, contacts_tiers = %d", len(catalog.Plans), len(catalog.ContactsTiers))
	}
}
