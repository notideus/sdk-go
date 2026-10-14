package notideus

import (
	"net/http"
	"time"
)

const defaultBaseURL = "https://api.notideus.io"

// Client is the Notideus API client. Construct with New; the key may be
// empty for the unauthenticated groups (Plans, Unsubscribe).
type Client struct {
	apiKey     string
	baseURL    string
	timeout    time.Duration
	maxRetries int
	http       *http.Client

	Emails      *EmailsService
	WhatsApp    *WhatsAppService
	Contacts    *ContactsService
	Webhooks    *WebhooksService
	Inbound     *InboundService
	Unsubscribe *UnsubscribeService
	Plans       *PlansService
}

// Option configures a Client.
type Option func(*Client)

// WithBaseURL overrides the API origin (self-hosting, tests).
func WithBaseURL(u string) Option { return func(c *Client) { c.baseURL = u } }

// WithTimeout sets the per-attempt HTTP timeout (default 30s).
func WithTimeout(d time.Duration) Option { return func(c *Client) { c.timeout = d } }

// WithMaxRetries sets retries beyond the initial attempt (default 2).
func WithMaxRetries(n int) Option { return func(c *Client) { c.maxRetries = n } }

// WithHTTPClient overrides the underlying *http.Client (tests).
func WithHTTPClient(h *http.Client) Option { return func(c *Client) { c.http = h } }

// New constructs a Client.
func New(apiKey string, opts ...Option) *Client {
	c := &Client{
		apiKey:     apiKey,
		baseURL:    defaultBaseURL,
		timeout:    30 * time.Second,
		maxRetries: 2,
		http:       &http.Client{},
	}
	for _, o := range opts {
		o(c)
	}
	c.Emails = &EmailsService{client: c}
	c.WhatsApp = &WhatsAppService{client: c}
	c.Contacts = &ContactsService{client: c}
	c.Webhooks = &WebhooksService{client: c, Deliveries: &WebhookDeliveriesService{client: c}}
	c.Inbound = &InboundService{
		client:   c,
		Routes:   &InboundRoutesService{client: c},
		Messages: &InboundMessagesService{client: c},
	}
	c.Unsubscribe = &UnsubscribeService{client: c}
	c.Plans = &PlansService{client: c}
	c.WhatsApp.Messages = &WhatsAppMessagesService{client: c}
	return c
}

// Ptr returns a pointer to v — shorthand for optional params.
func Ptr[T any](v T) *T { return &v }
