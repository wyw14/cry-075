package domain

import "fmt"

type CampaignStatus string

const (
	StatusDraft     CampaignStatus = "draft"
	StatusReview    CampaignStatus = "pending_review"
	StatusPublished CampaignStatus = "published"
	StatusPaused    CampaignStatus = "paused"
	StatusExpired   CampaignStatus = "expired"
)

func (s CampaignStatus) CanTransition(next CampaignStatus) bool {
	switch s {
	case StatusDraft:
		return next == StatusReview
	case StatusReview:
		return next == StatusDraft || next == StatusPublished
	case StatusPublished:
		return next == StatusPaused || next == StatusExpired
	case StatusPaused:
		return next == StatusReview || next == StatusExpired
	case StatusExpired:
		return false
	default:
		return false
	}
}

func (s CampaignStatus) Validate() error {
	switch s {
	case StatusDraft, StatusReview, StatusPublished, StatusPaused, StatusExpired:
		return nil
	default:
		return fmt.Errorf("unknown campaign status %q", s)
	}
}

type ApprovalDecision string

const (
	DecisionApprove ApprovalDecision = "approve"
	DecisionReject  ApprovalDecision = "reject"
)
