-- +goose Up
CREATE TABLE document_workflow_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    nama TEXT NOT NULL,
    aktif BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE document_workflow_stages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    template_id UUID NOT NULL REFERENCES document_workflow_templates(id) ON DELETE RESTRICT,
    urutan INT NOT NULL CHECK (urutan >= 0),
    nama_tahap TEXT NOT NULL,
    position_id UUID NOT NULL REFERENCES positions(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (template_id, urutan)
);

CREATE TABLE document_approval_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    template_id UUID NOT NULL REFERENCES document_workflow_templates(id) ON DELETE RESTRICT,
    document_url TEXT NOT NULL,
    document_url_normalized TEXT NOT NULL,
    judul TEXT NOT NULL,
    pemohon_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    stage_index INT NOT NULL DEFAULT 0 CHECK (stage_index >= 0),
    status TEXT NOT NULL CHECK (status IN ('berjalan', 'disetujui', 'ditolak', 'dibatalkan', 'bermasalah')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Core idempotency mechanism (PRD §5.2): only one running request per normalized URL at a
-- time. A partial unique index (not a plain UNIQUE) so a finished/rejected/cancelled
-- request never blocks a later resubmission of the same URL.
CREATE UNIQUE INDEX document_approval_requests_active_url_key
    ON document_approval_requests (document_url_normalized)
    WHERE status = 'berjalan';

CREATE INDEX document_approval_requests_pemohon_idx
    ON document_approval_requests (pemohon_user_id);

CREATE TABLE document_approval_decisions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    request_id UUID NOT NULL REFERENCES document_approval_requests(id) ON DELETE RESTRICT,
    urutan_tahap INT NOT NULL CHECK (urutan_tahap >= 0),
    approver_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    keputusan TEXT NOT NULL CHECK (keputusan IN ('approve', 'reject')),
    catatan TEXT,
    decided_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX document_approval_decisions_request_idx
    ON document_approval_decisions (request_id);

-- +goose Down
DROP TABLE IF EXISTS document_approval_decisions;
DROP TABLE IF EXISTS document_approval_requests;
DROP TABLE IF EXISTS document_workflow_stages;
DROP TABLE IF EXISTS document_workflow_templates;