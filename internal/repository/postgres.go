package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wyw14/cry-075/internal/domain"
)

type PostgresStore struct{ pool *pgxpool.Pool }
type transactionContextKey struct{}

func OpenPostgres(ctx context.Context, databaseURL string) (*PostgresStore, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("create pg pool: %w", err)
	}
	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return &PostgresStore{pool: pool}, nil
}

func (p *PostgresStore) Close()                         { p.pool.Close() }
func (p *PostgresStore) Ping(ctx context.Context) error { return p.pool.Ping(ctx) }

func (p *PostgresStore) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	return pgx.BeginFunc(ctx, p.pool, func(tx pgx.Tx) error { return fn(context.WithValue(ctx, transactionContextKey{}, tx)) })
}

type sqlExecutor interface {
	Exec(context.Context, string, ...any) (pgconnCommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

type pgconnCommandTag interface{ RowsAffected() int64 }

func (p *PostgresStore) executor(ctx context.Context) interface {
	Exec(context.Context, string, ...any) (pgconnCommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
} {
	if tx, ok := ctx.Value(transactionContextKey{}).(pgx.Tx); ok {
		return txAdapter{tx}
	}
	return poolAdapter{p.pool}
}

type txAdapter struct{ pgx.Tx }

func (a txAdapter) Exec(ctx context.Context, sql string, args ...any) (pgconnCommandTag, error) {
	return a.Tx.Exec(ctx, sql, args...)
}

type poolAdapter struct{ *pgxpool.Pool }

func (a poolAdapter) Exec(ctx context.Context, sql string, args ...any) (pgconnCommandTag, error) {
	return a.Pool.Exec(ctx, sql, args...)
}

func encode(value any) ([]byte, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode entity: %w", err)
	}
	return payload, nil
}
func translate(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	return err
}

func (p *PostgresStore) CreateCampaign(ctx context.Context, value domain.Campaign, key, digest string) (domain.Campaign, error) {
	payload, err := encode(value)
	if err != nil {
		return domain.Campaign{}, err
	}
	exec := p.executor(ctx)
	var existing, responseID string
	err = exec.QueryRow(ctx, `SELECT request_hash,response_id FROM idempotency_keys WHERE scope='campaign.create' AND key=$1`, key).Scan(&existing, &responseID)
	if err == nil {
		if existing != digest {
			return domain.Campaign{}, domain.ErrDuplicateRequest
		}
		return p.GetCampaign(ctx, domain.ID(responseID))
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return domain.Campaign{}, fmt.Errorf("check idempotency: %w", err)
	}
	if _, err = exec.Exec(ctx, `INSERT INTO campaigns(id,status,placement_id,environment,starts_at,ends_at,version,payload) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, value.ID, value.Status, value.PlacementID, value.Environment, value.Window.StartsAt, value.Window.EndsAt, value.Version, payload); err != nil {
		return domain.Campaign{}, fmt.Errorf("insert campaign: %w", err)
	}
	_, err = exec.Exec(ctx, `INSERT INTO idempotency_keys(scope,key,request_hash,response_id) VALUES('campaign.create',$1,$2,$3)`, key, digest, value.ID)
	return value, err
}

func (p *PostgresStore) GetCampaign(ctx context.Context, id domain.ID) (domain.Campaign, error) {
	var payload []byte
	err := p.executor(ctx).QueryRow(ctx, `SELECT payload FROM campaigns WHERE id=$1`, id).Scan(&payload)
	if err != nil {
		return domain.Campaign{}, translate(err)
	}
	var value domain.Campaign
	if err = json.Unmarshal(payload, &value); err != nil {
		return value, err
	}
	return value, nil
}

func (p *PostgresStore) UpdateCampaign(ctx context.Context, value domain.Campaign, expected int64) error {
	payload, err := encode(value)
	if err != nil {
		return err
	}
	tag, err := p.executor(ctx).Exec(ctx, `UPDATE campaigns SET status=$2,placement_id=$3,environment=$4,starts_at=$5,ends_at=$6,version=$7,payload=$8,updated_at=now() WHERE id=$1 AND version=$9`, value.ID, value.Status, value.PlacementID, value.Environment, value.Window.StartsAt, value.Window.EndsAt, value.Version, payload, expected)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return domain.ErrVersionConflict
	}
	return nil
}

func (p *PostgresStore) ListCampaigns(ctx context.Context, q domain.ListQuery) (domain.Page[domain.Campaign], error) {
	offset := (q.Page - 1) * q.PerPage
	status, env := q.Filters["status"], q.Filters["environment"]
	direction := "DESC"
	if q.Sort == "updated_at:asc" {
		direction = "ASC"
	}
	sql := `SELECT payload,count(*) OVER() FROM campaigns WHERE ($1='' OR status=$1) AND ($2='' OR environment=$2) ORDER BY updated_at ` + direction + ` LIMIT $3 OFFSET $4`
	rows, err := p.executor(ctx).Query(ctx, sql, status, env, q.PerPage, offset)
	if err != nil {
		return domain.Page[domain.Campaign]{}, err
	}
	defer rows.Close()
	items := make([]domain.Campaign, 0)
	total := 0
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload, &total); err != nil {
			return domain.Page[domain.Campaign]{}, err
		}
		var v domain.Campaign
		if err := json.Unmarshal(payload, &v); err != nil {
			return domain.Page[domain.Campaign]{}, err
		}
		items = append(items, v)
	}
	return domain.Page[domain.Campaign]{Items: items, Page: q.Page, PerPage: q.PerPage, Total: total}, rows.Err()
}

func (p *PostgresStore) FindPublishedOverlaps(ctx context.Context, placement domain.ID, env string, window domain.TimeWindow, exclude domain.ID) ([]domain.Campaign, error) {
	rows, err := p.executor(ctx).Query(ctx, `SELECT payload FROM campaigns WHERE placement_id=$1 AND environment=$2 AND status='published' AND id<>$3 AND tstzrange(starts_at,ends_at,'[)') && tstzrange($4,$5,'[)')`, placement, env, exclude, window.StartsAt, window.EndsAt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := make([]domain.Campaign, 0)
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		var v domain.Campaign
		if err := json.Unmarshal(payload, &v); err != nil {
			return nil, err
		}
		values = append(values, v)
	}
	return values, rows.Err()
}

func (p *PostgresStore) putEntity(ctx context.Context, kind string, id domain.ID, version int64, value any) error {
	payload, err := encode(value)
	if err != nil {
		return err
	}
	_, err = p.executor(ctx).Exec(ctx, `INSERT INTO entity_store(kind,id,version,payload) VALUES($1,$2,$3,$4)`, kind, id, version, payload)
	return err
}
func (p *PostgresStore) getEntity(ctx context.Context, kind string, id domain.ID, value any) error {
	var payload []byte
	err := p.executor(ctx).QueryRow(ctx, `SELECT payload FROM entity_store WHERE kind=$1 AND id=$2`, kind, id).Scan(&payload)
	if err != nil {
		return translate(err)
	}
	return json.Unmarshal(payload, value)
}
func (p *PostgresStore) updateEntity(ctx context.Context, kind string, id domain.ID, expected, next int64, value any) error {
	payload, err := encode(value)
	if err != nil {
		return err
	}
	tag, err := p.executor(ctx).Exec(ctx, `UPDATE entity_store SET version=$3,payload=$4,updated_at=now() WHERE kind=$1 AND id=$2 AND version=$5`, kind, id, next, payload, expected)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return domain.ErrVersionConflict
	}
	return nil
}
