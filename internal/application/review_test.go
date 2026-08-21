package application

import (
	"context"
	"errors"
	"testing"

	"github.com/wyw14/cry-075/internal/domain"
)

func TestReviewRejectsOverlappingPublishedPlacement(t *testing.T) {
	services := setupServices(t)
	first, _, _, placement, audience := seedComposition(t, services, "专题甲")
	ctx := context.Background()
	operator := domain.Actor{ID: "operator", Role: domain.RoleOperator}
	reviewer := domain.Actor{ID: "reviewer", Role: domain.RoleReviewer}
	first, err := services.review.Submit(ctx, first.ID, first.Version, operator, "submit-a")
	if err != nil {
		t.Fatal(err)
	}
	first, err = services.review.Decide(ctx, first.ID, domain.DecisionApprove, "通过", first.Version, reviewer, "review-a")
	if err != nil {
		t.Fatal(err)
	}
	second, pack, _, _, _ := seedComposition(t, services, "专题乙")
	second.PackageID = pack.ID
	second.PlacementID = placement.ID
	second.AudienceID = audience.ID
	current, _ := services.store.GetCampaign(ctx, second.ID)
	second.Version = current.Version + 1
	if err := services.store.UpdateCampaign(ctx, second, current.Version); err != nil {
		t.Fatal(err)
	}
	second, err = services.review.Submit(ctx, second.ID, second.Version, operator, "submit-b")
	if err != nil {
		t.Fatal(err)
	}
	_, err = services.review.Decide(ctx, second.ID, domain.DecisionApprove, "也通过", second.Version, reviewer, "review-b")
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected placement conflict, got %v", err)
	}
}
