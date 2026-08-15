-- +goose Up
CREATE TABLE farms (
    id          UUID PRIMARY KEY,
    player_id   UUID NOT NULL UNIQUE,
    name        TEXT NOT NULL,
    epoch       TIMESTAMPTZ NOT NULL,
    revision    BIGINT NOT NULL DEFAULT 0,
    state       JSONB NOT NULL,
    cash_cents  BIGINT NOT NULL DEFAULT 0,
    biomass_ug  BIGINT NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE farm_events (
    farm_id     UUID NOT NULL REFERENCES farms ON DELETE CASCADE,
    seq         BIGINT NOT NULL,
    kind        TEXT NOT NULL,
    from_tick   BIGINT NOT NULL,
    to_tick     BIGINT NOT NULL,
    tank_id     BIGINT NOT NULL DEFAULT 0,
    fish        BIGINT NOT NULL DEFAULT 0,
    mass_ug     BIGINT NOT NULL DEFAULT 0,
    cash_cents  BIGINT NOT NULL DEFAULT 0,
    reason      TEXT NOT NULL DEFAULT 'none',
    PRIMARY KEY (farm_id, seq)
);

CREATE INDEX farm_events_recent ON farm_events (farm_id, seq DESC);

CREATE TABLE farm_actions (
    farm_id         UUID NOT NULL REFERENCES farms ON DELETE CASCADE,
    idempotency_key BIGINT NOT NULL,
    applied         BOOLEAN NOT NULL,
    reason          TEXT NOT NULL,
    at_tick         BIGINT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (farm_id, idempotency_key)
);

DROP TABLE IF EXISTS products;

-- +goose Down
CREATE TABLE products (
    id          UUID PRIMARY KEY,
    name        TEXT NOT NULL,
    price_cents BIGINT NOT NULL CHECK (price_cents > 0),
    created_at  TIMESTAMPTZ NOT NULL
);

DROP TABLE farm_actions;
DROP TABLE farm_events;
DROP TABLE farms;
