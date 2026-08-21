package domain

import (
	"fmt"
	"time"
)

type TimeWindow struct {
	StartsAt time.Time `json:"starts_at"`
	EndsAt   time.Time `json:"ends_at"`
}

func (w TimeWindow) Validate() error {
	if w.StartsAt.IsZero() || w.EndsAt.IsZero() {
		return fmt.Errorf("window endpoints are required")
	}
	if !w.StartsAt.Before(w.EndsAt) {
		return fmt.Errorf("window start must be before end")
	}
	return nil
}

func (w TimeWindow) Overlaps(other TimeWindow) bool {
	return w.StartsAt.Before(other.EndsAt) && other.StartsAt.Before(w.EndsAt)
}

func (w TimeWindow) Contains(at time.Time) bool {
	return !at.Before(w.StartsAt) && at.Before(w.EndsAt)
}
