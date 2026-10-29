-- name: UpsertSource :one
INSERT INTO sources (key, name, kind, market, url, parser, weight, fetch_interval, text_policy, terms_url, issuer_company_id)
VALUES (
    @key, @name, @kind, @market, @url, @parser,
    COALESCE(sqlc.narg('weight')::numeric, 0.50),
    COALESCE(sqlc.narg('fetch_interval')::interval, '10 minutes'),
    COALESCE(sqlc.narg('text_policy')::text_policy, 'none'),
    sqlc.narg('terms_url'),
    sqlc.narg('issuer_company_id')
)
ON CONFLICT (key) DO UPDATE
SET name = EXCLUDED.name, kind = EXCLUDED.kind, market = EXCLUDED.market, url = EXCLUDED.url, parser = EXCLUDED.parser,
    weight = EXCLUDED.weight, fetch_interval = EXCLUDED.fetch_interval,
    text_policy = EXCLUDED.text_policy, terms_url = EXCLUDED.terms_url,
    issuer_company_id = EXCLUDED.issuer_company_id
RETURNING *;

-- name: GetSourceByKey :one
SELECT * FROM sources WHERE key = @key;

-- name: UpsertCompany :one
INSERT INTO companies (slug, name, legal_name, country, lei, cik)
VALUES (@slug, @name, sqlc.narg('legal_name'), @country, sqlc.narg('lei'), sqlc.narg('cik'))
ON CONFLICT (slug) DO UPDATE
SET name = EXCLUDED.name, legal_name = EXCLUDED.legal_name, country = EXCLUDED.country,
    lei = EXCLUDED.lei,
    -- A CIK filled in by the SEC import survives a seed row without one.
    cik = COALESCE(EXCLUDED.cik, companies.cik)
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

-- name: CompanyByCIK :one
SELECT id, curated FROM companies WHERE cik = @cik;

-- name: CompanyByListing :one
SELECT c.id, c.curated, c.cik FROM listings l JOIN companies c ON c.id = l.company_id
WHERE l.mic = @mic AND l.ticker = @ticker;

-- name: FillCompanyCIK :exec
UPDATE companies SET cik = @cik WHERE id = @id AND cik IS NULL;

-- name: InsertUncuratedCompany :one
INSERT INTO companies (slug, name, legal_name, country, cik, curated)
VALUES (@slug, @name, @name, 'US', @cik, false)
RETURNING id;

-- name: InsertListingIfAbsent :exec
INSERT INTO listings (company_id, ticker, mic, currency)
VALUES (@company_id, @ticker, @mic, 'USD')
ON CONFLICT (mic, ticker) DO NOTHING;

-- name: CuratedNamesWithoutCIK :many
-- Full names (with legal form) of curated companies that still lack a CIK.
SELECT c.id, c.legal_name::text AS name FROM companies c
WHERE c.curated AND c.cik IS NULL AND c.legal_name IS NOT NULL
UNION ALL
SELECT c.id, a.alias FROM company_aliases a JOIN companies c ON c.id = a.company_id
WHERE c.curated AND c.cik IS NULL AND a.kind IN ('issuer_name', 'former_name');

-- name: DeleteUncuratedByCIK :exec
-- Makes room for a curated company with this CIK; listings and links cascade.
DELETE FROM companies WHERE cik = @cik AND NOT curated;
