-- name: UpsertSource :one
INSERT INTO sources (key, name, kind, market, url)
VALUES (@key, @name, @kind, @market, @url)
ON CONFLICT (key) DO UPDATE
SET name = EXCLUDED.name, kind = EXCLUDED.kind, market = EXCLUDED.market, url = EXCLUDED.url
RETURNING *;

-- name: GetSourceByKey :one
SELECT * FROM sources WHERE key = @key;
