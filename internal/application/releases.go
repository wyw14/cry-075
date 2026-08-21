package application

import (
	"context"
	"fmt"

	"github.com/wyw14/cry-075/internal/domain"
	"github.com/wyw14/cry-075/internal/repository"
)

type ReleaseService struct {
	Campaigns    repository.CampaignRepository
	Catalog      repository.CatalogRepository
	Releases     repository.ReleaseRepository
	Transactions repository.UnitOfWork
	Audit        AuditWriter
	Clock        Clock
}

func (s ReleaseService) Snapshot(ctx context.Context, campaignID domain.ID, actor domain.Actor, requestID string) (domain.ReleaseSnapshot, error) {
	if err := require(actor, "release.publish"); err != nil {
		return domain.ReleaseSnapshot{}, err
	}
	campaign, err := s.Campaigns.GetCampaign(ctx, campaignID)
	if err != nil {
		return domain.ReleaseSnapshot{}, err
	}
	pack, err := s.Catalog.GetPackage(ctx, campaign.PackageID)
	if err != nil {
		return domain.ReleaseSnapshot{}, err
	}
	snapshot, err := domain.NewSnapshot(campaign, pack, actor.ID, s.Clock())
	if err != nil {
		return domain.ReleaseSnapshot{}, err
	}
	err = s.Transactions.WithinTransaction(ctx, func(tx context.Context) error {
		if err := s.Releases.SaveSnapshot(tx, snapshot); err != nil {
			return err
		}
		return s.Audit.Record(tx, AuditChange{Actor: actor, Action: "release.snapshot_created", Subject: "campaign", SubjectID: campaignID, RequestID: requestID, Metadata: map[string]any{"snapshot_id": snapshot.ID, "checksum": snapshot.Checksum}})
	})
	return snapshot, err
}

func (s ReleaseService) Publish(ctx context.Context, snapshotID domain.ID, reason string, actor domain.Actor, requestID string) (domain.ReleaseVersion, error) {
	if err := require(actor, "release.publish"); err != nil {
		return domain.ReleaseVersion{}, err
	}
	snapshot, err := s.Releases.GetSnapshot(ctx, snapshotID)
	if err != nil {
		return domain.ReleaseVersion{}, err
	}
	campaign, err := s.Campaigns.GetCampaign(ctx, snapshot.CampaignID)
	if err != nil {
		return domain.ReleaseVersion{}, err
	}
	if campaign.Status != domain.StatusPublished || campaign.Version != snapshot.Campaign.Version {
		return domain.ReleaseVersion{}, fmt.Errorf("%w: snapshot is stale or campaign is not published", domain.ErrVersionConflict)
	}
	sequence := int64(1)
	if latest, err := s.Releases.LatestRelease(ctx, campaign.ID, campaign.Environment); err == nil {
		sequence = latest.Sequence + 1
	}
	now := s.Clock()
	release := domain.ReleaseVersion{ID: domain.NewID("rel"), CampaignID: campaign.ID, Environment: campaign.Environment, Sequence: sequence, SnapshotID: snapshot.ID, ApprovedBy: actor.ID, ApprovalReason: reason, PublishedAt: &now}
	err = s.Transactions.WithinTransaction(ctx, func(tx context.Context) error {
		if err := s.Releases.SaveRelease(tx, release); err != nil {
			return err
		}
		return s.Audit.Record(tx, AuditChange{Actor: actor, Action: "release.published", Subject: "campaign", SubjectID: campaign.ID, RequestID: requestID, Metadata: map[string]any{"release_id": release.ID, "sequence": sequence, "snapshot_id": snapshot.ID}})
	})
	return release, err
}

func (s ReleaseService) Rollback(ctx context.Context, targetID domain.ID, actor domain.Actor, requestID string) (domain.ReleaseVersion, error) {
	if err := require(actor, "release.rollback"); err != nil {
		return domain.ReleaseVersion{}, err
	}
	plan, err := s.prepareRollback(ctx, targetID, actor)
	if err != nil {
		return domain.ReleaseVersion{}, err
	}
	err = s.Transactions.WithinTransaction(ctx, func(tx context.Context) error {
		if err := s.Campaigns.UpdateCampaign(tx, plan.Restored, plan.Current.Version); err != nil {
			return err
		}
		if err := s.Releases.SaveRelease(tx, plan.Release); err != nil {
			return err
		}
		return s.Audit.Record(tx, AuditChange{
			Actor: actor, Action: "release.rolled_back", Subject: "campaign", SubjectID: plan.Current.ID, RequestID: requestID,
			Metadata: map[string]any{"target_snapshot": plan.Target.ID, "release_id": plan.Release.ID},
		})
	})
	return plan.Release, err
}

type rollbackPlan struct {
	Target   domain.ReleaseSnapshot
	Current  domain.Campaign
	Restored domain.Campaign
	Release  domain.ReleaseVersion
}

func (s ReleaseService) prepareRollback(ctx context.Context, targetID domain.ID, actor domain.Actor) (rollbackPlan, error) {
	target, err := s.Releases.GetSnapshot(ctx, targetID)
	if err != nil {
		return rollbackPlan{}, err
	}
	current, err := s.Campaigns.GetCampaign(ctx, target.CampaignID)
	if err != nil {
		return rollbackPlan{}, err
	}
	restored := restoreSnapshotCampaign(target, current)
	sequence, err := s.nextRollbackSequence(ctx, current)
	if err != nil {
		return rollbackPlan{}, err
	}
	release := s.newRollbackVersion(target, current, actor, sequence)
	return rollbackPlan{Target: target, Current: current, Restored: restored, Release: release}, nil
}

func restoreSnapshotCampaign(target domain.ReleaseSnapshot, current domain.Campaign) domain.Campaign {
	restored := target.Campaign
	restored.Version = current.Version + 1
	return restored
}

func (s ReleaseService) nextRollbackSequence(ctx context.Context, current domain.Campaign) (int64, error) {
	latest, err := s.Releases.LatestRelease(ctx, current.ID, current.Environment)
	if err != nil {
		return 1, nil
	}
	return latest.Sequence + 1, nil
}

func (s ReleaseService) newRollbackVersion(target domain.ReleaseSnapshot, current domain.Campaign, actor domain.Actor, sequence int64) domain.ReleaseVersion {
	now := s.Clock()
	return domain.ReleaseVersion{
		ID:             domain.NewID("rel"),
		CampaignID:     current.ID,
		Environment:    target.Environment,
		Sequence:       sequence,
		SnapshotID:     target.ID,
		ApprovedBy:     actor.ID,
		ApprovalReason: "rollback",
		PublishedAt:    &now,
		RolledBackFrom: target.ID,
	}
}
