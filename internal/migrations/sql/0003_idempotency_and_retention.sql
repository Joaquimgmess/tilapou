-- +goose Up
ALTER TABLE farm_actions ADD COLUMN needed_cents BIGINT NOT NULL DEFAULT 0;

CREATE INDEX farm_events_prune ON farm_events (farm_id, seq);

-- +goose Down
DROP INDEX farm_events_prune;

ALTER TABLE farm_actions DROP COLUMN needed_cents;
