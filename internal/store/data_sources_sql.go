package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"

	"reporter/internal/domain"
)

func (s *Store) dataSourcesFromSQL(ctx context.Context) ([]domain.DataSource, error) {
	db, err := s.openConfiguredDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.QueryContext(ctx, `
SELECT id, name, protocol, endpoint, COALESCE(CAST(config_json AS CHAR), '{}'), COALESCE(CAST(dictionaries_json AS CHAR), '[]'), COALESCE(CAST(field_mapping_json AS CHAR), '[]'), created_at, updated_at
FROM data_sources
ORDER BY updated_at DESC, created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.DataSource{}
	for rows.Next() {
		item, err := scanDataSource(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) dataSourceFromSQL(ctx context.Context, id string) (domain.DataSource, bool, error) {
	db, err := s.openConfiguredDB()
	if err != nil {
		return domain.DataSource{}, false, err
	}
	defer db.Close()
	row := db.QueryRowContext(ctx, `
SELECT id, name, protocol, endpoint, COALESCE(CAST(config_json AS CHAR), '{}'), COALESCE(CAST(dictionaries_json AS CHAR), '[]'), COALESCE(CAST(field_mapping_json AS CHAR), '[]'), created_at, updated_at
FROM data_sources
WHERE id = ?`, id)
	item, err := scanDataSource(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.DataSource{}, false, nil
		}
		return domain.DataSource{}, false, err
	}
	return item, true, nil
}

func (s *Store) createDataSourceInSQL(ctx context.Context, source domain.DataSource) (domain.DataSource, error) {
	db, err := s.openConfiguredDB()
	if err != nil {
		return domain.DataSource{}, err
	}
	defer db.Close()
	now := time.Now().UTC()
	source.ID = firstNonEmptyStore(source.ID, uuid.NewString())
	source.CreatedAt = now
	source.UpdatedAt = now
	configJSON, dictionariesJSON, mappingJSON, err := marshalDataSourcePayloads(source)
	if err != nil {
		return domain.DataSource{}, err
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO data_sources (id, name, protocol, endpoint, config_json, dictionaries_json, field_mapping_json, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		source.ID, source.Name, source.Protocol, source.Endpoint, configJSON, dictionariesJSON, mappingJSON, source.CreatedAt, source.UpdatedAt,
	); err != nil {
		return domain.DataSource{}, err
	}
	return source, nil
}

func (s *Store) updateDataSourceInSQL(ctx context.Context, id string, patch domain.DataSource) (domain.DataSource, error) {
	source, ok, err := s.dataSourceFromSQL(ctx, id)
	if err != nil {
		return domain.DataSource{}, err
	}
	if !ok {
		return domain.DataSource{}, ErrNotFound
	}
	if patch.Name != "" {
		source.Name = patch.Name
	}
	if patch.Protocol != "" {
		source.Protocol = patch.Protocol
	}
	if patch.Endpoint != "" {
		source.Endpoint = patch.Endpoint
	}
	if patch.Config != nil {
		source.Config = patch.Config
	}
	if patch.Dictionaries != nil {
		source.Dictionaries = patch.Dictionaries
	}
	if patch.FieldMapping != nil {
		source.FieldMapping = patch.FieldMapping
	}
	source.UpdatedAt = time.Now().UTC()
	configJSON, dictionariesJSON, mappingJSON, err := marshalDataSourcePayloads(source)
	if err != nil {
		return domain.DataSource{}, err
	}
	db, err := s.openConfiguredDB()
	if err != nil {
		return domain.DataSource{}, err
	}
	defer db.Close()
	if _, err := db.ExecContext(ctx, `
UPDATE data_sources
SET name = ?, protocol = ?, endpoint = ?, config_json = ?, dictionaries_json = ?, field_mapping_json = ?, updated_at = ?
WHERE id = ?`,
		source.Name, source.Protocol, source.Endpoint, configJSON, dictionariesJSON, mappingJSON, source.UpdatedAt, id,
	); err != nil {
		return domain.DataSource{}, err
	}
	return source, nil
}

func (s *Store) deleteDataSourceInSQL(ctx context.Context, id string) (domain.DataSource, error) {
	source, ok, err := s.dataSourceFromSQL(ctx, id)
	if err != nil {
		return domain.DataSource{}, err
	}
	if !ok {
		return domain.DataSource{}, ErrNotFound
	}
	db, err := s.openConfiguredDB()
	if err != nil {
		return domain.DataSource{}, err
	}
	defer db.Close()
	if _, err := db.ExecContext(ctx, `DELETE FROM data_sources WHERE id = ?`, id); err != nil {
		return domain.DataSource{}, err
	}
	return source, nil
}

func marshalDataSourcePayloads(source domain.DataSource) (string, string, string, error) {
	configJSON, err := json.Marshal(source.Config)
	if err != nil {
		return "", "", "", err
	}
	if string(configJSON) == "null" {
		configJSON = []byte("{}")
	}
	dictionariesJSON, err := json.Marshal(source.Dictionaries)
	if err != nil {
		return "", "", "", err
	}
	if string(dictionariesJSON) == "null" {
		dictionariesJSON = []byte("[]")
	}
	mappingJSON, err := json.Marshal(source.FieldMapping)
	if err != nil {
		return "", "", "", err
	}
	if string(mappingJSON) == "null" {
		mappingJSON = []byte("[]")
	}
	return string(configJSON), string(dictionariesJSON), string(mappingJSON), nil
}

type dataSourceScanner interface {
	Scan(dest ...interface{}) error
}

func scanDataSource(scanner dataSourceScanner) (domain.DataSource, error) {
	var item domain.DataSource
	var configRaw, dictionariesRaw, mappingRaw string
	if err := scanner.Scan(&item.ID, &item.Name, &item.Protocol, &item.Endpoint, &configRaw, &dictionariesRaw, &mappingRaw, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return domain.DataSource{}, err
	}
	if err := json.Unmarshal([]byte(configRaw), &item.Config); err != nil {
		return domain.DataSource{}, err
	}
	if err := json.Unmarshal([]byte(dictionariesRaw), &item.Dictionaries); err != nil {
		return domain.DataSource{}, err
	}
	if err := json.Unmarshal([]byte(mappingRaw), &item.FieldMapping); err != nil {
		return domain.DataSource{}, err
	}
	return item, nil
}
