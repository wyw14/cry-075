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
	target, err := s.Releases.GetSnapshot(ctx, targetID)
	if err != nil {
		return domain.ReleaseVersion{}, err
	}
	current, err := s.Campaigns.GetCampaign(ctx, target.CampaignID)
	if err != nil {
		return domain.ReleaseVersion{}, err
	}
	if current.Environment != target.Environment {
		return domain.ReleaseVersion{}, fmt.Errorf("%w: cross-environment rollback", domain.ErrInvalidReference)
	}
	restored := target.Campaign
	restored.Version = current.Version + 1
	before := current.Version
	sequence := int64(1)
	if latest, err := s.Releases.LatestRelease(ctx, current.ID, current.Environment); err == nil {
		sequence = latest.Sequence + 1
	}
	now := s.Clock()
	release := domain.ReleaseVersion{ID: domain.NewID("rel"), CampaignID: current.ID, Environment: current.Environment, Sequence: sequence, SnapshotID: target.ID, ApprovedBy: actor.ID, ApprovalReason: "rollback", PublishedAt: &now, RolledBackFrom: target.ID}
	err = s.Transactions.WithinTransaction(ctx, func(tx context.Context) error {
		if err := s.Campaigns.UpdateCampaign(tx, restored, before); err != nil {
			return err
		}
		if err := s.Releases.SaveRelease(tx, release); err != nil {
			return err
		}
		return s.Audit.Record(tx, AuditChange{Actor: actor, Action: "release.rolled_back", Subject: "campaign", SubjectID: current.ID, RequestID: requestID, Metadata: map[string]any{"target_snapshot": target.ID, "release_id": release.ID}})
	})
	return release, err
}
