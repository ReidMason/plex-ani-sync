-- name: UpsertUser :one
INSERT INTO users (oidc_issuer, oidc_subject)
VALUES ($1, $2)
ON CONFLICT (oidc_issuer, oidc_subject) DO UPDATE SET oidc_issuer = EXCLUDED.oidc_issuer
RETURNING id;

-- name: CreateSession :exec
INSERT INTO sessions (id_hash, user_id, display_name, expires_at)
VALUES ($1, $2, $3, $4);

-- name: GetSessionUser :one
SELECT user_id, display_name
FROM sessions
WHERE id_hash = $1 AND expires_at > now() AND revoked_at IS NULL;

-- name: RevokeSession :exec
UPDATE sessions SET revoked_at = now() WHERE id_hash = $1 AND revoked_at IS NULL;
