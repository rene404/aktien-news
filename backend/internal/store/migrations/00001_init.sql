-- +goose Up
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TYPE source_kind AS ENUM ('adhoc', 'press_release', 'news_api', 'ir', 'filing');
CREATE TYPE text_policy AS ENUM ('none', 'feed_fields', 'full_text');
CREATE TYPE link_status AS ENUM ('linked', 'pending', 'rejected');
CREATE TYPE match_method AS ENUM ('issuer', 'isin', 'cashtag', 'ticker', 'full_name', 'core_name', 'admin');
CREATE TYPE event_type AS ENUM (
    'earnings', 'guidance', 'm_and_a', 'rating_change', 'litigation', 'insider',
    'management_change', 'capital_markets', 'dividend', 'regulatory', 'adhoc_other', 'other'
);
CREATE TYPE run_status AS ENUM ('ok', 'partial', 'failed', 'skipped');

-- One row per feed/endpoint. text_policy decides what may reach the LLM;
-- a detected TDM reservation (tdm_reserved) overrides it to 'none'.
CREATE TABLE sources (
    id             bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    key            text        NOT NULL UNIQUE,
    name           text        NOT NULL,
    kind           source_kind NOT NULL,
    market         text        NOT NULL CHECK (market IN ('AT', 'DE', 'US', 'MULTI')),
    url            text        NOT NULL,
    weight         numeric(3,2) NOT NULL DEFAULT 0.50 CHECK (weight BETWEEN 0 AND 1),
    enabled        boolean     NOT NULL DEFAULT true,
    fetch_interval interval    NOT NULL DEFAULT '10 minutes',
    text_policy    text_policy NOT NULL DEFAULT 'none',
    tdm_reserved   boolean     NOT NULL DEFAULT false,
    tdm_checked_at timestamptz,
    terms_url      text,
    notes          text,
    created_at     timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE companies (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name       text NOT NULL,
    legal_name text,
    lei        text UNIQUE,
    cik        text UNIQUE,
    country    char(2) NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE listings (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    company_id bigint NOT NULL REFERENCES companies (id) ON DELETE CASCADE,
    isin       text,
    ticker     text NOT NULL,
    mic        text NOT NULL,
    currency   char(3),
    active     boolean NOT NULL DEFAULT true,
    UNIQUE (mic, ticker)
);
CREATE INDEX listings_isin_idx ON listings (isin);
CREATE INDEX listings_company_idx ON listings (company_id);

-- Point-in-time index composition.
CREATE TABLE index_memberships (
    listing_id bigint NOT NULL REFERENCES listings (id) ON DELETE CASCADE,
    index_key  text   NOT NULL,
    valid_from date   NOT NULL,
    valid_to   date,
    PRIMARY KEY (listing_id, index_key, valid_from),
    CHECK (valid_to IS NULL OR valid_to > valid_from)
);

CREATE TABLE company_aliases (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    company_id bigint NOT NULL REFERENCES companies (id) ON DELETE CASCADE,
    alias      text   NOT NULL,
    kind       text   NOT NULL CHECK (kind IN ('core_name', 'former_name', 'brand', 'issuer_name')),
    UNIQUE (company_id, alias)
);
CREATE INDEX company_aliases_trgm_idx ON company_aliases USING gin (alias gin_trgm_ops);

CREATE TABLE clusters (
    id                        bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    first_seen_at             timestamptz NOT NULL,
    representative_article_id bigint,
    size                      integer NOT NULL DEFAULT 1 CHECK (size >= 1)
);

CREATE TABLE articles (
    id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    source_id     bigint      NOT NULL REFERENCES sources (id),
    external_id   text        NOT NULL,
    url           text        NOT NULL,
    canonical_url text        NOT NULL UNIQUE,
    title         text        NOT NULL,
    lang          text,
    published_at  timestamptz NOT NULL,
    ingested_at   timestamptz NOT NULL DEFAULT now(),
    content_hash  bytea       NOT NULL,
    simhash       bigint,
    cluster_id    bigint REFERENCES clusters (id),
    issuer_hint   text,
    issuer_isin   text,
    UNIQUE (source_id, external_id)
);
CREATE INDEX articles_published_idx ON articles (published_at DESC);
CREATE INDEX articles_cluster_idx ON articles (cluster_id);
CREATE INDEX articles_content_hash_idx ON articles (content_hash);

ALTER TABLE clusters
    ADD CONSTRAINT clusters_representative_fk
    FOREIGN KEY (representative_article_id) REFERENCES articles (id)
    DEFERRABLE INITIALLY DEFERRED;

-- Internal only, purged after purge_after. Never served by any API.
CREATE TABLE article_texts (
    article_id  bigint PRIMARY KEY REFERENCES articles (id) ON DELETE CASCADE,
    body        text        NOT NULL,
    fetched_at  timestamptz NOT NULL DEFAULT now(),
    purge_after timestamptz NOT NULL
);

-- Append-only: a status change is a new row. Current state = latest decided_at.
CREATE TABLE article_entities (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    article_id bigint       NOT NULL REFERENCES articles (id) ON DELETE CASCADE,
    company_id bigint       NOT NULL REFERENCES companies (id) ON DELETE CASCADE,
    confidence numeric(4,3) NOT NULL CHECK (confidence BETWEEN 0 AND 1),
    method     match_method NOT NULL,
    status     link_status  NOT NULL,
    decided_at timestamptz  NOT NULL DEFAULT now(),
    decided_by text         NOT NULL CHECK (decided_by IN ('engine', 'admin'))
);
CREATE INDEX article_entities_lookup_idx ON article_entities (article_id, company_id, decided_at DESC);
CREATE INDEX article_entities_company_idx ON article_entities (company_id, decided_at DESC);

-- Append-only: reclassification is a new row.
CREATE TABLE classifications (
    id             bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    article_id     bigint       NOT NULL REFERENCES articles (id) ON DELETE CASCADE,
    company_id     bigint       NOT NULL REFERENCES companies (id) ON DELETE CASCADE,
    event_type     event_type   NOT NULL,
    sentiment      numeric(4,3) NOT NULL CHECK (sentiment BETWEEN -1 AND 1),
    confidence     numeric(4,3) NOT NULL CHECK (confidence BETWEEN 0 AND 1),
    novelty        numeric(4,3) NOT NULL CHECK (novelty BETWEEN 0 AND 1),
    model          text         NOT NULL,
    prompt_version text         NOT NULL,
    classified_at  timestamptz  NOT NULL DEFAULT now()
);
CREATE INDEX classifications_lookup_idx ON classifications (company_id, classified_at DESC);
CREATE INDEX classifications_article_idx ON classifications (article_id, company_id, classified_at DESC);

CREATE TABLE external_sentiment (
    id                    bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    provider              text    NOT NULL,
    article_id            bigint  REFERENCES articles (id) ON DELETE SET NULL,
    listing_id            bigint  NOT NULL REFERENCES listings (id) ON DELETE CASCADE,
    score                 numeric NOT NULL,
    label                 text,
    relevance             numeric,
    provider_published_at timestamptz,
    fetched_at            timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX external_sentiment_listing_idx ON external_sentiment (listing_id, fetched_at DESC);

CREATE TABLE ingest_runs (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    source_id   bigint      NOT NULL REFERENCES sources (id),
    started_at  timestamptz NOT NULL DEFAULT now(),
    finished_at timestamptz,
    status      run_status,
    fetched     integer NOT NULL DEFAULT 0,
    new         integer NOT NULL DEFAULT 0,
    error       text
);
CREATE INDEX ingest_runs_source_idx ON ingest_runs (source_id, started_at DESC);

-- Invariants enforced in the database, not just in code.
-- +goose StatementBegin
CREATE FUNCTION forbid_update() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION '% is append-only', TG_TABLE_NAME;
END;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION keep_ingested_at() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.ingested_at IS DISTINCT FROM OLD.ingested_at THEN
        RAISE EXCEPTION 'articles.ingested_at is immutable';
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER article_entities_append_only BEFORE UPDATE ON article_entities
    FOR EACH ROW EXECUTE FUNCTION forbid_update();
CREATE TRIGGER classifications_append_only BEFORE UPDATE ON classifications
    FOR EACH ROW EXECUTE FUNCTION forbid_update();
CREATE TRIGGER articles_ingested_at_immutable BEFORE UPDATE ON articles
    FOR EACH ROW EXECUTE FUNCTION keep_ingested_at();

-- +goose Down
DROP TRIGGER IF EXISTS articles_ingested_at_immutable ON articles;
DROP TRIGGER IF EXISTS classifications_append_only ON classifications;
DROP TRIGGER IF EXISTS article_entities_append_only ON article_entities;
DROP FUNCTION IF EXISTS keep_ingested_at();
DROP FUNCTION IF EXISTS forbid_update();

DROP TABLE IF EXISTS ingest_runs;
DROP TABLE IF EXISTS external_sentiment;
DROP TABLE IF EXISTS classifications;
DROP TABLE IF EXISTS article_entities;
DROP TABLE IF EXISTS article_texts;
ALTER TABLE IF EXISTS clusters DROP CONSTRAINT IF EXISTS clusters_representative_fk;
DROP TABLE IF EXISTS articles;
DROP TABLE IF EXISTS clusters;
DROP TABLE IF EXISTS company_aliases;
DROP TABLE IF EXISTS index_memberships;
DROP TABLE IF EXISTS listings;
DROP TABLE IF EXISTS companies;
DROP TABLE IF EXISTS sources;

DROP TYPE IF EXISTS run_status;
DROP TYPE IF EXISTS event_type;
DROP TYPE IF EXISTS match_method;
DROP TYPE IF EXISTS link_status;
DROP TYPE IF EXISTS text_policy;
DROP TYPE IF EXISTS source_kind;

DROP EXTENSION IF EXISTS pg_trgm;
