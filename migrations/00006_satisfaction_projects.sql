-- +goose Up
CREATE TABLE IF NOT EXISTS satisfaction_projects (
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
);

ALTER TABLE survey_share_links
  ADD COLUMN project_id CHAR(36) NULL AFTER id,
  ADD INDEX idx_survey_share_links_project (project_id);

CREATE TABLE IF NOT EXISTS survey_submissions (
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
);

CREATE TABLE IF NOT EXISTS survey_submission_answers (
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
);

-- +goose Down
DROP TABLE IF EXISTS survey_submission_answers;
DROP TABLE IF EXISTS survey_submissions;
ALTER TABLE survey_share_links DROP INDEX idx_survey_share_links_project;
ALTER TABLE survey_share_links DROP COLUMN project_id;
DROP TABLE IF EXISTS satisfaction_projects;
