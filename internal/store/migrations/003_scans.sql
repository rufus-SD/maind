CREATE TABLE IF NOT EXISTS scans (
    id TEXT PRIMARY KEY,
    project TEXT NOT NULL,
    source TEXT NOT NULL DEFAULT 'cli' CHECK(source IN ('cli','ide','api')),
    status TEXT NOT NULL DEFAULT 'running' CHECK(status IN ('running','completed','failed')),
    summary TEXT,
    summary_encrypted INTEGER NOT NULL DEFAULT 0,
    thoughts TEXT NOT NULL DEFAULT '',
    thoughts_encrypted INTEGER NOT NULL DEFAULT 0,
    entries_created INTEGER NOT NULL DEFAULT 0,
    started_at TEXT NOT NULL DEFAULT (datetime('now')),
    completed_at TEXT
);

CREATE INDEX IF NOT EXISTS idx_scans_project ON scans(project);
CREATE INDEX IF NOT EXISTS idx_scans_started ON scans(started_at DESC);

ALTER TABLE entries ADD COLUMN scan_id TEXT REFERENCES scans(id);
