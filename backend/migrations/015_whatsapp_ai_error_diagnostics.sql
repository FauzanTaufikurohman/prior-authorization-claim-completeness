BEGIN;

ALTER TABLE whatsapp_pa_intakes
    ADD COLUMN last_ai_error_code VARCHAR(32)
        CHECK (
            last_ai_error_code IS NULL OR last_ai_error_code IN (
                'missing_api_key',
                'timeout',
                'authentication',
                'model_not_found',
                'rate_limited',
                'request_rejected',
                'provider_error',
                'request_failed',
                'connection_error',
                'invalid_response'
            )
        );

COMMIT;
