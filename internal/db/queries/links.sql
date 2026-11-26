-- name: DeleteTMDBLinks :exec
DELETE FROM tmdb_tvdb_links;

-- name: InsertTMDBLinks :exec
INSERT INTO tmdb_tvdb_links (tmdb_id, tvdb_id)
SELECT r.tmdb_id, r.tvdb_id
FROM jsonb_to_recordset(sqlc.arg(rows)::jsonb) AS r(tmdb_id bigint, tvdb_id bigint);

-- name: ListTVDBIDsForTMDB :many
SELECT tmdb_id, tvdb_id FROM tmdb_tvdb_links WHERE tmdb_id = ANY(sqlc.arg(tmdb_ids)::bigint[]);
