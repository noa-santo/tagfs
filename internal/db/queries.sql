-- name: InsertFile :exec
INSERT INTO files (id, orig_name, mode, size, mtime_cached, meta_json)
VALUES (?, ?, ?, ?, ?, ?);

-- name: InsertDynamicDirectory :exec
INSERT INTO dynamic_directories (id, parent_id, name, created_at)
VALUES (?, ?, ?, ?);

-- name: UpdateFileStats :exec
UPDATE files
SET size = ?, mtime_cached = ?, mode = ?
WHERE id = ?;

-- name: GetAllFiles :many
SELECT * FROM files;