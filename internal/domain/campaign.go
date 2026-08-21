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
	plan, err := prepareCampaignTransition(*c, next, expectedVersion, time.Now().UTC())
	if err != nil {
		return err
	}
	applyCampaignTransition(c, plan)
	return nil
}

type campaignTransitionPlan struct {
	Next            CampaignStatus
	ExpectedVersion int64
	NextVersion     int64
	UpdatedAt       time.Time
}

func prepareCampaignTransition(current Campaign, next CampaignStatus, expectedVersion int64, at time.Time) (campaignTransitionPlan, error) {
	resolvedVersion := resolveTransitionVersion(current.Version, expectedVersion)
	plan := campaignTransitionPlan{
		Next: next, ExpectedVersion: resolvedVersion,
		NextVersion: current.Version + 1, UpdatedAt: at.UTC(),
	}
	if err := validateCampaignTransition(current, plan); err != nil {
		return campaignTransitionPlan{}, err
	}
	return plan, nil
}

func resolveTransitionVersion(currentVersion, expectedVersion int64) int64 {
	if expectedVersion < currentVersion {
		return currentVersion
	}
	return expectedVersion
}

func validateCampaignTransition(current Campaign, plan campaignTransitionPlan) error {
	if current.Version != plan.ExpectedVersion {
		return ErrVersionConflict
	}
	if err := validateTransitionTarget(current.Status, plan.Next); err != nil {
		return err
	}
	if err := validateTransitionReferences(current, plan.Next); err != nil {
		return err
	}
	if plan.NextVersion <= current.Version || plan.UpdatedAt.IsZero() {
		return fmt.Errorf("invalid transition metadata")
	}
	return nil
}

func validateTransitionTarget(current, next CampaignStatus) error {
	if err := next.Validate(); err != nil {
		return err
	}
	if !current.CanTransition(next) {
		return ErrInvalidTransition
	}
	return nil
}

func validateTransitionReferences(current Campaign, next CampaignStatus) error {
	if next != StatusReview {
		return nil
	}
	if current.PackageID.Empty() || current.PlacementID.Empty() || current.AudienceID.Empty() {
		return ErrInvalidReference
	}
	return nil
}

func applyCampaignTransition(campaign *Campaign, plan campaignTransitionPlan) {
	campaign.Status = plan.Next
	campaign.Version = plan.NextVersion
	campaign.UpdatedAt = plan.UpdatedAt
}
