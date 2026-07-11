CREATE TABLE nodes (
    id TEXT PRIMARY KEY,           -- ULID
    orig_name TEXT NOT NULL,       -- e.g., "report.pdf"
    mode INTEGER NOT NULL         -- mode for inode creation
);

CREATE TABLE node_tags (
    node_id TEXT NOT NULL,
    tag_name TEXT NOT NULL,
    PRIMARY KEY (node_id, tag_name),
    FOREIGN KEY(node_id) REFERENCES nodes(id) ON DELETE CASCADE
);

CREATE INDEX idx_node_tags_tag_name ON node_tags(tag_name);