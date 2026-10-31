BEGIN;

ALTER TABLE whatsapp_pa_intakes
    ADD COLUMN last_message_at TIMESTAMPTZ,
    ADD COLUMN last_ai_attempt_at TIMESTAMPTZ,
    ADD COLUMN last_ai_status VARCHAR(20)
        CHECK (last_ai_status IS NULL OR last_ai_status IN ('processed', 'failed'));

COMMIT;
