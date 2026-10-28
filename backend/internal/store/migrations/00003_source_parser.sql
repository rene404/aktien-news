-- +goose Up
-- Which ingest parser handles a source. Explicit instead of derived from the key.
ALTER TABLE sources ADD COLUMN parser text NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE sources DROP COLUMN parser;
