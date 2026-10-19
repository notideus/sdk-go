# sdk-go

Official Go SDK for the [Notideus](https://notideus.io) email API. MIT
licensed.

Requires Go 1.22+. Zero runtime dependencies — the client is built on
`net/http` and the params/responses mirror the
[API reference](https://notideus.io/docs) 1:1 (snake_case JSON).

## Install

```bash
go get github.com/notideus/sdk-go
```

## Quickstart

```go
package main

import (
	"context"
	"fmt"
	"os"

	notideus "github.com/notideus/sdk-go"
)

func main() {
	client := notideus.New(os.Getenv("NT_KEY"))

	email, err := client.Emails.Send(context.Background(), notideus.SendEmailParams{
		From:    "Acme Inc <noreply@acme.com>",
		To:      []string{"jane@example.com"},
		Subject: "Welcome",
		HTML:    "<p>Hi {{name}}</p>",
		Variables: map[string]string{
			"name": "Jane",
		},
		Tags:           []string{"welcome"},
		IdempotencyKey: notideus.Ptr("req-123"),
	})
	if err != nil {
		// handle *notideus.NotideusError (see Error handling)
		panic(err)
	}
	fmt.Println(email.ID)
}
```

## Configuration

```go
client := notideus.New(apiKey, notideus.WithBaseURL(...), ...)
```

| Option            | Type            | Default                   | Notes                        |
| ----------------- | --------------- | ------------------------- | ---------------------------- |
| `apiKey`          | `string` (1st arg) | `""`                   | omit for unauthenticated resources |
| `WithBaseURL`     | `string`        | `https://api.notideus.io` | override for self-hosting    |
| `WithTimeout`     | `time.Duration` | `30s`                     | per-attempt HTTP timeout     |
| `WithMaxRetries`  | `int`           | `2`                       | see Retry behavior below     |
| `WithHTTPClient`  | `*http.Client`  | `http.DefaultClient`-ish  | inject a custom transport    |

The key may be empty: `Plans` and `Unsubscribe` work without one. Calling an
authenticated resource without a key surfaces the API's 401 as a
`*notideus.NotideusError`.

Use `notideus.Ptr(v)` for any optional pointer field (`*string`, `*int`, …).

## Resources

Params and responses use snake_case exactly like the API.

### Emails

```go
client.Emails.Send(ctx, notideus.SendEmailParams{
	From: "Acme Inc <noreply@acme.com>", To: []string{"jane@example.com"},
	Subject: "Welcome", HTML: "…", Text: "…",
	TemplateID: notideus.Ptr("tpl-1"), Variables: map[string]string{…},
	Tags: []string{…}, Headers: map[string]string{…}, ReplyTo: "…",
	IdempotencyKey: notideus.Ptr("req-123"),
})
client.Emails.SendBatch(ctx, []notideus.SendEmailParams{…}) // up to 100
```

### WhatsApp messages

```go
client.WhatsApp.Messages.Send(ctx, notideus.SendWhatsAppMessageParams{
	TemplateID: "tpl-1",
	To:         []string{"+33612345678"},
	Parameters: map[string]string{"body:1": "Jane"},
})
client.WhatsApp.Messages.SendBatch(ctx, []notideus.SendWhatsAppMessageParams{…})
```

`Send` returns one entry per recipient, in `To` order.

### Contacts

```go
client.Contacts.Create(ctx, notideus.CreateContactParams{
	Email: "jane@example.com", Properties: map[string]string{…}, TopicIDs: []string{…},
})
client.Contacts.Get(ctx, "jane@example.com")
client.Contacts.Update(ctx, "jane@example.com", notideus.UpdateContactParams{
	Properties: map[string]string{"plan": "pro"},
})
client.Contacts.Delete(ctx, "jane@example.com")
client.Contacts.Unsubscribe(ctx, "jane@example.com")
client.Contacts.Resubscribe(ctx, "jane@example.com")
client.Contacts.Bulk(ctx, []notideus.BulkContactInput{{Email: "…"}, …}) // up to 1000

page, err := client.Contacts.List(ctx, notideus.ListContactsParams{
	Limit: notideus.Ptr(100), Status: notideus.Ptr("subscribed"),
})
if page.NextCursor != nil {
	next, err := client.Contacts.List(ctx, notideus.ListContactsParams{
		Cursor: page.NextCursor,
	})
	_ = next
}
```

### Webhooks

```go
created, err := client.Webhooks.Create(ctx, notideus.CreateWebhookParams{
	Name: "Production API", URL: "https://example.com/hook", Events: []string{"email.delivered"},
})
// created.Secret — whsec_…, shown once
_, err = client.Webhooks.Update(ctx, created.Endpoint.ID, notideus.UpdateWebhookParams{
	Name: "Production API", URL: "https://example.com/hook",
	Events: []string{"email.delivered"}, Status: notideus.WebhookEndpointDisabled,
})
log, err := client.Webhooks.Deliveries.List(ctx, notideus.WebhookDeliveryListParams{
	EndpointID: notideus.Ptr(created.Endpoint.ID),
})
_, err = client.Webhooks.Deliveries.Redeliver(ctx, log.Data[0].ID)
events, err := client.Webhooks.Events(ctx)
```

### Inbound email

```go
routes, err := client.Inbound.Routes.List(ctx)
_, err = client.Inbound.Routes.Create(ctx, notideus.CreateInboundRouteParams{
	DomainID: domainID, Recipient: "support", Store: true,
})
inbox, err := client.Inbound.Messages.List(ctx, notideus.InboundMessageListParams{
	Q: notideus.Ptr("billing"),
})
mime, filename, contentType, err := client.Inbound.Messages.Raw(ctx, inbox.Data[0].ID)
```

### Hosted unsubscribe (token-based, unauthenticated)

```go
info, err := client.Unsubscribe.Info(ctx, token)
err = client.Unsubscribe.Unsubscribe(ctx, token)             // whole contact
err = client.Unsubscribe.Unsubscribe(ctx, token, topicIDs…)  // specific topics
err = client.Unsubscribe.Preferences(ctx, token, topicIDs)   // desired subscribed set
```

### Plans (unauthenticated)

```go
catalog, err := client.Plans.List(ctx, notideus.Ptr("fr")) // locale optional
```

## Error handling

Request-level failures return `*notideus.NotideusError`:

```go
_, err := client.Emails.Send(ctx, params)
var ne *notideus.NotideusError
if errors.As(err, &ne) {
	if ne.Code == "from_domain_not_verified" { /* … */ }
	if ne.Code == "rate_limited" {
		_ = ne.RetryAfter // seconds to wait
	}
	_ = ne.Status // HTTP status; 0 for transport failures (code "network_error")
	_ = ne.Reason // optional admin note on account_suspended
}
```

Batch sends never fail per item — inspect the results instead:

```go
results, err := client.Emails.SendBatch(ctx, batch) // err only on whole-batch failure
for _, r := range results {
	if r.Error != nil {
		log.Printf("%d %s %s", r.Index, r.Error.Code, r.Error.Message)
	} else {
		log.Printf("%d %s", r.Index, r.ID)
	}
}
```

A whole-batch failure (401, 402 `quota_exceeded`, 400 `batch_too_large`)
still returns an error.

## Retry behavior

The SDK retries network errors, 5xx and 429 (honoring `Retry-After`) with
exponential backoff — but only idempotent requests: non-POST calls, and POSTs
that carry an `idempotency_key` (matching the API's replay semantics, so a
retried send never double-sends). Set `WithMaxRetries(0)` to disable.

## Self-hosting / local development

```go
client := notideus.New(os.Getenv("NT_KEY"), notideus.WithBaseURL("http://localhost:8080"))
```

## Development

```bash
go test ./... -race   # mocked http.Client — no infra needed
go vet ./...
gofmt -l .
```

## Releasing

1. Merge the change set to `main`.
2. Create a GitHub Release with a `v<major>.<minor>.<patch>` tag (Go module
   semver) — e.g. `v1.0.0`. The `publish` workflow validates the tag shape
   and runs the full gate (`gofmt`, `vet`, `test -race`); it fails otherwise.
3. There is no publish step and **no secrets at all**: the tag IS the release,
   and the Go module proxy ingests it automatically.
4. Consumers upgrade with `go get github.com/notideus/sdk-go@v1.0.0`.
