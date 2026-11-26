-- name: CountMappings :one
SELECT count(*) FROM mappings;

-- name: DeleteDatasetMappings :exec
DELETE FROM mappings WHERE origin = 'dataset';

-- name: InsertMappings :exec
INSERT INTO mappings (tvdb_id, tvdb_season, first_episode, last_episode, anilist_id, entry_first_episode, entry_last_episode, anidb_id, origin)
SELECT r.tvdb_id, r.tvdb_season, r.first_episode, r.last_episode, r.anilist_id, r.entry_first_episode, r.entry_last_episode, r.anidb_id, sqlc.arg(origin)::text
FROM jsonb_to_recordset(sqlc.arg(rows)::jsonb) AS r(
    tvdb_id bigint, tvdb_season int, first_episode int, last_episode int,
    anilist_id bigint, entry_first_episode int, entry_last_episode int, anidb_id bigint
);

-- name: ListMappingsForShows :many
SELECT tvdb_id, tvdb_season, first_episode, last_episode, anilist_id, entry_first_episode, entry_last_episode
FROM mappings
WHERE tvdb_id = ANY(sqlc.arg(tvdb_ids)::bigint[])
ORDER BY tvdb_id, tvdb_season, first_episode;
