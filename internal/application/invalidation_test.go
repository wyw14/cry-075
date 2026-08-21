package application

import (
	"context"
	"testing"

	"github.com/wyw14/cry-075/internal/domain"
)

func TestEmergencyTakedownAtomicallyActivatesFallback(t *testing.T) {
	services := setupServices(t)
	campaign, _, asset, placement, audience := seedComposition(t, services, "紧急专题")
	ctx := context.Background()
	stored, _ := services.store.GetCampaign(ctx, campaign.ID)
	stored.Status = domain.StatusPublished
	stored.Version++
	if err := services.store.UpdateCampaign(ctx, stored, campaign.Version); err != nil {
		t.Fatal(err)
	}
	operator := domain.Actor{ID: "operator", Role: domain.RoleOperator}
	fallback, err := services.catalog.CreatePackage(ctx, "默认兜底", "production", true, operator, "fallback-package")
	if err != nil {
		t.Fatal(err)
	}
	fallbackAsset, err := services.catalog.CreateAsset(ctx, "默认视觉", "image/png", "fallback.png", nil, operator, "fallback-asset")
	if err != nil {
		t.Fatal(err)
	}
	fallback, err = services.catalog.AddItem(ctx, AddPackageItemCommand{PackageID: fallback.ID, AssetID: fallbackAsset.ID, ExpectedVersion: fallback.Version, Actor: operator, RequestID: "fallback-item"})
	if err != nil {
		t.Fatal(err)
	}
	if err := services.store.SaveFallback(ctx, domain.FallbackCandidate{PackageID: fallback.ID, PlacementID: placement.ID, AudienceID: audience.ID, Priority: 1, Window: stored.Window, Enabled: true}); err != nil {
		t.Fatal(err)
	}
	publisher := domain.Actor{ID: "publisher", Role: domain.RolePublisher}
	event, err := services.invalidation.EmergencyTakedown(ctx, asset.ID, domain.InvalidEmergency, publisher, "emergency")
	if err != nil {
		t.Fatal(err)
	}
	updated, _ := services.store.GetCampaign(ctx, campaign.ID)
	if updated.PackageID != fallback.ID {
		t.Fatalf("fallback not activated: %s", updated.PackageID)
	}
	if event.ActivatedFallbacks[campaign.ID] != fallback.ID {
		t.Fatal("event must record activated fallback")
	}
}

func TestEmergencyTakedownRollsBackWhenNoFallbackExists(t *testing.T) {
	services := setupServices(t)
	campaign, _, asset, _, _ := seedComposition(t, services, "无兜底专题")
	ctx := context.Background()
	stored, _ := services.store.GetCampaign(ctx, campaign.ID)
	stored.Status = domain.StatusPublished
	stored.Version++
	if err := services.store.UpdateCampaign(ctx, stored, campaign.Version); err != nil {
		t.Fatal(err)
	}
	publisher := domain.Actor{ID: "publisher", Role: domain.RolePublisher}
	if _, err := services.invalidation.EmergencyTakedown(ctx, asset.ID, domain.InvalidEmergency, publisher, "emergency-no-fallback"); err == nil {
		t.Fatal("takedown should fail without fallback")
	}
	unchanged, err := services.store.GetAsset(ctx, asset.ID)
	if err != nil {
		t.Fatal(err)
	}
	if unchanged.State != domain.AssetActive || unchanged.Version != asset.Version {
		t.Fatalf("failed transaction leaked asset mutation: %#v", unchanged)
	}
}
