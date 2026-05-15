package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"

	"github.com/google/uuid"

	"reporter/internal/domain"
)

func (s *Store) complaintDB(ctx context.Context) (*sql.DB, error) {
	s.mu.RLock()
	driver, dsn := s.dbDriver, s.dbDSN
	s.mu.RUnlock()
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
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func (s *Store) EnsureEvaluationComplaintTables(ctx context.Context) error {
	db, err := s.complaintDB(ctx)
	if err != nil {
		return err
	}
	defer db.Close()
	statements := []string{
		`CREATE TABLE IF NOT EXISTS evaluation_complaints (
  id CHAR(36) PRIMARY KEY,
  source VARCHAR(40) NOT NULL DEFAULT 'manual',
  kind VARCHAR(40) NOT NULL DEFAULT 'complaint',
  patient_id CHAR(36) NULL,
  patient_name VARCHAR(120) NULL,
  patient_phone VARCHAR(40) NULL,
  visit_id CHAR(36) NULL,
  channel VARCHAR(40) NULL,
  title VARCHAR(180) NOT NULL,
  content TEXT NOT NULL,
  rating INT NULL,
  category VARCHAR(120) NULL,
  authenticity VARCHAR(40) NOT NULL DEFAULT 'unconfirmed',
  status VARCHAR(40) NOT NULL DEFAULT 'new',
  responsible_department VARCHAR(120) NULL,
  responsible_person VARCHAR(120) NULL,
  audit_opinion TEXT NULL,
  handling_opinion TEXT NULL,
  rectification_measures TEXT NULL,
  tracking_opinion TEXT NULL,
  raw_payload JSON NULL,
  created_by CHAR(36) NULL,
  archived_at TIMESTAMP NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_eval_complaints_kind_status (kind, status),
  INDEX idx_eval_complaints_source (source),
  INDEX idx_eval_complaints_patient (patient_id),
  INDEX idx_eval_complaints_created_at (created_at)
)`,
		`CREATE TABLE IF NOT EXISTS evaluation_complaint_events (
  id CHAR(36) PRIMARY KEY,
  complaint_id CHAR(36) NOT NULL,
  actor_id CHAR(36) NULL,
  event_type VARCHAR(80) NOT NULL,
  comment TEXT NULL,
  payload_json JSON NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_eval_complaint_events_complaint (complaint_id),
  CONSTRAINT fk_eval_complaint_events_complaint FOREIGN KEY (complaint_id) REFERENCES evaluation_complaints(id)
)`,
	}
	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) EvaluationComplaints(ctx context.Context, status, kind string) ([]domain.EvaluationComplaint, error) {
	db, err := s.complaintDB(ctx)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	if err := s.EnsureEvaluationComplaintTables(ctx); err != nil {
		return nil, err
	}
	where := []string{"1=1"}
	args := []interface{}{}
	if status != "" {
		where = append(where, "status = ?")
		args = append(args, status)
	}
	if kind != "" {
		where = append(where, "kind = ?")
		args = append(args, kind)
	}
	rows, err := db.QueryContext(ctx, `
SELECT id, source, kind, COALESCE(patient_id, ''), COALESCE(patient_name, ''), COALESCE(patient_phone, ''),
       COALESCE(visit_id, ''), COALESCE(channel, ''), title, content, COALESCE(rating, 0),
       COALESCE(category, ''), authenticity, status, COALESCE(responsible_department, ''),
       COALESCE(responsible_person, ''), COALESCE(audit_opinion, ''), COALESCE(handling_opinion, ''),
       COALESCE(rectification_measures, ''), COALESCE(tracking_opinion, ''), COALESCE(CAST(raw_payload AS CHAR), '{}'),
       COALESCE(created_by, ''), archived_at, created_at, updated_at
FROM evaluation_complaints
WHERE `+strings.Join(where, " AND ")+`
ORDER BY created_at DESC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.EvaluationComplaint{}
	for rows.Next() {
		item, err := scanEvaluationComplaint(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) CreateEvaluationComplaint(ctx context.Context, item domain.EvaluationComplaint) (domain.EvaluationComplaint, error) {
	db, err := s.complaintDB(ctx)
	if err != nil {
		return domain.EvaluationComplaint{}, err
	}
	defer db.Close()
	if err := s.EnsureEvaluationComplaintTables(ctx); err != nil {
		return domain.EvaluationComplaint{}, err
	}
	if item.ID == "" {
		item.ID = uuid.NewString()
	}
	item.Source = firstNonEmptyStore(item.Source, "manual")
	item.Kind = firstNonEmptyStore(item.Kind, "complaint")
	item.Authenticity = firstNonEmptyStore(item.Authenticity, "unconfirmed")
	item.Status = firstNonEmptyStore(item.Status, "new")
	raw, err := json.Marshal(item.RawPayload)
	if err != nil {
		return domain.EvaluationComplaint{}, err
	}
	if string(raw) == "null" {
		raw = []byte("{}")
	}
	_, err = db.ExecContext(ctx, `
INSERT INTO evaluation_complaints (
  id, source, kind, patient_id, patient_name, patient_phone, visit_id, channel, title, content, rating, category,
  authenticity, status, responsible_department, responsible_person, audit_opinion, handling_opinion,
  rectification_measures, tracking_opinion, raw_payload, created_by
) VALUES (?, ?, ?, NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), ?, ?, NULLIF(?, 0), NULLIF(?, ''),
  ?, ?, NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), ?, NULLIF(?, ''))`,
		item.ID, item.Source, item.Kind, item.PatientID, item.PatientName, item.PatientPhone, item.VisitID, item.Channel,
		item.Title, item.Content, item.Rating, item.Category, item.Authenticity, item.Status, item.ResponsibleDepartment,
		item.ResponsiblePerson, item.AuditOpinion, item.HandlingOpinion, item.RectificationMeasures, item.TrackingOpinion,
		string(raw), item.CreatedBy)
	if err != nil {
		return domain.EvaluationComplaint{}, err
	}
	_ = insertComplaintEvent(ctx, db, item.ID, item.CreatedBy, "create", "创建评价投诉", item)
	return s.EvaluationComplaint(ctx, item.ID)
}

func (s *Store) EvaluationComplaint(ctx context.Context, id string) (domain.EvaluationComplaint, error) {
	db, err := s.complaintDB(ctx)
	if err != nil {
		return domain.EvaluationComplaint{}, err
	}
	defer db.Close()
	row := db.QueryRowContext(ctx, `
SELECT id, source, kind, COALESCE(patient_id, ''), COALESCE(patient_name, ''), COALESCE(patient_phone, ''),
       COALESCE(visit_id, ''), COALESCE(channel, ''), title, content, COALESCE(rating, 0),
       COALESCE(category, ''), authenticity, status, COALESCE(responsible_department, ''),
       COALESCE(responsible_person, ''), COALESCE(audit_opinion, ''), COALESCE(handling_opinion, ''),
       COALESCE(rectification_measures, ''), COALESCE(tracking_opinion, ''), COALESCE(CAST(raw_payload AS CHAR), '{}'),
       COALESCE(created_by, ''), archived_at, created_at, updated_at
FROM evaluation_complaints
WHERE id = ?`, id)
	item, err := scanEvaluationComplaint(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.EvaluationComplaint{}, ErrNotFound
	}
	return item, err
}

func (s *Store) UpdateEvaluationComplaint(ctx context.Context, id string, patch domain.EvaluationComplaint, actorID string) (domain.EvaluationComplaint, error) {
	db, err := s.complaintDB(ctx)
	if err != nil {
		return domain.EvaluationComplaint{}, err
	}
	defer db.Close()
	if _, err := s.EvaluationComplaint(ctx, id); err != nil {
		return domain.EvaluationComplaint{}, err
	}
	raw, err := json.Marshal(patch.RawPayload)
	if err != nil {
		return domain.EvaluationComplaint{}, err
	}
	if string(raw) == "null" {
		raw = []byte("{}")
	}
	archiveExpr := "archived_at"
	if patch.Status == "archived" {
		archiveExpr = "COALESCE(archived_at, CURRENT_TIMESTAMP)"
	}
	_, err = db.ExecContext(ctx, `
UPDATE evaluation_complaints SET
  source = COALESCE(NULLIF(?, ''), source),
  kind = COALESCE(NULLIF(?, ''), kind),
  patient_id = COALESCE(NULLIF(?, ''), patient_id),
  patient_name = COALESCE(NULLIF(?, ''), patient_name),
  patient_phone = COALESCE(NULLIF(?, ''), patient_phone),
  visit_id = COALESCE(NULLIF(?, ''), visit_id),
  channel = COALESCE(NULLIF(?, ''), channel),
  title = COALESCE(NULLIF(?, ''), title),
  content = COALESCE(NULLIF(?, ''), content),
  rating = COALESCE(NULLIF(?, 0), rating),
  category = COALESCE(NULLIF(?, ''), category),
  authenticity = COALESCE(NULLIF(?, ''), authenticity),
  status = COALESCE(NULLIF(?, ''), status),
  responsible_department = COALESCE(NULLIF(?, ''), responsible_department),
  responsible_person = COALESCE(NULLIF(?, ''), responsible_person),
  audit_opinion = COALESCE(NULLIF(?, ''), audit_opinion),
  handling_opinion = COALESCE(NULLIF(?, ''), handling_opinion),
  rectification_measures = COALESCE(NULLIF(?, ''), rectification_measures),
  tracking_opinion = COALESCE(NULLIF(?, ''), tracking_opinion),
  raw_payload = IF(? = '{}', raw_payload, ?),
  archived_at = `+archiveExpr+`
WHERE id = ?`,
		patch.Source, patch.Kind, patch.PatientID, patch.PatientName, patch.PatientPhone, patch.VisitID, patch.Channel,
		patch.Title, patch.Content, patch.Rating, patch.Category, patch.Authenticity, patch.Status, patch.ResponsibleDepartment,
		patch.ResponsiblePerson, patch.AuditOpinion, patch.HandlingOpinion, patch.RectificationMeasures, patch.TrackingOpinion,
		string(raw), string(raw), id)
	if err != nil {
		return domain.EvaluationComplaint{}, err
	}
	_ = insertComplaintEvent(ctx, db, id, actorID, firstNonEmptyStore(patch.Status, "update"), "更新评价投诉", patch)
	return s.EvaluationComplaint(ctx, id)
}

func (s *Store) DeleteEvaluationComplaint(ctx context.Context, id, actorID string) error {
	db, err := s.complaintDB(ctx)
	if err != nil {
		return err
	}
	defer db.Close()
	if _, err := s.EvaluationComplaint(ctx, id); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, `UPDATE evaluation_complaints SET status = 'deleted' WHERE id = ?`, id); err != nil {
		return err
	}
	return insertComplaintEvent(ctx, db, id, actorID, "delete", "删除评价投诉", map[string]string{"id": id})
}

func (s *Store) EvaluationComplaintStats(ctx context.Context) (map[string]interface{}, error) {
	db, err := s.complaintDB(ctx)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	if err := s.EnsureEvaluationComplaintTables(ctx); err != nil {
		return nil, err
	}
	result := map[string]interface{}{}
	for _, group := range []string{"kind", "status", "source", "category"} {
		rows, err := db.QueryContext(ctx, `SELECT COALESCE(`+group+`, '未分类'), COUNT(*) FROM evaluation_complaints WHERE status <> 'deleted' GROUP BY COALESCE(`+group+`, '未分类') ORDER BY COUNT(*) DESC`)
		if err != nil {
			return nil, err
		}
		items := []map[string]interface{}{}
		for rows.Next() {
			var name string
			var count int
			if err := rows.Scan(&name, &count); err != nil {
				rows.Close()
				return nil, err
			}
			items = append(items, map[string]interface{}{"name": name, "count": count})
		}
		rows.Close()
		result[group] = items
	}
	return result, nil
}

type complaintScanner interface {
	Scan(dest ...interface{}) error
}

func scanEvaluationComplaint(scanner complaintScanner) (domain.EvaluationComplaint, error) {
	var item domain.EvaluationComplaint
	var raw string
	var archived sql.NullTime
	err := scanner.Scan(
		&item.ID, &item.Source, &item.Kind, &item.PatientID, &item.PatientName, &item.PatientPhone, &item.VisitID,
		&item.Channel, &item.Title, &item.Content, &item.Rating, &item.Category, &item.Authenticity, &item.Status,
		&item.ResponsibleDepartment, &item.ResponsiblePerson, &item.AuditOpinion, &item.HandlingOpinion,
		&item.RectificationMeasures, &item.TrackingOpinion, &raw, &item.CreatedBy, &archived, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return item, err
	}
	if archived.Valid {
		item.ArchivedAt = &archived.Time
	}
	if raw != "" {
		_ = json.Unmarshal([]byte(raw), &item.RawPayload)
	}
	return item, nil
}

func insertComplaintEvent(ctx context.Context, db *sql.DB, complaintID, actorID, eventType, comment string, payload interface{}) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, `
INSERT INTO evaluation_complaint_events (id, complaint_id, actor_id, event_type, comment, payload_json)
VALUES (?, ?, NULLIF(?, ''), ?, ?, ?)`,
		uuid.NewString(), complaintID, actorID, eventType, comment, string(raw))
	return err
}
