package application

import (
	"context"
	"fmt"
	"time"

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
	if err := require(actor, invalidationPermission(reason)); err != nil {
		return domain.InvalidationEvent{}, err
	}
	asset, err := s.Catalog.GetAsset(ctx, assetID)
	if err != nil {
		return domain.InvalidationEvent{}, err
	}
	affected, err := s.affectedPublishedCampaigns(ctx, assetID)
	if err != nil {
		return domain.InvalidationEvent{}, err
	}
	// Mutate the asset in memory only; persisting it inside the transaction
	// below keeps the invalidation atomic with fallback activation. If a
	// fallback selection fails and the transaction rolls back, the asset is
	// restored too — otherwise it would be left invalid with no fallback and
	// the storefront would go empty despite the operation reporting an error.
	before := asset.Version
	asset.Invalidate(string(reason), s.Clock())
	event := newInvalidationEvent(assetID, reason, actor, s.Clock())
	err = s.Transactions.WithinTransaction(ctx, func(tx context.Context) error {
		if err := s.Catalog.UpdateAsset(tx, asset, before); err != nil {
			return err
		}
		for _, campaign := range affected {
			chosen, err := s.selectFallback(tx, campaign)
			if err != nil {
				return err
			}
			event.AffectedCampaignIDs = append(event.AffectedCampaignIDs, campaign.ID)
			event.ActivatedFallbacks[campaign.ID] = chosen
			if err := s.activateFallback(tx, campaign, chosen); err != nil {
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

func invalidationPermission(reason domain.InvalidationReason) string {
	if reason == domain.InvalidEmergency {
		return "release.emergency"
	}
	return "asset.invalidate"
}

func newInvalidationEvent(assetID domain.ID, reason domain.InvalidationReason, actor domain.Actor, at time.Time) domain.InvalidationEvent {
	return domain.InvalidationEvent{
		ID: domain.NewID("inv"), AssetID: assetID, Reason: reason, RequestedBy: actor.ID, RequestedAt: at,
		AffectedCampaignIDs: []domain.ID{}, ActivatedFallbacks: map[domain.ID]domain.ID{},
	}
}

func (s InvalidationService) affectedPublishedCampaigns(ctx context.Context, assetID domain.ID) ([]domain.Campaign, error) {
	page, err := s.Campaigns.ListCampaigns(ctx, domain.ListQuery{Page: 1, PerPage: 100, Sort: "updated_at:desc", Filters: map[string]string{"status": string(domain.StatusPublished)}})
	if err != nil {
		return nil, err
	}
	result := make([]domain.Campaign, 0)
	for _, campaign := range page.Items {
		pack, err := s.Catalog.GetPackage(ctx, campaign.PackageID)
		if err != nil {
			return nil, err
		}
		for _, item := range pack.Items {
			if item.AssetID == assetID || item.Replacement == assetID {
				result = append(result, campaign)
				break
			}
		}
	}
	return result, nil
}

func (s InvalidationService) selectFallback(ctx context.Context, campaign domain.Campaign) (domain.ID, error) {
	fallbacks, err := s.Releases.ListFallbacks(ctx, campaign.PlacementID, campaign.AudienceID, s.Clock())
	if err != nil {
		return "", err
	}
	for _, candidate := range fallbacks {
		pack, err := s.Catalog.GetPackage(ctx, candidate.PackageID)
		if err == nil && s.packageEligible(ctx, pack) {
			return pack.ID, nil
		}
	}
	return "", fmt.Errorf("%w for campaign %s", domain.ErrNoFallback, campaign.ID)
}

func (s InvalidationService) activateFallback(ctx context.Context, campaign domain.Campaign, packageID domain.ID) error {
	before := campaign.Version
	campaign.PackageID = packageID
	campaign.Version++
	return s.Campaigns.UpdateCampaign(ctx, campaign, before)
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
