package domain

import "time"

type OperationNote struct {
	ID         ID        `json:"id"`
	CampaignID ID        `json:"campaign_id"`
	AuthorID   ID        `json:"author_id"`
	Body       string    `json:"body"`
	CreatedAt  time.Time `json:"created_at"`
}

type Retrospective struct {
	ID          ID        `json:"id"`
	CampaignID  ID        `json:"campaign_id"`
	Summary     string    `json:"summary"`
	Learnings   []string  `json:"learnings"`
	FollowUps   []string  `json:"follow_ups"`
	CompletedBy ID        `json:"completed_by"`
	CompletedAt time.Time `json:"completed_at"`
}

type PerformanceCounter struct {
	CampaignID ID        `json:"campaign_id"`
	AssetID    ID        `json:"asset_id"`
	Views      int64     `json:"views"`
	Clicks     int64     `json:"clicks"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (c PerformanceCounter) ClickThroughRate() float64 {
	if c.Views == 0 {
		return 0
	}
	return float64(c.Clicks) / float64(c.Views)
}
