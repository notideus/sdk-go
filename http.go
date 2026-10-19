package notideus

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"time"
)

const version = "1.1.0"

type apiRequest struct {
	method string
	path   string
	query  url.Values
	body   any  // struct or map; nil for no body
	noAuth bool // suppress the Authorization header (public routes on a keyed client)
}

func (c *Client) do(ctx context.Context, req apiRequest, out any) error {
	idempotent := req.method != http.MethodPost || hasIdempotencyKey(req.body)

	var bodyBytes []byte
	if req.body != nil {
		b, err := json.Marshal(req.body)
		if err != nil {
			return fmt.Errorf("notideus: encode request: %w", err)
		}
		bodyBytes = b
	}

	var lastErr error
	for attempt := 0; ; attempt++ {
		err := c.attempt(ctx, req, bodyBytes, out)
		if err == nil {
			return nil
		}
		lastErr = err
		if !idempotent || !isRetriable(err) || attempt >= c.maxRetries {
			return lastErr
		}
		delay := retryDelay(err, attempt)
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func (c *Client) attempt(ctx context.Context, req apiRequest, bodyBytes []byte, out any) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	u := c.baseURL + req.path
	if len(req.query) > 0 {
		u += "?" + req.query.Encode()
	}
	httpReq, err := http.NewRequestWithContext(ctx, req.method, u, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	httpReq.Header.Set("User-Agent", "notideus-go/"+version)
	if bodyBytes != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}
	if !req.noAuth && c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return &NotideusError{Code: "network_error", Message: err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}
	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return &NotideusError{Code: "network_error", Message: err.Error()}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return envelopeError(payload, resp.StatusCode, resp.Header)
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(payload, out)
}

// doBinary downloads a binary endpoint (raw MIME, attachments): it mirrors
// attempt's request building (auth, user-agent, no body) but returns the raw
// response bytes instead of unmarshaling JSON. On a non-2xx status it runs
// the same envelope error path as do. There are no retries — callers can
// re-invoke. The filename is parsed from Content-Disposition ("" when the
// header is absent or unparseable); contentType is the Content-Type header
// ("" when absent).
func (c *Client) doBinary(ctx context.Context, req apiRequest) (data []byte, filename string, contentType string, err error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	u := c.baseURL + req.path
	if len(req.query) > 0 {
		u += "?" + req.query.Encode()
	}
	httpReq, err := http.NewRequestWithContext(ctx, req.method, u, nil)
	if err != nil {
		return nil, "", "", err
	}
	httpReq.Header.Set("User-Agent", "notideus-go/"+version)
	if !req.noAuth && c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, "", "", &NotideusError{Code: "network_error", Message: err.Error()}
	}
	defer resp.Body.Close()

	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", "", &NotideusError{Code: "network_error", Message: err.Error()}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "", "", envelopeError(payload, resp.StatusCode, resp.Header)
	}
	return payload, contentDispositionFilename(resp.Header.Get("Content-Disposition")), resp.Header.Get("Content-Type"), nil
}

var contentDispositionFilenameRe = regexp.MustCompile(`filename="?([^";]+)"?`)

// contentDispositionFilename extracts the filename from a Content-Disposition
// header value, or "" when absent or unparseable.
func contentDispositionFilename(v string) string {
	m := contentDispositionFilenameRe.FindStringSubmatch(v)
	if m == nil {
		return ""
	}
	return m[1]
}

func envelopeError(payload []byte, status int, h http.Header) *NotideusError {
	var parsed struct {
		Error *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
			Reason  string `json:"reason"`
		} `json:"error"`
	}
	code, message := "unknown_error", fmt.Sprintf("HTTP %d", status)
	var reason string
	if json.Unmarshal(payload, &parsed) == nil && parsed.Error != nil {
		if parsed.Error.Code != "" {
			code = parsed.Error.Code
		}
		if parsed.Error.Message != "" {
			message = parsed.Error.Message
		}
		reason = parsed.Error.Reason
	}
	e := &NotideusError{Code: code, Message: message, Status: status, Reason: reason}
	if ra := h.Get("Retry-After"); ra != "" {
		if n, err := strconv.Atoi(ra); err == nil {
			e.RetryAfter = n
		}
	}
	return e
}

func hasIdempotencyKey(body any) bool {
	if body == nil {
		return false
	}
	b, err := json.Marshal(body)
	if err != nil {
		return false
	}
	var m map[string]any
	if json.Unmarshal(b, &m) != nil {
		return false
	}
	v, ok := m["idempotency_key"]
	return ok && v != nil
}

func isRetriable(err error) bool {
	var ne *NotideusError
	if !errors.As(err, &ne) {
		return false
	}
	if ne.Code == "network_error" {
		return true
	}
	return ne.Status == 429 || ne.Status >= 500
}

func retryDelay(err error, attempt int) time.Duration {
	var ne *NotideusError
	if errors.As(err, &ne) && ne.RetryAfter > 0 {
		return time.Duration(ne.RetryAfter) * time.Second
	}
	d := time.Second << attempt
	if d > 8*time.Second {
		d = 8 * time.Second
	}
	return d + time.Duration(rand.Intn(250))*time.Millisecond
}
