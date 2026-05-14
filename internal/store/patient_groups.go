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

func (s *MemoryStore) groupDB(ctx context.Context) (*sql.DB, error) {
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

func (s *MemoryStore) EnsurePatientGroupTables(ctx context.Context) error {
	db, err := s.groupDB(ctx)
	if err != nil {
		return err
	}
	defer db.Close()
	statements := []string{
		`CREATE TABLE IF NOT EXISTS patient_tags (
  id CHAR(36) PRIMARY KEY,
  name VARCHAR(120) NOT NULL UNIQUE,
  color VARCHAR(40) NOT NULL DEFAULT '#2563eb',
  description TEXT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
)`,
		`CREATE TABLE IF NOT EXISTS patient_tag_assignments (
  patient_id CHAR(36) NOT NULL,
  tag_id CHAR(36) NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (patient_id, tag_id),
  INDEX idx_patient_tag_assignments_tag (tag_id),
  CONSTRAINT fk_patient_tag_assignments_tag FOREIGN KEY (tag_id) REFERENCES patient_tags(id)
)`,
		`CREATE TABLE IF NOT EXISTS patient_groups (
  id CHAR(36) PRIMARY KEY,
  name VARCHAR(160) NOT NULL,
  category VARCHAR(80) NOT NULL DEFAULT '专病',
  mode VARCHAR(40) NOT NULL DEFAULT 'person',
  assignment_mode VARCHAR(40) NOT NULL DEFAULT 'manual',
  followup_plan_id VARCHAR(80) NULL,
  rules_json JSON NULL,
  permissions_json JSON NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_patient_groups_category (category),
  INDEX idx_patient_groups_plan (followup_plan_id)
)`,
		`CREATE TABLE IF NOT EXISTS patient_group_members (
  group_id CHAR(36) NOT NULL,
  patient_id CHAR(36) NOT NULL,
  visit_id CHAR(36) NULL,
  added_by CHAR(36) NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (group_id, patient_id),
  INDEX idx_patient_group_members_patient (patient_id),
  CONSTRAINT fk_patient_group_members_group FOREIGN KEY (group_id) REFERENCES patient_groups(id)
)`,
	}
	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}

func (s *MemoryStore) PatientTags(ctx context.Context) ([]domain.PatientTag, error) {
	db, err := s.groupDB(ctx)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	if err := s.EnsurePatientGroupTables(ctx); err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx, `SELECT id, name, color, COALESCE(description, ''), created_at, updated_at FROM patient_tags ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.PatientTag{}
	for rows.Next() {
		var item domain.PatientTag
		if err := rows.Scan(&item.ID, &item.Name, &item.Color, &item.Description, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *MemoryStore) UpsertPatientTag(ctx context.Context, item domain.PatientTag) (domain.PatientTag, error) {
	db, err := s.groupDB(ctx)
	if err != nil {
		return domain.PatientTag{}, err
	}
	defer db.Close()
	if err := s.EnsurePatientGroupTables(ctx); err != nil {
		return domain.PatientTag{}, err
	}
	if item.ID == "" {
		item.ID = uuid.NewString()
	}
	item.Color = firstNonEmptyStore(item.Color, "#2563eb")
	_, err = db.ExecContext(ctx, `
INSERT INTO patient_tags (id, name, color, description)
VALUES (?, ?, ?, ?)
ON DUPLICATE KEY UPDATE name = VALUES(name), color = VALUES(color), description = VALUES(description)`,
		item.ID, item.Name, item.Color, item.Description)
	if err != nil {
		return domain.PatientTag{}, err
	}
	return item, nil
}

func (s *MemoryStore) PatientGroups(ctx context.Context) ([]domain.PatientGroup, error) {
	db, err := s.groupDB(ctx)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	if err := s.EnsurePatientGroupTables(ctx); err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx, `
SELECT g.id, g.name, g.category, g.mode, g.assignment_mode, COALESCE(g.followup_plan_id, ''),
       COALESCE(CAST(g.rules_json AS CHAR), '{}'), COALESCE(CAST(g.permissions_json AS CHAR), '{}'),
       COUNT(m.patient_id), g.created_at, g.updated_at
FROM patient_groups g
LEFT JOIN patient_group_members m ON m.group_id = g.id
GROUP BY g.id
ORDER BY g.updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.PatientGroup{}
	for rows.Next() {
		var item domain.PatientGroup
		var rules, permissions string
		if err := rows.Scan(&item.ID, &item.Name, &item.Category, &item.Mode, &item.AssignmentMode, &item.FollowupPlanID, &rules, &permissions, &item.MemberCount, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(rules), &item.Rules)
		_ = json.Unmarshal([]byte(permissions), &item.Permissions)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *MemoryStore) UpsertPatientGroup(ctx context.Context, item domain.PatientGroup) (domain.PatientGroup, error) {
	db, err := s.groupDB(ctx)
	if err != nil {
		return domain.PatientGroup{}, err
	}
	defer db.Close()
	if err := s.EnsurePatientGroupTables(ctx); err != nil {
		return domain.PatientGroup{}, err
	}
	if item.ID == "" {
		item.ID = uuid.NewString()
	}
	item.Category = firstNonEmptyStore(item.Category, "专病")
	item.Mode = firstNonEmptyStore(item.Mode, "person")
	item.AssignmentMode = firstNonEmptyStore(item.AssignmentMode, "manual")
	rules, err := json.Marshal(item.Rules)
	if err != nil {
		return domain.PatientGroup{}, err
	}
	permissions, err := json.Marshal(item.Permissions)
	if err != nil {
		return domain.PatientGroup{}, err
	}
	if string(rules) == "null" {
		rules = []byte("{}")
	}
	if string(permissions) == "null" {
		permissions = []byte("{}")
	}
	_, err = db.ExecContext(ctx, `
INSERT INTO patient_groups (id, name, category, mode, assignment_mode, followup_plan_id, rules_json, permissions_json)
VALUES (?, ?, ?, ?, ?, NULLIF(?, ''), CAST(? AS JSON), CAST(? AS JSON))
ON DUPLICATE KEY UPDATE name = VALUES(name), category = VALUES(category), mode = VALUES(mode),
  assignment_mode = VALUES(assignment_mode), followup_plan_id = VALUES(followup_plan_id),
  rules_json = VALUES(rules_json), permissions_json = VALUES(permissions_json)`,
		item.ID, item.Name, item.Category, item.Mode, item.AssignmentMode, item.FollowupPlanID, string(rules), string(permissions))
	if err != nil {
		return domain.PatientGroup{}, err
	}
	return item, nil
}

func (s *MemoryStore) AssignPatientGroupMembers(ctx context.Context, groupID string, patientIDs []string, actorID string) error {
	db, err := s.groupDB(ctx)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := s.EnsurePatientGroupTables(ctx); err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM patient_group_members WHERE group_id = ?`, groupID); err != nil {
		return err
	}
	for _, patientID := range patientIDs {
		patientID = strings.TrimSpace(patientID)
		if patientID == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO patient_group_members (group_id, patient_id, added_by) VALUES (?, ?, NULLIF(?, ''))`, groupID, patientID, actorID); err != nil {
			return err
		}
	}
	return tx.Commit()
}
