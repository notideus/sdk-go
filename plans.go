package notideus

import (
	"context"
	"net/url"
)

// Plan is one catalog plan row — field-for-field parity with the plans entry
// in GET /v1/plans in notideus-api/docs/public-api.md. Limits are nil when
// the plan does not define them; Tiers is nil on free and custom plans.
type Plan struct {
	Key               string     `json:"key"`
	Kind              string     `json:"kind"`
	Name              string     `json:"name"`
	Blurb             string     `json:"blurb"`
	Cta               *string    `json:"cta"`
	Featured          bool       `json:"featured"`
	Benefits          []string   `json:"benefits"`
	EmailLimit        *int64     `json:"email_limit"`
	ContactLimit      *int64     `json:"contact_limit"`
	DomainsLimit      *int64     `json:"domains_limit"`
	StorageGb         *int64     `json:"storage_gb"`
	SendRatePerSecond *int64     `json:"send_rate_per_second"`
	Tiers             []PlanTier `json:"tiers"`
}

// PlanTier is one purchasable emails-volume tier of a paid plan.
type PlanTier struct {
	Emails       int64  `json:"emails"`
	MonthlyCents int64  `json:"monthly_cents"`
	AnnualCents  *int64 `json:"annual_cents"`
}

// ContactsTier is one rung of the marketing contacts ladder.
type ContactsTier struct {
	Contacts     int64 `json:"contacts"`
	MonthlyCents int64 `json:"monthly_cents"`
}

// FreeAllowances is what the free plan grants.
type FreeAllowances struct {
	Emails   int64 `json:"emails"`
	Contacts int64 `json:"contacts"`
}

// PlanCatalog is the public subscription catalog (GET /v1/plans).
type PlanCatalog struct {
	FreeAllowances      FreeAllowances `json:"free_allowances"`
	OveragePer1000Cents *int64         `json:"overage_per_1000_cents"`
	Plans               []Plan         `json:"plans"`
	ContactsTiers       []ContactsTier `json:"contacts_tiers"`
}

// PlansService exposes the public plan catalog. The route is unauthenticated,
// so every call suppresses the Authorization header.
type PlansService struct{ client *Client }

// List returns the subscription catalog (GET /v1/plans) → 200. Locale nil
// leaves the server default (en); name/blurb/cta/benefits come back already
// localized.
func (s *PlansService) List(ctx context.Context, locale *string) (*PlanCatalog, error) {
	q := url.Values{}
	if locale != nil {
		q.Set("locale", *locale)
	}
	var catalog PlanCatalog
	err := s.client.do(ctx, apiRequest{method: "GET", path: "/v1/plans", query: q, noAuth: true}, &catalog)
	if err != nil {
		return nil, err
	}
	return &catalog, nil
}
