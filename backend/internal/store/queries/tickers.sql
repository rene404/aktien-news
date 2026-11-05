-- Ticker counts for the stock list and "Heute im Gespräch". They count what
-- the public feed (ListNews) shows: clusters whose representative matches the
-- source kinds and time, per ticker CURRENTLY linked to any member. A cluster
-- counts once per ticker, however many members it has.

-- name: TopTickers :many
-- The most-mentioned tickers, restricted to current members of an index if
-- one is given (else all tickers).
WITH feed AS (
    SELECT c.id AS cluster_id
    FROM clusters c
    JOIN articles a ON a.id = c.representative_article_id AND a.cluster_id = c.id
    JOIN sources s  ON s.id = a.source_id
    WHERE (sqlc.narg('kinds')::text[] IS NULL OR s.kind::text = ANY(sqlc.narg('kinds')::text[]))
      AND a.published_at >= sqlc.arg('from_ts')::timestamptz
),
linked AS (
    SELECT DISTINCT m.cluster_id, l.id AS listing_id
    FROM feed f
    JOIN articles m      ON m.cluster_id = f.cluster_id
    JOIN article_links k ON k.article_id = m.id AND k.status = 'linked'
    JOIN listings l      ON l.company_id = k.company_id AND l.active
)
SELECT l.ticker, co.name, count(DISTINCT k.cluster_id)::int AS mentions
FROM linked k
JOIN listings l   ON l.id = k.listing_id
JOIN companies co ON co.id = l.company_id
WHERE sqlc.narg('index_key')::text IS NULL OR EXISTS (
        SELECT 1 FROM index_memberships im
        WHERE im.listing_id = l.id
          AND im.index_key = sqlc.narg('index_key')::text
          AND im.valid_from <= current_date
          AND (im.valid_to IS NULL OR im.valid_to > current_date))
GROUP BY l.ticker, co.name
ORDER BY mentions DESC, l.ticker
LIMIT sqlc.arg('lim')::int;

-- name: ListIndexMembers :many
-- Every current member of an index with its count, members without news
-- included (0). No time filter when from_ts is null.
WITH feed AS (
    SELECT c.id AS cluster_id
    FROM clusters c
    JOIN articles a ON a.id = c.representative_article_id AND a.cluster_id = c.id
    JOIN sources s  ON s.id = a.source_id
    WHERE (sqlc.narg('kinds')::text[] IS NULL OR s.kind::text = ANY(sqlc.narg('kinds')::text[]))
      AND (sqlc.narg('from_ts')::timestamptz IS NULL OR a.published_at >= sqlc.narg('from_ts')::timestamptz)
),
counts AS (
    SELECT l.id AS listing_id, count(DISTINCT m.cluster_id) AS n
    FROM feed f
    JOIN articles m      ON m.cluster_id = f.cluster_id
    JOIN article_links k ON k.article_id = m.id AND k.status = 'linked'
    JOIN listings l      ON l.company_id = k.company_id AND l.active
    GROUP BY l.id
)
SELECT l.ticker, co.name, coalesce(c.n, 0)::int AS mentions
FROM index_memberships im
JOIN listings l   ON l.id = im.listing_id AND l.active
JOIN companies co ON co.id = l.company_id
LEFT JOIN counts c ON c.listing_id = l.id
WHERE im.index_key = sqlc.arg('index_key')::text
  AND im.valid_from <= current_date
  AND (im.valid_to IS NULL OR im.valid_to > current_date)
ORDER BY co.name, l.ticker;
