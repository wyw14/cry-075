package application

import (
	"context"
	"fmt"

	"github.com/wyw14/cry-075/internal/domain"
	"github.com/wyw14/cry-075/internal/repository"
)

type ReviewService struct {
	Campaigns    repository.CampaignRepository
	Catalog      repository.CatalogRepository
	Releases     repository.ReleaseRepository
	Transactions repository.UnitOfWork
	Audit        AuditWriter
	Clock        Clock
}

func (s ReviewService) Submit(ctx context.Context, id domain.ID, expected int64, actor domain.Actor, requestID string) (domain.Campaign, error) {
	if err := require(actor, "campaign.edit"); err != nil {
		return domain.Campaign{}, err
	}
	campaign, err := s.Campaigns.GetCampaign(ctx, id)
	if err != nil {
		return domain.Campaign{}, err
	}
	pack, err := s.Catalog.GetPackage(ctx, campaign.PackageID)
	if err != nil {
		return domain.Campaign{}, err
	}
	if len(pack.Items) == 0 {
		return domain.Campaign{}, fmt.Errorf("%w: package is empty", domain.ErrInvalidReference)
	}
	for _, item := range pack.Items {
		asset, err := s.Catalog.GetAsset(ctx, item.AssetID)
		if err != nil || !asset.Eligible(s.Clock()) {
			return domain.Campaign{}, fmt.Errorf("%w: package contains unavailable asset", domain.ErrInvalidReference)
		}
	}
	before := campaign.Version
	if err = campaign.Transition(domain.StatusReview, expected); err != nil {
		return domain.Campaign{}, err
	}
	err = s.Transactions.WithinTransaction(ctx, func(tx context.Context) error {
		if err := s.Campaigns.UpdateCampaign(tx, campaign, before); err != nil {
			return err
		}
		return s.Audit.Record(tx, AuditChange{Actor: actor, Action: "campaign.review_submitted", Subject: "campaign", SubjectID: campaign.ID, RequestID: requestID, Metadata: map[string]any{"version": campaign.Version}})
	})
	return campaign, err
}

func (s ReviewService) Decide(ctx context.Context, id domain.ID, decision domain.ApprovalDecision, comment string, expected int64, actor domain.Actor, requestID string) (domain.Campaign, error) {
	if err := require(actor, "review.decide"); err != nil {
		return domain.Campaign{}, err
	}
	campaign, err := s.Campaigns.GetCampaign(ctx, id)
	if err != nil {
		return domain.Campaign{}, err
	}
	next := domain.StatusPublished
	if decision == domain.DecisionReject {
		next = domain.StatusDraft
	} else if decision != domain.DecisionApprove {
		return domain.Campaign{}, fmt.Errorf("invalid decision")
	}
	if decision == domain.DecisionApprove {
		conflicts, err := s.Campaigns.FindPublishedOverlaps(ctx, campaign.PlacementID, campaign.Environment, campaign.Window, campaign.ID)
		if err != nil {
			return domain.Campaign{}, err
		}
		if len(conflicts) > 0 {
			return domain.Campaign{}, fmt.Errorf("%w: placement window occupied", domain.ErrConflict)
		}
	}
	before := campaign.Version
	if err = campaign.Transition(next, expected); err != nil {
		return domain.Campaign{}, err
	}
	approval := domain.Approval{ID: domain.NewID("apr"), CampaignID: id, Environment: campaign.Environment, ReviewerID: actor.ID, Decision: decision, Comment: comment, CreatedAt: s.Clock(), Version: before}
	err = s.Transactions.WithinTransaction(ctx, func(tx context.Context) error {
		if err := s.Campaigns.UpdateCampaign(tx, campaign, before); err != nil {
			return err
		}
		if err := s.Releases.SaveApproval(tx, approval); err != nil {
			return err
		}
		return s.Audit.Record(tx, AuditChange{Actor: actor, Action: "campaign.review_decided", Subject: "campaign", SubjectID: id, RequestID: requestID, Metadata: map[string]any{"decision": decision, "comment": comment}})
	})
	return campaign, err
}
