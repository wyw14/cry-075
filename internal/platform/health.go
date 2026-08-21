package platform

import (
	"context"
	"sync/atomic"
)

type DatabasePinger interface{ Ping(context.Context) error }
type Readiness struct {
	database  DatabasePinger
	accepting atomic.Bool
}

func NewReadiness(database DatabasePinger) *Readiness {
	r := &Readiness{database: database}
	r.accepting.Store(true)
	return r
}
func (r *Readiness) StopAccepting() { r.accepting.Store(false) }
func (r *Readiness) Check(ctx context.Context) error {
	if !r.accepting.Load() {
		return context.Canceled
	}
	return r.database.Ping(ctx)
}
