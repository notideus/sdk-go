// Package notideus is the official Go SDK for the Notideus email API.
package notideus

// NotideusError is returned for request-level failures: the whole call was
// rejected (including whole-batch failures). Batch item errors are NOT
// NotideusError — they arrive per item in the result slice.
type NotideusError struct {
	Code       string // API error code, e.g. "from_domain_not_verified"; "network_error" with Status 0 for transport failures
	Message    string
	Status     int    // HTTP status; 0 for transport failures
	RetryAfter int    // seconds from the Retry-After header; 0 when absent
	Reason     string // admin-provided note, present on account_suspended errors; empty otherwise
}

func (e *NotideusError) Error() string {
	return e.Code + ": " + e.Message
}
