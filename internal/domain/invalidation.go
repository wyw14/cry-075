package domain

import "time"

type InvalidationReason string

const (
	InvalidCopyright InvalidationReason = "copyright_expired"
	InvalidManual    InvalidationReason = "manual_invalidation"
	InvalidEmergency InvalidationReason = "emergency_takedown"
)

type InvalidationEvent struct {
	ID                  ID                 `json:"id"`
	AssetID             ID                 `json:"asset_id"`
	Reason              InvalidationReason `json:"reason"`
	RequestedBy         ID                 `json:"requested_by"`
	RequestedAt         time.Time          `json:"requested_at"`
	AffectedCampaignIDs []ID               `json:"affected_campaign_ids"`
	ActivatedFallbacks  map[ID]ID          `json:"activated_fallbacks"`
	ResolvedAt          *time.Time         `json:"resolved_at,omitempty"`
}

type FallbackCandidate struct {
	PackageID   ID         `json:"package_id"`
	PlacementID ID         `json:"placement_id"`
	AudienceID  ID         `json:"audience_id"`
	Priority    int        `json:"priority"`
	Window      TimeWindow `json:"window"`
	Enabled     bool       `json:"enabled"`
}

func (c FallbackCandidate) Eligible(placement, audience ID, at time.Time) bool {
	return c.Enabled && c.PlacementID == placement && c.AudienceID == audience && c.Window.Contains(at)
}
