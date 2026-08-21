package repository

import (
	"context"
	"sort"
	"sync"

	"github.com/wyw14/cry-075/internal/domain"
)

type MemoryStore struct {
	mu            sync.RWMutex
	campaigns     map[domain.ID]domain.Campaign
	assets        map[domain.ID]domain.Asset
	packages      map[domain.ID]domain.AssetPackage
	placements    map[domain.ID]domain.Placement
	audiences     map[domain.ID]domain.Audience
	schedules     map[domain.ID]domain.Schedule
	executions    []domain.ExecutionLog
	snapshots     map[domain.ID]domain.ReleaseSnapshot
	releases      []domain.ReleaseVersion
	approvals     []domain.Approval
	fallbacks     []domain.FallbackCandidate
	audits        []domain.AuditEvent
	invalidations []domain.InvalidationEvent
	notes         []domain.OperationNote
	retros        []domain.Retrospective
	counters      map[string]domain.PerformanceCounter
	idempotency   map[string]memoryIdempotency
	scheduleKeys  map[string]memoryScheduleKey
}

type memoryIdempotency struct {
	Digest     string
	CampaignID domain.ID
}

type memoryScheduleKey struct {
	Digest     string
	ScheduleID domain.ID
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		campaigns: make(map[domain.ID]domain.Campaign), assets: make(map[domain.ID]domain.Asset), packages: make(map[domain.ID]domain.AssetPackage),
		placements: make(map[domain.ID]domain.Placement), audiences: make(map[domain.ID]domain.Audience), schedules: make(map[domain.ID]domain.Schedule),
		snapshots: make(map[domain.ID]domain.ReleaseSnapshot), counters: make(map[string]domain.PerformanceCounter), idempotency: make(map[string]memoryIdempotency), scheduleKeys: make(map[string]memoryScheduleKey),
	}
}

func (m *MemoryStore) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	backup := m.snapshot()
	if err := fn(context.WithValue(ctx, memoryTransactionKey{}, true)); err != nil {
		m.restore(backup)
		return err
	}
	return nil
}

type memoryTransactionKey struct{}

type memoryState struct {
	campaigns     map[domain.ID]domain.Campaign
	assets        map[domain.ID]domain.Asset
	packages      map[domain.ID]domain.AssetPackage
	placements    map[domain.ID]domain.Placement
	audiences     map[domain.ID]domain.Audience
	schedules     map[domain.ID]domain.Schedule
	executions    []domain.ExecutionLog
	snapshots     map[domain.ID]domain.ReleaseSnapshot
	releases      []domain.ReleaseVersion
	approvals     []domain.Approval
	fallbacks     []domain.FallbackCandidate
	audits        []domain.AuditEvent
	invalidations []domain.InvalidationEvent
	notes         []domain.OperationNote
	retros        []domain.Retrospective
	counters      map[string]domain.PerformanceCounter
	idempotency   map[string]memoryIdempotency
	scheduleKeys  map[string]memoryScheduleKey
}

func (m *MemoryStore) snapshot() memoryState {
	state := memoryState{
		campaigns: cloneMap(m.campaigns), assets: cloneMap(m.assets), placements: cloneMap(m.placements), schedules: cloneMap(m.schedules),
		executions: append([]domain.ExecutionLog(nil), m.executions...), snapshots: cloneMap(m.snapshots), releases: append([]domain.ReleaseVersion(nil), m.releases...),
		approvals: append([]domain.Approval(nil), m.approvals...), fallbacks: append([]domain.FallbackCandidate(nil), m.fallbacks...), audits: append([]domain.AuditEvent(nil), m.audits...),
		invalidations: append([]domain.InvalidationEvent(nil), m.invalidations...), notes: append([]domain.OperationNote(nil), m.notes...), retros: append([]domain.Retrospective(nil), m.retros...),
		counters: cloneMap(m.counters), idempotency: cloneMap(m.idempotency), scheduleKeys: cloneMap(m.scheduleKeys), packages: make(map[domain.ID]domain.AssetPackage, len(m.packages)), audiences: make(map[domain.ID]domain.Audience, len(m.audiences)),
	}
	for id, value := range m.packages {
		state.packages[id] = clonePackage(value)
	}
	for id, value := range m.audiences {
		state.audiences[id] = cloneAudience(value)
	}
	return state
}

func (m *MemoryStore) restore(state memoryState) {
	m.campaigns, m.assets, m.packages = state.campaigns, state.assets, state.packages
	m.placements, m.audiences, m.schedules = state.placements, state.audiences, state.schedules
	m.executions, m.snapshots, m.releases = state.executions, state.snapshots, state.releases
	m.approvals, m.fallbacks, m.audits = state.approvals, state.fallbacks, state.audits
	m.invalidations, m.notes, m.retros = state.invalidations, state.notes, state.retros
	m.counters, m.idempotency, m.scheduleKeys = state.counters, state.idempotency, state.scheduleKeys
}

func cloneMap[K comparable, V any](source map[K]V) map[K]V {
	clone := make(map[K]V, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func (m *MemoryStore) locked(ctx context.Context) bool {
	value, _ := ctx.Value(memoryTransactionKey{}).(bool)
	return value
}
func (m *MemoryStore) readLock(ctx context.Context) func() {
	if m.locked(ctx) {
		return func() {}
	}
	m.mu.RLock()
	return m.mu.RUnlock
}
func (m *MemoryStore) writeLock(ctx context.Context) func() {
	if m.locked(ctx) {
		return func() {}
	}
	m.mu.Lock()
	return m.mu.Unlock
}

func (m *MemoryStore) CreateCampaign(ctx context.Context, c domain.Campaign, key, digest string) (domain.Campaign, error) {
	done := m.writeLock(ctx)
	defer done()
	if previous, ok := m.idempotency[key]; ok {
		if previous.Digest != digest {
			return domain.Campaign{}, domain.ErrDuplicateRequest
		}
		return m.campaigns[previous.CampaignID], nil
	}
	if _, exists := m.campaigns[c.ID]; exists {
		return domain.Campaign{}, domain.ErrConflict
	}
	m.campaigns[c.ID] = c
	m.idempotency[key] = memoryIdempotency{Digest: digest, CampaignID: c.ID}
	return c, nil
}

func (m *MemoryStore) GetCampaign(ctx context.Context, id domain.ID) (domain.Campaign, error) {
	done := m.readLock(ctx)
	defer done()
	value, ok := m.campaigns[id]
	if !ok {
		return domain.Campaign{}, domain.ErrNotFound
	}
	return value, nil
}

func (m *MemoryStore) UpdateCampaign(ctx context.Context, c domain.Campaign, expected int64) error {
	done := m.writeLock(ctx)
	defer done()
	current, ok := m.campaigns[c.ID]
	if !ok {
		return domain.ErrNotFound
	}
	if current.Version != expected {
		return domain.ErrVersionConflict
	}
	m.campaigns[c.ID] = c
	return nil
}

func (m *MemoryStore) ListCampaigns(ctx context.Context, query domain.ListQuery) (domain.Page[domain.Campaign], error) {
	done := m.readLock(ctx)
	defer done()
	items := make([]domain.Campaign, 0)
	for _, item := range m.campaigns {
		if status := query.Filters["status"]; status != "" && string(item.Status) != status {
			continue
		}
		if env := query.Filters["environment"]; env != "" && item.Environment != env {
			continue
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		if query.Sort == "updated_at:asc" {
			return items[i].UpdatedAt.Before(items[j].UpdatedAt)
		}
		return items[i].UpdatedAt.After(items[j].UpdatedAt)
	})
	total := len(items)
	start := (query.Page - 1) * query.PerPage
	if start > total {
		start = total
	}
	end := start + query.PerPage
	if end > total {
		end = total
	}
	return domain.Page[domain.Campaign]{Items: items[start:end], Page: query.Page, PerPage: query.PerPage, Total: total}, nil
}

func (m *MemoryStore) FindPublishedOverlaps(ctx context.Context, placement domain.ID, env string, window domain.TimeWindow, exclude domain.ID) ([]domain.Campaign, error) {
	done := m.readLock(ctx)
	defer done()
	result := make([]domain.Campaign, 0)
	for _, item := range m.campaigns {
		if item.ID != exclude && item.PlacementID == placement && item.Environment == env && item.Status == domain.StatusPublished && item.Window.Overlaps(window) {
			result = append(result, item)
		}
	}
	return result, nil
}
