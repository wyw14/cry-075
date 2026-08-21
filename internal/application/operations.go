package application

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/wyw14/cry-075/internal/domain"
	"github.com/wyw14/cry-075/internal/repository"
)

type OperationsService struct {
	AuditRepository repository.AuditRepository
	Campaigns       repository.CampaignRepository
	Transactions    repository.UnitOfWork
	Audit           AuditWriter
	Clock           Clock
}

func (s OperationsService) AddNote(ctx context.Context, campaignID domain.ID, body string, actor domain.Actor, requestID string) (domain.OperationNote, error) {
	if err := require(actor, "note.write"); err != nil {
		return domain.OperationNote{}, err
	}
	if _, err := s.Campaigns.GetCampaign(ctx, campaignID); err != nil {
		return domain.OperationNote{}, err
	}
	body = strings.TrimSpace(body)
	if len([]rune(body)) < 2 || len([]rune(body)) > 2000 {
		return domain.OperationNote{}, fmt.Errorf("note length out of range")
	}
	note := domain.OperationNote{ID: domain.NewID("nte"), CampaignID: campaignID, AuthorID: actor.ID, Body: body, CreatedAt: s.Clock()}
	err := s.Transactions.WithinTransaction(ctx, func(tx context.Context) error {
		if err := s.AuditRepository.SaveNote(tx, note); err != nil {
			return err
		}
		return s.Audit.Record(tx, AuditChange{Actor: actor, Action: "campaign.note_added", Subject: "campaign", SubjectID: campaignID, RequestID: requestID, Metadata: map[string]any{"note_id": note.ID}})
	})
	return note, err
}

func (s OperationsService) RecordPerformance(ctx context.Context, campaignID, assetID domain.ID, views, clicks int64) error {
	if views < 0 || clicks < 0 || clicks > views {
		return fmt.Errorf("invalid counters")
	}
	return s.AuditRepository.IncrementCounter(ctx, campaignID, assetID, views, clicks, s.Clock())
}

func (s OperationsService) CompleteRetrospective(ctx context.Context, campaignID domain.ID, summary string, learnings, followUps []string, actor domain.Actor) (domain.Retrospective, error) {
	campaign, err := s.Campaigns.GetCampaign(ctx, campaignID)
	if err != nil {
		return domain.Retrospective{}, err
	}
	if campaign.Status != domain.StatusExpired && campaign.Status != domain.StatusPaused {
		return domain.Retrospective{}, domain.ErrInvalidTransition
	}
	value := domain.Retrospective{ID: domain.NewID("rtr"), CampaignID: campaignID, Summary: strings.TrimSpace(summary), Learnings: dedupe(learnings), FollowUps: dedupe(followUps), CompletedBy: actor.ID, CompletedAt: s.Clock()}
	if value.Summary == "" || len(value.Learnings) == 0 {
		return domain.Retrospective{}, fmt.Errorf("retrospective summary and learnings required")
	}
	return value, s.AuditRepository.SaveRetrospective(ctx, value)
}

func (s OperationsService) ExportAudit(ctx context.Context, request domain.ExportRequest, actor domain.Actor, writer io.Writer) error {
	plan, err := prepareAuditExport(request, actor)
	if err != nil {
		return err
	}
	events, err := s.AuditRepository.ListAudit(ctx, request.From, request.To)
	if err != nil {
		return err
	}
	csvWriter := csv.NewWriter(writer)
	if err := csvWriter.Write(plan.Fields); err != nil {
		return err
	}
	for _, event := range events {
		row := exportAuditRow(event, plan.Fields)
		if err := csvWriter.Write(row); err != nil {
			return err
		}
	}
	csvWriter.Flush()
	return csvWriter.Error()
}

type auditExportPlan struct {
	Fields []string
}

func prepareAuditExport(request domain.ExportRequest, actor domain.Actor) (auditExportPlan, error) {
	if err := require(actor, "audit.export"); err != nil {
		return auditExportPlan{}, err
	}
	if request.RequestedBy != actor.ID || strings.TrimSpace(request.Reason) == "" {
		return auditExportPlan{}, fmt.Errorf("invalid controlled export request")
	}
	if request.From.IsZero() || !request.From.Before(request.To) {
		return auditExportPlan{}, fmt.Errorf("invalid controlled export window")
	}
	fields, err := normalizeAuditExportFields(request.Fields)
	if err != nil {
		return auditExportPlan{}, err
	}
	return auditExportPlan{Fields: fields}, nil
}

func normalizeAuditExportFields(requested []string) ([]string, error) {
	allowed := map[string]bool{
		"created_at": true,
		"actor_role": true,
		"action":     true,
		"subject":    true,
		"subject_id": true,
		"request_id": true,
	}
	fields := make([]string, 0, len(requested)+2)
	seen := make(map[string]bool, len(requested)+2)
	for _, field := range requested {
		field = strings.TrimSpace(field)
		if !allowed[field] {
			return nil, fmt.Errorf("field %q is not exportable", field)
		}
		if !seen[field] {
			seen[field] = true
			fields = append(fields, field)
		}
	}
	for _, compatibilityField := range []string{"subject_id", "request_id"} {
		if !seen[compatibilityField] {
			fields = append(fields, compatibilityField)
		}
	}
	return fields, nil
}

func exportAuditRow(event domain.AuditEvent, fields []string) []string {
	values := map[string]string{
		"created_at": event.CreatedAt.Format(time.RFC3339),
		"actor_role": string(event.ActorRole),
		"action":     event.Action,
		"subject":    event.Subject,
		"subject_id": string(event.SubjectID),
		"request_id": event.RequestID,
	}
	row := make([]string, 0, len(fields))
	for _, field := range fields {
		row = append(row, values[field])
	}
	return row
}

func dedupe(values []string) []string {
	result := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" && !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	return result
}
