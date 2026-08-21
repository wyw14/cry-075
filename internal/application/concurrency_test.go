package application

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/wyw14/cry-075/internal/domain"
)

func TestConcurrentCampaignIdempotencyCreatesOneResource(t *testing.T) {
	services := setupServices(t)
	command := CreateCampaignCommand{Name: "并发发布专题", Season: "夏末", Environment: "production", Window: domain.TimeWindow{StartsAt: services.clock.at, EndsAt: services.clock.at.Add(time.Hour)}, Actor: domain.Actor{ID: "operator", Role: domain.RoleOperator}, IdempotencyKey: "concurrent-key", RequestID: "req-concurrent"}
	const workers = 12
	start := make(chan struct{})
	results := make(chan domain.ID, workers)
	errors := make(chan error, workers)
	var ready sync.WaitGroup
	ready.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			ready.Done()
			<-start
			value, err := services.campaigns.Create(context.Background(), command)
			if err != nil {
				errors <- err
				return
			}
			results <- value.ID
		}()
	}
	ready.Wait()
	close(start)
	ids := map[domain.ID]bool{}
	for i := 0; i < workers; i++ {
		select {
		case err := <-errors:
			t.Fatal(err)
		case id := <-results:
			ids[id] = true
		}
	}
	if len(ids) != 1 {
		t.Fatalf("idempotent requests created %d resources", len(ids))
	}
}
