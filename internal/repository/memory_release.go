package repository

import (
	"context"
	"sort"
	"time"

	"github.com/wyw14/cry-075/internal/domain"
)

func (m *MemoryStore) SaveApproval(ctx context.Context, v domain.Approval) error {
	done := m.writeLock(ctx)
	defer done()
	m.approvals = append(m.approvals, v)
	return nil
}
func (m *MemoryStore) SaveSnapshot(ctx context.Context, v domain.ReleaseSnapshot) error {
	done := m.writeLock(ctx)
	defer done()
	if _, ok := m.snapshots[v.ID]; ok {
		return domain.ErrConflict
	}
	m.snapshots[v.ID] = v
	return nil
}
func (m *MemoryStore) GetSnapshot(ctx context.Context, id domain.ID) (domain.ReleaseSnapshot, error) {
	done := m.readLock(ctx)
	defer done()
	v, ok := m.snapshots[id]
	if !ok {
		return domain.ReleaseSnapshot{}, domain.ErrNotFound
	}
	return v, nil
}
func (m *MemoryStore) SaveRelease(ctx context.Context, v domain.ReleaseVersion) error {
	done := m.writeLock(ctx)
	defer done()
	m.releases = append(m.releases, v)
	return nil
}
func (m *MemoryStore) LatestRelease(ctx context.Context, campaign domain.ID, env string) (domain.ReleaseVersion, error) {
	done := m.readLock(ctx)
	defer done()
	var found domain.ReleaseVersion
	ok := false
	for _, v := range m.releases {
		if v.CampaignID == campaign && v.Environment == env && (!ok || v.Sequence > found.Sequence) {
			found = v
			ok = true
		}
	}
	if !ok {
		return domain.ReleaseVersion{}, domain.ErrNotFound
	}
	return found, nil
}
func (m *MemoryStore) ListFallbacks(ctx context.Context, placement, audience domain.ID, at time.Time) ([]domain.FallbackCandidate, error) {
	done := m.readLock(ctx)
	defer done()
	values := make([]domain.FallbackCandidate, 0)
	for _, v := range m.fallbacks {
		if v.Eligible(placement, audience, at) {
			values = append(values, v)
		}
	}
	sort.Slice(values, func(i, j int) bool { return values[i].Priority < values[j].Priority })
	return values, nil
}
func (m *MemoryStore) SaveFallback(ctx context.Context, v domain.FallbackCandidate) error {
	done := m.writeLock(ctx)
	defer done()
	m.fallbacks = append(m.fallbacks, v)
	return nil
}
