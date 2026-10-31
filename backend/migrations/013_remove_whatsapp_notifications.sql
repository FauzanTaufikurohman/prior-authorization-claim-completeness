BEGIN;

ALTER TABLE pa_requests
    DROP COLUMN IF EXISTS whatsapp_number,
    DROP COLUMN IF EXISTS whatsapp_notifications_consented_at;

COMMIT;
