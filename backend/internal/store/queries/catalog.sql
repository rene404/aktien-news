-- name: UpsertSource :one
INSERT INTO sources (key, name, kind, market, url, weight, fetch_interval, text_policy, terms_url)
VALUES (
    @key, @name, @kind, @market, @url,
    COALESCE(sqlc.narg('weight')::numeric, 0.50),
    COALESCE(sqlc.narg('fetch_interval')::interval, '10 minutes'),
    COALESCE(sqlc.narg('text_policy')::text_policy, 'none'),
    sqlc.narg('terms_url')
)
ON CONFLICT (key) DO UPDATE
SET name = EXCLUDED.name, kind = EXCLUDED.kind, market = EXCLUDED.market, url = EXCLUDED.url,
    weight = EXCLUDED.weight, fetch_interval = EXCLUDED.fetch_interval,
    text_policy = EXCLUDED.text_policy, terms_url = EXCLUDED.terms_url
RETURNING *;

-- name: GetSourceByKey :one
SELECT * FROM sources WHERE key = @key;

-- name: UpsertCompany :one
INSERT INTO companies (slug, name, legal_name, country, lei, cik)
VALUES (@slug, @name, sqlc.narg('legal_name'), @country, sqlc.narg('lei'), sqlc.narg('cik'))
ON CONFLICT (slug) DO UPDATE
SET name = EXCLUDED.name, legal_name = EXCLUDED.legal_name, country = EXCLUDED.country,
    lei = EXCLUDED.lei, cik = EXCLUDED.cik
RETURNING id;

-- name: UpsertListing :one
INSERT INTO listings (company_id, isin, ticker, mic, currency)
VALUES (@company_id, sqlc.narg('isin'), @ticker, @mic, sqlc.narg('currency'))
ON CONFLICT (mic, ticker) DO UPDATE
SET company_id = EXCLUDED.company_id, isin = EXCLUDED.isin, currency = EXCLUDED.currency, active = true
RETURNING id;

-- name: UpsertIndexMembership :exec
INSERT INTO index_memberships (listing_id, index_key, valid_from)
SELECT l.id, @index_key, @valid_from
FROM listings l WHERE l.mic = @mic AND l.ticker = @ticker
ON CONFLICT (listing_id, index_key, valid_from) DO NOTHING;

-- name: UpsertAlias :exec
INSERT INTO company_aliases (company_id, alias, kind)
SELECT c.id, @alias, @kind FROM companies c WHERE c.slug = @company_slug
ON CONFLICT (company_id, alias) DO UPDATE SET kind = EXCLUDED.kind;

-- name: CompanyByISIN :one
SELECT DISTINCT c.* FROM companies c
JOIN listings l ON l.company_id = c.id
WHERE l.isin = @isin;

-- name: CompanyByTicker :one
SELECT c.* FROM companies c
JOIN listings l ON l.company_id = c.id
WHERE l.mic = @mic AND l.ticker = @ticker;
