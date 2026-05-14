-- +goose Up
CREATE TABLE IF NOT EXISTS evaluation_complaints (
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
);

CREATE TABLE IF NOT EXISTS evaluation_complaint_events (
  id CHAR(36) PRIMARY KEY,
  complaint_id CHAR(36) NOT NULL,
  actor_id CHAR(36) NULL,
  event_type VARCHAR(80) NOT NULL,
  comment TEXT NULL,
  payload_json JSON NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_eval_complaint_events_complaint (complaint_id),
  CONSTRAINT fk_eval_complaint_events_complaint FOREIGN KEY (complaint_id) REFERENCES evaluation_complaints(id)
);

-- +goose Down
DROP TABLE IF EXISTS evaluation_complaint_events;
DROP TABLE IF EXISTS evaluation_complaints;
