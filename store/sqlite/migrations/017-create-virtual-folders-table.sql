-- Virtual folders store hierarchical organization and file placements in a
-- single table with two clearly distinguished record types.
--
-- Folder rows have entry_id IS NULL and a non-empty name. File association
-- rows have entry_id IS NOT NULL, name IS NULL, and parent_id IS NOT NULL.
-- Files without an association row belong to the implicit root folder.
-- Moving a file to root deletes its association row.
CREATE TABLE virtual_folders (
    id TEXT PRIMARY KEY,
    name TEXT,
    parent_id TEXT,
    entry_id TEXT UNIQUE,
    created_time TEXT NOT NULL CHECK (
        datetime(created_time) IS NOT NULL
        AND datetime(created_time) >= datetime('2022-02-20')
    ),
    CHECK (
        (
            entry_id IS NULL AND name IS NOT NULL
            AND length(name) BETWEEN 1 AND 255
        )
        OR (entry_id IS NOT NULL AND name IS NULL AND parent_id IS NOT NULL)
    ),
    FOREIGN KEY (parent_id) REFERENCES virtual_folders (id) ON DELETE RESTRICT,
    FOREIGN KEY (entry_id) REFERENCES entries (id) ON DELETE CASCADE
) STRICT;

-- Fast lookup of folder children and file placements.
CREATE INDEX idx_virtual_folders_parent_id ON virtual_folders (parent_id);

CREATE INDEX idx_virtual_folders_entry_id ON virtual_folders (entry_id);

-- Enforce unique sibling folder names. SQLite treats NULL values as distinct
-- in UNIQUE constraints, so root folders need a separate partial index.
CREATE UNIQUE INDEX idx_virtual_folders_root_name ON virtual_folders (name)
WHERE parent_id IS NULL AND entry_id IS NULL;

CREATE UNIQUE INDEX idx_virtual_folders_sibling_name
ON virtual_folders (parent_id, name)
WHERE parent_id IS NOT NULL AND entry_id IS NULL;
