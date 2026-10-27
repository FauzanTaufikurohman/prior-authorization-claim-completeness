BEGIN;

ALTER TABLE checklist_version_audit_logs
    DROP CONSTRAINT IF EXISTS checklist_version_audit_logs_action_check;

ALTER TABLE checklist_version_audit_logs
    ADD CONSTRAINT checklist_version_audit_logs_action_check
    CHECK (action IN ('created', 'activated', 'deactivated'));

COMMIT;
