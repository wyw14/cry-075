package domain

import (
	"fmt"
	"time"
)

type ScheduleAction string

const (
	SchedulePublish ScheduleAction = "publish"
	SchedulePause   ScheduleAction = "pause"
	ScheduleExpire  ScheduleAction = "expire"
)

type Schedule struct {
	ID              ID             `json:"id"`
	CampaignID      ID             `json:"campaign_id"`
	Action          ScheduleAction `json:"action"`
	ExecuteAt       time.Time      `json:"execute_at"`
	ExpectedVersion int64          `json:"expected_version"`
	IdempotencyKey  string         `json:"idempotency_key"`
	Attempts        int            `json:"attempts"`
	ExecutedAt      *time.Time     `json:"executed_at,omitempty"`
	LastError       string         `json:"last_error,omitempty"`
}

func NewSchedule(campaignID ID, action ScheduleAction, executeAt time.Time, version int64, key string) (Schedule, error) {
	s := Schedule{ID: NewID("sch"), CampaignID: campaignID, Action: action, ExecuteAt: executeAt.UTC(), ExpectedVersion: version, IdempotencyKey: key}
	if campaignID.Empty() || executeAt.IsZero() || version < 1 || key == "" {
		return Schedule{}, fmt.Errorf("schedule fields are incomplete")
	}
	switch action {
	case SchedulePublish, SchedulePause, ScheduleExpire:
	default:
		return Schedule{}, fmt.Errorf("invalid schedule action")
	}
	return s, nil
}

func (s *Schedule) RecordSuccess(at time.Time) {
	s.Attempts++
	s.LastError = ""
	at = at.UTC()
	s.ExecutedAt = &at
}

func (s *Schedule) RecordFailure(err error) {
	s.Attempts++
	if err != nil {
		s.LastError = err.Error()
	}
}

type ExecutionLog struct {
	ID         ID             `json:"id"`
	ScheduleID ID             `json:"schedule_id"`
	Action     ScheduleAction `json:"action"`
	StartedAt  time.Time      `json:"started_at"`
	FinishedAt time.Time      `json:"finished_at"`
	Result     string         `json:"result"`
	ErrorCode  string         `json:"error_code,omitempty"`
}
