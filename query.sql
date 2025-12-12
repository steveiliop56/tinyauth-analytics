-- name: GetInstance :one
SELECT * FROM instances
WHERE uuid = ? LIMIT 1;

-- name: GetAllInstances :many
SELECT * FROM instances;

-- name: CreateInstance :exec
INSERT INTO instances (
  uuid, version, last_seen
) VALUES (
  ?, ?, ?
);

-- name: UpdateInstance :exec
UPDATE instances
set last_seen = ?
WHERE uuid = ?;

-- name: DeleteInstance :exec
DELETE FROM instances
WHERE uuid = ?;

-- name: DeleteOldInstances :many
DELETE FROM instances
WHERE last_seen < ?
RETURNING *;