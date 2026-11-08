-- +goose Up
-- docs/plan-eu-sources.md, task 1: sources for France, Spain and the Netherlands.
ALTER TABLE sources DROP CONSTRAINT sources_market_check;
ALTER TABLE sources ADD CONSTRAINT sources_market_check
    CHECK (market IN ('AT', 'DE', 'US', 'FR', 'ES', 'NL', 'MULTI'));

-- +goose Down
-- Fails while FR, ES or NL sources exist: remove them first, nothing is dropped silently.
ALTER TABLE sources DROP CONSTRAINT sources_market_check;
ALTER TABLE sources ADD CONSTRAINT sources_market_check
    CHECK (market IN ('AT', 'DE', 'US', 'MULTI'));
