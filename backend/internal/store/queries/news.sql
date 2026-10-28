-- name: ListNews :many
-- Public feed: articles with at least one company whose CURRENT link state
-- (newest row in the append-only article_entities) is 'linked'.
-- Keyset pagination over (published_at, id), newest first.
WITH current_links AS (
    SELECT DISTINCT ON (e.article_id, e.company_id) e.article_id, e.company_id, e.status
    FROM article_entities e
    ORDER BY e.article_id, e.company_id, e.decided_at DESC, e.id DESC
), linked AS (
    SELECT article_id, company_id FROM current_links WHERE status = 'linked'
)
SELECT a.id, a.title, a.url, a.published_at,
       s.key AS source_key, s.name AS source_name,
       array_agg(DISTINCT l.ticker ORDER BY l.ticker)::text[] AS tickers
FROM articles a
JOIN sources s  ON s.id = a.source_id
JOIN linked k   ON k.article_id = a.id
JOIN listings l ON l.company_id = k.company_id AND l.active
WHERE (sqlc.narg('source')::text IS NULL OR s.key = sqlc.narg('source')::text)
  AND (sqlc.narg('from_ts')::timestamptz IS NULL OR a.published_at >= sqlc.narg('from_ts')::timestamptz)
  AND (sqlc.narg('to_ts')::timestamptz IS NULL OR a.published_at < sqlc.narg('to_ts')::timestamptz)
  AND (sqlc.narg('ticker')::text IS NULL OR EXISTS (
        SELECT 1 FROM linked k2
        JOIN listings l2 ON l2.company_id = k2.company_id AND l2.active
        WHERE k2.article_id = a.id AND upper(l2.ticker) = upper(sqlc.narg('ticker')::text)))
  AND (sqlc.narg('index_key')::text IS NULL OR EXISTS (
        SELECT 1 FROM linked k3
        JOIN listings l3 ON l3.company_id = k3.company_id AND l3.active
        JOIN index_memberships m ON m.listing_id = l3.id
        WHERE k3.article_id = a.id
          AND m.index_key = sqlc.narg('index_key')::text
          AND m.valid_from <= current_date
          AND (m.valid_to IS NULL OR m.valid_to > current_date)))
  AND (sqlc.narg('cursor_ts')::timestamptz IS NULL
       OR (a.published_at, a.id) < (sqlc.narg('cursor_ts')::timestamptz, sqlc.narg('cursor_id')::bigint))
GROUP BY a.id, s.key, s.name
ORDER BY a.published_at DESC, a.id DESC
LIMIT sqlc.arg('lim')::int;

-- name: SearchInstruments :many
SELECT l.ticker, l.mic, c.name, l.isin
FROM listings l JOIN companies c ON c.id = l.company_id
WHERE l.active
  AND (upper(l.ticker) LIKE upper(sqlc.arg('q')::text) || '%'
       OR c.name ILIKE '%' || sqlc.arg('q')::text || '%'
       OR l.isin = upper(sqlc.arg('q')::text))
ORDER BY (upper(l.ticker) = upper(sqlc.arg('q')::text)) DESC, c.name, l.mic
LIMIT 10;

-- name: InstrumentByTicker :one
SELECT l.id, l.ticker, l.mic, l.isin, c.name, c.country
FROM listings l JOIN companies c ON c.id = l.company_id
WHERE l.active AND upper(l.ticker) = upper(sqlc.arg('ticker')::text)
ORDER BY l.mic
LIMIT 1;

-- name: ListingIndices :many
SELECT index_key FROM index_memberships
WHERE listing_id = @listing_id
  AND valid_from <= current_date AND (valid_to IS NULL OR valid_to > current_date)
ORDER BY index_key;

-- name: ListEnabledSources :many
SELECT key, name, market FROM sources WHERE enabled ORDER BY name;
