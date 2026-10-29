BEGIN;

ALTER TABLE pa_requests
    ADD COLUMN public_upload_token_hash VARCHAR(64) UNIQUE,
    ADD COLUMN whatsapp_number VARCHAR(20),
    ADD COLUMN whatsapp_notifications_consented_at TIMESTAMPTZ;

COMMIT;
