BEGIN;

ALTER TABLE pa_requests
    DROP CONSTRAINT IF EXISTS pa_requests_status_check;

ALTER TABLE pa_requests
    ADD CONSTRAINT pa_requests_status_check
    CHECK (status IN (
        'draft', 'ready_for_review', 'in_review', 'needs_information',
        'ready_for_submission', 'submitted', 'completed', 'cancelled',
        -- Retained for existing rows; new workflow actions do not write these.
        'processing', 'error'
    ));

ALTER TABLE pa_requests
    ADD COLUMN IF NOT EXISTS payer_reference VARCHAR(200),
    ADD COLUMN IF NOT EXISTS submitted_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS payer_decision VARCHAR(30),
    ADD COLUMN IF NOT EXISTS payer_decision_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS payer_decision_note TEXT;

ALTER TABLE pa_requests
    DROP CONSTRAINT IF EXISTS pa_requests_payer_decision_check;

ALTER TABLE pa_requests
    ADD CONSTRAINT pa_requests_payer_decision_check
    CHECK (
        payer_decision IS NULL OR
        payer_decision IN ('approved', 'denied', 'partially_approved', 'withdrawn')
    );

CREATE TABLE IF NOT EXISTS pa_request_status_history (
    id BIGSERIAL PRIMARY KEY,
    pa_request_id BIGINT NOT NULL
        REFERENCES pa_requests(id) ON DELETE CASCADE,
    actor_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    from_status VARCHAR(30),
    to_status VARCHAR(30) NOT NULL,
    action VARCHAR(60) NOT NULL,
    note TEXT,
    details JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pa_request_status_history_request
    ON pa_request_status_history(pa_request_id, created_at DESC);

INSERT INTO pa_request_status_history
    (pa_request_id, actor_id, from_status, to_status, action, note)
SELECT id, created_by, NULL, status, 'workflow_migration',
       'Status awal dicatat saat migrasi alur kerja.'
FROM pa_requests
WHERE NOT EXISTS (
    SELECT 1 FROM pa_request_status_history history
    WHERE history.pa_request_id = pa_requests.id
);

COMMIT;
