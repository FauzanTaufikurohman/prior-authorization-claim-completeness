BEGIN;

ALTER TABLE pa_requests
    ADD COLUMN IF NOT EXISTS ordering_provider_npi VARCHAR(10),
    ADD COLUMN IF NOT EXISTS servicing_provider_npi VARCHAR(10);

UPDATE pa_requests
SET ordering_provider_npi = provider_npi
WHERE ordering_provider_npi IS NULL AND provider_npi ~ '^[0-9]{10}$';

ALTER TABLE pa_requests
    ADD CONSTRAINT pa_requests_ordering_provider_npi_format
        CHECK (ordering_provider_npi IS NULL OR ordering_provider_npi ~ '^[0-9]{10}$'),
    ADD CONSTRAINT pa_requests_servicing_provider_npi_format
        CHECK (servicing_provider_npi IS NULL OR servicing_provider_npi ~ '^[0-9]{10}$');

COMMIT;
