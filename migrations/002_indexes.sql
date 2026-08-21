CREATE INDEX IF NOT EXISTS campaigns_status_environment_idx ON campaigns(status, environment, updated_at DESC);
CREATE INDEX IF NOT EXISTS event_store_kind_created_idx ON event_store(kind, created_at DESC);
CREATE INDEX IF NOT EXISTS entity_store_kind_updated_idx ON entity_store(kind, updated_at DESC);

