-- name: UpsertTracker :exec
INSERT INTO trackers (user_id, kind, username) VALUES ($1, $2, $3)
ON CONFLICT (user_id, kind) DO UPDATE SET
    username = EXCLUDED.username,
    token_encrypted = CASE WHEN trackers.username = EXCLUDED.username THEN trackers.token_encrypted END;

-- name: UpsertAuthorisedTracker :exec
INSERT INTO trackers (user_id, kind, username, token_encrypted) VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id, kind) DO UPDATE SET username = EXCLUDED.username, token_encrypted = EXCLUDED.token_encrypted;

-- name: GetTrackerUsername :one
SELECT username FROM trackers WHERE user_id = $1 AND kind = $2;

-- name: GetTrackerStatus :one
SELECT username, (token_encrypted IS NOT NULL)::boolean AS authorised FROM trackers WHERE user_id = $1 AND kind = $2;

-- name: DeleteTracker :exec
DELETE FROM trackers WHERE user_id = $1 AND kind = $2;
