package application

import (
	"context"
	"testing"
	"time"

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

// TestPreviewHonorsAudienceWhenMemberCampaignWasUpdatedLater guards against a
// regression where a freshly-updated member campaign on the same placement would
// win the preview over a matching visitor campaign because the audience was
// never compared against the request.
func TestPreviewHonorsAudienceWhenMemberCampaignWasUpdatedLater(t *testing.T) {
	services := setupServices(t)
	ctx := context.Background()

	visitor, _, _, placement, _ := seedComposition(t, services, "访客专题")
	publishCampaign(t, services, visitor)

	// A second campaign on the SAME placement targets members only.
	member, _, _, _, _ := seedComposition(t, services, "会员专题")
	member.PlacementID = placement.ID
	memberAudience := domain.Audience{ID: domain.NewID("aud"), Name: "会员", Attributes: map[string]string{"tier": "member"}}
	if err := services.store.PutAudience(ctx, memberAudience); err != nil {
		t.Fatal(err)
	}
	member.AudienceID = memberAudience.ID
	current, _ := services.store.GetCampaign(ctx, member.ID)
	member.Version = current.Version + 1
	if err := services.store.UpdateCampaign(ctx, member, current.Version); err != nil {
		t.Fatal(err)
	}
	publishCampaign(t, services, member)

	// Touch the member campaign AFTER the visitor campaign so that, without the
	// audience filter, it would sort first by updated_at:desc.
	bumpUpdatedAt(t, services, member, services.clock.at.Add(time.Second))

	// A plain visitor preview must select the visitor campaign, not the member one.
	result, err := services.preview.Render(ctx, domain.PreviewRequest{
		Environment: "production",
		At:          services.clock.at,
		Audience:    map[string]string{},
		PlacementID: placement.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Empty {
		t.Fatalf("preview was empty: %#v", result)
	}
	if result.CampaignID != visitor.ID {
		t.Fatalf("visitor preview selected %s, want %s", result.CampaignID, visitor.ID)
	}
	if result.CampaignID == member.ID {
		t.Fatalf("visitor preview wrongly selected the member campaign %s; audience was ignored", member.ID)
	}
}

func publishCampaign(t *testing.T, services testServices, campaign domain.Campaign) {
	t.Helper()
	ctx := context.Background()
	stored, err := services.store.GetCampaign(ctx, campaign.ID)
	if err != nil {
		t.Fatal(err)
	}
	stored.Status = domain.StatusPublished
	stored.Version++
	if err := services.store.UpdateCampaign(ctx, stored, campaign.Version); err != nil {
		t.Fatal(err)
	}
}

func bumpUpdatedAt(t *testing.T, services testServices, campaign domain.Campaign, at time.Time) {
	t.Helper()
	ctx := context.Background()
	stored, err := services.store.GetCampaign(ctx, campaign.ID)
	if err != nil {
		t.Fatal(err)
	}
	stored.UpdatedAt = at.UTC()
	stored.Version++
	if err := services.store.UpdateCampaign(ctx, stored, stored.Version-1); err != nil {
		t.Fatal(err)
	}
}
