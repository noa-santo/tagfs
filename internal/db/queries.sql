-- name: InsertNode :exec
INSERT INTO nodes (id, orig_name, mode)
VALUES (?, ?, ?);

-- name: UpdateNodeMode :exec
UPDATE nodes
SET mode = ?
WHERE id = ?;

-- name: GetAllNodes :many
SELECT * FROM nodes;

-- name: GetNode :one
SELECT * FROM nodes WHERE id = ?;

-- name: ClearTags :exec
DELETE FROM node_tags WHERE node_id = ?;

-- name: InsertTag :exec
INSERT INTO node_tags (node_id, tag_name) VALUES (?, ?);