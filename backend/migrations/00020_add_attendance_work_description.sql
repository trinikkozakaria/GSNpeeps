-- +goose Up
ALTER TABLE attendances ADD COLUMN uraian_pekerjaan VARCHAR(500);

-- +goose Down
ALTER TABLE attendances DROP COLUMN uraian_pekerjaan;
