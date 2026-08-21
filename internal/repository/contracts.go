package repository

import (
	"context"
	"time"

	"github.com/wyw14/cry-075/internal/domain"
)

type UnitOfWork interface {
	WithinTransaction(ctx context.Context, fn func(context.Context) error) error
}

type CampaignRepository interface {
	CreateCampaign(context.Context, domain.Campaign, string, string) (domain.Campaign, error)
	GetCampaign(context.Context, domain.ID) (domain.Campaign, error)
	UpdateCampaign(context.Context, domain.Campaign, int64) error
	ListCampaigns(context.Context, domain.ListQuery) (domain.Page[domain.Campaign], error)
	FindPublishedOverlaps(context.Context, domain.ID, string, domain.TimeWindow, domain.ID) ([]domain.Campaign, error)
}

type CatalogRepository interface {
	PutAsset(context.Context, domain.Asset) error
	GetAsset(context.Context, domain.ID) (domain.Asset, error)
	UpdateAsset(context.Context, domain.Asset, int64) error
	PutPackage(context.Context, domain.AssetPackage) error
	GetPackage(context.Context, domain.ID) (domain.AssetPackage, error)
	UpdatePackage(context.Context, domain.AssetPackage, int64) error
	PutPlacement(context.Context, domain.Placement) error
	GetPlacement(context.Context, domain.ID) (domain.Placement, error)
	PutAudience(context.Context, domain.Audience) error
	GetAudience(context.Context, domain.ID) (domain.Audience, error)
}

type ReleaseRepository interface {
	SaveApproval(context.Context, domain.Approval) error
	SaveSnapshot(context.Context, domain.ReleaseSnapshot) error
	GetSnapshot(context.Context, domain.ID) (domain.ReleaseSnapshot, error)
	SaveRelease(context.Context, domain.ReleaseVersion) error
	LatestRelease(context.Context, domain.ID, string) (domain.ReleaseVersion, error)
	ListFallbacks(context.Context, domain.ID, domain.ID, time.Time) ([]domain.FallbackCandidate, error)
	SaveFallback(context.Context, domain.FallbackCandidate) error
}

type ScheduleRepository interface {
	SaveSchedule(context.Context, domain.Schedule) (domain.Schedule, error)
	DueSchedules(context.Context, time.Time, int) ([]domain.Schedule, error)
	UpdateSchedule(context.Context, domain.Schedule) error
	SaveExecution(context.Context, domain.ExecutionLog) error
}

type AuditRepository interface {
	AppendAudit(context.Context, domain.AuditEvent) error
	ListAudit(context.Context, time.Time, time.Time) ([]domain.AuditEvent, error)
	AppendInvalidation(context.Context, domain.InvalidationEvent) error
	SaveNote(context.Context, domain.OperationNote) error
	SaveRetrospective(context.Context, domain.Retrospective) error
	IncrementCounter(context.Context, domain.ID, domain.ID, int64, int64, time.Time) error
}
