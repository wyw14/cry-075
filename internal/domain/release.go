package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

type ReleaseVersion struct {
	ID             ID         `json:"id"`
	CampaignID     ID         `json:"campaign_id"`
	Environment    string     `json:"environment"`
	Sequence       int64      `json:"sequence"`
	SnapshotID     ID         `json:"snapshot_id"`
	ApprovedBy     ID         `json:"approved_by"`
	ApprovalReason string     `json:"approval_reason"`
	PublishedAt    *time.Time `json:"published_at,omitempty"`
	RolledBackFrom ID         `json:"rolled_back_from,omitempty"`
}

type ReleaseSnapshot struct {
	ID          ID           `json:"id"`
	CampaignID  ID           `json:"campaign_id"`
	Environment string       `json:"environment"`
	CreatedAt   time.Time    `json:"created_at"`
	CreatedBy   ID           `json:"created_by"`
	Checksum    string       `json:"checksum"`
	Campaign    Campaign     `json:"campaign"`
	Package     AssetPackage `json:"package"`
}

func NewSnapshot(campaign Campaign, pack AssetPackage, actor ID, now time.Time) (ReleaseSnapshot, error) {
	if campaign.Environment != pack.Environment || actor.Empty() {
		return ReleaseSnapshot{}, ErrInvalidReference
	}
	payload, err := json.Marshal(struct {
		Campaign Campaign
		Package  AssetPackage
	}{campaign, pack})
	if err != nil {
		return ReleaseSnapshot{}, fmt.Errorf("marshal snapshot: %w", err)
	}
	sum := sha256.Sum256(payload)
	return ReleaseSnapshot{ID: NewID("snp"), CampaignID: campaign.ID, Environment: campaign.Environment, CreatedAt: now.UTC(), CreatedBy: actor, Checksum: hex.EncodeToString(sum[:]), Campaign: campaign, Package: pack}, nil
}

type Approval struct {
	ID          ID               `json:"id"`
	CampaignID  ID               `json:"campaign_id"`
	Environment string           `json:"environment"`
	ReviewerID  ID               `json:"reviewer_id"`
	Decision    ApprovalDecision `json:"decision"`
	Comment     string           `json:"comment"`
	CreatedAt   time.Time        `json:"created_at"`
	Version     int64            `json:"version"`
}
