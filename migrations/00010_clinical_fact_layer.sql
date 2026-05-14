-- +goose Up
CREATE TABLE IF NOT EXISTS patient_diagnoses (
  id CHAR(36) PRIMARY KEY,
  patient_id CHAR(36) NOT NULL,
  visit_id CHAR(36) NULL,
  diagnosis_code VARCHAR(80) NULL,
  diagnosis_name VARCHAR(180) NOT NULL,
  diagnosis_type VARCHAR(60) NOT NULL DEFAULT 'primary',
  diagnosed_at DATETIME NULL,
  department_name VARCHAR(120) NULL,
  doctor_name VARCHAR(120) NULL,
  source_system VARCHAR(80) NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_patient_diagnoses_patient (patient_id),
  INDEX idx_patient_diagnoses_visit (visit_id),
  INDEX idx_patient_diagnoses_code (diagnosis_code)
);

CREATE TABLE IF NOT EXISTS patient_histories (
  id CHAR(36) PRIMARY KEY,
  patient_id CHAR(36) NOT NULL,
  history_type VARCHAR(60) NOT NULL,
  title VARCHAR(180) NOT NULL,
  content TEXT NULL,
  recorded_at DATETIME NULL,
  source_system VARCHAR(80) NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_patient_histories_patient (patient_id),
  INDEX idx_patient_histories_type (history_type)
);

CREATE TABLE IF NOT EXISTS medication_orders (
  id CHAR(36) PRIMARY KEY,
  patient_id CHAR(36) NOT NULL,
  visit_id CHAR(36) NULL,
  order_no VARCHAR(120) NULL,
  prescription_no VARCHAR(120) NULL,
  drug_code VARCHAR(80) NULL,
  drug_name VARCHAR(180) NOT NULL,
  generic_name VARCHAR(180) NULL,
  specification VARCHAR(180) NULL,
  dosage VARCHAR(80) NULL,
  dosage_unit VARCHAR(40) NULL,
  frequency VARCHAR(80) NULL,
  route VARCHAR(80) NULL,
  start_at DATETIME NULL,
  end_at DATETIME NULL,
  days INT NULL,
  quantity DECIMAL(10,2) NULL,
  manufacturer VARCHAR(180) NULL,
  doctor_name VARCHAR(120) NULL,
  pharmacist_name VARCHAR(120) NULL,
  status VARCHAR(40) NOT NULL DEFAULT 'active',
  adverse_reaction TEXT NULL,
  compliance VARCHAR(60) NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_medication_orders_patient (patient_id),
  INDEX idx_medication_orders_visit (visit_id),
  INDEX idx_medication_orders_drug (drug_code)
);

CREATE TABLE IF NOT EXISTS lab_reports (
  id CHAR(36) PRIMARY KEY,
  patient_id CHAR(36) NOT NULL,
  visit_id CHAR(36) NULL,
  report_no VARCHAR(120) NOT NULL,
  report_name VARCHAR(180) NOT NULL,
  specimen VARCHAR(80) NULL,
  ordered_at DATETIME NULL,
  reported_at DATETIME NULL,
  department_name VARCHAR(120) NULL,
  doctor_name VARCHAR(120) NULL,
  status VARCHAR(40) NOT NULL DEFAULT 'reported',
  source_system VARCHAR(80) NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uniq_lab_report_no (report_no),
  INDEX idx_lab_reports_patient (patient_id),
  INDEX idx_lab_reports_visit (visit_id)
);

CREATE TABLE IF NOT EXISTS lab_results (
  id CHAR(36) PRIMARY KEY,
  report_id CHAR(36) NOT NULL,
  item_code VARCHAR(80) NULL,
  item_name VARCHAR(180) NOT NULL,
  result_value VARCHAR(120) NULL,
  unit VARCHAR(60) NULL,
  reference_range VARCHAR(120) NULL,
  abnormal_flag VARCHAR(40) NULL,
  numeric_value DECIMAL(12,4) NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_lab_results_report (report_id),
  INDEX idx_lab_results_item (item_code),
  INDEX idx_lab_results_abnormal (abnormal_flag)
);

CREATE TABLE IF NOT EXISTS exam_reports (
  id CHAR(36) PRIMARY KEY,
  patient_id CHAR(36) NOT NULL,
  visit_id CHAR(36) NULL,
  exam_no VARCHAR(120) NOT NULL,
  exam_type VARCHAR(80) NULL,
  exam_name VARCHAR(180) NOT NULL,
  body_part VARCHAR(120) NULL,
  report_conclusion TEXT NULL,
  report_findings TEXT NULL,
  ordered_at DATETIME NULL,
  reported_at DATETIME NULL,
  department_name VARCHAR(120) NULL,
  doctor_name VARCHAR(120) NULL,
  source_system VARCHAR(80) NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uniq_exam_report_no (exam_no),
  INDEX idx_exam_reports_patient (patient_id),
  INDEX idx_exam_reports_visit (visit_id)
);

CREATE TABLE IF NOT EXISTS surgery_records (
  id CHAR(36) PRIMARY KEY,
  patient_id CHAR(36) NOT NULL,
  visit_id CHAR(36) NULL,
  operation_code VARCHAR(80) NULL,
  operation_name VARCHAR(180) NOT NULL,
  operation_date DATETIME NULL,
  surgeon_name VARCHAR(120) NULL,
  anesthesia_type VARCHAR(80) NULL,
  operation_level VARCHAR(60) NULL,
  wound_grade VARCHAR(60) NULL,
  outcome VARCHAR(120) NULL,
  source_system VARCHAR(80) NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_surgery_records_patient (patient_id),
  INDEX idx_surgery_records_visit (visit_id),
  INDEX idx_surgery_records_operation (operation_code)
);

CREATE TABLE IF NOT EXISTS followup_records (
  id CHAR(36) PRIMARY KEY,
  patient_id CHAR(36) NOT NULL,
  visit_id CHAR(36) NULL,
  task_id VARCHAR(80) NULL,
  project_id VARCHAR(80) NULL,
  followup_type VARCHAR(80) NULL,
  channel VARCHAR(40) NULL,
  status VARCHAR(40) NOT NULL DEFAULT 'completed',
  summary TEXT NULL,
  satisfaction_score DECIMAL(5,2) NULL,
  risk_level VARCHAR(40) NULL,
  followed_at DATETIME NULL,
  operator_name VARCHAR(120) NULL,
  source_system VARCHAR(80) NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_followup_records_patient (patient_id),
  INDEX idx_followup_records_task (task_id),
  INDEX idx_followup_records_project (project_id)
);

CREATE TABLE IF NOT EXISTS interview_extracted_facts (
  id CHAR(36) PRIMARY KEY,
  patient_id CHAR(36) NOT NULL,
  visit_id CHAR(36) NULL,
  interview_id CHAR(36) NULL,
  fact_type VARCHAR(80) NOT NULL,
  fact_key VARCHAR(120) NOT NULL,
  fact_label VARCHAR(180) NOT NULL,
  fact_value TEXT NULL,
  confidence DECIMAL(5,4) NULL,
  extracted_at DATETIME NULL,
  source_text TEXT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_interview_facts_patient (patient_id),
  INDEX idx_interview_facts_key (fact_key),
  INDEX idx_interview_facts_interview (interview_id)
);

CREATE TABLE IF NOT EXISTS satisfaction_indicator_scores (
  id CHAR(36) PRIMARY KEY,
  project_id VARCHAR(80) NOT NULL,
  indicator_id CHAR(36) NOT NULL,
  patient_id CHAR(36) NULL,
  visit_id CHAR(36) NULL,
  department_name VARCHAR(120) NULL,
  doctor_name VARCHAR(120) NULL,
  nurse_name VARCHAR(120) NULL,
  disease_name VARCHAR(180) NULL,
  visit_type VARCHAR(60) NULL,
  score DECIMAL(10,2) NOT NULL DEFAULT 0,
  sample_count INT NOT NULL DEFAULT 0,
  score_period DATE NULL,
  source_json JSON NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_satisfaction_scores_project (project_id),
  INDEX idx_satisfaction_scores_indicator (indicator_id),
  INDEX idx_satisfaction_scores_patient (patient_id),
  INDEX idx_satisfaction_scores_department (department_name),
  INDEX idx_satisfaction_scores_period (score_period)
);

-- +goose Down
DROP TABLE IF EXISTS satisfaction_indicator_scores;
DROP TABLE IF EXISTS interview_extracted_facts;
DROP TABLE IF EXISTS followup_records;
DROP TABLE IF EXISTS surgery_records;
DROP TABLE IF EXISTS exam_reports;
DROP TABLE IF EXISTS lab_results;
DROP TABLE IF EXISTS lab_reports;
DROP TABLE IF EXISTS medication_orders;
DROP TABLE IF EXISTS patient_histories;
DROP TABLE IF EXISTS patient_diagnoses;
