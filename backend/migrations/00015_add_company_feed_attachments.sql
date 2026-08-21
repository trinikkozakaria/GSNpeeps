-- +goose Up
CREATE TABLE company_feed_attachments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    feed_id UUID NOT NULL REFERENCES company_feeds(id) ON DELETE CASCADE,
    stored_path TEXT NOT NULL UNIQUE,
    file_name VARCHAR(255) NOT NULL,
    media_type VARCHAR(100) NOT NULL CHECK (media_type IN ('application/pdf', 'image/png', 'image/jpeg')),
    file_size BIGINT NOT NULL CHECK (file_size > 0 AND file_size <= 5242880),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_company_feed_attachments_feed ON company_feed_attachments (feed_id, created_at);

-- +goose Down
DROP TABLE IF EXISTS company_feed_attachments;
