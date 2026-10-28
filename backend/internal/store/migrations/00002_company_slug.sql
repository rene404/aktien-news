-- +goose Up
-- Stable natural key so the master-data seed can upsert companies.
ALTER TABLE companies ADD COLUMN slug text NOT NULL UNIQUE;

-- +goose Down
ALTER TABLE companies DROP COLUMN slug;
