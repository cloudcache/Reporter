-- +goose Up
CREATE TABLE IF NOT EXISTS departments (
  id VARCHAR(80) PRIMARY KEY,
  code VARCHAR(80) NOT NULL UNIQUE,
  name VARCHAR(180) NOT NULL,
  kind VARCHAR(60) NOT NULL DEFAULT 'clinical',
  status VARCHAR(40) NOT NULL DEFAULT 'active',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS dictionaries (
  id VARCHAR(80) PRIMARY KEY,
  code VARCHAR(120) NOT NULL UNIQUE,
  name VARCHAR(180) NOT NULL,
  category VARCHAR(120) NOT NULL,
  description TEXT NULL,
  items_json JSON NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS form_library_items (
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

CREATE TABLE IF NOT EXISTS followup_plans (
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

CREATE TABLE IF NOT EXISTS followup_tasks (
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

INSERT INTO roles (id, name, description)
VALUES
  ('doctor', '医生', '查看患者档案、制定随访方案、处理异常结果'),
  ('nurse', '护士', '维护护理随访、宣教和患者基础信息'),
  ('agent', '随访员/调查员', '可查看患者并执行电话随访、问卷调查')
ON DUPLICATE KEY UPDATE
  name = VALUES(name),
  description = VALUES(description);

-- +goose Down
DROP TABLE IF EXISTS followup_tasks;
DROP TABLE IF EXISTS followup_plans;
DROP TABLE IF EXISTS form_library_items;
DROP TABLE IF EXISTS dictionaries;
DROP TABLE IF EXISTS departments;
