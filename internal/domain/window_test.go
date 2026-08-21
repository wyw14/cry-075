package domain

import (
	"testing"
	"time"
)

func TestTimeWindowUsesHalfOpenIntervals(t *testing.T) {
	start := time.Date(2026, 8, 22, 8, 0, 0, 0, time.UTC)
	first := TimeWindow{StartsAt: start, EndsAt: start.Add(time.Hour)}
	touching := TimeWindow{StartsAt: start.Add(time.Hour), EndsAt: start.Add(2 * time.Hour)}
	overlap := TimeWindow{StartsAt: start.Add(30 * time.Minute), EndsAt: start.Add(90 * time.Minute)}
	if first.Overlaps(touching) {
		t.Fatal("touching half-open windows must not overlap")
	}
	if !first.Overlaps(overlap) {
		t.Fatal("intersecting windows must overlap")
	}
}
