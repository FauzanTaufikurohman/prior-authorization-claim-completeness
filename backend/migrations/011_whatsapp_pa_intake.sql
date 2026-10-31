BEGIN;

CREATE TABLE whatsapp_pa_intakes (
    id BIGSERIAL PRIMARY KEY,
    device_number VARCHAR(20) NOT NULL,
    sender_number VARCHAR(20) NOT NULL,
    staff_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status VARCHAR(30) NOT NULL DEFAULT 'awaiting_consent'
        CHECK (status IN ('awaiting_consent', 'collecting', 'awaiting_confirmation', 'completed', 'cancelled')),
    collected_data JSONB NOT NULL DEFAULT '{}'::jsonb,
    pa_request_id BIGINT REFERENCES pa_requests(id) ON DELETE SET NULL,
    encrypted_completion_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (device_number, sender_number)
);

CREATE INDEX idx_whatsapp_pa_intakes_staff
    ON whatsapp_pa_intakes(staff_user_id, updated_at DESC);

COMMIT;
