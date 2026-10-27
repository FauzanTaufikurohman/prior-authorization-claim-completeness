BEGIN;

ALTER TABLE pa_requests
    DROP CONSTRAINT IF EXISTS pa_requests_status_check;

ALTER TABLE pa_requests
    ADD CONSTRAINT pa_requests_status_check
    CHECK (status IN (
        'draft', 'ready_for_review', 'in_review', 'needs_information',
        'ready_for_submission', 'submitted', 'payer_needs_information',
        'appeal_submitted', 'completed', 'cancelled', 'processing', 'error'
    ));

ALTER TABLE pa_requests
    ADD COLUMN IF NOT EXISTS appeal_reference VARCHAR(200),
    ADD COLUMN IF NOT EXISTS appeal_submitted_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS appeal_decision VARCHAR(30),
    ADD COLUMN IF NOT EXISTS appeal_decision_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS appeal_decision_note TEXT;

ALTER TABLE pa_requests
    DROP CONSTRAINT IF EXISTS pa_requests_appeal_decision_check;

ALTER TABLE pa_requests
    ADD CONSTRAINT pa_requests_appeal_decision_check
    CHECK (
        appeal_decision IS NULL OR
        appeal_decision IN ('approved', 'denied', 'partially_approved', 'withdrawn')
    );

COMMIT;
