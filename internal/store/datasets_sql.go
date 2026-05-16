package store

import (
	"context"
	"database/sql"
	"strings"

	"github.com/google/uuid"

	"reporter/internal/domain"
)

func (s *Store) DatasetsStrict(ctx context.Context, keyword string) ([]domain.Dataset, error) {
	db, err := s.openConfiguredDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()
	keyword = strings.TrimSpace(keyword)
	like := "%" + strings.ToLower(keyword) + "%"
	rows, err := db.QueryContext(ctx, `
SELECT id, name, COALESCE(description, ''), COALESCE(owner, ''), record_count, form_count, status, created_at, updated_at
FROM datasets
WHERE ? = '' OR LOWER(id) LIKE ? OR LOWER(name) LIKE ? OR LOWER(COALESCE(description, '')) LIKE ? OR LOWER(COALESCE(owner, '')) LIKE ?
ORDER BY updated_at DESC, created_at DESC`, keyword, like, like, like, like)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.Dataset{}
	for rows.Next() {
		item, err := scanDataset(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) DatasetStrict(ctx context.Context, id string) (domain.Dataset, bool, error) {
	db, err := s.openConfiguredDB()
	if err != nil {
		return domain.Dataset{}, false, err
	}
	defer db.Close()
	row := db.QueryRowContext(ctx, `SELECT id, name, COALESCE(description, ''), COALESCE(owner, ''), record_count, form_count, status, created_at, updated_at FROM datasets WHERE id = ?`, id)
	item, err := scanDataset(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.Dataset{}, false, nil
		}
		return domain.Dataset{}, false, err
	}
	return item, true, nil
}

func (s *Store) CreateDatasetStrict(ctx context.Context, item domain.Dataset) (domain.Dataset, error) {
	db, err := s.openConfiguredDB()
	if err != nil {
		return domain.Dataset{}, err
	}
	defer db.Close()
	if strings.TrimSpace(item.ID) == "" {
		item.ID = uuid.NewString()
	}
	if strings.TrimSpace(item.Status) == "" {
		item.Status = "active"
	}
	_, err = db.ExecContext(ctx, `INSERT INTO datasets (id, name, description, owner, record_count, form_count, status) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		item.ID, item.Name, nullableString(item.Description), nullableString(item.Owner), item.RecordCount, item.FormCount, item.Status)
	if err != nil {
		return domain.Dataset{}, err
	}
	saved, _, err := s.DatasetStrict(ctx, item.ID)
	return saved, err
}

func (s *Store) UpdateDatasetStrict(ctx context.Context, id string, patch domain.Dataset) (domain.Dataset, error) {
	db, err := s.openConfiguredDB()
	if err != nil {
		return domain.Dataset{}, err
	}
	defer db.Close()
	if strings.TrimSpace(patch.Status) == "" {
		patch.Status = "active"
	}
	result, err := db.ExecContext(ctx, `UPDATE datasets SET name = ?, description = ?, owner = ?, record_count = ?, form_count = ?, status = ? WHERE id = ?`,
		patch.Name, nullableString(patch.Description), nullableString(patch.Owner), patch.RecordCount, patch.FormCount, patch.Status, id)
	if err != nil {
		return domain.Dataset{}, err
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return domain.Dataset{}, ErrNotFound
	}
	saved, _, err := s.DatasetStrict(ctx, id)
	return saved, err
}

func (s *Store) DeleteDatasetStrict(ctx context.Context, id string) (domain.Dataset, error) {
	before, ok, err := s.DatasetStrict(ctx, id)
	if err != nil {
		return domain.Dataset{}, err
	}
	if !ok {
		return domain.Dataset{}, ErrNotFound
	}
	db, err := s.openConfiguredDB()
	if err != nil {
		return domain.Dataset{}, err
	}
	defer db.Close()
	if _, err := db.ExecContext(ctx, `DELETE FROM datasets WHERE id = ?`, id); err != nil {
		return domain.Dataset{}, err
	}
	return before, nil
}

type datasetScanner interface {
	Scan(dest ...interface{}) error
}

func scanDataset(scanner datasetScanner) (domain.Dataset, error) {
	var item domain.Dataset
	err := scanner.Scan(&item.ID, &item.Name, &item.Description, &item.Owner, &item.RecordCount, &item.FormCount, &item.Status, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}
