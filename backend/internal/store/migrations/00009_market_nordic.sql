-- +goose Up
-- docs/plan-eu-sources.md, task 8: Nasdaq Nordic covers Stockholm, Helsinki
-- and Copenhagen in one feed, so its sources get one market.
ALTER TABLE sources DROP CONSTRAINT sources_market_check;
ALTER TABLE sources ADD CONSTRAINT sources_market_check
    CHECK (market IN ('AT', 'DE', 'US', 'FR', 'ES', 'NL', 'NORDIC', 'MULTI'));

-- +goose Down
-- Fails while NORDIC sources exist: remove them first, nothing is dropped silently.
ALTER TABLE sources DROP CONSTRAINT sources_market_check;
ALTER TABLE sources ADD CONSTRAINT sources_market_check
    CHECK (market IN ('AT', 'DE', 'US', 'FR', 'ES', 'NL', 'MULTI'));
