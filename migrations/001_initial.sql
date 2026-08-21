CREATE EXTENSION IF NOT EXISTS btree_gist;

CREATE TABLE IF NOT EXISTS campaigns (
  id text PRIMARY KEY,
  status text NOT NULL,
  placement_id text NOT NULL,
  environment text NOT NULL CHECK (environment IN ('development','staging','production')),
  starts_at timestamptz NOT NULL,
  ends_at timestamptz NOT NULL,
  version bigint NOT NULL CHECK (version > 0),
  payload jsonb NOT NULL,
  updated_at timestamptz NOT NULL DEFAULT now(),
  CHECK (starts_at < ends_at)
);

CREATE TABLE IF NOT EXISTS entity_store (
  kind text NOT NULL,
  id text NOT NULL,
  version bigint NOT NULL CHECK (version > 0),
  payload jsonb NOT NULL,
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (kind,id)
);

CREATE TABLE IF NOT EXISTS event_store (
  sequence bigserial PRIMARY KEY,
  kind text NOT NULL,
  id text NOT NULL,
  payload jsonb NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE(kind,id)
);

CREATE TABLE IF NOT EXISTS idempotency_keys (
  scope text NOT NULL,
  key text NOT NULL,
  request_hash text NOT NULL,
  response_id text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY(scope,key)
);

CREATE TABLE IF NOT EXISTS performance_counters (
  campaign_id text NOT NULL,
  asset_id text NOT NULL,
  views bigint NOT NULL DEFAULT 0 CHECK (views >= 0),
  clicks bigint NOT NULL DEFAULT 0 CHECK (clicks >= 0 AND clicks <= views),
  updated_at timestamptz NOT NULL,
  PRIMARY KEY(campaign_id,asset_id)
);

DO $$ BEGIN
  ALTER TABLE campaigns ADD CONSTRAINT published_placement_window_no_overlap
  EXCLUDE USING gist (placement_id WITH =, environment WITH =, tstzrange(starts_at,ends_at,'[)') WITH &&)
  WHERE (status='published');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

