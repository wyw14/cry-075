package application

import (
	"context"
	"sort"
	"time"

	"github.com/wyw14/cry-075/internal/domain"
	"github.com/wyw14/cry-075/internal/repository"
)

type PreviewService struct {
	Campaigns repository.CampaignRepository
	Catalog   repository.CatalogRepository
	Releases  repository.ReleaseRepository
	Clock     Clock
}

func (s PreviewService) Render(ctx context.Context, request domain.PreviewRequest) (domain.PreviewResult, error) {
	request = normalizePreviewRequest(request, s.Clock)
	page, err := s.loadPreviewCampaigns(ctx, request)
	if err != nil {
		return domain.PreviewResult{}, err
	}
	plan, err := s.buildPreviewPlan(ctx, page.Items, request)
	if err != nil {
		return domain.PreviewResult{}, err
	}
	if len(plan) == 0 {
		return s.renderFallback(ctx, request, "no active campaign")
	}
	for _, entry := range plan {
		result, err := s.renderCampaign(ctx, entry.Campaign, request.At, false)
		if err != nil {
			return domain.PreviewResult{}, err
		}
		if !result.Empty {
			return result, nil
		}
	}
	return s.renderFallback(ctx, request, "active campaigns have no eligible assets")
}

type previewPlanEntry struct {
	Campaign domain.Campaign
	Rank     int
}

func normalizePreviewRequest(request domain.PreviewRequest, clock Clock) domain.PreviewRequest {
	if request.At.IsZero() {
		request.At = clock()
	}
	request.At = request.At.UTC()
	if request.Audience == nil {
		request.Audience = map[string]string{}
	}
	return request
}

func (s PreviewService) loadPreviewCampaigns(ctx context.Context, request domain.PreviewRequest) (domain.Page[domain.Campaign], error) {
	return s.Campaigns.ListCampaigns(ctx, domain.ListQuery{
		Page:    1,
		PerPage: 100,
		Sort:    "updated_at:desc",
		Filters: map[string]string{
			"status":      string(domain.StatusPublished),
			"environment": request.Environment,
		},
	})
}

func (s PreviewService) buildPreviewPlan(ctx context.Context, campaigns []domain.Campaign, request domain.PreviewRequest) ([]previewPlanEntry, error) {
	entries := make([]previewPlanEntry, 0, len(campaigns))
	for _, campaign := range campaigns {
		if campaign.PlacementID != request.PlacementID {
			continue
		}
		if !campaign.Window.Contains(request.At) {
			continue
		}
		audience, err := s.Catalog.GetAudience(ctx, campaign.AudienceID)
		if err != nil {
			return nil, err
		}
		if !audience.Matches(request.Audience) {
			continue
		}
		entries = append(entries, previewPlanEntry{Campaign: campaign, Rank: len(entries)})
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Campaign.UpdatedAt.Equal(entries[j].Campaign.UpdatedAt) {
			return entries[i].Rank < entries[j].Rank
		}
		return entries[i].Campaign.UpdatedAt.After(entries[j].Campaign.UpdatedAt)
	})
	return entries, nil
}

func (s PreviewService) renderCampaign(ctx context.Context, campaign domain.Campaign, at time.Time, fallback bool) (domain.PreviewResult, error) {
	pack, err := s.Catalog.GetPackage(ctx, campaign.PackageID)
	if err != nil {
		return domain.PreviewResult{}, err
	}
	items := make([]domain.PreviewItem, 0, len(pack.Items))
	for _, entry := range pack.Items {
		asset, err := s.Catalog.GetAsset(ctx, entry.AssetID)
		if err != nil {
			return domain.PreviewResult{}, err
		}
		selected := asset
		if !asset.Eligible(at) && !entry.Replacement.Empty() {
			selected, err = s.Catalog.GetAsset(ctx, entry.Replacement)
			if err != nil {
				return domain.PreviewResult{}, err
			}
		}
		if !selected.Eligible(at) {
			continue
		}
		items = append(items, domain.PreviewItem{AssetID: selected.ID, Name: selected.Name, StorageKey: selected.StorageKey, Position: entry.Position, Fallback: fallback})
	}
	return domain.PreviewResult{
		PreviewTarget:       domain.PreviewTarget{CampaignID: campaign.ID, PackageID: pack.ID, Placement: campaign.PlacementID},
		PreviewPresentation: domain.PreviewPresentation{Items: items, Empty: len(items) == 0}, Generated: s.Clock(),
	}, nil
}

func (s PreviewService) renderFallback(ctx context.Context, request domain.PreviewRequest, reason string) (domain.PreviewResult, error) {
	page, err := s.Campaigns.ListCampaigns(ctx, domain.ListQuery{Page: 1, PerPage: 100, Sort: "updated_at:desc", Filters: map[string]string{"environment": request.Environment}})
	if err != nil {
		return domain.PreviewResult{}, err
	}
	for _, campaign := range page.Items {
		if campaign.PlacementID != request.PlacementID {
			continue
		}
		audience, err := s.Catalog.GetAudience(ctx, campaign.AudienceID)
		if err != nil || !audience.Matches(request.Audience) {
			continue
		}
		fallbacks, err := s.Releases.ListFallbacks(ctx, campaign.PlacementID, campaign.AudienceID, request.At)
		if err != nil {
			return domain.PreviewResult{}, err
		}
		for _, candidate := range fallbacks {
			pack, err := s.Catalog.GetPackage(ctx, candidate.PackageID)
			if err != nil || !pack.Fallback {
				continue
			}
			shadow := campaign
			shadow.PackageID = pack.ID
			result, err := s.renderCampaign(ctx, shadow, request.At, true)
			if err == nil && !result.Empty {
				result.Reason = reason
				return result, nil
			}
		}
	}
	return domain.PreviewResult{
		PreviewTarget:       domain.PreviewTarget{Placement: request.PlacementID},
		PreviewPresentation: domain.PreviewPresentation{Items: []domain.PreviewItem{}, Empty: true, Reason: reason}, Generated: s.Clock(),
	}, nil
}

func (s PreviewService) Analyze(ctx context.Context, campaignID domain.ID) (domain.ImpactAnalysis, error) {
	campaign, err := s.Campaigns.GetCampaign(ctx, campaignID)
	if err != nil {
		return domain.ImpactAnalysis{}, err
	}
	pack, err := s.Catalog.GetPackage(ctx, campaign.PackageID)
	if err != nil {
		return domain.ImpactAnalysis{}, err
	}
	assets := make([]domain.ID, 0, len(pack.Items))
	warnings := make([]string, 0)
	for _, item := range pack.Items {
		assets = append(assets, item.AssetID)
		asset, err := s.Catalog.GetAsset(ctx, item.AssetID)
		if err != nil || !asset.Eligible(s.Clock()) {
			warnings = append(warnings, "素材不可用: "+string(item.AssetID))
		}
	}
	overlaps, err := s.Campaigns.FindPublishedOverlaps(ctx, campaign.PlacementID, campaign.Environment, campaign.Window, campaign.ID)
	if err != nil {
		return domain.ImpactAnalysis{}, err
	}
	conflicts := make([]domain.ID, 0, len(overlaps))
	for _, item := range overlaps {
		conflicts = append(conflicts, item.ID)
	}
	analysis := domain.ImpactAnalysis{
		ImpactSubject: domain.ImpactSubject{CampaignID: campaign.ID, AffectedAssets: assets},
		ImpactRisks:   domain.ImpactRisks{Conflicts: conflicts, Warnings: warnings}, AudienceEstimate: int64(len(assets)) * 1000,
	}
	fallbacks, err := s.Releases.ListFallbacks(ctx, campaign.PlacementID, campaign.AudienceID, s.Clock())
	if err == nil && len(fallbacks) > 0 {
		analysis.FallbackPackage = fallbacks[0].PackageID
	}
	return analysis, nil
}
