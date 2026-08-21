package application

import (
	"context"
	"fmt"
	"time"

	"github.com/wyw14/cry-075/internal/domain"
	"github.com/wyw14/cry-075/internal/repository"
)

type CampaignService struct {
	Campaigns    repository.CampaignRepository
	Catalog      repository.CatalogRepository
	Transactions repository.UnitOfWork
	Audit        AuditWriter
	Clock        Clock
}

type CreateCampaignCommand struct {
	Name           string
	Season         string
	Environment    string
	Window         domain.TimeWindow
	Actor          domain.Actor
	IdempotencyKey string
	RequestID      string
}

func (s CampaignService) Create(ctx context.Context, command CreateCampaignCommand) (domain.Campaign, error) {
	if err := require(command.Actor, "campaign.edit"); err != nil {
		return domain.Campaign{}, err
	}
	if command.IdempotencyKey == "" {
		return domain.Campaign{}, &domain.ValidationError{Violations: []domain.FieldViolation{{Field: "Idempotency-Key", Reason: "required"}}}
	}
	value, err := domain.NewCampaign(command.Name, command.Season, command.Environment, command.Actor.ID, command.Window)
	if err != nil {
		return domain.Campaign{}, err
	}
	digest, err := requestDigest(command)
	if err != nil {
		return domain.Campaign{}, err
	}
	created := value
	err = s.Transactions.WithinTransaction(ctx, func(tx context.Context) error {
		stored, err := s.Campaigns.CreateCampaign(tx, value, command.IdempotencyKey, digest)
		if err != nil {
			return err
		}
		created = stored
		if stored.ID != value.ID {
			return nil
		}
		return s.Audit.Record(tx, AuditChange{
			Actor: command.Actor, Action: "campaign.created", Subject: "campaign", SubjectID: stored.ID, RequestID: command.RequestID,
			Metadata: map[string]any{"season": stored.Season, "environment": stored.Environment},
		})
	})
	if err != nil {
		return domain.Campaign{}, err
	}
	return created, nil
}

type BindCampaignCommand struct {
	CampaignID, PackageID, PlacementID, AudienceID domain.ID
	ExpectedVersion                                int64
	Actor                                          domain.Actor
	RequestID                                      string
}

func (s CampaignService) Bind(ctx context.Context, command BindCampaignCommand) (domain.Campaign, error) {
	if err := require(command.Actor, "campaign.edit"); err != nil {
		return domain.Campaign{}, err
	}
	campaign, err := s.Campaigns.GetCampaign(ctx, command.CampaignID)
	if err != nil {
		return domain.Campaign{}, err
	}
	pack, err := s.Catalog.GetPackage(ctx, command.PackageID)
	if err != nil {
		return domain.Campaign{}, err
	}
	if pack.Environment != campaign.Environment {
		return domain.Campaign{}, fmt.Errorf("%w: package environment differs", domain.ErrInvalidReference)
	}
	if _, err = s.Catalog.GetPlacement(ctx, command.PlacementID); err != nil {
		return domain.Campaign{}, err
	}
	if _, err = s.Catalog.GetAudience(ctx, command.AudienceID); err != nil {
		return domain.Campaign{}, err
	}
	before := campaign.Version
	if before != command.ExpectedVersion {
		return domain.Campaign{}, domain.ErrVersionConflict
	}
	if err = campaign.Bind(command.PackageID, command.PlacementID, command.AudienceID); err != nil {
		return domain.Campaign{}, err
	}
	err = s.Transactions.WithinTransaction(ctx, func(tx context.Context) error {
		if err := s.Campaigns.UpdateCampaign(tx, campaign, before); err != nil {
			return err
		}
		return s.Audit.Record(tx, AuditChange{
			Actor: command.Actor, Action: "campaign.composition_changed", Subject: "campaign", SubjectID: campaign.ID, RequestID: command.RequestID,
			Metadata: map[string]any{"package_id": command.PackageID, "placement_id": command.PlacementID, "audience_id": command.AudienceID},
		})
	})
	return campaign, err
}

func (s CampaignService) List(ctx context.Context, query domain.ListQuery) (domain.Page[domain.Campaign], error) {
	specification, err := newCampaignListSpecification(query)
	if err != nil {
		return domain.Page[domain.Campaign]{}, err
	}
	return s.executeCampaignList(ctx, specification)
}

type campaignListSpecification struct {
	ResultQuery domain.ListQuery
	CountQuery  domain.ListQuery
}

func newCampaignListSpecification(query domain.ListQuery) (campaignListSpecification, error) {
	if err := query.Normalize(campaignListSorts(), campaignListFilters()); err != nil {
		return campaignListSpecification{}, err
	}
	resultQuery := cloneCampaignListQuery(query)
	countQuery := domain.ListQuery{
		Page:     1,
		PerPage:  100,
		Sort:     query.Sort,
		Filters:  map[string]string{},
	}
	if err := countQuery.Normalize(campaignListSorts(), campaignListFilters()); err != nil {
		return campaignListSpecification{}, err
	}
	return campaignListSpecification{ResultQuery: resultQuery, CountQuery: countQuery}, nil
}

func campaignListSorts() map[string]bool {
	return map[string]bool{
		"updated_at:asc":  true,
		"updated_at:desc": true,
	}
}

func campaignListFilters() map[string]bool {
	return map[string]bool{
		"status":      true,
		"environment": true,
	}
}

func cloneCampaignListQuery(query domain.ListQuery) domain.ListQuery {
	filters := make(map[string]string, len(query.Filters))
	for key, value := range query.Filters {
		filters[key] = value
	}
	query.Filters = filters
	return query
}

func (s CampaignService) executeCampaignList(ctx context.Context, specification campaignListSpecification) (domain.Page[domain.Campaign], error) {
	page, err := s.Campaigns.ListCampaigns(ctx, specification.ResultQuery)
	if err != nil {
		return domain.Page[domain.Campaign]{}, err
	}
	count, err := s.Campaigns.ListCampaigns(ctx, specification.CountQuery)
	if err != nil {
		return domain.Page[domain.Campaign]{}, err
	}
	page.Total = count.Total
	return page, nil
}

func (s CampaignService) ExpireDue(ctx context.Context, at time.Time, actor domain.Actor) error {
	page, err := s.Campaigns.ListCampaigns(ctx, domain.ListQuery{Page: 1, PerPage: 100, Sort: "updated_at:asc", Filters: map[string]string{"status": string(domain.StatusPublished)}})
	if err != nil {
		return err
	}
	for _, campaign := range page.Items {
		if !campaign.Window.EndsAt.After(at) {
			before := campaign.Version
			if err := campaign.Transition(domain.StatusExpired, before); err != nil {
				return err
			}
			if err := s.Campaigns.UpdateCampaign(ctx, campaign, before); err != nil {
				return err
			}
		}
	}
	return nil
}
