-- +goose Up
CREATE TABLE IF NOT EXISTS satisfaction_indicators (
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
);

CREATE TABLE IF NOT EXISTS satisfaction_issues (
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
);

-- +goose Down
DROP TABLE IF EXISTS satisfaction_issues;
DROP TABLE IF EXISTS satisfaction_indicators;
