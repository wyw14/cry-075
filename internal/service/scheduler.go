package service

import (
	"context"
	"time"

	"github.com/wyw14/cry-075/internal/application"
	"github.com/wyw14/cry-075/internal/domain"
	"go.uber.org/zap"
)

type LocalScheduler struct {
	Service   application.ScheduleService
	Actor     domain.Actor
	Interval  time.Duration
	BatchSize int
	Logger    *zap.Logger
}

func (s LocalScheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(s.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runCtx, cancel := context.WithTimeout(ctx, s.Interval)
			err := s.Service.ExecuteDue(runCtx, s.Actor, s.BatchSize)
			cancel()
			if err != nil {
				s.Logger.Error("schedule tick failed", zap.Error(err))
			} else {
				s.Logger.Debug("schedule tick complete")
			}
		}
	}
}
