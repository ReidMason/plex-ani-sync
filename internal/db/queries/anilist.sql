-- name: ListAniListMedia :many
SELECT anilist_id, episodes, title_romaji, title_english, fetched_at
FROM anilist_media
WHERE anilist_id = ANY(sqlc.arg(anilist_ids)::bigint[]);

-- name: UpsertAniListMedia :exec
INSERT INTO anilist_media (anilist_id, episodes, title_romaji, title_english, fetched_at)
SELECT r.anilist_id, r.episodes, r.title_romaji, r.title_english, sqlc.arg(fetched_at)::timestamptz
FROM jsonb_to_recordset(sqlc.arg(rows)::jsonb) AS r(anilist_id bigint, episodes int, title_romaji text, title_english text)
ON CONFLICT (anilist_id) DO UPDATE SET
    episodes = EXCLUDED.episodes,
    title_romaji = EXCLUDED.title_romaji,
    title_english = EXCLUDED.title_english,
    fetched_at = EXCLUDED.fetched_at;
