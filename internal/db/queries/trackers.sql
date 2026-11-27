-- name: UpsertTracker :exec
INSERT INTO trackers (user_id, kind, username) VALUES ($1, $2, $3)
ON CONFLICT (user_id, kind) DO UPDATE SET username = EXCLUDED.username;

-- name: GetTrackerUsername :one
SELECT username FROM trackers WHERE user_id = $1 AND kind = $2;

-- name: DeleteTracker :exec
DELETE FROM trackers WHERE user_id = $1 AND kind = $2;
