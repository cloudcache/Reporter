CREATE TABLE users (
  id CHAR(36) PRIMARY KEY,
  username VARCHAR(80) NOT NULL UNIQUE,
  display_name VARCHAR(120) NOT NULL,
  password_hash TEXT NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE roles (
  id VARCHAR(80) PRIMARY KEY,
  name VARCHAR(120) NOT NULL,
  description TEXT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE permissions (
  id CHAR(36) PRIMARY KEY,
  resource VARCHAR(160) NOT NULL,
  action VARCHAR(80) NOT NULL,
  description TEXT NULL,
  UNIQUE KEY uniq_permission (resource, action)
);

CREATE TABLE user_roles (
  user_id CHAR(36) NOT NULL,
  role_id VARCHAR(80) NOT NULL,
  PRIMARY KEY (user_id, role_id),
  CONSTRAINT fk_user_roles_user FOREIGN KEY (user_id) REFERENCES users(id),
  CONSTRAINT fk_user_roles_role FOREIGN KEY (role_id) REFERENCES roles(id)
);

CREATE TABLE role_permissions (
  role_id VARCHAR(80) NOT NULL,
  permission_id CHAR(36) NOT NULL,
  PRIMARY KEY (role_id, permission_id),
  CONSTRAINT fk_role_permissions_role FOREIGN KEY (role_id) REFERENCES roles(id),
  CONSTRAINT fk_role_permissions_permission FOREIGN KEY (permission_id) REFERENCES permissions(id)
);

CREATE TABLE patients (
  id CHAR(36) PRIMARY KEY,
  patient_no VARCHAR(80) NOT NULL UNIQUE,
  medical_record_no VARCHAR(80) NULL,
  name VARCHAR(120) NOT NULL,
  gender VARCHAR(20) NULL,
  birth_date DATE NULL,
  age INT NULL,
  id_card_no VARCHAR(80) NULL,
  phone VARCHAR(40) NULL,
  address TEXT NULL,
  nationality VARCHAR(80) NULL,
  ethnicity VARCHAR(80) NULL,
  marital_status VARCHAR(40) NULL,
  insurance_type VARCHAR(80) NULL,
  blood_type VARCHAR(20) NULL,
  allergies_json JSON NULL,
  emergency_contact VARCHAR(120) NULL,
  emergency_phone VARCHAR(40) NULL,
  diagnosis VARCHAR(240) NULL,
  status ENUM('active','follow_up','inactive') NOT NULL DEFAULT 'active',
  last_visit_at DATE NULL,
  source_refs_json JSON NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE clinical_visits (
  id CHAR(36) PRIMARY KEY,
  patient_id CHAR(36) NOT NULL,
  visit_no VARCHAR(100) NOT NULL,
  visit_type VARCHAR(60) NULL,
  department_code VARCHAR(80) NULL,
  department_name VARCHAR(120) NULL,
  ward VARCHAR(120) NULL,
  bed_no VARCHAR(40) NULL,
  attending_doctor VARCHAR(120) NULL,
  visit_at DATETIME NULL,
  discharge_at DATETIME NULL,
  diagnosis_code VARCHAR(80) NULL,
  diagnosis_name VARCHAR(240) NULL,
  status VARCHAR(40) NOT NULL DEFAULT 'active',
  source_refs_json JSON NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uniq_clinical_visit_no (visit_no),
  CONSTRAINT fk_clinical_visits_patient FOREIGN KEY (patient_id) REFERENCES patients(id)
);

CREATE TABLE medical_records (
  id CHAR(36) PRIMARY KEY,
  patient_id CHAR(36) NOT NULL,
  visit_id CHAR(36) NULL,
  record_no VARCHAR(120) NOT NULL,
  record_type VARCHAR(80) NOT NULL,
  title VARCHAR(180) NOT NULL,
  summary TEXT NULL,
  chief_complaint TEXT NULL,
  present_illness TEXT NULL,
  diagnosis_code VARCHAR(80) NULL,
  diagnosis_name VARCHAR(240) NULL,
  procedure_name VARCHAR(180) NULL,
  study_uid VARCHAR(180) NULL,
  study_desc VARCHAR(240) NULL,
  recorded_at DATETIME NULL,
  source_refs_json JSON NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uniq_medical_record_no (record_no),
  CONSTRAINT fk_medical_records_patient FOREIGN KEY (patient_id) REFERENCES patients(id),
  CONSTRAINT fk_medical_records_visit FOREIGN KEY (visit_id) REFERENCES clinical_visits(id)
);

CREATE TABLE datasets (
  id CHAR(36) PRIMARY KEY,
  name VARCHAR(180) NOT NULL,
  description TEXT NULL,
  owner VARCHAR(120) NULL,
  record_count INT NOT NULL DEFAULT 0,
  form_count INT NOT NULL DEFAULT 0,
  status ENUM('active','archived') NOT NULL DEFAULT 'active',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE departments (
  id VARCHAR(80) PRIMARY KEY,
  code VARCHAR(80) NOT NULL UNIQUE,
  name VARCHAR(180) NOT NULL,
  kind VARCHAR(60) NOT NULL DEFAULT 'clinical',
  status VARCHAR(40) NOT NULL DEFAULT 'active',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE dictionaries (
  id VARCHAR(80) PRIMARY KEY,
  code VARCHAR(120) NOT NULL UNIQUE,
  name VARCHAR(180) NOT NULL,
  category VARCHAR(120) NOT NULL,
  description TEXT NULL,
  items_json JSON NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE forms (
  id CHAR(36) PRIMARY KEY,
  name VARCHAR(180) NOT NULL,
  description TEXT NULL,
  status ENUM('draft','published','archived') NOT NULL DEFAULT 'draft',
  current_version_id CHAR(36) NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE form_versions (
  id CHAR(36) PRIMARY KEY,
  form_id CHAR(36) NOT NULL,
  version INT NOT NULL,
  schema_json JSON NOT NULL,
  created_by CHAR(36) NULL,
  published BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uniq_form_version (form_id, version),
  CONSTRAINT fk_form_versions_form FOREIGN KEY (form_id) REFERENCES forms(id),
  CONSTRAINT fk_form_versions_creator FOREIGN KEY (created_by) REFERENCES users(id)
);

CREATE TABLE form_components (
  id CHAR(36) PRIMARY KEY,
  form_version_id CHAR(36) NOT NULL,
  parent_component_id CHAR(36) NULL,
  component_key VARCHAR(120) NOT NULL,
  component_type VARCHAR(60) NOT NULL,
  label VARCHAR(180) NOT NULL,
  required BOOLEAN NOT NULL DEFAULT FALSE,
  config_json JSON NULL,
  binding_json JSON NULL,
  sort_order INT NOT NULL DEFAULT 0,
  CONSTRAINT fk_form_components_version FOREIGN KEY (form_version_id) REFERENCES form_versions(id)
);

CREATE TABLE form_library_items (
  id VARCHAR(120) PRIMARY KEY,
  kind ENUM('template','common','atom') NOT NULL,
  label VARCHAR(180) NOT NULL,
  hint TEXT NULL,
  scenario VARCHAR(40) NULL,
  components_json JSON NOT NULL,
  sort_order INT NOT NULL DEFAULT 0,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE form_submissions (
  id CHAR(36) PRIMARY KEY,
  form_id CHAR(36) NOT NULL,
  form_version_id CHAR(36) NOT NULL,
  submitter_id CHAR(36) NOT NULL,
  status ENUM('draft','submitted','approved','rejected') NOT NULL DEFAULT 'submitted',
  data_json JSON NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  CONSTRAINT fk_form_submissions_form FOREIGN KEY (form_id) REFERENCES forms(id),
  CONSTRAINT fk_form_submissions_version FOREIGN KEY (form_version_id) REFERENCES form_versions(id),
  CONSTRAINT fk_form_submissions_submitter FOREIGN KEY (submitter_id) REFERENCES users(id)
);

CREATE TABLE submission_events (
  id CHAR(36) PRIMARY KEY,
  submission_id CHAR(36) NOT NULL,
  actor_id CHAR(36) NULL,
  event_type VARCHAR(80) NOT NULL,
  payload_json JSON NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_submission_events_submission FOREIGN KEY (submission_id) REFERENCES form_submissions(id)
);

CREATE TABLE data_sources (
  id CHAR(36) PRIMARY KEY,
  name VARCHAR(180) NOT NULL,
  protocol ENUM('mysql','postgres','http','soap','xml','grpc','hl7','dicom','custom') NOT NULL,
  endpoint TEXT NOT NULL,
  config_json JSON NULL,
  dictionaries_json JSON NULL,
  field_mapping_json JSON NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE data_source_credentials (
  id CHAR(36) PRIMARY KEY,
  data_source_id CHAR(36) NOT NULL,
  secret_ciphertext BLOB NOT NULL,
  key_version VARCHAR(40) NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_data_source_credentials_source FOREIGN KEY (data_source_id) REFERENCES data_sources(id)
);

CREATE TABLE data_source_bindings (
  id CHAR(36) PRIMARY KEY,
  form_component_id CHAR(36) NULL,
  data_source_id CHAR(36) NOT NULL,
  operation VARCHAR(160) NOT NULL,
  params_json JSON NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_data_source_bindings_component FOREIGN KEY (form_component_id) REFERENCES form_components(id),
  CONSTRAINT fk_data_source_bindings_source FOREIGN KEY (data_source_id) REFERENCES data_sources(id)
);

CREATE TABLE followup_plans (
  id VARCHAR(80) PRIMARY KEY,
  name VARCHAR(180) NOT NULL,
  scenario VARCHAR(80) NOT NULL,
  disease_code VARCHAR(80) NULL,
  department_id VARCHAR(80) NULL,
  form_template_id VARCHAR(120) NOT NULL,
  trigger_type VARCHAR(80) NOT NULL,
  trigger_offset INT NOT NULL DEFAULT 0,
  channel VARCHAR(40) NOT NULL DEFAULT 'phone',
  assignee_role VARCHAR(80) NOT NULL DEFAULT 'agent',
  status VARCHAR(40) NOT NULL DEFAULT 'active',
  rules_json JSON NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE followup_tasks (
  id VARCHAR(80) PRIMARY KEY,
  plan_id VARCHAR(80) NULL,
  patient_id CHAR(36) NOT NULL,
  visit_id CHAR(36) NULL,
  form_id CHAR(36) NULL,
  form_template_id VARCHAR(120) NULL,
  assignee_id CHAR(36) NULL,
  role VARCHAR(80) NULL,
  channel VARCHAR(40) NOT NULL DEFAULT 'phone',
  status VARCHAR(40) NOT NULL DEFAULT 'pending',
  priority VARCHAR(40) NOT NULL DEFAULT 'normal',
  due_at DATE NULL,
  result_json JSON NULL,
  last_event TEXT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE patient_tags (
  id CHAR(36) PRIMARY KEY,
  name VARCHAR(120) NOT NULL UNIQUE,
  color VARCHAR(40) NOT NULL DEFAULT '#2563eb',
  description TEXT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE patient_tag_assignments (
  patient_id CHAR(36) NOT NULL,
  tag_id CHAR(36) NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (patient_id, tag_id),
  INDEX idx_patient_tag_assignments_tag (tag_id),
  CONSTRAINT fk_patient_tag_assignments_tag FOREIGN KEY (tag_id) REFERENCES patient_tags(id)
);

CREATE TABLE patient_groups (
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
);

CREATE TABLE patient_group_members (
  group_id CHAR(36) NOT NULL,
  patient_id CHAR(36) NOT NULL,
  visit_id CHAR(36) NULL,
  added_by CHAR(36) NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (group_id, patient_id),
  INDEX idx_patient_group_members_patient (patient_id),
  CONSTRAINT fk_patient_group_members_group FOREIGN KEY (group_id) REFERENCES patient_groups(id)
);

CREATE TABLE integration_channels (
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
);

CREATE TABLE survey_share_links (
  id CHAR(36) PRIMARY KEY,
  project_id CHAR(36) NULL,
  form_template_id VARCHAR(120) NOT NULL,
  title VARCHAR(180) NOT NULL,
  channel VARCHAR(40) NOT NULL DEFAULT 'web',
  token VARCHAR(80) NOT NULL UNIQUE,
  expires_at TIMESTAMP NULL,
  config_json JSON NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_survey_share_links_project (project_id),
  INDEX idx_survey_share_links_template (form_template_id),
  INDEX idx_survey_share_links_channel (channel)
);

CREATE TABLE survey_interviews (
  id CHAR(36) PRIMARY KEY,
  share_id CHAR(36) NOT NULL,
  patient_id CHAR(36) NULL,
  status VARCHAR(40) NOT NULL DEFAULT 'active',
  answers_json JSON NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_survey_interviews_share (share_id),
  CONSTRAINT fk_survey_interviews_share FOREIGN KEY (share_id) REFERENCES survey_share_links(id)
);

CREATE TABLE satisfaction_projects (
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

CREATE TABLE survey_submissions (
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

CREATE TABLE survey_submission_answers (
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

CREATE TABLE reports (
  id CHAR(36) PRIMARY KEY,
  name VARCHAR(180) NOT NULL,
  description TEXT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE report_versions (
  id CHAR(36) PRIMARY KEY,
  report_id CHAR(36) NOT NULL,
  version INT NOT NULL,
  layout_json JSON NOT NULL,
  created_by CHAR(36) NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uniq_report_version (report_id, version),
  CONSTRAINT fk_report_versions_report FOREIGN KEY (report_id) REFERENCES reports(id),
  CONSTRAINT fk_report_versions_creator FOREIGN KEY (created_by) REFERENCES users(id)
);

CREATE TABLE report_widgets (
  id CHAR(36) PRIMARY KEY,
  report_id CHAR(36) NOT NULL,
  widget_type VARCHAR(60) NOT NULL,
  title VARCHAR(180) NOT NULL,
  query_json JSON NULL,
  vis_spec_json JSON NULL,
  data_source_id CHAR(36) NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_report_widgets_report FOREIGN KEY (report_id) REFERENCES reports(id),
  CONSTRAINT fk_report_widgets_source FOREIGN KEY (data_source_id) REFERENCES data_sources(id)
);

CREATE TABLE report_query_results (
  id CHAR(36) PRIMARY KEY,
  report_id CHAR(36) NOT NULL,
  dimensions_json JSON NULL,
  measures_json JSON NULL,
  rows_json JSON NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_report_query_results_report FOREIGN KEY (report_id) REFERENCES reports(id)
);

CREATE TABLE report_queries (
  id CHAR(36) PRIMARY KEY,
  report_id CHAR(36) NOT NULL,
  data_source_id CHAR(36) NULL,
  query_template TEXT NOT NULL,
  params_schema JSON NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_report_queries_report FOREIGN KEY (report_id) REFERENCES reports(id),
  CONSTRAINT fk_report_queries_source FOREIGN KEY (data_source_id) REFERENCES data_sources(id)
);

CREATE TABLE evaluation_complaints (
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

CREATE TABLE evaluation_complaint_events (
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

CREATE TABLE agent_seats (
  id CHAR(36) PRIMARY KEY,
  user_id CHAR(36) NULL,
  name VARCHAR(120) NOT NULL,
  extension VARCHAR(40) NOT NULL,
  sip_uri VARCHAR(180) NOT NULL,
  status ENUM('available','busy','offline','wrap_up') NOT NULL DEFAULT 'offline',
  skills_json JSON NULL,
  current_call_id CHAR(36) NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  CONSTRAINT fk_agent_seats_user FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE TABLE sip_endpoints (
  id CHAR(36) PRIMARY KEY,
  name VARCHAR(160) NOT NULL,
  wss_url TEXT NOT NULL,
  domain VARCHAR(160) NOT NULL,
  proxy TEXT NULL,
  config_json JSON NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE storage_configs (
  id CHAR(36) PRIMARY KEY,
  name VARCHAR(160) NOT NULL,
  kind VARCHAR(40) NOT NULL,
  endpoint TEXT NULL,
  bucket VARCHAR(180) NULL,
  base_path TEXT NULL,
  base_uri TEXT NULL,
  credential_ref VARCHAR(180) NULL,
  config_json JSON NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE recording_configs (
  id CHAR(36) PRIMARY KEY,
  name VARCHAR(160) NOT NULL,
  mode VARCHAR(40) NOT NULL,
  storage_config_id CHAR(36) NOT NULL,
  format VARCHAR(40) NOT NULL,
  retention_days INT NOT NULL DEFAULT 365,
  auto_start BOOLEAN NOT NULL DEFAULT TRUE,
  auto_stop BOOLEAN NOT NULL DEFAULT TRUE,
  config_json JSON NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  CONSTRAINT fk_recording_configs_storage FOREIGN KEY (storage_config_id) REFERENCES storage_configs(id)
);

CREATE TABLE call_sessions (
  id CHAR(36) PRIMARY KEY,
  seat_id CHAR(36) NOT NULL,
  patient_id CHAR(36) NULL,
  direction ENUM('inbound','outbound') NOT NULL,
  phone_number VARCHAR(40) NOT NULL,
  status ENUM('dialing','ringing','connected','recording','recorded','ended','failed') NOT NULL DEFAULT 'dialing',
  started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  ended_at TIMESTAMP NULL,
  recording_id CHAR(36) NULL,
  transcript_id CHAR(36) NULL,
  analysis_id CHAR(36) NULL,
  interview_form VARCHAR(120) NULL,
  CONSTRAINT fk_call_sessions_seat FOREIGN KEY (seat_id) REFERENCES agent_seats(id),
  CONSTRAINT fk_call_sessions_patient FOREIGN KEY (patient_id) REFERENCES patients(id)
);

CREATE TABLE recordings (
  id CHAR(36) PRIMARY KEY,
  call_id CHAR(36) NOT NULL,
  storage_uri TEXT NOT NULL,
  duration INT NOT NULL DEFAULT 0,
  filename VARCHAR(240) NULL,
  mime_type VARCHAR(120) NULL,
  size_bytes BIGINT NOT NULL DEFAULT 0,
  source VARCHAR(80) NOT NULL DEFAULT 'browser',
  backend VARCHAR(40) NOT NULL DEFAULT 'local',
  object_name VARCHAR(512) NULL,
  status ENUM('recording','ready','failed') NOT NULL DEFAULT 'recording',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_recordings_call FOREIGN KEY (call_id) REFERENCES call_sessions(id)
);

CREATE TABLE model_providers (
  id CHAR(36) PRIMARY KEY,
  name VARCHAR(160) NOT NULL,
  kind VARCHAR(80) NOT NULL,
  mode ENUM('realtime','offline','both') NOT NULL DEFAULT 'offline',
  endpoint TEXT NOT NULL,
  model VARCHAR(120) NOT NULL,
  credential_ref VARCHAR(180) NULL,
  config_json JSON NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE realtime_assist_sessions (
  id CHAR(36) PRIMARY KEY,
  call_id CHAR(36) NOT NULL,
  patient_id CHAR(36) NULL,
  form_id VARCHAR(120) NOT NULL,
  provider_id CHAR(36) NOT NULL,
  status ENUM('active','completed','failed') NOT NULL DEFAULT 'active',
  transcript_json JSON NULL,
  form_draft_json JSON NULL,
  last_suggestion TEXT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  CONSTRAINT fk_realtime_assist_call FOREIGN KEY (call_id) REFERENCES call_sessions(id),
  CONSTRAINT fk_realtime_assist_patient FOREIGN KEY (patient_id) REFERENCES patients(id),
  CONSTRAINT fk_realtime_assist_provider FOREIGN KEY (provider_id) REFERENCES model_providers(id)
);

CREATE TABLE offline_analysis_jobs (
  id CHAR(36) PRIMARY KEY,
  call_id CHAR(36) NOT NULL,
  recording_id CHAR(36) NOT NULL,
  provider_id CHAR(36) NOT NULL,
  status ENUM('queued','running','completed','failed') NOT NULL DEFAULT 'queued',
  result_json JSON NULL,
  error TEXT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  CONSTRAINT fk_offline_analysis_call FOREIGN KEY (call_id) REFERENCES call_sessions(id),
  CONSTRAINT fk_offline_analysis_recording FOREIGN KEY (recording_id) REFERENCES recordings(id),
  CONSTRAINT fk_offline_analysis_provider FOREIGN KEY (provider_id) REFERENCES model_providers(id)
);

CREATE TABLE call_analyses (
  id CHAR(36) PRIMARY KEY,
  call_id CHAR(36) NOT NULL,
  provider_id CHAR(36) NOT NULL,
  patient_emotion VARCHAR(120) NULL,
  true_satisfaction DECIMAL(4,2) NULL,
  risk_level VARCHAR(40) NULL,
  patient_status VARCHAR(120) NULL,
  summary TEXT NULL,
  extracted_form_data JSON NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_call_analyses_call FOREIGN KEY (call_id) REFERENCES call_sessions(id),
  CONSTRAINT fk_call_analyses_provider FOREIGN KEY (provider_id) REFERENCES model_providers(id)
);

CREATE TABLE interview_sessions (
  id CHAR(36) PRIMARY KEY,
  patient_id CHAR(36) NOT NULL,
  form_id VARCHAR(120) NOT NULL,
  call_id CHAR(36) NULL,
  mode ENUM('chat','call','chat_call') NOT NULL DEFAULT 'chat',
  status ENUM('draft','active','completed','abandoned') NOT NULL DEFAULT 'draft',
  messages_json JSON NULL,
  form_draft_json JSON NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  CONSTRAINT fk_interview_sessions_patient FOREIGN KEY (patient_id) REFERENCES patients(id),
  CONSTRAINT fk_interview_sessions_call FOREIGN KEY (call_id) REFERENCES call_sessions(id)
);

CREATE TABLE audit_logs (
  id CHAR(36) PRIMARY KEY,
  actor_id CHAR(36) NULL,
  action VARCHAR(120) NOT NULL,
  resource VARCHAR(240) NOT NULL,
  before_json JSON NULL,
  after_json JSON NULL,
  ip VARCHAR(80) NULL,
  user_agent TEXT NULL,
  trace_id VARCHAR(80) NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE system_settings (
  setting_key VARCHAR(120) PRIMARY KEY,
  setting_value JSON NULL,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

INSERT INTO roles (id, name, description) VALUES
('admin', '系统管理员', '拥有平台全部管理权限'),
('doctor', '医生', '查看患者档案、制定随访方案、处理异常结果'),
('nurse', '护士', '维护护理随访、宣教和患者基础信息'),
('analyst', '数据分析员', '可管理表单、报表并查看数据源'),
('agent', '随访员/调查员', '可查看患者并执行电话随访、问卷调查')
ON DUPLICATE KEY UPDATE name = VALUES(name), description = VALUES(description);

INSERT INTO storage_configs (id, name, kind, base_path, config_json) VALUES
('STOR001', '本地录音存储', 'local', 'data/recordings', JSON_OBJECT('pathStrategy', 'yyyy/mm/dd'))
ON DUPLICATE KEY UPDATE name = VALUES(name), kind = VALUES(kind), base_path = VALUES(base_path), config_json = VALUES(config_json);

INSERT INTO recording_configs (id, name, mode, storage_config_id, format, retention_days, auto_start, auto_stop, config_json) VALUES
('REC-CFG-001', '默认通话录音策略', 'server', 'STOR001', 'wav', 365, TRUE, TRUE, JSON_OBJECT('source', 'pbx_or_diago'))
ON DUPLICATE KEY UPDATE name = VALUES(name), mode = VALUES(mode), storage_config_id = VALUES(storage_config_id), format = VALUES(format), retention_days = VALUES(retention_days), auto_start = VALUES(auto_start), auto_stop = VALUES(auto_stop), config_json = VALUES(config_json);
