package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	"reporter/internal/domain"
)

func (s *Store) StorageConfigsStrict(ctx context.Context) ([]domain.StorageConfig, error) {
	db, err := s.openConfiguredDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.QueryContext(ctx, `SELECT id, name, kind, COALESCE(endpoint, ''), COALESCE(bucket, ''), COALESCE(base_path, ''), COALESCE(base_uri, ''), COALESCE(credential_ref, ''), COALESCE(CAST(config_json AS CHAR), '{}'), created_at, updated_at FROM storage_configs ORDER BY created_at DESC, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.StorageConfig{}
	for rows.Next() {
		item, err := scanStorageConfig(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) StorageConfigStrict(ctx context.Context, id string) (domain.StorageConfig, bool, error) {
	db, err := s.openConfiguredDB()
	if err != nil {
		return domain.StorageConfig{}, false, err
	}
	defer db.Close()

	row := db.QueryRowContext(ctx, `SELECT id, name, kind, COALESCE(endpoint, ''), COALESCE(bucket, ''), COALESCE(base_path, ''), COALESCE(base_uri, ''), COALESCE(credential_ref, ''), COALESCE(CAST(config_json AS CHAR), '{}'), created_at, updated_at FROM storage_configs WHERE id = ?`, id)
	item, err := scanStorageConfig(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.StorageConfig{}, false, nil
		}
		return domain.StorageConfig{}, false, err
	}
	return item, true, nil
}

func (s *Store) CreateStorageConfigStrict(ctx context.Context, item domain.StorageConfig) (domain.StorageConfig, error) {
	db, err := s.openConfiguredDB()
	if err != nil {
		return domain.StorageConfig{}, err
	}
	defer db.Close()
	if strings.TrimSpace(item.ID) == "" {
		item.ID = uuid.NewString()
	}
	if strings.TrimSpace(item.Kind) == "" {
		item.Kind = "local"
	}
	configJSON, err := json.Marshal(nonNilMap(item.Config))
	if err != nil {
		return domain.StorageConfig{}, err
	}
	_, err = db.ExecContext(ctx, `INSERT INTO storage_configs (id, name, kind, endpoint, bucket, base_path, base_uri, credential_ref, config_json) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.ID, item.Name, item.Kind, nullableString(item.Endpoint), nullableString(item.Bucket), nullableString(item.BasePath), nullableString(item.BaseURI), nullableString(item.CredentialRef), string(configJSON))
	if err != nil {
		return domain.StorageConfig{}, err
	}
	saved, _, err := s.StorageConfigStrict(ctx, item.ID)
	return saved, err
}

func (s *Store) UpdateStorageConfigStrict(ctx context.Context, id string, patch domain.StorageConfig) (domain.StorageConfig, error) {
	db, err := s.openConfiguredDB()
	if err != nil {
		return domain.StorageConfig{}, err
	}
	defer db.Close()
	if strings.TrimSpace(patch.Kind) == "" {
		patch.Kind = "local"
	}
	configJSON, err := json.Marshal(nonNilMap(patch.Config))
	if err != nil {
		return domain.StorageConfig{}, err
	}
	result, err := db.ExecContext(ctx, `UPDATE storage_configs SET name = ?, kind = ?, endpoint = ?, bucket = ?, base_path = ?, base_uri = ?, credential_ref = ?, config_json = ? WHERE id = ?`,
		patch.Name, patch.Kind, nullableString(patch.Endpoint), nullableString(patch.Bucket), nullableString(patch.BasePath), nullableString(patch.BaseURI), nullableString(patch.CredentialRef), string(configJSON), id)
	if err != nil {
		return domain.StorageConfig{}, err
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return domain.StorageConfig{}, ErrNotFound
	}
	saved, _, err := s.StorageConfigStrict(ctx, id)
	return saved, err
}

func (s *Store) DeleteStorageConfigStrict(ctx context.Context, id string) (domain.StorageConfig, error) {
	before, ok, err := s.StorageConfigStrict(ctx, id)
	if err != nil {
		return domain.StorageConfig{}, err
	}
	if !ok {
		return domain.StorageConfig{}, ErrNotFound
	}
	db, err := s.openConfiguredDB()
	if err != nil {
		return domain.StorageConfig{}, err
	}
	defer db.Close()
	if _, err := db.ExecContext(ctx, `DELETE FROM storage_configs WHERE id = ?`, id); err != nil {
		return domain.StorageConfig{}, err
	}
	return before, nil
}

func (s *Store) RecordingConfigsStrict(ctx context.Context) ([]domain.RecordingConfig, error) {
	db, err := s.openConfiguredDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.QueryContext(ctx, `SELECT id, name, mode, storage_config_id, format, retention_days, auto_start, auto_stop, COALESCE(CAST(config_json AS CHAR), '{}'), created_at, updated_at FROM recording_configs ORDER BY created_at DESC, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.RecordingConfig{}
	for rows.Next() {
		item, err := scanRecordingConfig(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) RecordingConfigStrict(ctx context.Context, id string) (domain.RecordingConfig, bool, error) {
	db, err := s.openConfiguredDB()
	if err != nil {
		return domain.RecordingConfig{}, false, err
	}
	defer db.Close()
	row := db.QueryRowContext(ctx, `SELECT id, name, mode, storage_config_id, format, retention_days, auto_start, auto_stop, COALESCE(CAST(config_json AS CHAR), '{}'), created_at, updated_at FROM recording_configs WHERE id = ?`, id)
	item, err := scanRecordingConfig(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.RecordingConfig{}, false, nil
		}
		return domain.RecordingConfig{}, false, err
	}
	return item, true, nil
}

func (s *Store) DefaultRecordingConfigStrict(ctx context.Context) (domain.RecordingConfig, bool, error) {
	db, err := s.openConfiguredDB()
	if err != nil {
		return domain.RecordingConfig{}, false, err
	}
	defer db.Close()

	query := `SELECT id, name, mode, storage_config_id, format, retention_days, auto_start, auto_stop, COALESCE(CAST(config_json AS CHAR), '{}'), created_at, updated_at
FROM recording_configs
ORDER BY CASE WHEN id = 'REC-CFG-001' THEN 0 ELSE 1 END, created_at
LIMIT 1`
	var item domain.RecordingConfig
	var configRaw string
	err = db.QueryRowContext(ctx, query).Scan(&item.ID, &item.Name, &item.Mode, &item.StorageConfigID, &item.Format, &item.RetentionDays, &item.AutoStart, &item.AutoStop, &configRaw, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.RecordingConfig{}, false, nil
		}
		return domain.RecordingConfig{}, false, err
	}
	_ = json.Unmarshal([]byte(configRaw), &item.Config)
	return item, true, nil
}

func (s *Store) CreateRecordingConfigStrict(ctx context.Context, item domain.RecordingConfig) (domain.RecordingConfig, error) {
	db, err := s.openConfiguredDB()
	if err != nil {
		return domain.RecordingConfig{}, err
	}
	defer db.Close()
	if strings.TrimSpace(item.ID) == "" {
		item.ID = uuid.NewString()
	}
	if strings.TrimSpace(item.Mode) == "" {
		item.Mode = "server"
	}
	if strings.TrimSpace(item.Format) == "" {
		item.Format = "wav"
	}
	if item.RetentionDays <= 0 {
		item.RetentionDays = 365
	}
	configJSON, err := json.Marshal(nonNilMap(item.Config))
	if err != nil {
		return domain.RecordingConfig{}, err
	}
	_, err = db.ExecContext(ctx, `INSERT INTO recording_configs (id, name, mode, storage_config_id, format, retention_days, auto_start, auto_stop, config_json) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.ID, item.Name, item.Mode, item.StorageConfigID, item.Format, item.RetentionDays, item.AutoStart, item.AutoStop, string(configJSON))
	if err != nil {
		return domain.RecordingConfig{}, err
	}
	saved, _, err := s.RecordingConfigStrict(ctx, item.ID)
	return saved, err
}

func (s *Store) UpdateRecordingConfigStrict(ctx context.Context, id string, patch domain.RecordingConfig) (domain.RecordingConfig, error) {
	db, err := s.openConfiguredDB()
	if err != nil {
		return domain.RecordingConfig{}, err
	}
	defer db.Close()
	if strings.TrimSpace(patch.Mode) == "" {
		patch.Mode = "server"
	}
	if strings.TrimSpace(patch.Format) == "" {
		patch.Format = "wav"
	}
	if patch.RetentionDays <= 0 {
		patch.RetentionDays = 365
	}
	configJSON, err := json.Marshal(nonNilMap(patch.Config))
	if err != nil {
		return domain.RecordingConfig{}, err
	}
	result, err := db.ExecContext(ctx, `UPDATE recording_configs SET name = ?, mode = ?, storage_config_id = ?, format = ?, retention_days = ?, auto_start = ?, auto_stop = ?, config_json = ? WHERE id = ?`,
		patch.Name, patch.Mode, patch.StorageConfigID, patch.Format, patch.RetentionDays, patch.AutoStart, patch.AutoStop, string(configJSON), id)
	if err != nil {
		return domain.RecordingConfig{}, err
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return domain.RecordingConfig{}, ErrNotFound
	}
	saved, _, err := s.RecordingConfigStrict(ctx, id)
	return saved, err
}

func (s *Store) DeleteRecordingConfigStrict(ctx context.Context, id string) (domain.RecordingConfig, error) {
	before, ok, err := s.RecordingConfigStrict(ctx, id)
	if err != nil {
		return domain.RecordingConfig{}, err
	}
	if !ok {
		return domain.RecordingConfig{}, ErrNotFound
	}
	db, err := s.openConfiguredDB()
	if err != nil {
		return domain.RecordingConfig{}, err
	}
	defer db.Close()
	if _, err := db.ExecContext(ctx, `DELETE FROM recording_configs WHERE id = ?`, id); err != nil {
		return domain.RecordingConfig{}, err
	}
	return before, nil
}

type storageConfigScanner interface {
	Scan(dest ...interface{}) error
}

func scanStorageConfig(scanner storageConfigScanner) (domain.StorageConfig, error) {
	var item domain.StorageConfig
	var configRaw string
	err := scanner.Scan(&item.ID, &item.Name, &item.Kind, &item.Endpoint, &item.Bucket, &item.BasePath, &item.BaseURI, &item.CredentialRef, &configRaw, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return domain.StorageConfig{}, err
	}
	_ = json.Unmarshal([]byte(configRaw), &item.Config)
	return item, nil
}

type recordingConfigScanner interface {
	Scan(dest ...interface{}) error
}

func scanRecordingConfig(scanner recordingConfigScanner) (domain.RecordingConfig, error) {
	var item domain.RecordingConfig
	var configRaw string
	err := scanner.Scan(&item.ID, &item.Name, &item.Mode, &item.StorageConfigID, &item.Format, &item.RetentionDays, &item.AutoStart, &item.AutoStop, &configRaw, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return domain.RecordingConfig{}, err
	}
	_ = json.Unmarshal([]byte(configRaw), &item.Config)
	return item, nil
}

func nullableString(value string) interface{} {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}
