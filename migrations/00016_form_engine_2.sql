CREATE TABLE IF NOT EXISTS form_schema_registry (
  id CHAR(36) PRIMARY KEY,
  form_id CHAR(36) NOT NULL,
  version_id CHAR(36) NOT NULL,
  schema_name VARCHAR(180) NOT NULL,
  schema_hash VARCHAR(64) NOT NULL,
  status VARCHAR(40) NOT NULL DEFAULT 'draft',
  description TEXT NULL,
  json_schema JSON NULL,
  created_by CHAR(36) NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uniq_form_schema_version (version_id),
  INDEX idx_form_schema_form (form_id, status),
  INDEX idx_form_schema_hash (schema_hash),
  CONSTRAINT fk_form_schema_form FOREIGN KEY (form_id) REFERENCES forms(id),
  CONSTRAINT fk_form_schema_version FOREIGN KEY (version_id) REFERENCES form_versions(id),
  CONSTRAINT fk_form_schema_creator FOREIGN KEY (created_by) REFERENCES users(id)
);

CREATE TABLE IF NOT EXISTS component_templates (
  id VARCHAR(120) PRIMARY KEY,
  category VARCHAR(80) NOT NULL,
  name VARCHAR(180) NOT NULL,
  description TEXT NULL,
  component_type VARCHAR(60) NOT NULL,
  schema_json JSON NOT NULL,
  preview_json JSON NULL,
  tags_json JSON NULL,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_component_templates_category (category, enabled),
  INDEX idx_component_templates_type (component_type)
);

CREATE TABLE IF NOT EXISTS question_bank_items (
  id CHAR(36) PRIMARY KEY,
  category VARCHAR(80) NOT NULL,
  question_id VARCHAR(120) NOT NULL,
  label VARCHAR(255) NOT NULL,
  question_type VARCHAR(60) NOT NULL,
  options_json JSON NULL,
  validation_json JSON NULL,
  tags_json JSON NULL,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uniq_question_bank_question (category, question_id),
  INDEX idx_question_bank_type (question_type),
  INDEX idx_question_bank_enabled (enabled)
);

CREATE TABLE IF NOT EXISTS form_attachments (
  id CHAR(36) PRIMARY KEY,
  submission_id CHAR(36) NULL,
  form_id CHAR(36) NULL,
  form_version_id CHAR(36) NULL,
  component_id VARCHAR(120) NOT NULL,
  file_name VARCHAR(255) NOT NULL,
  mime_type VARCHAR(120) NOT NULL,
  file_kind VARCHAR(40) NOT NULL,
  size_bytes BIGINT NOT NULL DEFAULT 0,
  storage_config_id CHAR(36) NULL,
  storage_uri TEXT NOT NULL,
  object_name VARCHAR(512) NULL,
  checksum VARCHAR(128) NULL,
  metadata_json JSON NULL,
  created_by CHAR(36) NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_form_attachments_submission (submission_id),
  INDEX idx_form_attachments_form (form_id, form_version_id),
  INDEX idx_form_attachments_component (component_id),
  INDEX idx_form_attachments_kind (file_kind),
  CONSTRAINT fk_form_attachments_submission FOREIGN KEY (submission_id) REFERENCES form_submissions(id),
  CONSTRAINT fk_form_attachments_form FOREIGN KEY (form_id) REFERENCES forms(id),
  CONSTRAINT fk_form_attachments_version FOREIGN KEY (form_version_id) REFERENCES form_versions(id),
  CONSTRAINT fk_form_attachments_storage FOREIGN KEY (storage_config_id) REFERENCES storage_configs(id),
  CONSTRAINT fk_form_attachments_creator FOREIGN KEY (created_by) REFERENCES users(id)
);

INSERT INTO component_templates (id, category, name, description, component_type, schema_json, preview_json, tags_json, enabled)
VALUES
('multi-sheet-table', '多维数据', '多维表格', '用于明细、指标、规格、用药和费用等二维/多维采集', 'table',
 '{"type":"table","label":"多维表格","columns":["项目","数值","单位"],"rows":["记录1"],"config":{"addRows":true,"addColumns":false}}',
 '{"columns":["项目","数值","单位"],"sample":[["血压","135/82","mmHg"]]}',
 '["table","sheet","multimodal"]', TRUE),
('computed-field', '计算逻辑', '计算字段', '按表达式从其他字段实时计算得分、费用、风险值', 'computed',
 '{"type":"computed","label":"计算字段","config":{"expression":"","precision":2,"readonly":true}}',
 '{"expression":"score_a * weight_a + score_b * weight_b"}',
 '["formula","calculation"]', TRUE),
('media-upload', '多模态附件', '文件/图片/视频/录音上传', '统一上传附件到对象存储并写回附件索引', 'attachment',
 '{"type":"attachment","label":"附件上传","config":{"accept":["image/*","video/*","audio/*","application/pdf"],"maxSizeMb":200,"multiple":true}}',
 '{"accept":"image/video/audio/pdf"}',
 '["file","image","video","audio","object-storage"]', TRUE)
ON DUPLICATE KEY UPDATE name = VALUES(name), description = VALUES(description), schema_json = VALUES(schema_json), preview_json = VALUES(preview_json), tags_json = VALUES(tags_json), enabled = VALUES(enabled);

INSERT INTO question_bank_items (id, category, question_id, label, question_type, options_json, validation_json, tags_json, enabled)
VALUES
(UUID(), '满意度', 'overall_satisfaction', '总体满意度', 'single_select',
 '[{"label":"很不满意","value":"1"},{"label":"不满意","value":"2"},{"label":"一般","value":"3"},{"label":"满意","value":"4"},{"label":"非常满意","value":"5"}]',
 '{"required":true}', '["satisfaction","score"]', TRUE),
(UUID(), '满意度', 'recommend_score', '推荐意愿', 'single_select',
 '[{"label":"0","value":"0"},{"label":"1","value":"1"},{"label":"2","value":"2"},{"label":"3","value":"3"},{"label":"4","value":"4"},{"label":"5","value":"5"},{"label":"6","value":"6"},{"label":"7","value":"7"},{"label":"8","value":"8"},{"label":"9","value":"9"},{"label":"10","value":"10"}]',
 '{"required":true}', '["nps","score"]', TRUE)
ON DUPLICATE KEY UPDATE label = VALUES(label), question_type = VALUES(question_type), options_json = VALUES(options_json), validation_json = VALUES(validation_json), tags_json = VALUES(tags_json), enabled = VALUES(enabled);

