package repository

import (
	"context"

	"github.com/wyw14/cry-075/internal/domain"
)

func (m *MemoryStore) PutAsset(ctx context.Context, value domain.Asset) error {
	done := m.writeLock(ctx)
	defer done()
	if _, ok := m.assets[value.ID]; ok {
		return domain.ErrConflict
	}
	m.assets[value.ID] = value
	return nil
}
func (m *MemoryStore) GetAsset(ctx context.Context, id domain.ID) (domain.Asset, error) {
	done := m.readLock(ctx)
	defer done()
	v, ok := m.assets[id]
	if !ok {
		return domain.Asset{}, domain.ErrNotFound
	}
	return v, nil
}
func (m *MemoryStore) UpdateAsset(ctx context.Context, value domain.Asset, expected int64) error {
	done := m.writeLock(ctx)
	defer done()
	v, ok := m.assets[value.ID]
	if !ok {
		return domain.ErrNotFound
	}
	if v.Version != expected {
		return domain.ErrVersionConflict
	}
	m.assets[value.ID] = value
	return nil
}
func (m *MemoryStore) PutPackage(ctx context.Context, value domain.AssetPackage) error {
	done := m.writeLock(ctx)
	defer done()
	if _, ok := m.packages[value.ID]; ok {
		return domain.ErrConflict
	}
	m.packages[value.ID] = clonePackage(value)
	return nil
}
func (m *MemoryStore) GetPackage(ctx context.Context, id domain.ID) (domain.AssetPackage, error) {
	done := m.readLock(ctx)
	defer done()
	v, ok := m.packages[id]
	if !ok {
		return domain.AssetPackage{}, domain.ErrNotFound
	}
	return clonePackage(v), nil
}
func (m *MemoryStore) UpdatePackage(ctx context.Context, value domain.AssetPackage, expected int64) error {
	done := m.writeLock(ctx)
	defer done()
	v, ok := m.packages[value.ID]
	if !ok {
		return domain.ErrNotFound
	}
	if v.Version != expected {
		return domain.ErrVersionConflict
	}
	m.packages[value.ID] = clonePackage(value)
	return nil
}
func (m *MemoryStore) PutPlacement(ctx context.Context, value domain.Placement) error {
	done := m.writeLock(ctx)
	defer done()
	if _, ok := m.placements[value.ID]; ok {
		return domain.ErrConflict
	}
	m.placements[value.ID] = value
	return nil
}
func (m *MemoryStore) GetPlacement(ctx context.Context, id domain.ID) (domain.Placement, error) {
	done := m.readLock(ctx)
	defer done()
	v, ok := m.placements[id]
	if !ok {
		return domain.Placement{}, domain.ErrNotFound
	}
	return v, nil
}
func (m *MemoryStore) PutAudience(ctx context.Context, value domain.Audience) error {
	done := m.writeLock(ctx)
	defer done()
	if _, ok := m.audiences[value.ID]; ok {
		return domain.ErrConflict
	}
	m.audiences[value.ID] = cloneAudience(value)
	return nil
}
func (m *MemoryStore) GetAudience(ctx context.Context, id domain.ID) (domain.Audience, error) {
	done := m.readLock(ctx)
	defer done()
	v, ok := m.audiences[id]
	if !ok {
		return domain.Audience{}, domain.ErrNotFound
	}
	return cloneAudience(v), nil
}

func clonePackage(value domain.AssetPackage) domain.AssetPackage {
	value.Items = append([]domain.PackageItem(nil), value.Items...)
	return value
}
func cloneAudience(value domain.Audience) domain.Audience {
	attributes := make(map[string]string, len(value.Attributes))
	for k, v := range value.Attributes {
		attributes[k] = v
	}
	value.Attributes = attributes
	return value
}
