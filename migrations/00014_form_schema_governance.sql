-- +goose Up
ALTER TABLE form_versions
  ADD COLUMN schema_hash VARCHAR(64) NULL AFTER schema_json,
  ADD COLUMN change_note TEXT NULL AFTER schema_hash,
  ADD COLUMN locked_at TIMESTAMP NULL AFTER published,
  ADD COLUMN published_at TIMESTAMP NULL AFTER locked_at,
  ADD INDEX idx_form_versions_hash (form_id, schema_hash);

-- +goose Down
ALTER TABLE form_versions
  DROP INDEX idx_form_versions_hash,
  DROP COLUMN published_at,
  DROP COLUMN locked_at,
  DROP COLUMN change_note,
  DROP COLUMN schema_hash;
