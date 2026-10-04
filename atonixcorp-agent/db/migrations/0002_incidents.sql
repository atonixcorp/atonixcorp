CREATE TABLE incidents (
    id TEXT PRIMARY KEY,
    service TEXT NOT NULL,
    root_cause TEXT NOT NULL,
    resolution TEXT NOT NULL,
    details_json TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL
);

CREATE INDEX idx_incidents_service_created
    ON incidents(service, created_at DESC);
