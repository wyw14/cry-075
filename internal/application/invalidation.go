package application

import (
	"context"
	"fmt"

	"github.com/wyw14/cry-075/internal/domain"
	"github.com/wyw14/cry-075/internal/repository"
)

type InvalidationService struct {
	Campaigns       repository.CampaignRepository
	Catalog         repository.CatalogRepository
	Releases        repository.ReleaseRepository
	AuditRepository repository.AuditRepository
	Transactions    repository.UnitOfWork
	Audit           AuditWriter
	Clock           Clock
}

func (s InvalidationService) EmergencyTakedown(ctx context.Context, assetID domain.ID, reason domain.InvalidationReason, actor domain.Actor, requestID string) (domain.InvalidationEvent, error) {
	permission := "asset.invalidate"
	if reason == domain.InvalidEmergency {
		permission = "release.emergency"
	}
	if err := require(actor, permission); err != nil {
		return domain.InvalidationEvent{}, err
	}
	asset, err := s.Catalog.GetAsset(ctx, assetID)
	if err != nil {
		return domain.InvalidationEvent{}, err
	}
	before := asset.Version
	asset.Invalidate(string(reason), s.Clock())
	page, err := s.Campaigns.ListCampaigns(ctx, domain.ListQuery{Page: 1, PerPage: 100, Sort: "updated_at:desc", Filters: map[string]string{"status": string(domain.StatusPublished)}})
	if err != nil {
		return domain.InvalidationEvent{}, err
	}
	event := domain.InvalidationEvent{ID: domain.NewID("inv"), AssetID: assetID, Reason: reason, RequestedBy: actor.ID, RequestedAt: s.Clock(), AffectedCampaignIDs: []domain.ID{}, ActivatedFallbacks: map[domain.ID]domain.ID{}}
	err = s.Transactions.WithinTransaction(ctx, func(tx context.Context) error {
		if err := s.Catalog.UpdateAsset(tx, asset, before); err != nil {
			return err
		}
		for _, campaign := range page.Items {
			pack, err := s.Catalog.GetPackage(tx, campaign.PackageID)
			if err != nil {
				return err
			}
			affected := false
			for _, item := range pack.Items {
				if item.AssetID == assetID || item.Replacement == assetID {
					affected = true
					break
				}
			}
			if !affected {
				continue
			}
			event.AffectedCampaignIDs = append(event.AffectedCampaignIDs, campaign.ID)
			fallbacks, err := s.Releases.ListFallbacks(tx, campaign.PlacementID, campaign.AudienceID, s.Clock())
			if err != nil {
				return err
			}
			chosen := domain.ID("")
			for _, candidate := range fallbacks {
				pack, err := s.Catalog.GetPackage(tx, candidate.PackageID)
				if err != nil {
					continue
				}
				if s.packageEligible(tx, pack) {
					chosen = pack.ID
					break
				}
			}
			if chosen.Empty() {
				return fmt.Errorf("%w for campaign %s", domain.ErrNoFallback, campaign.ID)
			}
			event.ActivatedFallbacks[campaign.ID] = chosen
			old := campaign.Version
			campaign.PackageID = chosen
			campaign.Version++
			if err := s.Campaigns.UpdateCampaign(tx, campaign, old); err != nil {
				return err
			}
		}
		resolved := s.Clock()
		event.ResolvedAt = &resolved
		if err := s.AuditRepository.AppendInvalidation(tx, event); err != nil {
			return err
		}
		return s.Audit.Record(tx, AuditChange{
			Actor: actor, Action: "asset.emergency_takedown", Subject: "asset", SubjectID: assetID, RequestID: requestID,
			Metadata: map[string]any{"reason": reason, "affected": event.AffectedCampaignIDs, "fallbacks": event.ActivatedFallbacks},
		})
	})
	if err != nil {
		return domain.InvalidationEvent{}, err
	}
	return event, nil
}

func (s InvalidationService) packageEligible(ctx context.Context, pack domain.AssetPackage) bool {
	if len(pack.Items) == 0 {
		return false
	}
	for _, item := range pack.Items {
		asset, err := s.Catalog.GetAsset(ctx, item.AssetID)
		if err != nil || !asset.Eligible(s.Clock()) {
			return false
		}
	}
	return true
}

func (s InvalidationService) ExpireRights(ctx context.Context, actor domain.Actor, assetIDs []domain.ID) error {
	for _, id := range assetIDs {
		asset, err := s.Catalog.GetAsset(ctx, id)
		if err != nil {
			return err
		}
		if asset.RightsEndsAt != nil && !asset.RightsEndsAt.After(s.Clock()) && asset.State == domain.AssetActive {
			if _, err := s.EmergencyTakedown(ctx, id, domain.InvalidCopyright, actor, "local-scheduler"); err != nil {
				return err
			}
		}
	}
	return nil
}
