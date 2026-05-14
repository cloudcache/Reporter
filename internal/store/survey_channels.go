package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"reporter/internal/domain"
)

func (s *MemoryStore) surveyDB(ctx context.Context) (*sql.DB, error) {
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

func (s *MemoryStore) EnsureSurveyChannelTables(ctx context.Context) error {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return err
	}
	defer db.Close()
	statements := []string{
		`CREATE TABLE IF NOT EXISTS integration_channels (
  id CHAR(36) PRIMARY KEY,
  kind VARCHAR(40) NOT NULL,
  name VARCHAR(160) NOT NULL,
  endpoint TEXT NULL,
  app_id VARCHAR(180) NULL,
  credential_ref VARCHAR(180) NULL,
  config_json JSON NULL,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_integration_channels_kind (kind)
)`,
		`CREATE TABLE IF NOT EXISTS survey_share_links (
  id CHAR(36) PRIMARY KEY,
  form_template_id VARCHAR(120) NOT NULL,
  title VARCHAR(180) NOT NULL,
  channel VARCHAR(40) NOT NULL DEFAULT 'web',
  token VARCHAR(80) NOT NULL UNIQUE,
  expires_at TIMESTAMP NULL,
  config_json JSON NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_survey_share_links_template (form_template_id),
  INDEX idx_survey_share_links_channel (channel)
)`,
		`CREATE TABLE IF NOT EXISTS survey_interviews (
  id CHAR(36) PRIMARY KEY,
  share_id CHAR(36) NOT NULL,
  patient_id CHAR(36) NULL,
  status VARCHAR(40) NOT NULL DEFAULT 'active',
  answers_json JSON NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_survey_interviews_share (share_id),
  CONSTRAINT fk_survey_interviews_share FOREIGN KEY (share_id) REFERENCES survey_share_links(id)
)`,
		`CREATE TABLE IF NOT EXISTS satisfaction_projects (
  id CHAR(36) PRIMARY KEY,
  name VARCHAR(180) NOT NULL,
  target_type VARCHAR(40) NOT NULL DEFAULT 'outpatient',
  form_template_id VARCHAR(120) NOT NULL,
  start_date DATE NULL,
  end_date DATE NULL,
  target_sample_size INT NOT NULL DEFAULT 0,
  actual_sample_size INT NOT NULL DEFAULT 0,
  anonymous BOOLEAN NOT NULL DEFAULT TRUE,
  requires_verification BOOLEAN NOT NULL DEFAULT FALSE,
  status VARCHAR(40) NOT NULL DEFAULT 'draft',
  config_json JSON NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_satisfaction_projects_status (status),
  INDEX idx_satisfaction_projects_target (target_type),
  INDEX idx_satisfaction_projects_template (form_template_id)
)`,
		`CREATE TABLE IF NOT EXISTS survey_submissions (
  id CHAR(36) PRIMARY KEY,
  project_id CHAR(36) NULL,
  share_id CHAR(36) NULL,
  form_template_id VARCHAR(120) NOT NULL,
  channel VARCHAR(40) NOT NULL DEFAULT 'web',
  patient_id CHAR(36) NULL,
  visit_id CHAR(36) NULL,
  anonymous BOOLEAN NOT NULL DEFAULT TRUE,
  status VARCHAR(40) NOT NULL DEFAULT 'submitted',
  quality_status VARCHAR(40) NOT NULL DEFAULT 'pending',
  quality_reason VARCHAR(255) NULL,
  started_at TIMESTAMP NULL,
  submitted_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  duration_seconds INT NOT NULL DEFAULT 0,
  ip_address VARCHAR(64) NULL,
  user_agent TEXT NULL,
  answers_json JSON NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_survey_submissions_project (project_id),
  INDEX idx_survey_submissions_share (share_id),
  INDEX idx_survey_submissions_template (form_template_id),
  INDEX idx_survey_submissions_quality (quality_status),
  INDEX idx_survey_submissions_submitted (submitted_at)
)`,
		`CREATE TABLE IF NOT EXISTS survey_submission_answers (
  id CHAR(36) PRIMARY KEY,
  submission_id CHAR(36) NOT NULL,
  question_id VARCHAR(120) NOT NULL,
  question_label VARCHAR(255) NOT NULL,
  question_type VARCHAR(60) NOT NULL,
  answer_json JSON NULL,
  score DECIMAL(10,2) NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_submission_answers_submission (submission_id),
  INDEX idx_submission_answers_question (question_id),
  CONSTRAINT fk_submission_answers_submission FOREIGN KEY (submission_id) REFERENCES survey_submissions(id)
)`,
		`CREATE TABLE IF NOT EXISTS satisfaction_indicators (
  id CHAR(36) PRIMARY KEY,
  project_id CHAR(36) NULL,
  target_type VARCHAR(40) NOT NULL DEFAULT 'outpatient',
  level_no INT NOT NULL DEFAULT 1,
  parent_id CHAR(36) NULL,
  name VARCHAR(180) NOT NULL,
  question_id VARCHAR(120) NULL,
  weight DECIMAL(10,2) NOT NULL DEFAULT 1,
  national_dimension VARCHAR(120) NULL,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_satisfaction_indicators_project (project_id),
  INDEX idx_satisfaction_indicators_question (question_id),
  INDEX idx_satisfaction_indicators_parent (parent_id)
)`,
		`CREATE TABLE IF NOT EXISTS satisfaction_issues (
  id CHAR(36) PRIMARY KEY,
  project_id CHAR(36) NULL,
  submission_id CHAR(36) NULL,
  indicator_id CHAR(36) NULL,
  title VARCHAR(240) NOT NULL,
  source VARCHAR(60) NOT NULL DEFAULT 'manual',
  responsible_department VARCHAR(120) NULL,
  responsible_person VARCHAR(120) NULL,
  severity VARCHAR(40) NOT NULL DEFAULT 'medium',
  suggestion TEXT NULL,
  status VARCHAR(40) NOT NULL DEFAULT 'open',
  due_date DATE NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_satisfaction_issues_project (project_id),
  INDEX idx_satisfaction_issues_status (status),
  INDEX idx_satisfaction_issues_submission (submission_id)
)`,
		`CREATE TABLE IF NOT EXISTS satisfaction_indicator_questions (
  id CHAR(36) PRIMARY KEY,
  project_id CHAR(36) NULL,
  indicator_id CHAR(36) NOT NULL,
  form_template_id VARCHAR(120) NOT NULL,
  question_id VARCHAR(120) NOT NULL,
  question_label VARCHAR(255) NULL,
  score_direction VARCHAR(40) NOT NULL DEFAULT 'positive',
  weight DECIMAL(10,2) NOT NULL DEFAULT 1,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uniq_indicator_question (project_id, form_template_id, question_id),
  INDEX idx_indicator_questions_indicator (indicator_id),
  INDEX idx_indicator_questions_project (project_id)
)`,
		`CREATE TABLE IF NOT EXISTS satisfaction_cleaning_rules (
  id CHAR(36) PRIMARY KEY,
  project_id CHAR(36) NULL,
  name VARCHAR(180) NOT NULL,
  rule_type VARCHAR(80) NOT NULL,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  config_json JSON NULL,
  action VARCHAR(40) NOT NULL DEFAULT 'mark_suspicious',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_cleaning_rules_project (project_id),
  INDEX idx_cleaning_rules_type (rule_type)
)`,
		`CREATE TABLE IF NOT EXISTS survey_submission_audit_logs (
  id CHAR(36) PRIMARY KEY,
  submission_id CHAR(36) NOT NULL,
  project_id CHAR(36) NULL,
  action VARCHAR(80) NOT NULL,
  from_status VARCHAR(40) NULL,
  to_status VARCHAR(40) NULL,
  reason VARCHAR(255) NULL,
  actor_id CHAR(36) NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_submission_audit_submission (submission_id),
  INDEX idx_submission_audit_project (project_id)
)`,
		`CREATE TABLE IF NOT EXISTS satisfaction_issue_events (
  id CHAR(36) PRIMARY KEY,
  issue_id CHAR(36) NOT NULL,
  action VARCHAR(80) NOT NULL,
  from_status VARCHAR(40) NULL,
  to_status VARCHAR(40) NULL,
  content TEXT NULL,
  attachments_json JSON NULL,
  actor_id CHAR(36) NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_issue_events_issue (issue_id),
  INDEX idx_issue_events_action (action)
)`,
	}
	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	if err := ensureColumn(ctx, db, "survey_share_links", "project_id", "CHAR(36) NULL AFTER id"); err != nil {
		return err
	}
	if err := ensureIndex(ctx, db, "survey_share_links", "idx_survey_share_links_project", "project_id"); err != nil {
		return err
	}
	for _, item := range []struct {
		table, column, definition string
	}{
		{"satisfaction_indicators", "service_stage", "VARCHAR(120) NULL AFTER name"},
		{"satisfaction_indicators", "service_node", "VARCHAR(120) NULL AFTER service_stage"},
		{"satisfaction_indicators", "include_total_score", "BOOLEAN NOT NULL DEFAULT TRUE AFTER weight"},
		{"satisfaction_indicators", "include_national", "BOOLEAN NOT NULL DEFAULT FALSE AFTER national_dimension"},
		{"satisfaction_issues", "measure", "TEXT NULL AFTER suggestion"},
		{"satisfaction_issues", "material_urls", "JSON NULL AFTER measure"},
		{"satisfaction_issues", "verification_result", "TEXT NULL AFTER material_urls"},
		{"satisfaction_issues", "closed_at", "TIMESTAMP NULL AFTER due_date"},
	} {
		if err := ensureColumn(ctx, db, item.table, item.column, item.definition); err != nil {
			return err
		}
	}
	return seedIntegrationChannels(ctx, db)
}

func ensureColumn(ctx context.Context, db *sql.DB, table, column, definition string) error {
	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = ? AND column_name = ?`, table, column).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	_, err := db.ExecContext(ctx, fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, definition))
	return err
}

func ensureIndex(ctx context.Context, db *sql.DB, table, index, column string) error {
	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = ? AND index_name = ?`, table, index).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	_, err := db.ExecContext(ctx, fmt.Sprintf("ALTER TABLE %s ADD INDEX %s (%s)", table, index, column))
	return err
}

func seedIntegrationChannels(ctx context.Context, db *sql.DB) error {
	defaults := []domain.IntegrationChannel{
		{ID: "CHAN-SMS", Kind: "sms", Name: "短信接口", Endpoint: "https://sms.example.local/send", CredentialRef: "secret://sms/default", Enabled: true, Config: map[string]interface{}{"signature": "医院", "templateMode": true}},
		{ID: "CHAN-WECHAT", Kind: "wechat", Name: "微信公众号接口", Endpoint: "https://api.weixin.qq.com", AppID: "wx-app-id", CredentialRef: "secret://wechat/default", Enabled: true, Config: map[string]interface{}{"messageType": "template"}},
		{ID: "CHAN-QQ", Kind: "qq", Name: "QQ 分享接口", Endpoint: "https://connect.qq.com", AppID: "qq-app-id", CredentialRef: "secret://qq/default", Enabled: false, Config: map[string]interface{}{}},
		{ID: "CHAN-WEB", Kind: "web", Name: "Web 链接", Endpoint: "http://127.0.0.1:4321/survey", Enabled: true, Config: map[string]interface{}{"allowAnonymous": true}},
	}
	for _, item := range defaults {
		raw, err := json.Marshal(item.Config)
		if err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, `
INSERT IGNORE INTO integration_channels (id, kind, name, endpoint, app_id, credential_ref, config_json, enabled)
VALUES (?, ?, ?, ?, ?, ?, CAST(? AS JSON), ?)`,
			item.ID, item.Kind, item.Name, item.Endpoint, item.AppID, item.CredentialRef, string(raw), item.Enabled); err != nil {
			return err
		}
	}
	return nil
}

func (s *MemoryStore) IntegrationChannels(ctx context.Context) ([]domain.IntegrationChannel, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	if err := s.EnsureSurveyChannelTables(ctx); err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx, `SELECT id, kind, name, COALESCE(endpoint, ''), COALESCE(app_id, ''), COALESCE(credential_ref, ''), COALESCE(CAST(config_json AS CHAR), '{}'), enabled, created_at, updated_at FROM integration_channels ORDER BY kind, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.IntegrationChannel{}
	for rows.Next() {
		var item domain.IntegrationChannel
		var raw string
		if err := rows.Scan(&item.ID, &item.Kind, &item.Name, &item.Endpoint, &item.AppID, &item.CredentialRef, &raw, &item.Enabled, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(raw), &item.Config)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *MemoryStore) UpsertIntegrationChannel(ctx context.Context, item domain.IntegrationChannel) (domain.IntegrationChannel, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return domain.IntegrationChannel{}, err
	}
	defer db.Close()
	if err := s.EnsureSurveyChannelTables(ctx); err != nil {
		return domain.IntegrationChannel{}, err
	}
	if item.ID == "" {
		item.ID = uuid.NewString()
	}
	raw, err := json.Marshal(item.Config)
	if err != nil {
		return domain.IntegrationChannel{}, err
	}
	if string(raw) == "null" {
		raw = []byte("{}")
	}
	_, err = db.ExecContext(ctx, `
INSERT INTO integration_channels (id, kind, name, endpoint, app_id, credential_ref, config_json, enabled)
VALUES (?, ?, ?, ?, ?, ?, CAST(? AS JSON), ?)
ON DUPLICATE KEY UPDATE kind=VALUES(kind), name=VALUES(name), endpoint=VALUES(endpoint), app_id=VALUES(app_id), credential_ref=VALUES(credential_ref), config_json=VALUES(config_json), enabled=VALUES(enabled)`,
		item.ID, item.Kind, item.Name, item.Endpoint, item.AppID, item.CredentialRef, string(raw), item.Enabled)
	return item, err
}

func (s *MemoryStore) SurveyShareLinks(ctx context.Context) ([]domain.SurveyShareLink, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	if err := s.EnsureSurveyChannelTables(ctx); err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx, `SELECT id, COALESCE(project_id, ''), form_template_id, title, channel, token, COALESCE(CAST(config_json AS CHAR), '{}'), created_at, updated_at FROM survey_share_links ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.SurveyShareLink{}
	for rows.Next() {
		var item domain.SurveyShareLink
		var raw string
		if err := rows.Scan(&item.ID, &item.ProjectID, &item.FormTemplateID, &item.Title, &item.Channel, &item.Token, &raw, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		item.URL = "/survey?token=" + item.Token
		_ = json.Unmarshal([]byte(raw), &item.Config)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *MemoryStore) CreateSurveyShareLink(ctx context.Context, item domain.SurveyShareLink) (domain.SurveyShareLink, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return domain.SurveyShareLink{}, err
	}
	defer db.Close()
	if err := s.EnsureSurveyChannelTables(ctx); err != nil {
		return domain.SurveyShareLink{}, err
	}
	item.ID = uuid.NewString()
	item.Channel = firstNonEmptyStore(item.Channel, "web")
	item.Token = randomToken()
	raw, err := json.Marshal(item.Config)
	if err != nil {
		return domain.SurveyShareLink{}, err
	}
	if string(raw) == "null" {
		raw = []byte("{}")
	}
	_, err = db.ExecContext(ctx, `INSERT INTO survey_share_links (id, project_id, form_template_id, title, channel, token, config_json) VALUES (?, NULLIF(?, ''), ?, ?, ?, ?, CAST(? AS JSON))`, item.ID, item.ProjectID, item.FormTemplateID, item.Title, item.Channel, item.Token, string(raw))
	if err != nil {
		return domain.SurveyShareLink{}, err
	}
	item.URL = "/survey?token=" + item.Token
	return item, nil
}

func (s *MemoryStore) SurveyShareByToken(ctx context.Context, token string) (domain.SurveyShareLink, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return domain.SurveyShareLink{}, err
	}
	defer db.Close()
	var item domain.SurveyShareLink
	var raw string
	err = db.QueryRowContext(ctx, `SELECT id, COALESCE(project_id, ''), form_template_id, title, channel, token, COALESCE(CAST(config_json AS CHAR), '{}'), created_at, updated_at FROM survey_share_links WHERE token = ?`, token).Scan(&item.ID, &item.ProjectID, &item.FormTemplateID, &item.Title, &item.Channel, &item.Token, &raw, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return item, ErrNotFound
	}
	if err != nil {
		return item, err
	}
	item.URL = "/survey?token=" + item.Token
	_ = json.Unmarshal([]byte(raw), &item.Config)
	return item, nil
}

func (s *MemoryStore) CreateSurveyInterview(ctx context.Context, shareID, patientID string) (domain.SurveyInterview, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return domain.SurveyInterview{}, err
	}
	defer db.Close()
	item := domain.SurveyInterview{ID: uuid.NewString(), ShareID: shareID, PatientID: patientID, Status: "active", Answers: map[string]interface{}{}}
	_, err = db.ExecContext(ctx, `INSERT INTO survey_interviews (id, share_id, patient_id, answers_json) VALUES (?, ?, NULLIF(?, ''), JSON_OBJECT())`, item.ID, item.ShareID, item.PatientID)
	return item, err
}

func (s *MemoryStore) SatisfactionProjects(ctx context.Context) ([]domain.SatisfactionProject, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	if err := s.EnsureSurveyChannelTables(ctx); err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx, `
SELECT p.id, p.name, p.target_type, p.form_template_id, COALESCE(DATE_FORMAT(p.start_date, '%Y-%m-%d'), ''), COALESCE(DATE_FORMAT(p.end_date, '%Y-%m-%d'), ''),
       p.target_sample_size, COUNT(s.id), p.anonymous, p.requires_verification, p.status, COALESCE(CAST(p.config_json AS CHAR), '{}'), p.created_at, p.updated_at
FROM satisfaction_projects p
LEFT JOIN survey_submissions s ON s.project_id = p.id
GROUP BY p.id
ORDER BY p.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.SatisfactionProject{}
	for rows.Next() {
		var item domain.SatisfactionProject
		var raw string
		if err := rows.Scan(&item.ID, &item.Name, &item.TargetType, &item.FormTemplateID, &item.StartDate, &item.EndDate, &item.TargetSampleSize, &item.ActualSampleSize, &item.Anonymous, &item.RequiresVerification, &item.Status, &raw, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(raw), &item.Config)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *MemoryStore) UpsertSatisfactionProject(ctx context.Context, item domain.SatisfactionProject) (domain.SatisfactionProject, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return domain.SatisfactionProject{}, err
	}
	defer db.Close()
	if err := s.EnsureSurveyChannelTables(ctx); err != nil {
		return domain.SatisfactionProject{}, err
	}
	if item.ID == "" {
		item.ID = uuid.NewString()
	}
	item.TargetType = firstNonEmptyStore(item.TargetType, "outpatient")
	item.Status = firstNonEmptyStore(item.Status, "draft")
	raw, err := json.Marshal(item.Config)
	if err != nil {
		return domain.SatisfactionProject{}, err
	}
	if string(raw) == "null" {
		raw = []byte("{}")
	}
	_, err = db.ExecContext(ctx, `
INSERT INTO satisfaction_projects (id, name, target_type, form_template_id, start_date, end_date, target_sample_size, anonymous, requires_verification, status, config_json)
VALUES (?, ?, ?, ?, NULLIF(?, ''), NULLIF(?, ''), ?, ?, ?, ?, CAST(? AS JSON))
ON DUPLICATE KEY UPDATE name=VALUES(name), target_type=VALUES(target_type), form_template_id=VALUES(form_template_id), start_date=VALUES(start_date), end_date=VALUES(end_date), target_sample_size=VALUES(target_sample_size), anonymous=VALUES(anonymous), requires_verification=VALUES(requires_verification), status=VALUES(status), config_json=VALUES(config_json)`,
		item.ID, item.Name, item.TargetType, item.FormTemplateID, item.StartDate, item.EndDate, item.TargetSampleSize, item.Anonymous, item.RequiresVerification, item.Status, string(raw))
	if err != nil {
		return item, err
	}
	return item, nil
}

func (s *MemoryStore) CreateSurveySubmission(ctx context.Context, item domain.SurveySubmission, components []map[string]interface{}) (domain.SurveySubmission, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return domain.SurveySubmission{}, err
	}
	defer db.Close()
	if err := s.EnsureSurveyChannelTables(ctx); err != nil {
		return domain.SurveySubmission{}, err
	}
	item.ID = uuid.NewString()
	item.Status = firstNonEmptyStore(item.Status, "submitted")
	item.QualityStatus = qualityStatus(item)
	if reasons := cleaningReasons(item); len(reasons) > 0 {
		item.QualityReason = strings.Join(reasons, "；")
	}
	if duplicate, reason := s.duplicateSubmission(ctx, db, item); duplicate {
		item.QualityStatus = "suspicious"
		item.QualityReason = firstNonEmptyStore(item.QualityReason, reason)
	}
	raw, err := json.Marshal(item.Answers)
	if err != nil {
		return item, err
	}
	if string(raw) == "null" {
		raw = []byte("{}")
	}
	startedAt := strings.TrimSpace(item.StartedAt)
	if startedAt != "" && len(startedAt) > 19 {
		startedAt = startedAt[:19]
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return item, err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `
INSERT INTO survey_submissions (id, project_id, share_id, form_template_id, channel, patient_id, visit_id, anonymous, status, quality_status, quality_reason, started_at, duration_seconds, ip_address, user_agent, answers_json)
VALUES (?, NULLIF(?, ''), NULLIF(?, ''), ?, ?, NULLIF(?, ''), NULLIF(?, ''), ?, ?, ?, ?, NULLIF(?, ''), ?, ?, ?, CAST(? AS JSON))`,
		item.ID, item.ProjectID, item.ShareID, item.FormTemplateID, item.Channel, item.PatientID, item.VisitID, item.Anonymous, item.Status, item.QualityStatus, item.QualityReason, startedAt, item.DurationSeconds, item.IPAddress, item.UserAgent, string(raw))
	if err != nil {
		return item, err
	}
	for _, component := range components {
		id, _ := component["id"].(string)
		if id == "" || component["type"] == "section" {
			continue
		}
		answer, ok := item.Answers[id]
		if !ok {
			continue
		}
		answerRaw, err := json.Marshal(answer)
		if err != nil {
			return item, err
		}
		label, _ := component["label"].(string)
		kind, _ := component["type"].(string)
		score := scoreAnswer(answer)
		var scoreArg interface{}
		if score != nil {
			scoreArg = *score
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO survey_submission_answers (id, submission_id, question_id, question_label, question_type, answer_json, score) VALUES (?, ?, ?, ?, ?, CAST(? AS JSON), ?)`, uuid.NewString(), item.ID, id, label, kind, string(answerRaw), scoreArg)
		if err != nil {
			return item, err
		}
	}
	if item.ProjectID != "" {
		_, _ = tx.ExecContext(ctx, `UPDATE satisfaction_projects SET actual_sample_size = (SELECT COUNT(*) FROM survey_submissions WHERE project_id = ?) WHERE id = ?`, item.ProjectID, item.ProjectID)
	}
	if err := tx.Commit(); err != nil {
		return item, err
	}
	item.SubmittedAt = time.Now()
	return item, nil
}

func (s *MemoryStore) duplicateSubmission(ctx context.Context, db *sql.DB, item domain.SurveySubmission) (bool, string) {
	phone := ""
	if item.Answers != nil {
		phone = strings.TrimSpace(fmt.Sprint(item.Answers["patient_phone"]))
	}
	var count int
	if item.PatientID != "" {
		_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM survey_submissions WHERE share_id = NULLIF(?, '') AND patient_id = NULLIF(?, '')`, item.ShareID, item.PatientID).Scan(&count)
		if count > 0 {
			return true, "同一患者重复提交"
		}
	}
	if item.VisitID != "" {
		_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM survey_submissions WHERE share_id = NULLIF(?, '') AND visit_id = NULLIF(?, '')`, item.ShareID, item.VisitID).Scan(&count)
		if count > 0 {
			return true, "同一就诊重复提交"
		}
	}
	if phone != "" {
		like := `%\"patient_phone\":\"` + phone + `\"%`
		_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM survey_submissions WHERE share_id = NULLIF(?, '') AND CAST(answers_json AS CHAR) LIKE ?`, item.ShareID, like).Scan(&count)
		if count > 0 {
			return true, "同一手机号重复提交"
		}
	}
	return false, ""
}

func (s *MemoryStore) SurveySubmissions(ctx context.Context, projectID string) ([]domain.SurveySubmission, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	if err := s.EnsureSurveyChannelTables(ctx); err != nil {
		return nil, err
	}
	query := `SELECT id, COALESCE(project_id, ''), COALESCE(share_id, ''), form_template_id, channel, COALESCE(patient_id, ''), COALESCE(visit_id, ''), anonymous, status, quality_status, COALESCE(quality_reason, ''), COALESCE(DATE_FORMAT(started_at, '%Y-%m-%dT%H:%i:%s'), ''), submitted_at, duration_seconds, COALESCE(ip_address, ''), COALESCE(user_agent, ''), COALESCE(CAST(answers_json AS CHAR), '{}'), created_at, updated_at FROM survey_submissions`
	args := []interface{}{}
	if strings.TrimSpace(projectID) != "" {
		query += ` WHERE project_id = ?`
		args = append(args, projectID)
	}
	query += ` ORDER BY submitted_at DESC LIMIT 500`
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.SurveySubmission{}
	for rows.Next() {
		var item domain.SurveySubmission
		var raw string
		if err := rows.Scan(&item.ID, &item.ProjectID, &item.ShareID, &item.FormTemplateID, &item.Channel, &item.PatientID, &item.VisitID, &item.Anonymous, &item.Status, &item.QualityStatus, &item.QualityReason, &item.StartedAt, &item.SubmittedAt, &item.DurationSeconds, &item.IPAddress, &item.UserAgent, &raw, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(raw), &item.Answers)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *MemoryStore) SatisfactionIndicators(ctx context.Context, projectID string) ([]domain.SatisfactionIndicator, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	if err := s.EnsureSurveyChannelTables(ctx); err != nil {
		return nil, err
	}
	query := `SELECT id, COALESCE(project_id, ''), target_type, level_no, COALESCE(parent_id, ''), name, COALESCE(service_stage, ''), COALESCE(service_node, ''), COALESCE(question_id, ''), weight, include_total_score, COALESCE(national_dimension, ''), include_national, enabled, created_at, updated_at FROM satisfaction_indicators`
	args := []interface{}{}
	if strings.TrimSpace(projectID) != "" {
		query += ` WHERE project_id IS NULL OR project_id = ?`
		args = append(args, projectID)
	}
	query += ` ORDER BY level_no, name`
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.SatisfactionIndicator{}
	for rows.Next() {
		var item domain.SatisfactionIndicator
		if err := rows.Scan(&item.ID, &item.ProjectID, &item.TargetType, &item.Level, &item.ParentID, &item.Name, &item.ServiceStage, &item.ServiceNode, &item.QuestionID, &item.Weight, &item.IncludeTotalScore, &item.NationalDimension, &item.IncludeNational, &item.Enabled, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if len(items) == 0 {
		defaults := defaultSatisfactionIndicators(projectID)
		for _, item := range defaults {
			if _, err := s.UpsertSatisfactionIndicator(ctx, item); err != nil {
				return nil, err
			}
		}
		return defaults, nil
	}
	return items, rows.Err()
}

func (s *MemoryStore) UpsertSatisfactionIndicator(ctx context.Context, item domain.SatisfactionIndicator) (domain.SatisfactionIndicator, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return domain.SatisfactionIndicator{}, err
	}
	defer db.Close()
	if item.ID == "" {
		item.ID = uuid.NewString()
	}
	if item.Level == 0 {
		item.Level = 1
	}
	if item.Weight == 0 {
		item.Weight = 1
	}
	item.TargetType = firstNonEmptyStore(item.TargetType, "outpatient")
	_, err = db.ExecContext(ctx, `
INSERT INTO satisfaction_indicators (id, project_id, target_type, level_no, parent_id, name, service_stage, service_node, question_id, weight, include_total_score, national_dimension, include_national, enabled)
VALUES (?, NULLIF(?, ''), ?, ?, NULLIF(?, ''), ?, NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), ?, ?, NULLIF(?, ''), ?, ?)
ON DUPLICATE KEY UPDATE project_id=VALUES(project_id), target_type=VALUES(target_type), level_no=VALUES(level_no), parent_id=VALUES(parent_id), name=VALUES(name), service_stage=VALUES(service_stage), service_node=VALUES(service_node), question_id=VALUES(question_id), weight=VALUES(weight), include_total_score=VALUES(include_total_score), national_dimension=VALUES(national_dimension), include_national=VALUES(include_national), enabled=VALUES(enabled)`,
		item.ID, item.ProjectID, item.TargetType, item.Level, item.ParentID, item.Name, item.ServiceStage, item.ServiceNode, item.QuestionID, item.Weight, item.IncludeTotalScore, item.NationalDimension, item.IncludeNational, item.Enabled)
	return item, err
}

func defaultSatisfactionIndicators(projectID string) []domain.SatisfactionIndicator {
	return []domain.SatisfactionIndicator{
		{ID: uuid.NewString(), ProjectID: projectID, TargetType: "outpatient", Level: 1, Name: "综合体验", ServiceStage: "全流程", ServiceNode: "总体评价", QuestionID: "overall_satisfaction", Weight: 1, IncludeTotalScore: true, NationalDimension: "综合体验", IncludeNational: true, Enabled: true},
		{ID: uuid.NewString(), ProjectID: projectID, TargetType: "outpatient", Level: 2, Name: "推荐意愿", ServiceStage: "全流程", ServiceNode: "推荐", QuestionID: "recommend_score", Weight: 1, IncludeTotalScore: true, NationalDimension: "综合体验", IncludeNational: true, Enabled: true},
		{ID: uuid.NewString(), ProjectID: projectID, TargetType: "outpatient", Level: 2, Name: "候诊体验", ServiceStage: "候诊", ServiceNode: "候诊时间", QuestionID: "service_matrix", Weight: 1, IncludeTotalScore: true, NationalDimension: "诊疗流程", IncludeNational: true, Enabled: true},
	}
}

func (s *MemoryStore) SatisfactionIndicatorQuestions(ctx context.Context, projectID string) ([]domain.SatisfactionIndicatorQuestion, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	query := `SELECT id, COALESCE(project_id, ''), indicator_id, form_template_id, question_id, COALESCE(question_label, ''), score_direction, weight, created_at, updated_at FROM satisfaction_indicator_questions`
	args := []interface{}{}
	if strings.TrimSpace(projectID) != "" {
		query += ` WHERE project_id IS NULL OR project_id = ?`
		args = append(args, projectID)
	}
	query += ` ORDER BY form_template_id, question_id`
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.SatisfactionIndicatorQuestion{}
	for rows.Next() {
		var item domain.SatisfactionIndicatorQuestion
		if err := rows.Scan(&item.ID, &item.ProjectID, &item.IndicatorID, &item.FormTemplateID, &item.QuestionID, &item.QuestionLabel, &item.ScoreDirection, &item.Weight, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *MemoryStore) UpsertSatisfactionIndicatorQuestion(ctx context.Context, item domain.SatisfactionIndicatorQuestion) (domain.SatisfactionIndicatorQuestion, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return domain.SatisfactionIndicatorQuestion{}, err
	}
	defer db.Close()
	if item.ID == "" {
		item.ID = uuid.NewString()
	}
	item.ScoreDirection = firstNonEmptyStore(item.ScoreDirection, "positive")
	if item.Weight == 0 {
		item.Weight = 1
	}
	_, err = db.ExecContext(ctx, `
INSERT INTO satisfaction_indicator_questions (id, project_id, indicator_id, form_template_id, question_id, question_label, score_direction, weight)
VALUES (?, NULLIF(?, ''), ?, ?, ?, NULLIF(?, ''), ?, ?)
ON DUPLICATE KEY UPDATE indicator_id=VALUES(indicator_id), question_label=VALUES(question_label), score_direction=VALUES(score_direction), weight=VALUES(weight)`,
		item.ID, item.ProjectID, item.IndicatorID, item.FormTemplateID, item.QuestionID, item.QuestionLabel, item.ScoreDirection, item.Weight)
	return item, err
}

func (s *MemoryStore) SatisfactionCleaningRules(ctx context.Context, projectID string) ([]domain.SatisfactionCleaningRule, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	query := `SELECT id, COALESCE(project_id, ''), name, rule_type, enabled, COALESCE(CAST(config_json AS CHAR), '{}'), action, created_at, updated_at FROM satisfaction_cleaning_rules`
	args := []interface{}{}
	if strings.TrimSpace(projectID) != "" {
		query += ` WHERE project_id IS NULL OR project_id = ?`
		args = append(args, projectID)
	}
	query += ` ORDER BY enabled DESC, rule_type, name`
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.SatisfactionCleaningRule{}
	for rows.Next() {
		var item domain.SatisfactionCleaningRule
		var raw string
		if err := rows.Scan(&item.ID, &item.ProjectID, &item.Name, &item.RuleType, &item.Enabled, &raw, &item.Action, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(raw), &item.Config)
		items = append(items, item)
	}
	if len(items) == 0 {
		defaults := defaultCleaningRules(projectID)
		for _, item := range defaults {
			if _, err := s.UpsertSatisfactionCleaningRule(ctx, item); err != nil {
				return nil, err
			}
		}
		return defaults, nil
	}
	return items, rows.Err()
}

func (s *MemoryStore) UpsertSatisfactionCleaningRule(ctx context.Context, item domain.SatisfactionCleaningRule) (domain.SatisfactionCleaningRule, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return domain.SatisfactionCleaningRule{}, err
	}
	defer db.Close()
	if item.ID == "" {
		item.ID = uuid.NewString()
	}
	item.RuleType = firstNonEmptyStore(item.RuleType, "duration")
	item.Action = firstNonEmptyStore(item.Action, "mark_suspicious")
	raw, _ := json.Marshal(item.Config)
	if string(raw) == "null" {
		raw = []byte("{}")
	}
	_, err = db.ExecContext(ctx, `
INSERT INTO satisfaction_cleaning_rules (id, project_id, name, rule_type, enabled, config_json, action)
VALUES (?, NULLIF(?, ''), ?, ?, ?, CAST(? AS JSON), ?)
ON DUPLICATE KEY UPDATE name=VALUES(name), rule_type=VALUES(rule_type), enabled=VALUES(enabled), config_json=VALUES(config_json), action=VALUES(action)`,
		item.ID, item.ProjectID, item.Name, item.RuleType, item.Enabled, string(raw), item.Action)
	return item, err
}

func defaultCleaningRules(projectID string) []domain.SatisfactionCleaningRule {
	return []domain.SatisfactionCleaningRule{
		{ID: uuid.NewString(), ProjectID: projectID, Name: "答题时长过短", RuleType: "duration", Enabled: true, Config: map[string]interface{}{"minSeconds": 20}, Action: "mark_suspicious"},
		{ID: uuid.NewString(), ProjectID: projectID, Name: "同项目重复提交", RuleType: "duplicate_project", Enabled: true, Config: map[string]interface{}{"windowHours": 24, "strategy": "keep_latest"}, Action: "mark_suspicious"},
		{ID: uuid.NewString(), ProjectID: projectID, Name: "全同选项", RuleType: "same_option", Enabled: true, Config: map[string]interface{}{"minQuestionCount": 5}, Action: "mark_suspicious"},
		{ID: uuid.NewString(), ProjectID: projectID, Name: "同 IP/设备高频提交", RuleType: "same_device", Enabled: false, Config: map[string]interface{}{"maxCount": 5, "windowHours": 1}, Action: "mark_suspicious"},
	}
}

func (s *MemoryStore) SatisfactionIssues(ctx context.Context, projectID string) ([]domain.SatisfactionIssue, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	query := `SELECT id, COALESCE(project_id, ''), COALESCE(submission_id, ''), COALESCE(indicator_id, ''), title, source, COALESCE(responsible_department, ''), COALESCE(responsible_person, ''), severity, COALESCE(suggestion, ''), COALESCE(measure, ''), COALESCE(CAST(material_urls AS CHAR), '[]'), COALESCE(verification_result, ''), status, COALESCE(DATE_FORMAT(due_date, '%Y-%m-%d'), ''), COALESCE(DATE_FORMAT(closed_at, '%Y-%m-%d %H:%i:%s'), ''), created_at, updated_at FROM satisfaction_issues`
	args := []interface{}{}
	if strings.TrimSpace(projectID) != "" {
		query += ` WHERE project_id = ?`
		args = append(args, projectID)
	}
	query += ` ORDER BY created_at DESC`
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.SatisfactionIssue{}
	for rows.Next() {
		var item domain.SatisfactionIssue
		var materials string
		if err := rows.Scan(&item.ID, &item.ProjectID, &item.SubmissionID, &item.IndicatorID, &item.Title, &item.Source, &item.ResponsibleDepartment, &item.ResponsiblePerson, &item.Severity, &item.Suggestion, &item.Measure, &materials, &item.VerificationResult, &item.Status, &item.DueDate, &item.ClosedAt, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(materials), &item.MaterialURLs)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *MemoryStore) UpsertSatisfactionIssue(ctx context.Context, item domain.SatisfactionIssue) (domain.SatisfactionIssue, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return domain.SatisfactionIssue{}, err
	}
	defer db.Close()
	if item.ID == "" {
		item.ID = uuid.NewString()
	}
	item.Source = firstNonEmptyStore(item.Source, "manual")
	item.Severity = firstNonEmptyStore(item.Severity, "medium")
	item.Status = firstNonEmptyStore(item.Status, "open")
	materials, _ := json.Marshal(item.MaterialURLs)
	if string(materials) == "null" {
		materials = []byte("[]")
	}
	closedAt := item.ClosedAt
	if item.Status == "closed" && strings.TrimSpace(closedAt) == "" {
		closedAt = time.Now().UTC().Format("2006-01-02 15:04:05")
	}
	_, err = db.ExecContext(ctx, `
INSERT INTO satisfaction_issues (id, project_id, submission_id, indicator_id, title, source, responsible_department, responsible_person, severity, suggestion, measure, material_urls, verification_result, status, due_date, closed_at)
VALUES (?, NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), ?, ?, NULLIF(?, ''), NULLIF(?, ''), ?, NULLIF(?, ''), NULLIF(?, ''), CAST(? AS JSON), NULLIF(?, ''), ?, NULLIF(?, ''), NULLIF(?, ''))
ON DUPLICATE KEY UPDATE title=VALUES(title), responsible_department=VALUES(responsible_department), responsible_person=VALUES(responsible_person), severity=VALUES(severity), suggestion=VALUES(suggestion), measure=VALUES(measure), material_urls=VALUES(material_urls), verification_result=VALUES(verification_result), status=VALUES(status), due_date=VALUES(due_date), closed_at=VALUES(closed_at)`,
		item.ID, item.ProjectID, item.SubmissionID, item.IndicatorID, item.Title, item.Source, item.ResponsibleDepartment, item.ResponsiblePerson, item.Severity, item.Suggestion, item.Measure, string(materials), item.VerificationResult, item.Status, item.DueDate, closedAt)
	item.ClosedAt = closedAt
	return item, err
}

func (s *MemoryStore) SatisfactionIssueEvents(ctx context.Context, issueID string) ([]domain.SatisfactionIssueEvent, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.QueryContext(ctx, `SELECT id, issue_id, action, COALESCE(from_status, ''), COALESCE(to_status, ''), COALESCE(content, ''), COALESCE(CAST(attachments_json AS CHAR), '[]'), COALESCE(actor_id, ''), created_at FROM satisfaction_issue_events WHERE issue_id = ? ORDER BY created_at DESC`, issueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.SatisfactionIssueEvent{}
	for rows.Next() {
		var item domain.SatisfactionIssueEvent
		var attachments string
		if err := rows.Scan(&item.ID, &item.IssueID, &item.Action, &item.FromStatus, &item.ToStatus, &item.Content, &attachments, &item.ActorID, &item.CreatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(attachments), &item.Attachments)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *MemoryStore) AddSatisfactionIssueEvent(ctx context.Context, item domain.SatisfactionIssueEvent) (domain.SatisfactionIssueEvent, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return domain.SatisfactionIssueEvent{}, err
	}
	defer db.Close()
	if item.ID == "" {
		item.ID = uuid.NewString()
	}
	item.Action = firstNonEmptyStore(item.Action, "note")
	attachments, _ := json.Marshal(item.Attachments)
	if string(attachments) == "null" {
		attachments = []byte("[]")
	}
	_, err = db.ExecContext(ctx, `INSERT INTO satisfaction_issue_events (id, issue_id, action, from_status, to_status, content, attachments_json, actor_id) VALUES (?, ?, ?, NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), CAST(? AS JSON), NULLIF(?, ''))`,
		item.ID, item.IssueID, item.Action, item.FromStatus, item.ToStatus, item.Content, string(attachments), item.ActorID)
	if err == nil && strings.TrimSpace(item.ToStatus) != "" {
		closed := ""
		if item.ToStatus == "closed" {
			closed = time.Now().UTC().Format("2006-01-02 15:04:05")
		}
		_, _ = db.ExecContext(ctx, `UPDATE satisfaction_issues SET status = ?, closed_at = CASE WHEN ? <> '' THEN ? ELSE closed_at END WHERE id = ?`, item.ToStatus, closed, closed, item.IssueID)
	}
	item.CreatedAt = time.Now().UTC()
	return item, err
}

func (s *MemoryStore) SurveySubmissionAuditLogs(ctx context.Context, submissionID string) ([]domain.SurveySubmissionAuditLog, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.QueryContext(ctx, `SELECT id, submission_id, COALESCE(project_id, ''), action, COALESCE(from_status, ''), COALESCE(to_status, ''), COALESCE(reason, ''), COALESCE(actor_id, ''), created_at FROM survey_submission_audit_logs WHERE submission_id = ? ORDER BY created_at DESC`, submissionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.SurveySubmissionAuditLog{}
	for rows.Next() {
		var item domain.SurveySubmissionAuditLog
		if err := rows.Scan(&item.ID, &item.SubmissionID, &item.ProjectID, &item.Action, &item.FromStatus, &item.ToStatus, &item.Reason, &item.ActorID, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *MemoryStore) AddSurveySubmissionAuditLog(ctx context.Context, item domain.SurveySubmissionAuditLog) (domain.SurveySubmissionAuditLog, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return domain.SurveySubmissionAuditLog{}, err
	}
	defer db.Close()
	if item.ID == "" {
		item.ID = uuid.NewString()
	}
	item.Action = firstNonEmptyStore(item.Action, "quality_update")
	_, err = db.ExecContext(ctx, `INSERT INTO survey_submission_audit_logs (id, submission_id, project_id, action, from_status, to_status, reason, actor_id) VALUES (?, ?, NULLIF(?, ''), ?, NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''))`,
		item.ID, item.SubmissionID, item.ProjectID, item.Action, item.FromStatus, item.ToStatus, item.Reason, item.ActorID)
	item.CreatedAt = time.Now().UTC()
	return item, err
}

func (s *MemoryStore) SurveySubmission(ctx context.Context, id string) (domain.SurveySubmission, error) {
	items, err := s.SurveySubmissions(ctx, "")
	if err != nil {
		return domain.SurveySubmission{}, err
	}
	for _, item := range items {
		if item.ID == id {
			answers, err := s.surveySubmissionAnswers(ctx, id)
			if err != nil {
				return item, err
			}
			item.AnswerItems = answers
			return item, nil
		}
	}
	return domain.SurveySubmission{}, ErrNotFound
}

func (s *MemoryStore) surveySubmissionAnswers(ctx context.Context, id string) ([]domain.SurveySubmissionAnswer, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.QueryContext(ctx, `SELECT id, submission_id, question_id, question_label, question_type, COALESCE(CAST(answer_json AS CHAR), 'null'), score, created_at FROM survey_submission_answers WHERE submission_id = ? ORDER BY created_at, question_id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.SurveySubmissionAnswer{}
	for rows.Next() {
		var item domain.SurveySubmissionAnswer
		var raw string
		var score sql.NullFloat64
		if err := rows.Scan(&item.ID, &item.SubmissionID, &item.QuestionID, &item.QuestionLabel, &item.QuestionType, &raw, &score, &item.CreatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(raw), &item.Answer)
		if score.Valid {
			item.Score = &score.Float64
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *MemoryStore) UpdateSurveySubmissionQuality(ctx context.Context, id, status, reason string) (domain.SurveySubmission, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return domain.SurveySubmission{}, err
	}
	defer db.Close()
	status = firstNonEmptyStore(status, "pending")
	var fromStatus, projectID string
	_ = db.QueryRowContext(ctx, `SELECT quality_status, COALESCE(project_id, '') FROM survey_submissions WHERE id = ?`, id).Scan(&fromStatus, &projectID)
	_, err = db.ExecContext(ctx, `UPDATE survey_submissions SET quality_status = ?, quality_reason = NULLIF(?, '') WHERE id = ?`, status, reason, id)
	if err != nil {
		return domain.SurveySubmission{}, err
	}
	_, _ = s.AddSurveySubmissionAuditLog(ctx, domain.SurveySubmissionAuditLog{SubmissionID: id, ProjectID: projectID, Action: "quality_update", FromStatus: fromStatus, ToStatus: status, Reason: reason})
	return s.SurveySubmission(ctx, id)
}

func qualityStatus(item domain.SurveySubmission) string {
	reasons := cleaningReasons(item)
	if len(reasons) > 0 {
		item.QualityReason = strings.Join(reasons, "；")
		return "suspicious"
	}
	return "pending"
}

func cleaningReasons(item domain.SurveySubmission) []string {
	reasons := []string{}
	if item.DurationSeconds > 0 && item.DurationSeconds < 10 {
		reasons = append(reasons, "答题时长过短")
	}
	if item.Answers != nil {
		if sameChoiceAnswers(item.Answers) {
			reasons = append(reasons, "疑似全同选项")
		}
		if strings.TrimSpace(fmt.Sprint(item.Answers["patient_phone"])) == "" && strings.TrimSpace(item.PatientID) == "" && !item.Anonymous {
			reasons = append(reasons, "实名调查缺少患者身份")
		}
	}
	return reasons
}

func sameChoiceAnswers(answers map[string]interface{}) bool {
	count := 0
	first := ""
	for key, value := range answers {
		if key == "patient_phone" || key == "patient_name" || key == "department" || key == "diagnosis" {
			continue
		}
		text := strings.TrimSpace(fmt.Sprint(value))
		if text == "" || strings.HasPrefix(text, "map[") || strings.HasPrefix(text, "[") {
			continue
		}
		if first == "" {
			first = text
		} else if text != first {
			return false
		}
		count++
	}
	return count >= 4
}

func scoreAnswer(answer interface{}) *float64 {
	switch value := answer.(type) {
	case float64:
		return &value
	case int:
		next := float64(value)
		return &next
	case string:
		var parsed float64
		if _, err := fmt.Sscanf(value, "%f", &parsed); err == nil {
			return &parsed
		}
	}
	return nil
}

func randomToken() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return uuid.NewString()
	}
	return hex.EncodeToString(buf)
}
