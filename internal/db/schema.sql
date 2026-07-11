CREATE TABLE files (
    id TEXT PRIMARY KEY,           -- ULID
    orig_name TEXT NOT NULL,       -- e.g., "report.pdf"
    mode INTEGER NOT NULL         -- mode for inode creation
);

CREATE TABLE file_tags (
    file_id TEXT NOT NULL,
    tag_name TEXT NOT NULL,
    PRIMARY KEY (file_id, tag_name),
    FOREIGN KEY(file_id) REFERENCES files(id) ON DELETE CASCADE
);

CREATE TABLE dynamic_directories (
    id TEXT PRIMARY KEY,           -- ULID for the directory
    parent_id TEXT,                -- Can refer to another dynamic dir or a static config path
    name TEXT NOT NULL             -- e.g., "vacation_photos"
);

CREATE TABLE directory_tags (
    dir_id TEXT NOT NULL,
    tag_name TEXT NOT NULL,
    PRIMARY KEY (dir_id, tag_name),
    FOREIGN KEY(dir_id) REFERENCES dynamic_directories(id) ON DELETE CASCADE
);

CREATE INDEX idx_file_tags_tag_name ON file_tags(tag_name);
CREATE INDEX idx_dir_tags_tag_name ON directory_tags(tag_name);