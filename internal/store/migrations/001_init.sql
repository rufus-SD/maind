CREATE TABLE IF NOT EXISTS entries (
    id TEXT PRIMARY KEY,
    kind TEXT NOT NULL DEFAULT 'note'
        CHECK(kind IN ('note','decision','bug','solution','context','snippet','learning')),
    title TEXT,
    body TEXT NOT NULL,
    body_encrypted INTEGER NOT NULL DEFAULT 0,
    importance INTEGER NOT NULL DEFAULT 5
        CHECK(importance BETWEEN 0 AND 10),
    source TEXT NOT NULL DEFAULT 'cli'
        CHECK(source IN ('cli','import','git','ide','api')),
    project TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    deleted_at TEXT
);

CREATE TABLE IF NOT EXISTS tags (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE COLLATE NOCASE,
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS entry_tags (
    entry_id TEXT NOT NULL REFERENCES entries(id) ON DELETE CASCADE,
    tag_id TEXT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (entry_id, tag_id)
);

CREATE TABLE IF NOT EXISTS links (
    id TEXT PRIMARY KEY,
    from_entry_id TEXT NOT NULL REFERENCES entries(id) ON DELETE CASCADE,
    to_entry_id TEXT NOT NULL REFERENCES entries(id) ON DELETE CASCADE,
    relation TEXT NOT NULL DEFAULT 'relates_to'
        CHECK(relation IN ('relates_to','caused_by','supersedes','solved_by','depends_on','part_of','derived_from')),
    weight REAL NOT NULL DEFAULT 1.0
        CHECK(weight BETWEEN 0.0 AND 10.0),
    metadata TEXT DEFAULT '{}',
    created_at TEXT NOT NULL,
    CHECK(from_entry_id != to_entry_id)
);

CREATE INDEX IF NOT EXISTS idx_entries_kind ON entries(kind);
CREATE INDEX IF NOT EXISTS idx_entries_created_at ON entries(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_entries_importance ON entries(importance DESC);
CREATE INDEX IF NOT EXISTS idx_entries_project ON entries(project) WHERE project IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_entries_not_deleted ON entries(id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_links_from ON links(from_entry_id, relation);
CREATE INDEX IF NOT EXISTS idx_links_to ON links(to_entry_id, relation);
CREATE INDEX IF NOT EXISTS idx_tags_name ON tags(name);
CREATE INDEX IF NOT EXISTS idx_entry_tags_tag ON entry_tags(tag_id);

CREATE VIRTUAL TABLE IF NOT EXISTS entries_fts USING fts5(
    entry_id UNINDEXED,
    title,
    body
);
