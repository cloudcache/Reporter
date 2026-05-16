package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"reporter/internal/domain"
)

func (s *Store) formsFromSQL(ctx context.Context) ([]domain.Form, error) {
	db, err := s.openConfiguredDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.QueryContext(ctx, `SELECT id, name, COALESCE(description, ''), status, COALESCE(current_version_id, ''), created_at, updated_at FROM forms ORDER BY updated_at DESC, created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	forms := []domain.Form{}
	for rows.Next() {
		var item domain.Form
		if err := rows.Scan(&item.ID, &item.Name, &item.Description, &item.Status, &item.CurrentVersionID, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		versions, err := formVersionsFromSQL(ctx, db, item.ID)
		if err != nil {
			return nil, err
		}
		item.Versions = versions
		forms = append(forms, item)
	}
	return forms, rows.Err()
}

func (s *Store) createFormInSQL(ctx context.Context, form domain.Form) (domain.Form, error) {
	db, err := s.openConfiguredDB()
	if err != nil {
		return domain.Form{}, err
	}
	defer db.Close()

	now := time.Now().UTC()
	form.ID = firstNonEmptyStore(form.ID, uuid.NewString())
	form.Status = firstNonEmptyStore(form.Status, "draft")
	form.CreatedAt = now
	form.UpdatedAt = now
	_, err = db.ExecContext(ctx, `INSERT INTO forms (id, name, description, status, current_version_id, created_at, updated_at) VALUES (?, ?, ?, ?, NULL, ?, ?)`,
		form.ID, form.Name, form.Description, form.Status, form.CreatedAt, form.UpdatedAt)
	if err != nil {
		return domain.Form{}, err
	}
	return form, nil
}

func (s *Store) createFormVersionInSQL(ctx context.Context, formID, actor string, schema []domain.FormComponent) (domain.FormVersion, error) {
	db, err := s.openConfiguredDB()
	if err != nil {
		return domain.FormVersion{}, err
	}
	defer db.Close()

	var exists int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM forms WHERE id = ?`, formID).Scan(&exists); err != nil {
		return domain.FormVersion{}, err
	}
	if exists == 0 {
		return domain.FormVersion{}, ErrNotFound
	}

	schemaJSON, hash, err := normalizedSchema(schema)
	if err != nil {
		return domain.FormVersion{}, err
	}
	var lastID string
	var lastVersion int
	var lastSchemaHash sql.NullString
	err = db.QueryRowContext(ctx, `SELECT id, version, schema_hash FROM form_versions WHERE form_id = ? ORDER BY version DESC LIMIT 1`, formID).Scan(&lastID, &lastVersion, &lastSchemaHash)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return domain.FormVersion{}, err
	}
	if lastSchemaHash.Valid && lastSchemaHash.String == hash {
		versions, err := formVersionsFromSQL(ctx, db, formID)
		if err != nil {
			return domain.FormVersion{}, err
		}
		for _, version := range versions {
			if version.ID == lastID {
				return version, nil
			}
		}
	}

	version := domain.FormVersion{
		ID:        uuid.NewString(),
		FormID:    formID,
		Version:   lastVersion + 1,
		Schema:    schema,
		CreatedBy: actor,
		CreatedAt: time.Now().UTC(),
	}
	if version.Version == 0 {
		version.Version = 1
	}
	_, err = db.ExecContext(ctx, `INSERT INTO form_versions (id, form_id, version, schema_json, schema_hash, created_by, published, created_at) VALUES (?, ?, ?, ?, ?, NULLIF(?, ''), FALSE, ?)`,
		version.ID, version.FormID, version.Version, string(schemaJSON), hash, version.CreatedBy, version.CreatedAt)
	if err != nil {
		return domain.FormVersion{}, err
	}
	_, err = db.ExecContext(ctx, `UPDATE forms SET status = IF(status = 'published', status, 'draft'), updated_at = ? WHERE id = ?`, version.CreatedAt, formID)
	if err != nil {
		return domain.FormVersion{}, err
	}
	forms, err := s.formsFromSQL(ctx)
	if err != nil {
		return domain.FormVersion{}, err
	}
	for _, form := range forms {
		if form.ID == formID {
			_ = s.RegisterFormSchema(ctx, form, version)
			break
		}
	}
	return version, nil
}

func (s *Store) publishFormInSQL(ctx context.Context, formID string) (domain.Form, error) {
	db, err := s.openConfiguredDB()
	if err != nil {
		return domain.Form{}, err
	}
	defer db.Close()

	versions, err := formVersionsFromSQL(ctx, db, formID)
	if err != nil {
		return domain.Form{}, err
	}
	if len(versions) == 0 {
		return domain.Form{}, errors.New("form has no version")
	}
	target := versions[len(versions)-1]
	now := time.Now().UTC()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Form{}, err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `UPDATE form_versions SET published = FALSE WHERE form_id = ?`, formID); err != nil {
		return domain.Form{}, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE form_versions SET published = TRUE, locked_at = COALESCE(locked_at, ?), published_at = COALESCE(published_at, ?) WHERE id = ?`, now, now, target.ID); err != nil {
		return domain.Form{}, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE forms SET status = 'published', current_version_id = ?, updated_at = ? WHERE id = ?`, target.ID, now, formID); err != nil {
		return domain.Form{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.Form{}, err
	}
	forms, err := s.formsFromSQL(ctx)
	if err != nil {
		return domain.Form{}, err
	}
	for _, form := range forms {
		if form.ID == formID {
			for _, version := range form.Versions {
				_ = s.RegisterFormSchema(ctx, form, version)
			}
			return form, nil
		}
	}
	return domain.Form{}, ErrNotFound
}

func (s *Store) createSubmissionInSQL(ctx context.Context, submission domain.Submission) (domain.Submission, error) {
	db, err := s.openConfiguredDB()
	if err != nil {
		return domain.Submission{}, err
	}
	defer db.Close()

	var currentVersionID string
	if err := db.QueryRowContext(ctx, `SELECT COALESCE(current_version_id, '') FROM forms WHERE id = ?`, submission.FormID).Scan(&currentVersionID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Submission{}, ErrNotFound
		}
		return domain.Submission{}, err
	}
	if submission.FormVersionID == "" {
		submission.FormVersionID = currentVersionID
	}
	if submission.FormVersionID == "" {
		return domain.Submission{}, errors.New("form has no published version")
	}
	data, err := json.Marshal(submission.Data)
	if err != nil {
		return domain.Submission{}, err
	}
	now := time.Now().UTC()
	submission.ID = firstNonEmptyStore(submission.ID, uuid.NewString())
	submission.Status = firstNonEmptyStore(submission.Status, "submitted")
	submission.CreatedAt = now
	submission.UpdatedAt = now
	_, err = db.ExecContext(ctx, `INSERT INTO form_submissions (id, form_id, form_version_id, submitter_id, status, data_json, created_at, updated_at) VALUES (?, ?, ?, NULLIF(?, ''), ?, ?, ?, ?)`,
		submission.ID, submission.FormID, submission.FormVersionID, submission.SubmitterID, submission.Status, string(data), submission.CreatedAt, submission.UpdatedAt)
	if err != nil {
		return domain.Submission{}, err
	}
	return submission, nil
}

func (s *Store) submissionsByFormFromSQL(ctx context.Context, formID string) ([]domain.Submission, error) {
	db, err := s.openConfiguredDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.QueryContext(ctx, `
SELECT id, form_id, form_version_id, COALESCE(submitter_id, ''), status, CAST(data_json AS CHAR), created_at, updated_at
FROM form_submissions
WHERE form_id = ?
ORDER BY created_at DESC`, formID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.Submission{}
	for rows.Next() {
		item, err := scanSubmission(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) submissionFromSQL(ctx context.Context, id string) (domain.Submission, bool, error) {
	db, err := s.openConfiguredDB()
	if err != nil {
		return domain.Submission{}, false, err
	}
	defer db.Close()
	row := db.QueryRowContext(ctx, `
SELECT id, form_id, form_version_id, COALESCE(submitter_id, ''), status, CAST(data_json AS CHAR), created_at, updated_at
FROM form_submissions
WHERE id = ?`, id)
	item, err := scanSubmission(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Submission{}, false, nil
		}
		return domain.Submission{}, false, err
	}
	return item, true, nil
}

type submissionScanner interface {
	Scan(dest ...interface{}) error
}

func scanSubmission(scanner submissionScanner) (domain.Submission, error) {
	var item domain.Submission
	var raw string
	if err := scanner.Scan(&item.ID, &item.FormID, &item.FormVersionID, &item.SubmitterID, &item.Status, &raw, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return domain.Submission{}, err
	}
	if strings.TrimSpace(raw) == "" {
		raw = "{}"
	}
	if err := json.Unmarshal([]byte(raw), &item.Data); err != nil {
		return domain.Submission{}, err
	}
	return item, nil
}

func (s *Store) openConfiguredDB() (*sql.DB, error) {
	driver, dsn := s.dbDriver, s.dbDSN
	if strings.TrimSpace(dsn) == "" {
		return nil, errors.New("database is not configured")
	}
	if strings.TrimSpace(driver) == "" {
		driver = "mysql"
	}
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func formVersionsFromSQL(ctx context.Context, db *sql.DB, formID string) ([]domain.FormVersion, error) {
	rows, err := db.QueryContext(ctx, `SELECT id, form_id, version, CAST(schema_json AS CHAR), COALESCE(created_by, ''), created_at, published FROM form_versions WHERE form_id = ? ORDER BY version ASC`, formID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	versions := []domain.FormVersion{}
	for rows.Next() {
		var item domain.FormVersion
		var raw string
		if err := rows.Scan(&item.ID, &item.FormID, &item.Version, &raw, &item.CreatedBy, &item.CreatedAt, &item.Published); err != nil {
			return nil, err
		}
		if strings.TrimSpace(raw) != "" {
			_ = json.Unmarshal([]byte(raw), &item.Schema)
		}
		versions = append(versions, item)
	}
	return versions, rows.Err()
}

func normalizedSchema(schema []domain.FormComponent) ([]byte, string, error) {
	data, err := json.Marshal(schema)
	if err != nil {
		return nil, "", err
	}
	sum := sha256.Sum256(data)
	return data, hex.EncodeToString(sum[:]), nil
}
