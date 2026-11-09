-- +goose Up
-- docs/plan-eu-sources.md, task 9: a source whose items may not reach the LLM
-- at all, not even their titles (FCA NSM: Content-Signal ai-train=no).
ALTER TYPE text_policy ADD VALUE IF NOT EXISTS 'no_llm';

-- +goose Down
-- Postgres cannot drop an enum value. Rebuild the type without 'no_llm'; this
-- fails while a source uses it, so nothing is loosened silently.
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM sources WHERE text_policy = 'no_llm') THEN
        RAISE EXCEPTION 'sources use text_policy no_llm; change them first';
    END IF;
END $$;
-- +goose StatementEnd
ALTER TABLE sources ALTER COLUMN text_policy DROP DEFAULT;
ALTER TYPE text_policy RENAME TO text_policy_old;
CREATE TYPE text_policy AS ENUM ('none', 'feed_fields', 'full_text');
ALTER TABLE sources ALTER COLUMN text_policy TYPE text_policy USING text_policy::text::text_policy;
ALTER TABLE sources ALTER COLUMN text_policy SET DEFAULT 'none';
DROP TYPE text_policy_old;
