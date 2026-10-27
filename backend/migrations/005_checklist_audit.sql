BEGIN;

CREATE TABLE IF NOT EXISTS checklist_version_audit_logs (
    id BIGSERIAL PRIMARY KEY,
    actor_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    checklist_version_id BIGINT NOT NULL
        REFERENCES checklist_versions(id) ON DELETE CASCADE,
    action VARCHAR(60) NOT NULL
        CHECK (action IN ('created', 'activated')),
    details JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_checklist_version_audit_logs_version
    ON checklist_version_audit_logs(checklist_version_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_checklist_version_audit_logs_actor
    ON checklist_version_audit_logs(actor_id, created_at DESC);

COMMIT;
