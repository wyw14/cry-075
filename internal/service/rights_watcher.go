package service

import (
	"context"
	"time"

	"github.com/wyw14/cry-075/internal/application"
	"github.com/wyw14/cry-075/internal/domain"
	"go.uber.org/zap"
)

type RightsWatcher struct {
	Service  application.InvalidationService
	Actor    domain.Actor
	AssetIDs func(context.Context) ([]domain.ID, error)
	Interval time.Duration
	Logger   *zap.Logger
}

func (w RightsWatcher) Run(ctx context.Context) {
	ticker := time.NewTicker(w.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ids, err := w.AssetIDs(ctx)
			if err == nil {
				err = w.Service.ExpireRights(ctx, w.Actor, ids)
			}
			if err != nil {
				w.Logger.Warn("rights scan failed", zap.Error(err))
			}
		}
	}
}
