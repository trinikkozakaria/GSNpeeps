-- +goose Up
ALTER TABLE employees ADD COLUMN foto_profil_url VARCHAR(255);

-- +goose Down
ALTER TABLE employees DROP COLUMN IF EXISTS foto_profil_url;
