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
	if err := require(actor, "audit.export"); err != nil {
		return err
	}
	if request.RequestedBy != actor.ID || request.Reason == "" || request.From.IsZero() || !request.From.Before(request.To) {
		return fmt.Errorf("invalid controlled export request")
	}
	allowed := map[string]bool{"created_at": true, "actor_role": true, "action": true, "subject": true, "subject_id": true, "request_id": true}
	for _, field := range request.Fields {
		if !allowed[field] {
			return fmt.Errorf("field %q is not exportable", field)
		}
	}
	events, err := s.AuditRepository.ListAudit(ctx, request.From, request.To)
	if err != nil {
		return err
	}
	csvWriter := csv.NewWriter(writer)
	if err := csvWriter.Write(request.Fields); err != nil {
		return err
	}
	for _, event := range events {
		row := make([]string, 0, len(request.Fields))
		for _, field := range request.Fields {
			switch field {
			case "created_at":
				row = append(row, event.CreatedAt.Format(time.RFC3339))
			case "actor_role":
				row = append(row, string(event.ActorRole))
			case "action":
				row = append(row, event.Action)
			case "subject":
				row = append(row, event.Subject)
			case "subject_id":
				row = append(row, string(event.SubjectID))
			case "request_id":
				row = append(row, event.RequestID)
			}
		}
		if err := csvWriter.Write(row); err != nil {
			return err
		}
	}
	csvWriter.Flush()
	return csvWriter.Error()
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
