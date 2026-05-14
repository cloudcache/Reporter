-- +goose Up
ALTER TABLE satisfaction_indicators
  ADD COLUMN service_stage VARCHAR(120) NULL AFTER name,
  ADD COLUMN service_node VARCHAR(120) NULL AFTER service_stage,
  ADD COLUMN include_total_score BOOLEAN NOT NULL DEFAULT TRUE AFTER weight,
  ADD COLUMN include_national BOOLEAN NOT NULL DEFAULT FALSE AFTER national_dimension;

CREATE TABLE IF NOT EXISTS satisfaction_indicator_questions (
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
);

CREATE TABLE IF NOT EXISTS satisfaction_cleaning_rules (
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
);

CREATE TABLE IF NOT EXISTS survey_submission_audit_logs (
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
);

ALTER TABLE satisfaction_issues
  ADD COLUMN measure TEXT NULL AFTER suggestion,
  ADD COLUMN material_urls JSON NULL AFTER measure,
  ADD COLUMN verification_result TEXT NULL AFTER material_urls,
  ADD COLUMN closed_at TIMESTAMP NULL AFTER due_date;

CREATE TABLE IF NOT EXISTS satisfaction_issue_events (
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
);

-- +goose Down
DROP TABLE IF EXISTS satisfaction_issue_events;
ALTER TABLE satisfaction_issues
  DROP COLUMN closed_at,
  DROP COLUMN verification_result,
  DROP COLUMN material_urls,
  DROP COLUMN measure;
DROP TABLE IF EXISTS survey_submission_audit_logs;
DROP TABLE IF EXISTS satisfaction_cleaning_rules;
DROP TABLE IF EXISTS satisfaction_indicator_questions;
ALTER TABLE satisfaction_indicators
  DROP COLUMN include_national,
  DROP COLUMN include_total_score,
  DROP COLUMN service_node,
  DROP COLUMN service_stage;
