CREATE TABLE actions (
    id TEXT PRIMARY KEY,
    action_name TEXT NOT NULL,
    details_json TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'recorded',
    created_at TEXT NOT NULL
);

CREATE INDEX idx_actions_name_created
    ON actions(action_name, created_at DESC);
