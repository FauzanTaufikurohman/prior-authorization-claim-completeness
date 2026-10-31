BEGIN;

ALTER TABLE whatsapp_pa_intakes
    DROP CONSTRAINT IF EXISTS whatsapp_pa_intakes_device_sender_key;

ALTER TABLE whatsapp_pa_intakes
    DROP CONSTRAINT IF EXISTS whatsapp_pa_intakes_device_number_sender_number_key;

CREATE UNIQUE INDEX IF NOT EXISTS idx_whatsapp_pa_intakes_one_active_per_sender
    ON whatsapp_pa_intakes (device_number, sender_number)
    WHERE status IN ('awaiting_consent', 'collecting', 'awaiting_confirmation');

COMMIT;
