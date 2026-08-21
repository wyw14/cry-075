package domain

import "time"

type PreviewRequest struct {
	Environment string            `json:"environment"`
	At          time.Time         `json:"at"`
	Audience    map[string]string `json:"audience"`
	PlacementID ID                `json:"placement_id"`
}

type PreviewItem struct {
	AssetID    ID     `json:"asset_id"`
	Name       string `json:"name"`
	StorageKey string `json:"storage_key"`
	Position   int    `json:"position"`
	Fallback   bool   `json:"fallback"`
}

type PreviewTarget struct {
	CampaignID ID `json:"campaign_id,omitempty"`
	PackageID  ID `json:"package_id,omitempty"`
	Placement  ID `json:"placement_id"`
}

type PreviewPresentation struct {
	Items  []PreviewItem `json:"items"`
	Empty  bool          `json:"empty"`
	Reason string        `json:"reason,omitempty"`
}

type PreviewResult struct {
	PreviewTarget
	PreviewPresentation
	Generated time.Time `json:"generated_at"`
}

type ImpactSubject struct {
	CampaignID     ID   `json:"campaign_id"`
	AffectedAssets []ID `json:"affected_assets"`
}

type ImpactRisks struct {
	Conflicts []ID     `json:"conflicting_campaign_ids"`
	Warnings  []string `json:"warnings"`
}

type ImpactAnalysis struct {
	ImpactSubject
	ImpactRisks
	FallbackPackage  ID    `json:"fallback_package_id,omitempty"`
	AudienceEstimate int64 `json:"audience_estimate"`
}
