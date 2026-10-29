-- +goose Up
-- false for companies bulk-imported from the SEC ticker list: the matcher
-- links them only on strong signals.
ALTER TABLE companies ADD COLUMN curated boolean NOT NULL DEFAULT true;

-- +goose Down
ALTER TABLE companies DROP COLUMN curated;
