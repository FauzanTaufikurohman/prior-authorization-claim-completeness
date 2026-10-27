
BEGIN;

ALTER TABLE pa_requests
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ NULL;

CREATE INDEX IF NOT EXISTS idx_pa_requests_active_created_at
    ON pa_requests (created_at DESC)
    WHERE deleted_at IS NULL;

COMMIT;
