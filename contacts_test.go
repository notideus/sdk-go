package notideus

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

const testContact = `{"id":"3fa8","email":"jane@example.com","status":"subscribed","properties":{"plan":"pro"},` +
	`"subscribed_topic_ids":["7b1c"],"created_at":"2026-09-10T12:00:00Z","updated_at":"2026-09-10T12:00:00Z"}`

func TestContactsListSerializesFiltersAndMapsPage(t *testing.T) {
	var gotQuery url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		_, _ = w.Write([]byte(`{"data":[` + testContact + `],"next_cursor":"20"}`))
	}))
	defer srv.Close()

	c := New("nt_live_k", WithBaseURL(srv.URL))
	page, err := c.Contacts.List(context.Background(), ListContactsParams{
		Cursor: Ptr("0"),
		Email:  Ptr("jane"),
		Status: Ptr("subscribed"),
		Limit:  Ptr(20),
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if gotQuery.Get("cursor") != "0" || gotQuery.Get("email") != "jane" ||
		gotQuery.Get("status") != "subscribed" || gotQuery.Get("limit") != "20" {
		t.Fatalf("query = %v", gotQuery)
	}
	if len(page.Data) != 1 || page.Data[0].ID != "3fa8" || page.Data[0].Status != "subscribed" {
		t.Fatalf("data = %+v", page.Data)
	}
	if page.Data[0].Properties["plan"] != "pro" || page.Data[0].SubscribedTopicIDs[0] != "7b1c" {
		t.Fatalf("contact = %+v", page.Data[0])
	}
	if page.NextCursor == nil || *page.NextCursor != "20" {
		t.Fatalf("next_cursor = %v", page.NextCursor)
	}
}

func TestContactsCreate(t *testing.T) {
	var method, path string
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.Path
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &body)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(testContact))
	}))
	defer srv.Close()

	c := New("nt_live_k", WithBaseURL(srv.URL))
	contact, err := c.Contacts.Create(context.Background(), CreateContactParams{
		Email:      "jane@example.com",
		Properties: map[string]string{"plan": "pro"},
		TopicIDs:   []string{"7b1c"},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if contact.ID != "3fa8" || contact.Status != "subscribed" {
		t.Fatalf("contact = %+v", contact)
	}
	if method != "POST" || path != "/v1/contacts" {
		t.Fatalf("%s %s", method, path)
	}
	if body["email"] != "jane@example.com" {
		t.Fatalf("body = %v", body)
	}
	if props, ok := body["properties"].(map[string]any); !ok || props["plan"] != "pro" {
		t.Fatalf("properties = %v", body["properties"])
	}
	if topics, ok := body["topic_ids"].([]any); !ok || len(topics) != 1 || topics[0] != "7b1c" {
		t.Fatalf("topic_ids = %v", body["topic_ids"])
	}
}

func TestContactsGetPathEscapesEmail(t *testing.T) {
	const email = "jane+news@example.com"
	var gotPath, gotEscaped string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotEscaped = r.URL.Path, r.URL.EscapedPath()
		_, _ = w.Write([]byte(testContact))
	}))
	defer srv.Close()

	c := New("nt_live_k", WithBaseURL(srv.URL))
	if _, err := c.Contacts.Get(context.Background(), email); err != nil {
		t.Fatalf("get: %v", err)
	}
	if gotPath != "/v1/contacts/"+email {
		t.Fatalf("path = %q", gotPath)
	}
	if gotEscaped != "/v1/contacts/"+url.PathEscape(email) {
		t.Fatalf("escaped path = %q, want /v1/contacts/%s", gotEscaped, url.PathEscape(email))
	}
}

func TestContactsUpdate(t *testing.T) {
	const email = "jane@example.com"
	var method, gotEscaped string
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, gotEscaped = r.Method, r.URL.EscapedPath()
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &body)
		_, _ = w.Write([]byte(testContact))
	}))
	defer srv.Close()

	c := New("nt_live_k", WithBaseURL(srv.URL))
	contact, err := c.Contacts.Update(context.Background(), email, UpdateContactParams{
		Properties: map[string]string{"plan": "scale"},
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if contact.ID != "3fa8" {
		t.Fatalf("contact = %+v", contact)
	}
	if method != "PATCH" || gotEscaped != "/v1/contacts/"+url.PathEscape(email) {
		t.Fatalf("%s %s", method, gotEscaped)
	}
	if props, ok := body["properties"].(map[string]any); !ok || props["plan"] != "scale" {
		t.Fatalf("properties = %v", body["properties"])
	}
	if _, hasTopics := body["topic_ids"]; hasTopics {
		t.Fatalf("topic_ids should be omitted: %v", body)
	}
}

func TestContactsDeleteNoContent(t *testing.T) {
	const email = "jane@example.com"
	var method, gotEscaped string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, gotEscaped = r.Method, r.URL.EscapedPath()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := New("nt_live_k", WithBaseURL(srv.URL))
	if err := c.Contacts.Delete(context.Background(), email); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if method != "DELETE" || gotEscaped != "/v1/contacts/"+url.PathEscape(email) {
		t.Fatalf("%s %s", method, gotEscaped)
	}
}

func TestContactsBulkReturnsPerItemResults(t *testing.T) {
	var path string
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &body)
		_, _ = w.Write([]byte(`{"data":[{"index":0,"id":"3fa8"},{"index":1,"error":` +
			`{"code":"bad_request","message":"invalid email address"}}]}`))
	}))
	defer srv.Close()

	c := New("nt_live_k", WithBaseURL(srv.URL))
	results, err := c.Contacts.Bulk(context.Background(), []BulkContactInput{
		{Email: "jane@example.com"},
		{Email: "not-an-email"},
	})
	if err != nil {
		t.Fatalf("bulk: %v", err)
	}
	if len(results) != 2 || results[0].ID != "3fa8" || results[0].Error != nil {
		t.Fatalf("results[0] = %+v", results[0])
	}
	if results[1].Error == nil || results[1].Error.Code != "bad_request" {
		t.Fatalf("results[1] = %+v", results[1])
	}
	if path != "/v1/contacts/bulk" {
		t.Fatalf("path = %q", path)
	}
	if items, ok := body["contacts"].([]any); !ok || len(items) != 2 {
		t.Fatalf("contacts = %v", body["contacts"])
	}
}

func TestContactsUnsubscribeAndResubscribe(t *testing.T) {
	const email = "jane@example.com"
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.EscapedPath())
		if r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/unsubscribe") {
			_, _ = w.Write([]byte(strings.Replace(testContact, `"subscribed"`, `"unsubscribed"`, 1)))
			return
		}
		_, _ = w.Write([]byte(testContact))
	}))
	defer srv.Close()

	c := New("nt_live_k", WithBaseURL(srv.URL))
	unsubscribed, err := c.Contacts.Unsubscribe(context.Background(), email)
	if err != nil {
		t.Fatalf("unsubscribe: %v", err)
	}
	if unsubscribed.Status != "unsubscribed" {
		t.Fatalf("status = %q", unsubscribed.Status)
	}
	if _, err := c.Contacts.Resubscribe(context.Background(), email); err != nil {
		t.Fatalf("resubscribe: %v", err)
	}
	wantUnsub := "/v1/contacts/" + url.PathEscape(email) + "/unsubscribe"
	wantResub := "/v1/contacts/" + url.PathEscape(email) + "/resubscribe"
	if len(paths) != 2 || paths[0] != wantUnsub || paths[1] != wantResub {
		t.Fatalf("paths = %v", paths)
	}
}
