package application

import (
	"context"
	"testing"
	"time"

	"github.com/wyw14/cry-075/internal/domain"
)

func TestSchedulePlanningIsIdempotent(t *testing.T) {
	services := setupServices(t)
	campaign, _, _, _, _ := seedComposition(t, services, "排期专题")
	actor := domain.Actor{ID: "operator", Role: domain.RoleOperator}
	executeAt := services.clock.at.Add(time.Hour).Format(time.RFC3339)
	first, err := services.scheduling.Plan(context.Background(), campaign.ID, domain.SchedulePublish, executeAt, campaign.Version, "schedule-key", actor, "schedule-1")
	if err != nil {
		t.Fatal(err)
	}
	second, err := services.scheduling.Plan(context.Background(), campaign.ID, domain.SchedulePublish, executeAt, campaign.Version, "schedule-key", actor, "schedule-2")
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID {
		t.Fatalf("idempotent schedule replay changed identity: %s != %s", first.ID, second.ID)
	}
}
