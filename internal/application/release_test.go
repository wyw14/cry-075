package application

import (
	"context"
	"testing"

	"github.com/wyw14/cry-075/internal/domain"
)

func TestRollbackCreatesNewVersionWithoutChangingEnvironment(t *testing.T) {
	services := setupServices(t)
	campaign, _, _, _, _ := seedComposition(t, services, "版本专题")
	ctx := context.Background()
	stored, _ := services.store.GetCampaign(ctx, campaign.ID)
	stored.Status = domain.StatusPublished
	stored.Version++
	if err := services.store.UpdateCampaign(ctx, stored, campaign.Version); err != nil {
		t.Fatal(err)
	}
	publisher := domain.Actor{ID: "publisher", Role: domain.RolePublisher}
	snapshot, err := services.releases.Snapshot(ctx, stored.ID, publisher, "snapshot")
	if err != nil {
		t.Fatal(err)
	}
	release, err := services.releases.Rollback(ctx, snapshot.ID, publisher, "rollback")
	if err != nil {
		t.Fatal(err)
	}
	updated, _ := services.store.GetCampaign(ctx, campaign.ID)
	if updated.Environment != "production" {
		t.Fatal("rollback crossed environment")
	}
	if updated.Version <= stored.Version {
		t.Fatal("rollback must create a newer campaign version")
	}
	if release.RolledBackFrom != snapshot.ID {
		t.Fatal("release must link rollback source")
	}
}
