-- +goose Up
-- docs/plan-eu-sources.md, task 2: the issuer's LEI as a matching signal.
ALTER TYPE match_method ADD VALUE IF NOT EXISTS 'lei';

-- +goose Down
-- Postgres cannot drop an enum value. Rebuild the type without 'lei'; this
-- fails while article_entities holds 'lei' rows, so nothing is lost silently.
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM article_entities WHERE method = 'lei') THEN
        RAISE EXCEPTION 'article_entities holds links with method lei; remove them first';
    END IF;
END $$;
-- +goose StatementEnd
ALTER TYPE match_method RENAME TO match_method_old;
CREATE TYPE match_method AS ENUM ('issuer', 'isin', 'cashtag', 'ticker', 'full_name', 'core_name', 'admin');
ALTER TABLE article_entities ALTER COLUMN method TYPE match_method USING method::text::match_method;
DROP TYPE match_method_old;
