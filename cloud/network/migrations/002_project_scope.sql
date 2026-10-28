ALTER TABLE networks
    ADD COLUMN IF NOT EXISTS project_id TEXT;

UPDATE networks n
SET project_id = COALESCE(
    (
        SELECT p.id::TEXT
        FROM projects p
        WHERE p.tenant_id::TEXT = n.tenant_id
        ORDER BY p.id
        LIMIT 1
    ),
    'legacy'
)
WHERE project_id IS NULL;

ALTER TABLE networks
    ALTER COLUMN project_id SET NOT NULL;

CREATE INDEX IF NOT EXISTS idx_networks_tenant_project
    ON networks (tenant_id, project_id, created_at);
