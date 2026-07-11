-- name: InsertFile :exec
INSERT INTO files (id, orig_name, mode)
VALUES (?, ?, ?);

-- name: InsertDynamicDirectory :exec
INSERT INTO dynamic_directories (id, parent_id, name)
VALUES (?, ?, ?);

-- name: UpdateFileStats :exec
UPDATE files
SET mode = ?
WHERE id = ?;

-- name: GetAllFiles :many
SELECT * FROM files;

-- name: GetFile :one
SELECT * FROM files WHERE id = ?;

-- name: ClearFileTags :exec
DELETE FROM file_tags WHERE file_id = ?;

-- name: InsertFileTag :exec
INSERT INTO file_tags (file_id, tag_name) VALUES (?, ?);