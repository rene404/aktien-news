-- name: DueSources :many
-- Enabled sources without a run inside their fetch interval. Failed runs
-- count too, so a broken source is not hammered (and FN's 10-minute rule
-- holds across worker restarts).
SELECT s.* FROM sources s
WHERE s.enabled
  AND NOT EXISTS (
      SELECT 1 FROM ingest_runs r
      WHERE r.source_id = s.id
        AND r.started_at + s.fetch_interval > sqlc.arg('now')::timestamptz
  )
ORDER BY s.id;

-- name: StartRun :one
INSERT INTO ingest_runs (source_id, started_at) VALUES (@source_id, @started_at) RETURNING id;

-- name: FinishRun :exec
UPDATE ingest_runs
SET finished_at = @finished_at, status = @status, fetched = @fetched, new = @new, error = sqlc.narg('error')
WHERE id = @id;
