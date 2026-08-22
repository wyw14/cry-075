package application

import (
	"context"
	"errors"
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

// 回滚只能发生在相同环境中：staging 专题不能被 production 快照覆盖回去。
func TestRollbackRejectsCrossEnvironmentSnapshot(t *testing.T) {
	services := setupServices(t)
	campaign, pack, _, _, _ := seedComposition(t, services, "跨环境专题")
	ctx := context.Background()

	// staging 专题：把当前专题改成 staging 并发布。
	stored, _ := services.store.GetCampaign(ctx, campaign.ID)
	stored.Environment = "staging"
	stored.Status = domain.StatusPublished
	stored.Version++
	if err := services.store.UpdateCampaign(ctx, stored, campaign.Version); err != nil {
		t.Fatal(err)
	}
	stagingPack := pack
	stagingPack.Environment = "staging"

	// 误选以前 production 专题的发布快照执行回滚。
	productionSnapshot, err := domain.NewSnapshot(campaign, pack, "publisher", services.clock.at)
	if err != nil {
		t.Fatal(err)
	}
	if err := services.store.SaveSnapshot(ctx, productionSnapshot); err != nil {
		t.Fatal(err)
	}

	publisher := domain.Actor{ID: "publisher", Role: domain.RolePublisher}
	if _, err := services.releases.Rollback(ctx, productionSnapshot.ID, publisher, "rollback"); err == nil {
		t.Fatal("rollback across environments must be rejected")
	} else if !errors.Is(err, domain.ErrInvalidReference) {
		t.Fatalf("expected invalid reference error, got %v", err)
	}

	// 被拒绝后 staging 专题不应被改回 production。
	after, _ := services.store.GetCampaign(ctx, campaign.ID)
	if after.Environment != "staging" {
		t.Fatalf("campaign environment must stay staging, got %q", after.Environment)
	}
}
