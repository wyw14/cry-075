package application

import (
	"context"
	"testing"
	"time"

	"github.com/wyw14/cry-075/internal/domain"
	"github.com/wyw14/cry-075/internal/repository"
)

type fixedClock struct{ at time.Time }

func (c fixedClock) Now() time.Time { return c.at }

type testServices struct {
	store        *repository.MemoryStore
	campaigns    CampaignService
	catalog      CatalogService
	review       ReviewService
	releases     ReleaseService
	preview      PreviewService
	invalidation InvalidationService
	scheduling   ScheduleService
	operations   OperationsService
	clock        fixedClock
}

func setupServices(t *testing.T) testServices {
	t.Helper()
	store := repository.NewMemoryStore()
	clock := fixedClock{at: time.Date(2026, 8, 22, 8, 0, 0, 0, time.UTC)}
	timeSource := Clock(clock.Now)
	audit := AuditWriter{Repository: store, Clock: timeSource}
	return testServices{store: store, clock: clock,
		campaigns:    CampaignService{Campaigns: store, Catalog: store, Transactions: store, Audit: audit, Clock: timeSource},
		catalog:      CatalogService{Catalog: store, Transactions: store, Audit: audit},
		review:       ReviewService{Campaigns: store, Catalog: store, Releases: store, Transactions: store, Audit: audit, Clock: timeSource},
		releases:     ReleaseService{Campaigns: store, Catalog: store, Releases: store, Transactions: store, Audit: audit, Clock: timeSource},
		preview:      PreviewService{Campaigns: store, Catalog: store, Releases: store, Clock: timeSource},
		invalidation: InvalidationService{Campaigns: store, Catalog: store, Releases: store, AuditRepository: store, Transactions: store, Audit: audit, Clock: timeSource},
		scheduling:   ScheduleService{Campaigns: store, Schedules: store, Transactions: store, Audit: audit, Clock: timeSource},
		operations:   OperationsService{AuditRepository: store, Campaigns: store, Transactions: store, Audit: audit, Clock: timeSource},
	}
}

func seedComposition(t *testing.T, services testServices, name string) (domain.Campaign, domain.AssetPackage, domain.Asset, domain.Placement, domain.Audience) {
	t.Helper()
	ctx := context.Background()
	operator := domain.Actor{ID: "operator", Role: domain.RoleOperator}
	asset, err := services.catalog.CreateAsset(ctx, name+"视觉", "image/png", name+".png", nil, operator, "req-asset")
	if err != nil {
		t.Fatal(err)
	}
	pack, err := services.catalog.CreatePackage(ctx, name+"素材包", "production", false, operator, "req-package")
	if err != nil {
		t.Fatal(err)
	}
	pack, err = services.catalog.AddItem(ctx, AddPackageItemCommand{PackageID: pack.ID, AssetID: asset.ID, ExpectedVersion: pack.Version, Actor: operator, RequestID: "req-item"})
	if err != nil {
		t.Fatal(err)
	}
	placement, _ := domain.NewPlacement("home.hero."+name, name+"推荐位", 3)
	if err := services.store.PutPlacement(ctx, placement); err != nil {
		t.Fatal(err)
	}
	audience := domain.Audience{ID: domain.NewID("aud"), Name: "全部访客", Attributes: map[string]string{}}
	if err := services.store.PutAudience(ctx, audience); err != nil {
		t.Fatal(err)
	}
	campaign, err := services.campaigns.Create(ctx, CreateCampaignCommand{Name: name, Season: "夏末", Environment: "production", Window: domain.TimeWindow{StartsAt: services.clock.at.Add(-time.Hour), EndsAt: services.clock.at.Add(24 * time.Hour)}, Actor: operator, IdempotencyKey: "create-" + name, RequestID: "req-create"})
	if err != nil {
		t.Fatal(err)
	}
	campaign, err = services.campaigns.Bind(ctx, BindCampaignCommand{CampaignID: campaign.ID, PackageID: pack.ID, PlacementID: placement.ID, AudienceID: audience.ID, ExpectedVersion: campaign.Version, Actor: operator, RequestID: "req-bind"})
	if err != nil {
		t.Fatal(err)
	}
	return campaign, pack, asset, placement, audience
}
