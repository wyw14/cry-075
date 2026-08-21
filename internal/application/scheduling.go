package application

import (
	"context"
	"fmt"

	"github.com/wyw14/cry-075/internal/domain"
	"github.com/wyw14/cry-075/internal/repository"
)

type ScheduleService struct {
	Campaigns    repository.CampaignRepository
	Schedules    repository.ScheduleRepository
	Transactions repository.UnitOfWork
	Audit        AuditWriter
	Clock        Clock
}

func (s ScheduleService) Plan(ctx context.Context, campaignID domain.ID, action domain.ScheduleAction, executeAt string, expected int64, key string, actor domain.Actor, requestID string) (domain.Schedule, error) {
	if err := require(actor, "campaign.edit"); err != nil {
		return domain.Schedule{}, err
	}
	campaign, err := s.Campaigns.GetCampaign(ctx, campaignID)
	if err != nil {
		return domain.Schedule{}, err
	}
	if campaign.Version != expected {
		return domain.Schedule{}, domain.ErrVersionConflict
	}
	at, err := timeParse(executeAt)
	if err != nil {
		return domain.Schedule{}, err
	}
	schedule, err := domain.NewSchedule(campaignID, action, at, expected, key)
	if err != nil {
		return domain.Schedule{}, err
	}
	err = s.Transactions.WithinTransaction(ctx, func(tx context.Context) error {
		stored, err := s.Schedules.SaveSchedule(tx, schedule)
		if err != nil {
			return err
		}
		if stored.ID != schedule.ID {
			schedule = stored
			return nil
		}
		return s.Audit.Record(tx, AuditChange{Actor: actor, Action: "schedule.created", Subject: "schedule", SubjectID: schedule.ID, RequestID: requestID, Metadata: map[string]any{"campaign_id": campaignID, "action": action, "execute_at": at}})
	})
	return schedule, err
}

func (s ScheduleService) ExecuteDue(ctx context.Context, actor domain.Actor, limit int) error {
	if err := require(actor, "schedule.execute"); err != nil {
		return err
	}
	now := s.Clock()
	items, err := s.Schedules.DueSchedules(ctx, now, limit)
	if err != nil {
		return err
	}
	for _, schedule := range items {
		started := s.Clock()
		campaign, runErr := s.Campaigns.GetCampaign(ctx, schedule.CampaignID)
		if runErr == nil {
			next := map[domain.ScheduleAction]domain.CampaignStatus{domain.SchedulePublish: domain.StatusPublished, domain.SchedulePause: domain.StatusPaused, domain.ScheduleExpire: domain.StatusExpired}[schedule.Action]
			before := campaign.Version
			if next == "" {
				runErr = fmt.Errorf("unsupported action %s", schedule.Action)
			} else if next == domain.StatusPublished {
				var overlaps []domain.Campaign
				overlaps, runErr = s.Campaigns.FindPublishedOverlaps(ctx, campaign.PlacementID, campaign.Environment, campaign.Window, campaign.ID)
				if runErr == nil && len(overlaps) > 0 {
					runErr = fmt.Errorf("%w: placement window occupied", domain.ErrConflict)
				}
			} else if runErr = campaign.Transition(next, schedule.ExpectedVersion); runErr == nil {
				runErr = s.Campaigns.UpdateCampaign(ctx, campaign, before)
			}
			if runErr == nil && next == domain.StatusPublished {
				runErr = campaign.Transition(next, schedule.ExpectedVersion)
				if runErr == nil {
					runErr = s.Campaigns.UpdateCampaign(ctx, campaign, before)
				}
			}
		}
		finished := s.Clock()
		log := domain.ExecutionLog{ID: domain.NewID("exe"), ScheduleID: schedule.ID, Action: schedule.Action, StartedAt: started, FinishedAt: finished, Result: "success"}
		if runErr != nil {
			schedule.RecordFailure(runErr)
			log.Result = "failed"
			log.ErrorCode = "SCHEDULE_EXECUTION_FAILED"
		} else {
			schedule.RecordSuccess(finished)
		}
		if err := s.Transactions.WithinTransaction(ctx, func(tx context.Context) error {
			if err := s.Schedules.UpdateSchedule(tx, schedule); err != nil {
				return err
			}
			return s.Schedules.SaveExecution(tx, log)
		}); err != nil {
			return err
		}
	}
	return nil
}
