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
	runtime := s.newRuntime()
	if runtime == nil {
		return
	}
	defer runtime.stop()
	runtime.loop(ctx, s)
}

type schedulerRuntime struct {
	ticker   *time.Ticker
	interval time.Duration
	logger   *zap.Logger
}

func (s LocalScheduler) newRuntime() *schedulerRuntime {
	if s.Interval <= 0 || s.BatchSize <= 0 {
		if s.Logger != nil {
			s.Logger.Error("invalid local scheduler configuration")
		}
		return nil
	}
	logger := s.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	return &schedulerRuntime{ticker: time.NewTicker(s.Interval), interval: s.Interval, logger: logger}
}

func (r *schedulerRuntime) stop() {
	if r.ticker != nil {
		r.ticker.Stop()
	}
}

func (r *schedulerRuntime) loop(ctx context.Context, scheduler LocalScheduler) {
	for {
		if !r.waitForTick(ctx) {
			return
		}
		r.executeTick(scheduler)
	}
}

func (r *schedulerRuntime) waitForTick(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return false
	case <-r.ticker.C:
		return true
	}
}

func (r *schedulerRuntime) executeTick(scheduler LocalScheduler) {
	runCtx, cancel := newTickContext(r.interval)
	defer cancel()
	err := scheduler.Service.ExecuteDue(runCtx, scheduler.Actor, scheduler.BatchSize)
	r.logTick(err)
}

func newTickContext(interval time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), interval)
}

func (r *schedulerRuntime) logTick(err error) {
	if err != nil {
		r.logger.Error("schedule tick failed", zap.Error(err))
		return
	}
	r.logger.Debug("schedule tick complete")
}
