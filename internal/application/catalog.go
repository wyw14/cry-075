package application

import (
	"context"
	"fmt"
	"time"

	"github.com/wyw14/cry-075/internal/domain"
	"github.com/wyw14/cry-075/internal/repository"
)

type CatalogService struct {
	Catalog      repository.CatalogRepository
	Transactions repository.UnitOfWork
	Audit        AuditWriter
}

func (s CatalogService) CreateAsset(ctx context.Context, name, mediaType, storageKey string, rights *time.Time, actor domain.Actor, requestID string) (domain.Asset, error) {
	if err := require(actor, "campaign.edit"); err != nil {
		return domain.Asset{}, err
	}
	asset, err := domain.NewAsset(name, mediaType, storageKey, rights)
	if err != nil {
		return domain.Asset{}, err
	}
	err = s.Transactions.WithinTransaction(ctx, func(tx context.Context) error {
		if err := s.Catalog.PutAsset(tx, asset); err != nil {
			return err
		}
		return s.Audit.Record(tx, AuditChange{Actor: actor, Action: "asset.created", Subject: "asset", SubjectID: asset.ID, RequestID: requestID, Metadata: map[string]any{"media_type": mediaType}})
	})
	return asset, err
}

func (s CatalogService) CreatePackage(ctx context.Context, name, env string, fallback bool, actor domain.Actor, requestID string) (domain.AssetPackage, error) {
	if err := require(actor, "campaign.edit"); err != nil {
		return domain.AssetPackage{}, err
	}
	pack, err := domain.NewAssetPackage(name, env, fallback)
	if err != nil {
		return domain.AssetPackage{}, err
	}
	err = s.Transactions.WithinTransaction(ctx, func(tx context.Context) error {
		if err := s.Catalog.PutPackage(tx, pack); err != nil {
			return err
		}
		return s.Audit.Record(tx, AuditChange{Actor: actor, Action: "package.created", Subject: "package", SubjectID: pack.ID, RequestID: requestID, Metadata: map[string]any{"fallback": fallback, "environment": env}})
	})
	return pack, err
}

type AddPackageItemCommand struct {
	PackageID, AssetID, ReplacementID domain.ID
	Pinned                            bool
	ExpectedVersion                   int64
	Actor                             domain.Actor
	RequestID                         string
}

func (s CatalogService) AddItem(ctx context.Context, command AddPackageItemCommand) (domain.AssetPackage, error) {
	if err := require(command.Actor, "campaign.edit"); err != nil {
		return domain.AssetPackage{}, err
	}
	pack, err := s.Catalog.GetPackage(ctx, command.PackageID)
	if err != nil {
		return domain.AssetPackage{}, err
	}
	asset, err := s.Catalog.GetAsset(ctx, command.AssetID)
	if err != nil {
		return domain.AssetPackage{}, err
	}
	if !asset.Eligible(time.Now().UTC()) {
		return domain.AssetPackage{}, fmt.Errorf("%w: primary asset is unavailable", domain.ErrInvalidReference)
	}
	if !command.ReplacementID.Empty() {
		replacement, err := s.Catalog.GetAsset(ctx, command.ReplacementID)
		if err != nil {
			return domain.AssetPackage{}, err
		}
		if !replacement.Eligible(time.Now().UTC()) {
			return domain.AssetPackage{}, fmt.Errorf("%w: replacement is unavailable", domain.ErrInvalidReference)
		}
	}
	if pack.Version != command.ExpectedVersion {
		return domain.AssetPackage{}, domain.ErrVersionConflict
	}
	before := pack.Version
	if err = pack.Add(command.AssetID, command.ReplacementID, command.Pinned); err != nil {
		return domain.AssetPackage{}, err
	}
	err = s.Transactions.WithinTransaction(ctx, func(tx context.Context) error {
		if err := s.Catalog.UpdatePackage(tx, pack, before); err != nil {
			return err
		}
		return s.Audit.Record(tx, AuditChange{
			Actor: command.Actor, Action: "package.asset_added", Subject: "package", SubjectID: pack.ID, RequestID: command.RequestID,
			Metadata: map[string]any{"asset_id": command.AssetID, "replacement_id": command.ReplacementID, "pinned": command.Pinned},
		})
	})
	return pack, err
}

func (s CatalogService) Reorder(ctx context.Context, packageID domain.ID, order []domain.ID, expected int64, actor domain.Actor, requestID string) (domain.AssetPackage, error) {
	if err := require(actor, "campaign.edit"); err != nil {
		return domain.AssetPackage{}, err
	}
	pack, err := s.Catalog.GetPackage(ctx, packageID)
	if err != nil {
		return domain.AssetPackage{}, err
	}
	if pack.Version != expected {
		return domain.AssetPackage{}, domain.ErrVersionConflict
	}
	before := pack.Version
	if err := pack.Reorder(order); err != nil {
		return domain.AssetPackage{}, err
	}
	err = s.Transactions.WithinTransaction(ctx, func(tx context.Context) error {
		if err := s.Catalog.UpdatePackage(tx, pack, before); err != nil {
			return err
		}
		return s.Audit.Record(tx, AuditChange{Actor: actor, Action: "package.reordered", Subject: "package", SubjectID: pack.ID, RequestID: requestID, Metadata: map[string]any{"order": order}})
	})
	return pack, err
}
