CREATE TABLE models (
    name TEXT PRIMARY KEY,
    current_version TEXT NOT NULL,
    production_version TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE model_versions (
    model_name TEXT NOT NULL REFERENCES models(name) ON DELETE CASCADE,
    version TEXT NOT NULL,
    stage TEXT NOT NULL CHECK (stage IN ('staging', 'production')),
    metadata_json TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL,
    promotion_history_json TEXT NOT NULL DEFAULT '[]',
    PRIMARY KEY (model_name, version)
);

CREATE INDEX idx_model_versions_stage
    ON model_versions(stage);

CREATE TABLE metric_runs (
    run_id TEXT PRIMARY KEY,
    model_name TEXT NOT NULL,
    model_version TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    FOREIGN KEY (model_name, model_version)
        REFERENCES model_versions(model_name, version)
        ON DELETE CASCADE
);

CREATE INDEX idx_metric_runs_model_updated
    ON metric_runs(model_name, updated_at DESC);

CREATE TABLE metrics (
    run_id TEXT NOT NULL REFERENCES metric_runs(run_id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    value REAL NOT NULL,
    PRIMARY KEY (run_id, name)
);

CREATE TABLE events (
    id TEXT PRIMARY KEY,
    event_type TEXT NOT NULL,
    payload_json TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE INDEX idx_events_type_created
    ON events(event_type, created_at DESC);
