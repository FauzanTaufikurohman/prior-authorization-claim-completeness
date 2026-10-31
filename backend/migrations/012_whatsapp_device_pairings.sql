BEGIN;

CREATE TABLE whatsapp_device_pairings (
    id BIGSERIAL PRIMARY KEY,
    staff_user_id BIGINT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    device_number VARCHAR(20) NOT NULL UNIQUE,
    device_name VARCHAR(30) NOT NULL,
    encrypted_device_token TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'connected', 'disconnected')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMIT;
