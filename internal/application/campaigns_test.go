package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/wyw14/cry-075/internal/domain"
)

func TestCreateCampaignIdempotencyReturnsOriginalResource(t *testing.T) {
	services := setupServices(t)
	command := CreateCampaignCommand{Name: "处暑专题", Season: "夏末", Environment: "production", Window: domain.TimeWindow{StartsAt: services.clock.at, EndsAt: services.clock.at.Add(time.Hour)}, Actor: domain.Actor{ID: "operator", Role: domain.RoleOperator}, IdempotencyKey: "stable-key", RequestID: "req-1"}
	first, err := services.campaigns.Create(context.Background(), command)
	if err != nil {
		t.Fatal(err)
	}
	second, err := services.campaigns.Create(context.Background(), command)
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID {
		t.Fatalf("idempotent replay changed identity: %s != %s", first.ID, second.ID)
	}
	command.Name = "另一个专题"
	if _, err := services.campaigns.Create(context.Background(), command); !errors.Is(err, domain.ErrDuplicateRequest) {
		t.Fatalf("expected duplicate request conflict, got %v", err)
	}
}
