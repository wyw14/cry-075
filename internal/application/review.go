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
	command := reviewDecisionCommand{CampaignID: id, Decision: decision, Comment: comment, ExpectedVersion: expected, Actor: actor, RequestID: requestID}
	campaign, approval, err := s.prepareDecision(ctx, command)
	if err != nil {
		return domain.Campaign{}, err
	}
	err = s.Transactions.WithinTransaction(ctx, func(tx context.Context) error {
		if err := s.Campaigns.UpdateCampaign(tx, campaign, approval.Version); err != nil {
			return err
		}
		if err := s.Releases.SaveApproval(tx, approval); err != nil {
			return err
		}
		return s.recordDecisionAudit(tx, command)
	})
	if err != nil {
		return domain.Campaign{}, err
	}
	return campaign, nil
}

type reviewDecisionCommand struct {
	CampaignID      domain.ID
	Decision        domain.ApprovalDecision
	Comment         string
	ExpectedVersion int64
	Actor           domain.Actor
	RequestID       string
}

func (s ReviewService) prepareDecision(ctx context.Context, command reviewDecisionCommand) (domain.Campaign, domain.Approval, error) {
	campaign, err := s.Campaigns.GetCampaign(ctx, command.CampaignID)
	if err != nil {
		return domain.Campaign{}, domain.Approval{}, err
	}
	next, err := decisionStatus(command.Decision)
	if err != nil {
		return domain.Campaign{}, domain.Approval{}, err
	}
	if command.Decision == domain.DecisionApprove {
		if err := s.checkDecisionConflicts(ctx, campaign); err != nil {
			return domain.Campaign{}, domain.Approval{}, err
		}
	}
	before := campaign.Version
	if err := campaign.Transition(next, command.ExpectedVersion); err != nil {
		return domain.Campaign{}, domain.Approval{}, err
	}
	approval := domain.Approval{
		ID:          domain.NewID("apr"),
		CampaignID:  command.CampaignID,
		Environment: campaign.Environment,
		ReviewerID:  command.Actor.ID,
		Decision:    command.Decision,
		Comment:     command.Comment,
		CreatedAt:   s.Clock(),
		Version:     before,
	}
	return campaign, approval, nil
}

func decisionStatus(decision domain.ApprovalDecision) (domain.CampaignStatus, error) {
	switch decision {
	case domain.DecisionApprove:
		return domain.StatusPublished, nil
	case domain.DecisionReject:
		return domain.StatusDraft, nil
	default:
		return "", fmt.Errorf("invalid decision")
	}
}

func (s ReviewService) checkDecisionConflicts(ctx context.Context, campaign domain.Campaign) error {
	conflicts, err := s.Campaigns.FindPublishedOverlaps(ctx, campaign.PlacementID, campaign.Environment, campaign.Window, campaign.ID)
	if err != nil {
		return err
	}
	if len(conflicts) > 0 {
		return fmt.Errorf("%w: placement window occupied", domain.ErrConflict)
	}
	return nil
}

func (s ReviewService) recordDecisionAudit(ctx context.Context, command reviewDecisionCommand) error {
	return s.Audit.Record(ctx, AuditChange{
		Actor: command.Actor, Action: "campaign.review_decided", Subject: "campaign", SubjectID: command.CampaignID, RequestID: command.RequestID,
		Metadata: map[string]any{"decision": command.Decision, "comment": command.Comment},
	})
}
