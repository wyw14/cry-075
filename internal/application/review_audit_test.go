package application

import (
	"context"
	"errors"
	"testing"

	"github.com/wyw14/cry-075/internal/domain"
	"github.com/wyw14/cry-075/internal/repository"
)

// flakyAuditRepository wraps an AuditRepository and forces AppendAudit to fail,
// simulating a temporarily unavailable audit store during a review decision.
type flakyAuditRepository struct {
	repository.AuditRepository
	appendErr error
}

func (f flakyAuditRepository) AppendAudit(ctx context.Context, event domain.AuditEvent) error {
	if f.appendErr != nil {
		return f.appendErr
	}
	return f.AuditRepository.AppendAudit(ctx, event)
}

// TestReviewApproveDoesNotLeakPublishedStatusWhenAuditFails reproduces the
// issue where approving a campaign during an audit-store outage left the
// campaign stuck in "published" even though the request returned an error,
// which then made retries hit a status conflict.
func TestReviewApproveDoesNotLeakPublishedStatusWhenAuditFails(t *testing.T) {
	services := setupServices(t)
	campaign, _, _, _, _ := seedComposition(t, services, "审核暂存专题")
	ctx := context.Background()
	operator := domain.Actor{ID: "operator", Role: domain.RoleOperator}
	reviewer := domain.Actor{ID: "reviewer", Role: domain.RoleReviewer}

	campaign, err := services.review.Submit(ctx, campaign.ID, campaign.Version, operator, "submit-a")
	if err != nil {
		t.Fatal(err)
	}

	// Make the audit store reject writes while the decision transaction runs.
	auditErr := errors.New("audit store temporarily unavailable")
	flaky := flakyAuditRepository{AuditRepository: services.store, appendErr: auditErr}
	timeSource := Clock(services.clock.Now)
	services.review = ReviewService{
		Campaigns:    services.store,
		Catalog:      services.store,
		Releases:     services.store,
		Transactions: services.store,
		Audit:        AuditWriter{Repository: flaky, Clock: timeSource},
		Clock:        timeSource,
	}

	if _, err := services.review.Decide(ctx, campaign.ID, domain.DecisionApprove, "通过", campaign.Version, reviewer, "review-a"); !errors.Is(err, auditErr) {
		t.Fatalf("expected audit failure to surface, got %v", err)
	}

	// The campaign must NOT be left published: the failed audit must roll back
	// the status transition together with the approval record, so the state
	// matches what it was before the decision was attempted.
	unchanged, err := services.store.GetCampaign(ctx, campaign.ID)
	if err != nil {
		t.Fatal(err)
	}
	if unchanged.Status == domain.StatusPublished {
		t.Fatalf("published status leaked after failed review: %#v", unchanged.Status)
	}
	if unchanged.Version != campaign.Version {
		t.Fatalf("campaign version advanced after failed review: got %d want %d", unchanged.Version, campaign.Version)
	}

	// A retry with the recovered audit store must succeed: no lingering state
	// conflict from the prior failure. This is the part that previously broke —
	// the campaign was already "published", so the pending_review->published
	// transition could no longer run.
	services.review.Audit = AuditWriter{Repository: services.store, Clock: timeSource}
	approved, err := services.review.Decide(ctx, campaign.ID, domain.DecisionApprove, "通过", campaign.Version, reviewer, "review-a-retry")
	if err != nil {
		t.Fatalf("retry should succeed after audit recovery, got %v", err)
	}
	if approved.Status != domain.StatusPublished {
		t.Fatalf("retry should publish campaign, got %s", approved.Status)
	}
}
