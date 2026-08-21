package application

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/wyw14/cry-075/internal/domain"
)

func TestControlledAuditExportUsesFieldAllowlist(t *testing.T) {
	services := setupServices(t)
	ctx := context.Background()
	auditor := domain.Actor{ID: "auditor", Role: domain.RoleAuditor}
	event := domain.NewAudit(domain.Actor{ID: "operator", Role: domain.RoleOperator}, "campaign.created", "campaign", "cmp_one", "req-secret", map[string]any{"token": "do-not-export"}, services.clock.at)
	if err := services.store.AppendAudit(ctx, event); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	request := domain.ExportRequest{ID: "exp", RequestedBy: auditor.ID, From: services.clock.at.Add(-time.Hour), To: services.clock.at.Add(time.Hour), Reason: "季度审计", Fields: []string{"created_at", "actor_role", "action", "subject_id"}}
	if err := services.operations.ExportAudit(ctx, request, auditor, &output); err != nil {
		t.Fatal(err)
	}
	csv := output.String()
	if strings.Contains(csv, "do-not-export") || !strings.Contains(csv, "campaign.created") {
		t.Fatalf("unexpected export: %s", csv)
	}
}

func TestOperationsNoteAndCountersValidateBusinessInput(t *testing.T) {
	services := setupServices(t)
	campaign, _, asset, _, _ := seedComposition(t, services, "复盘专题")
	ctx := context.Background()
	operator := domain.Actor{ID: "operator", Role: domain.RoleOperator}
	note, err := services.operations.AddNote(ctx, campaign.ID, "  复盘时关注首屏点击表现  ", operator, "note-request")
	if err != nil {
		t.Fatal(err)
	}
	if note.Body != "复盘时关注首屏点击表现" {
		t.Fatalf("note was not normalized: %q", note.Body)
	}
	if err := services.operations.RecordPerformance(ctx, campaign.ID, asset.ID, 100, 11); err != nil {
		t.Fatal(err)
	}
	if err := services.operations.RecordPerformance(ctx, campaign.ID, asset.ID, 5, 8); err == nil {
		t.Fatal("clicks greater than views should be rejected")
	}
}
