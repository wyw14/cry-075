package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/wyw14/cry-075/internal/domain"
)

func (p *PostgresStore) PutAsset(ctx context.Context, v domain.Asset) error {
	return p.putEntity(ctx, "asset", v.ID, v.Version, v)
}
func (p *PostgresStore) GetAsset(ctx context.Context, id domain.ID) (domain.Asset, error) {
	var v domain.Asset
	return v, p.getEntity(ctx, "asset", id, &v)
}
func (p *PostgresStore) UpdateAsset(ctx context.Context, v domain.Asset, e int64) error {
	return p.updateEntity(ctx, "asset", v.ID, e, v.Version, v)
}
func (p *PostgresStore) PutPackage(ctx context.Context, v domain.AssetPackage) error {
	return p.putEntity(ctx, "package", v.ID, v.Version, v)
}
func (p *PostgresStore) GetPackage(ctx context.Context, id domain.ID) (domain.AssetPackage, error) {
	var v domain.AssetPackage
	return v, p.getEntity(ctx, "package", id, &v)
}
func (p *PostgresStore) UpdatePackage(ctx context.Context, v domain.AssetPackage, e int64) error {
	return p.updateEntity(ctx, "package", v.ID, e, v.Version, v)
}
func (p *PostgresStore) PutPlacement(ctx context.Context, v domain.Placement) error {
	return p.putEntity(ctx, "placement", v.ID, v.Version, v)
}
func (p *PostgresStore) GetPlacement(ctx context.Context, id domain.ID) (domain.Placement, error) {
	var v domain.Placement
	return v, p.getEntity(ctx, "placement", id, &v)
}
func (p *PostgresStore) PutAudience(ctx context.Context, v domain.Audience) error {
	return p.putEntity(ctx, "audience", v.ID, 1, v)
}
func (p *PostgresStore) GetAudience(ctx context.Context, id domain.ID) (domain.Audience, error) {
	var v domain.Audience
	return v, p.getEntity(ctx, "audience", id, &v)
}

func (p *PostgresStore) append(ctx context.Context, kind string, id domain.ID, value any) error {
	payload, err := encode(value)
	if err != nil {
		return err
	}
	_, err = p.executor(ctx).Exec(ctx, `INSERT INTO event_store(kind,id,payload) VALUES($1,$2,$3)`, kind, id, payload)
	return err
}
func (p *PostgresStore) SaveApproval(ctx context.Context, v domain.Approval) error {
	return p.append(ctx, "approval", v.ID, v)
}
func (p *PostgresStore) SaveSnapshot(ctx context.Context, v domain.ReleaseSnapshot) error {
	return p.putEntity(ctx, "snapshot", v.ID, 1, v)
}
func (p *PostgresStore) GetSnapshot(ctx context.Context, id domain.ID) (domain.ReleaseSnapshot, error) {
	var v domain.ReleaseSnapshot
	return v, p.getEntity(ctx, "snapshot", id, &v)
}
func (p *PostgresStore) SaveRelease(ctx context.Context, v domain.ReleaseVersion) error {
	return p.append(ctx, "release", v.ID, v)
}
func (p *PostgresStore) LatestRelease(ctx context.Context, id domain.ID, env string) (domain.ReleaseVersion, error) {
	rows, err := p.executor(ctx).Query(ctx, `SELECT payload FROM event_store WHERE kind='release' ORDER BY created_at DESC`)
	if err != nil {
		return domain.ReleaseVersion{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			return domain.ReleaseVersion{}, err
		}
		var v domain.ReleaseVersion
		if json.Unmarshal(payload, &v) == nil && v.CampaignID == id && v.Environment == env {
			return v, nil
		}
	}
	return domain.ReleaseVersion{}, domain.ErrNotFound
}
func (p *PostgresStore) SaveFallback(ctx context.Context, v domain.FallbackCandidate) error {
	return p.append(ctx, "fallback", domain.NewID("fbk"), v)
}
func (p *PostgresStore) ListFallbacks(ctx context.Context, placement, audience domain.ID, at time.Time) ([]domain.FallbackCandidate, error) {
	rows, err := p.executor(ctx).Query(ctx, `SELECT payload FROM event_store WHERE kind='fallback'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := make([]domain.FallbackCandidate, 0)
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		var v domain.FallbackCandidate
		if json.Unmarshal(payload, &v) == nil && v.Eligible(placement, audience, at) {
			values = append(values, v)
		}
	}
	return values, rows.Err()
}

func (p *PostgresStore) SaveSchedule(ctx context.Context, v domain.Schedule) (domain.Schedule, error) {
	exec := p.executor(ctx)
	digest := digestSchedule(v)
	var existing, responseID string
	err := exec.QueryRow(ctx, `SELECT request_hash,response_id FROM idempotency_keys WHERE scope='schedule.plan' AND key=$1`, v.IdempotencyKey).Scan(&existing, &responseID)
	if err == nil {
		if existing != digest {
			return domain.Schedule{}, domain.ErrDuplicateRequest
		}
		var stored domain.Schedule
		if err := p.getEntity(ctx, "schedule", domain.ID(responseID), &stored); err != nil {
			return domain.Schedule{}, err
		}
		return stored, nil
	}
	if err != pgx.ErrNoRows {
		return domain.Schedule{}, err
	}
	if err := p.putEntity(ctx, "schedule", v.ID, 1, v); err != nil {
		return domain.Schedule{}, err
	}
	if _, err := exec.Exec(ctx, `INSERT INTO idempotency_keys(scope,key,request_hash,response_id) VALUES('schedule.plan',$1,$2,$3)`, v.IdempotencyKey, digest, v.ID); err != nil {
		return domain.Schedule{}, err
	}
	return v, nil
}
func (p *PostgresStore) DueSchedules(ctx context.Context, at time.Time, limit int) ([]domain.Schedule, error) {
	rows, err := p.executor(ctx).Query(ctx, `SELECT payload FROM entity_store WHERE kind='schedule' ORDER BY updated_at LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := make([]domain.Schedule, 0)
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		var v domain.Schedule
		if json.Unmarshal(payload, &v) == nil && v.ExecutedAt == nil && !v.ExecuteAt.After(at) {
			values = append(values, v)
		}
	}
	return values, rows.Err()
}
func (p *PostgresStore) UpdateSchedule(ctx context.Context, v domain.Schedule) error {
	payload, err := encode(v)
	if err != nil {
		return err
	}
	_, err = p.executor(ctx).Exec(ctx, `UPDATE entity_store SET version=version+1,payload=$3,updated_at=now() WHERE kind='schedule' AND id=$2`, "schedule", v.ID, payload)
	return err
}
func (p *PostgresStore) SaveExecution(ctx context.Context, v domain.ExecutionLog) error {
	return p.append(ctx, "execution", v.ID, v)
}
func (p *PostgresStore) AppendAudit(ctx context.Context, v domain.AuditEvent) error {
	return p.append(ctx, "audit", v.ID, v)
}
func (p *PostgresStore) ListAudit(ctx context.Context, from, to time.Time) ([]domain.AuditEvent, error) {
	rows, err := p.executor(ctx).Query(ctx, `SELECT payload FROM event_store WHERE kind='audit' AND created_at >= $1 AND created_at < $2 ORDER BY created_at`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := make([]domain.AuditEvent, 0)
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		var v domain.AuditEvent
		if err := json.Unmarshal(payload, &v); err != nil {
			return nil, err
		}
		values = append(values, v)
	}
	return values, rows.Err()
}
func (p *PostgresStore) AppendInvalidation(ctx context.Context, v domain.InvalidationEvent) error {
	return p.append(ctx, "invalidation", v.ID, v)
}
func (p *PostgresStore) SaveNote(ctx context.Context, v domain.OperationNote) error {
	return p.append(ctx, "note", v.ID, v)
}
func (p *PostgresStore) SaveRetrospective(ctx context.Context, v domain.Retrospective) error {
	return p.append(ctx, "retrospective", v.ID, v)
}
func (p *PostgresStore) IncrementCounter(ctx context.Context, campaign, asset domain.ID, views, clicks int64, at time.Time) error {
	_, err := p.executor(ctx).Exec(ctx, `INSERT INTO performance_counters(campaign_id,asset_id,views,clicks,updated_at) VALUES($1,$2,$3,$4,$5) ON CONFLICT(campaign_id,asset_id) DO UPDATE SET views=performance_counters.views+EXCLUDED.views,clicks=performance_counters.clicks+EXCLUDED.clicks,updated_at=EXCLUDED.updated_at`, campaign, asset, views, clicks, at)
	return err
}

var _ = pgx.ErrNoRows
