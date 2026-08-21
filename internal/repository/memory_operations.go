package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/wyw14/cry-075/internal/domain"
)

func (m *MemoryStore) SaveSchedule(ctx context.Context, v domain.Schedule) (domain.Schedule, error) {
	done := m.writeLock(ctx)
	defer done()
	digest := digestSchedule(v)
	if previous, ok := m.scheduleKeys[v.IdempotencyKey]; ok {
		if previous.Digest != digest {
			return domain.Schedule{}, domain.ErrDuplicateRequest
		}
		return m.schedules[previous.ScheduleID], nil
	}
	if _, ok := m.schedules[v.ID]; ok {
		return domain.Schedule{}, domain.ErrConflict
	}
	m.schedules[v.ID] = v
	m.scheduleKeys[v.IdempotencyKey] = memoryScheduleKey{Digest: digest, ScheduleID: v.ID}
	return v, nil
}

func digestSchedule(v domain.Schedule) string {
	v.ID = ""
	payload, _ := json.Marshal(v)
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}
func (m *MemoryStore) DueSchedules(ctx context.Context, at time.Time, limit int) ([]domain.Schedule, error) {
	done := m.readLock(ctx)
	defer done()
	values := make([]domain.Schedule, 0)
	for _, v := range m.schedules {
		if v.ExecutedAt == nil && !v.ExecuteAt.After(at) {
			values = append(values, v)
			if len(values) == limit {
				break
			}
		}
	}
	return values, nil
}
func (m *MemoryStore) UpdateSchedule(ctx context.Context, v domain.Schedule) error {
	done := m.writeLock(ctx)
	defer done()
	if _, ok := m.schedules[v.ID]; !ok {
		return domain.ErrNotFound
	}
	m.schedules[v.ID] = v
	return nil
}
func (m *MemoryStore) SaveExecution(ctx context.Context, v domain.ExecutionLog) error {
	done := m.writeLock(ctx)
	defer done()
	m.executions = append(m.executions, v)
	return nil
}
func (m *MemoryStore) AppendAudit(ctx context.Context, v domain.AuditEvent) error {
	done := m.writeLock(ctx)
	defer done()
	m.audits = append(m.audits, v)
	return nil
}
func (m *MemoryStore) ListAudit(ctx context.Context, from, to time.Time) ([]domain.AuditEvent, error) {
	done := m.readLock(ctx)
	defer done()
	values := make([]domain.AuditEvent, 0)
	for _, v := range m.audits {
		if !v.CreatedAt.Before(from) && v.CreatedAt.Before(to) {
			values = append(values, v)
		}
	}
	return values, nil
}
func (m *MemoryStore) AppendInvalidation(ctx context.Context, v domain.InvalidationEvent) error {
	done := m.writeLock(ctx)
	defer done()
	m.invalidations = append(m.invalidations, v)
	return nil
}
func (m *MemoryStore) SaveNote(ctx context.Context, v domain.OperationNote) error {
	done := m.writeLock(ctx)
	defer done()
	m.notes = append(m.notes, v)
	return nil
}
func (m *MemoryStore) SaveRetrospective(ctx context.Context, v domain.Retrospective) error {
	done := m.writeLock(ctx)
	defer done()
	m.retros = append(m.retros, v)
	return nil
}
func (m *MemoryStore) IncrementCounter(ctx context.Context, campaign, asset domain.ID, views, clicks int64, at time.Time) error {
	done := m.writeLock(ctx)
	defer done()
	key := string(campaign) + ":" + string(asset)
	v := m.counters[key]
	v.CampaignID = campaign
	v.AssetID = asset
	v.Views += views
	v.Clicks += clicks
	v.UpdatedAt = at
	m.counters[key] = v
	return nil
}
