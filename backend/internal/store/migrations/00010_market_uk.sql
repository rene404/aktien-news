-- +goose Up
-- docs/plan-eu-sources.md, task 9: the FCA National Storage Mechanism (UK).
ALTER TABLE sources DROP CONSTRAINT sources_market_check;
ALTER TABLE sources ADD CONSTRAINT sources_market_check
    CHECK (market IN ('AT', 'DE', 'US', 'FR', 'ES', 'NL', 'NORDIC', 'UK', 'MULTI'));

-- +goose Down
-- Fails while UK sources exist: remove them first, nothing is dropped silently.
ALTER TABLE sources DROP CONSTRAINT sources_market_check;
ALTER TABLE sources ADD CONSTRAINT sources_market_check
    CHECK (market IN ('AT', 'DE', 'US', 'FR', 'ES', 'NL', 'NORDIC', 'MULTI'));
