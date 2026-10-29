-- name: ListNews :many
-- Public feed, one item per cluster: the representative's title, link,
-- source and time, with the tickers of every member whose CURRENT link state
-- (article_links, the projection of the append-only article_entities) is
-- 'linked'. A cluster is listed when at least one member is linked.
-- Ticker/index filters match any member; source, source kinds, time and the
-- keyset cursor over (published_at, id), newest first, refer to the representative.
WITH linked AS (
    SELECT m.cluster_id, l.ticker, l.id AS listing_id
    FROM article_links k
    JOIN articles m ON m.id = k.article_id
    JOIN listings l ON l.company_id = k.company_id AND l.active
    WHERE k.status = 'linked' AND m.cluster_id IS NOT NULL
)
SELECT a.id, a.title, a.url, a.published_at,
       s.key AS source_key, s.name AS source_name, s.kind AS source_kind,
       array_agg(DISTINCT k.ticker ORDER BY k.ticker)::text[] AS tickers
FROM clusters c
JOIN articles a ON a.id = c.representative_article_id AND a.cluster_id = c.id
JOIN sources s  ON s.id = a.source_id
JOIN linked k   ON k.cluster_id = c.id
WHERE (sqlc.narg('source')::text IS NULL OR s.key = sqlc.narg('source')::text)
  AND (sqlc.narg('kinds')::text[] IS NULL OR s.kind::text = ANY(sqlc.narg('kinds')::text[]))
  AND (sqlc.narg('from_ts')::timestamptz IS NULL OR a.published_at >= sqlc.narg('from_ts')::timestamptz)
  AND (sqlc.narg('to_ts')::timestamptz IS NULL OR a.published_at < sqlc.narg('to_ts')::timestamptz)
  AND (sqlc.narg('ticker')::text IS NULL OR EXISTS (
        SELECT 1 FROM linked k2
        WHERE k2.cluster_id = c.id AND upper(k2.ticker) = upper(sqlc.narg('ticker')::text)))
  AND (sqlc.narg('index_key')::text IS NULL OR EXISTS (
        SELECT 1 FROM linked k3
        JOIN index_memberships im ON im.listing_id = k3.listing_id
        WHERE k3.cluster_id = c.id
          AND im.index_key = sqlc.narg('index_key')::text
          AND im.valid_from <= current_date
          AND (im.valid_to IS NULL OR im.valid_to > current_date)))
  AND (sqlc.narg('cursor_ts')::timestamptz IS NULL
       OR (a.published_at, a.id) < (sqlc.narg('cursor_ts')::timestamptz, sqlc.narg('cursor_id')::bigint))
GROUP BY a.id, s.key, s.name, s.kind
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
SELECT key, name, market, kind FROM sources WHERE enabled ORDER BY name;
