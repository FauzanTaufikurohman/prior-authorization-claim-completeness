BEGIN;

DO $$
DECLARE
    has_legacy_sender BOOLEAN;
BEGIN
    SELECT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'whatsapp_pa_intakes'
          AND column_name = 'sender'
    ) INTO has_legacy_sender;

    IF has_legacy_sender THEN
        ALTER TABLE whatsapp_pa_intakes
            DROP CONSTRAINT IF EXISTS whatsapp_pa_intakes_pkey;

        ALTER TABLE whatsapp_pa_intakes
            RENAME COLUMN sender TO sender_number;
        ALTER TABLE whatsapp_pa_intakes
            RENAME COLUMN fonnte_device TO device_number;

        ALTER TABLE whatsapp_pa_intakes
            ADD COLUMN id BIGSERIAL,
            ADD COLUMN staff_user_id BIGINT,
            ADD COLUMN collected_data JSONB NOT NULL DEFAULT '{}'::jsonb,
            ADD COLUMN encrypted_completion_message TEXT;

        UPDATE whatsapp_pa_intakes AS intake
        SET staff_user_id = pairing.staff_user_id
        FROM whatsapp_device_pairings AS pairing
        WHERE pairing.device_number = intake.device_number;

        IF EXISTS (
            SELECT 1
            FROM whatsapp_pa_intakes
            WHERE staff_user_id IS NULL
        ) THEN
            RAISE EXCEPTION 'Cannot repair WhatsApp intakes without a matching device pairing';
        END IF;

        ALTER TABLE whatsapp_pa_intakes
            ALTER COLUMN staff_user_id SET NOT NULL,
            ADD CONSTRAINT whatsapp_pa_intakes_staff_user_id_fkey
                FOREIGN KEY (staff_user_id) REFERENCES users(id) ON DELETE CASCADE,
            ADD CONSTRAINT whatsapp_pa_intakes_pkey PRIMARY KEY (id),
            ADD CONSTRAINT whatsapp_pa_intakes_device_sender_key
                UNIQUE (device_number, sender_number);

        ALTER TABLE whatsapp_pa_intakes
            DROP CONSTRAINT IF EXISTS whatsapp_pa_intakes_status_check;

        UPDATE whatsapp_pa_intakes
        SET status = CASE status
            WHEN 'awaiting_privacy_confirmation' THEN 'awaiting_consent'
            ELSE status
        END,
        collected_data = jsonb_strip_nulls(jsonb_build_object(
            'payer_name', payer_name,
            'service_type', service_type,
            'patient_reference', patient_reference,
            'diagnosis', diagnosis
        ));

        ALTER TABLE whatsapp_pa_intakes
            ADD CONSTRAINT whatsapp_pa_intakes_status_check
                CHECK (status IN (
                    'awaiting_consent',
                    'collecting',
                    'awaiting_confirmation',
                    'completed',
                    'cancelled'
                ));

        CREATE INDEX IF NOT EXISTS idx_whatsapp_pa_intakes_staff
            ON whatsapp_pa_intakes(staff_user_id, updated_at DESC);
    END IF;
END;
$$;

COMMIT;
