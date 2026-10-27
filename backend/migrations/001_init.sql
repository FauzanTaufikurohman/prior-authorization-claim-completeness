
BEGIN;

CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    full_name VARCHAR(150) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role VARCHAR(30) NOT NULL DEFAULT 'staff'
        CHECK (role IN ('admin', 'staff', 'reviewer')),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE checklist_versions (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    payer_name VARCHAR(200) NOT NULL,
    service_type VARCHAR(200) NOT NULL,
    version_number INTEGER NOT NULL CHECK (version_number > 0),
    source_url TEXT,
    effective_date DATE,
    reviewed_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    is_active BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (payer_name, service_type, version_number)
);

CREATE TABLE pa_requests (
    id BIGSERIAL PRIMARY KEY,
    request_number VARCHAR(50) NOT NULL UNIQUE,
    created_by BIGINT NOT NULL REFERENCES users(id),
    payer_name VARCHAR(200) NOT NULL,
    service_type VARCHAR(200) NOT NULL,
    provider_npi VARCHAR(10),
    patient_reference VARCHAR(100) NOT NULL,
    diagnosis TEXT,
    status VARCHAR(30) NOT NULL DEFAULT 'draft'
        CHECK (status IN (
            'draft', 'processing', 'in_review',
            'ready_for_review', 'completed', 'error'
        )),
    checklist_version_id BIGINT
        REFERENCES checklist_versions(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_pa_requests_created_by
    ON pa_requests(created_by);

CREATE INDEX idx_pa_requests_status
    ON pa_requests(status);

CREATE INDEX idx_pa_requests_created_at
    ON pa_requests(created_at DESC);

CREATE TABLE documents (
    id BIGSERIAL PRIMARY KEY,
    pa_request_id BIGINT NOT NULL
        REFERENCES pa_requests(id) ON DELETE CASCADE,
    original_filename VARCHAR(255) NOT NULL,
    storage_key TEXT NOT NULL,
    mime_type VARCHAR(100) NOT NULL,
    size_bytes BIGINT NOT NULL CHECK (size_bytes > 0),
    parsing_status VARCHAR(30) NOT NULL DEFAULT 'pending'
        CHECK (parsing_status IN (
            'pending', 'processing', 'completed', 'failed'
        )),
    uploaded_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_documents_pa_request_id
    ON documents(pa_request_id);

CREATE TABLE extracted_fields (
    id BIGSERIAL PRIMARY KEY,
    pa_request_id BIGINT NOT NULL
        REFERENCES pa_requests(id) ON DELETE CASCADE,
    document_id BIGINT REFERENCES documents(id) ON DELETE SET NULL,
    field_key VARCHAR(100) NOT NULL,
    field_value TEXT,
    confidence NUMERIC(5,4)
        CHECK (confidence IS NULL OR confidence BETWEEN 0 AND 1),
    evidence_text TEXT,
    verification_status VARCHAR(30) NOT NULL DEFAULT 'pending'
        CHECK (verification_status IN (
            'pending', 'verified', 'corrected', 'uncertain'
        )),
    reviewed_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    reviewed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_extracted_fields_request
    ON extracted_fields(pa_request_id);

CREATE TABLE checklist_rules (
    id BIGSERIAL PRIMARY KEY,
    checklist_version_id BIGINT NOT NULL
        REFERENCES checklist_versions(id) ON DELETE CASCADE,
    rule_key VARCHAR(100) NOT NULL,
    description TEXT NOT NULL,
    required_field_key VARCHAR(100),
    is_required BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (checklist_version_id, rule_key)
);

CREATE TABLE checklist_results (
    id BIGSERIAL PRIMARY KEY,
    pa_request_id BIGINT NOT NULL
        REFERENCES pa_requests(id) ON DELETE CASCADE,
    checklist_rule_id BIGINT NOT NULL
        REFERENCES checklist_rules(id) ON DELETE RESTRICT,
    status VARCHAR(30) NOT NULL
        CHECK (status IN ('fulfilled', 'missing', 'needs_review')),
    reason TEXT NOT NULL,
    evidence TEXT,
    reviewed_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    reviewed_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (pa_request_id, checklist_rule_id)
);

CREATE INDEX idx_checklist_results_request
    ON checklist_results(pa_request_id);

CREATE TABLE audit_logs (
    id BIGSERIAL PRIMARY KEY,
    actor_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    pa_request_id BIGINT REFERENCES pa_requests(id) ON DELETE SET NULL,
    action VARCHAR(100) NOT NULL,
    entity_type VARCHAR(100) NOT NULL,
    entity_id BIGINT,
    details JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_request
    ON audit_logs(pa_request_id);

CREATE INDEX idx_audit_logs_created_at
    ON audit_logs(created_at DESC);

COMMIT;
