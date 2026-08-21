package application

import (
	"context"
	"testing"

	"github.com/wyw14/cry-075/internal/domain"
)

func TestPreviewShowsOnlyEligibleAudienceAndAssets(t *testing.T) {
	services := setupServices(t)
	campaign, _, _, placement, _ := seedComposition(t, services, "前台专题")
	ctx := context.Background()
	stored, _ := services.store.GetCampaign(ctx, campaign.ID)
	stored.Status = domain.StatusPublished
	stored.Version++
	if err := services.store.UpdateCampaign(ctx, stored, campaign.Version); err != nil {
		t.Fatal(err)
	}
	result, err := services.preview.Render(ctx, domain.PreviewRequest{Environment: "production", At: services.clock.at, Audience: map[string]string{}, PlacementID: placement.ID})
	if err != nil {
		t.Fatal(err)
	}
	if result.Empty || len(result.Items) != 1 {
		t.Fatalf("unexpected preview: %#v", result)
	}
	if result.CampaignID != campaign.ID {
		t.Fatal("preview selected wrong campaign")
	}
}
