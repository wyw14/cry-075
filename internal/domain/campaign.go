package domain

import (
	"fmt"
	"strings"
	"time"
)

type Campaign struct {
	ID          ID             `json:"id"`
	Name        string         `json:"name"`
	Season      string         `json:"season"`
	Status      CampaignStatus `json:"status"`
	PackageID   ID             `json:"package_id"`
	PlacementID ID             `json:"placement_id"`
	AudienceID  ID             `json:"audience_id"`
	Window      TimeWindow     `json:"window"`
	Environment string         `json:"environment"`
	Version     int64          `json:"version"`
	CreatedBy   ID             `json:"created_by"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

func NewCampaign(name, season, environment string, actor ID, window TimeWindow) (Campaign, error) {
	c := Campaign{ID: NewID("cmp"), Name: strings.TrimSpace(name), Season: strings.TrimSpace(season), Environment: environment, Status: StatusDraft, CreatedBy: actor, Window: window, Version: 1, UpdatedAt: time.Now().UTC()}
	if err := c.Validate(); err != nil {
		return Campaign{}, err
	}
	return c, nil
}

func (c Campaign) Validate() error {
	if c.ID.Empty() || c.Name == "" || c.Season == "" || c.CreatedBy.Empty() {
		return fmt.Errorf("campaign identity, name, season and creator are required")
	}
	if !validEnvironment(c.Environment) {
		return fmt.Errorf("invalid campaign environment")
	}
	if err := c.Status.Validate(); err != nil {
		return err
	}
	return c.Window.Validate()
}

func (c *Campaign) Bind(packageID, placementID, audienceID ID) error {
	if c.Status != StatusDraft || packageID.Empty() || placementID.Empty() || audienceID.Empty() {
		return ErrInvalidTransition
	}
	c.PackageID, c.PlacementID, c.AudienceID = packageID, placementID, audienceID
	c.Version++
	c.UpdatedAt = time.Now().UTC()
	return nil
}

func (c *Campaign) Transition(next CampaignStatus, expectedVersion int64) error {
	if c.Version != expectedVersion {
		return ErrVersionConflict
	}
	if !c.Status.CanTransition(next) {
		return ErrInvalidTransition
	}
	if next == StatusReview && (c.PackageID.Empty() || c.PlacementID.Empty() || c.AudienceID.Empty()) {
		return ErrInvalidReference
	}
	c.Status = next
	c.Version++
	c.UpdatedAt = time.Now().UTC()
	return nil
}
