package application

import (
	"fmt"
	"strings"
	"time"
)

func timeParse(value string) (time.Time, error) {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return time.Time{}, fmt.Errorf("schedule time is required")
	}
	parsed, err := time.Parse(time.RFC3339, normalized)
	if err != nil {
		return time.Time{}, fmt.Errorf("schedule time must include an RFC3339 timezone: %w", err)
	}
	return parsed.Round(0).UTC(), nil
}
