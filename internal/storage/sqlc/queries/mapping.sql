-- name: GetMappings :many
-- Retrieve all mappings for a season
SELECT * FROM mappings
WHERE season_id = ?;

-- name: AddMapping :exec
-- Add a mapping
INSERT INTO mappings (

    anime_id,
    anime_episode_start,
    anime_episode_end,
    season_id,
    season_episode_start,
    season_episode_end 
) VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(anime_id, season_id) DO UPDATE SET
    anime_episode_start = EXCLUDED.anime_episode_start,
    anime_episode_end = EXCLUDED.anime_episode_end,
    season_episode_start = EXCLUDED.season_episode_start,
    season_episode_end = EXCLUDED.season_episode_end,
    updated_at = datetime('now');

-- name: DeleteMapping :exec
-- Delete a mapping
DELETE FROM mappings
WHERE id = ?;

