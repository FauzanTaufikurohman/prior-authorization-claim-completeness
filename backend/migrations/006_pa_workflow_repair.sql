BEGIN;

ALTER TABLE pa_requests
    DROP CONSTRAINT IF EXISTS pa_requests_status_check;

ALTER TABLE pa_requests
    ADD CONSTRAINT pa_requests_status_check
    CHECK (status IN (
        'draft', 'ready_for_review', 'in_review', 'needs_information',
        'ready_for_submission', 'submitted', 'completed', 'cancelled',
        'processing', 'error'
    ));

CREATE TABLE IF NOT EXISTS pa_request_status_history (
    id BIGSERIAL PRIMARY KEY,
    pa_request_id BIGINT NOT NULL REFERENCES pa_requests(id) ON DELETE CASCADE,
    actor_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    from_status VARCHAR(30),
    to_status VARCHAR(30) NOT NULL,
    action VARCHAR(60) NOT NULL,
    note TEXT,
    details JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE pa_request_status_history
    ADD COLUMN IF NOT EXISTS id BIGSERIAL,
    ADD COLUMN IF NOT EXISTS pa_request_id BIGINT REFERENCES pa_requests(id) ON DELETE CASCADE,
    ADD COLUMN IF NOT EXISTS actor_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS from_status VARCHAR(30),
    ADD COLUMN IF NOT EXISTS to_status VARCHAR(30) NOT NULL DEFAULT 'legacy',
    ADD COLUMN IF NOT EXISTS action VARCHAR(60) NOT NULL DEFAULT 'legacy',
    ADD COLUMN IF NOT EXISTS note TEXT,
    ADD COLUMN IF NOT EXISTS details JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

CREATE INDEX IF NOT EXISTS idx_pa_request_status_history_request
    ON pa_request_status_history(pa_request_id, created_at DESC);

COMMIT;
