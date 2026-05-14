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
	_, err = db.ExecContext(ctx, `UPDATE survey_submissions SET quality_status = ?, quality_reason = NULLIF(?, '') WHERE id = ?`, status, reason, id)
	if err != nil {
		return domain.SurveySubmission{}, err
	}
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
