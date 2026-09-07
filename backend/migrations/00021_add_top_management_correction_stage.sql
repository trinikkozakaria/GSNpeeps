-- +goose Up
ALTER TABLE attendance_corrections DROP CONSTRAINT attendance_corrections_status_check;
ALTER TABLE attendance_corrections ADD CONSTRAINT attendance_corrections_status_check
    CHECK (status IN ('menunggu_atasan','menunggu_hr','menunggu_top_management','disetujui','ditolak','dibatalkan'));

DROP INDEX IF EXISTS idx_attendance_corrections_pending;
CREATE INDEX idx_attendance_corrections_pending
    ON attendance_corrections (status) WHERE status IN ('menunggu_atasan','menunggu_hr','menunggu_top_management');

ALTER TABLE attendance_correction_approvals ALTER COLUMN tahap TYPE VARCHAR(20);
ALTER TABLE attendance_correction_approvals DROP CONSTRAINT attendance_correction_approvals_tahap_check;
ALTER TABLE attendance_correction_approvals ADD CONSTRAINT attendance_correction_approvals_tahap_check
    CHECK (tahap IN ('atasan','hr','top_management'));

-- +goose Down
ALTER TABLE attendance_correction_approvals DROP CONSTRAINT attendance_correction_approvals_tahap_check;
ALTER TABLE attendance_correction_approvals ADD CONSTRAINT attendance_correction_approvals_tahap_check
    CHECK (tahap IN ('atasan','hr'));
ALTER TABLE attendance_correction_approvals ALTER COLUMN tahap TYPE VARCHAR(10);

DROP INDEX IF EXISTS idx_attendance_corrections_pending;
CREATE INDEX idx_attendance_corrections_pending
    ON attendance_corrections (status) WHERE status IN ('menunggu_atasan','menunggu_hr');

ALTER TABLE attendance_corrections DROP CONSTRAINT attendance_corrections_status_check;
ALTER TABLE attendance_corrections ADD CONSTRAINT attendance_corrections_status_check
    CHECK (status IN ('menunggu_atasan','menunggu_hr','disetujui','ditolak','dibatalkan'));
