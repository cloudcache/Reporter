-- +goose Up
CREATE TABLE IF NOT EXISTS patient_tags (
  id CHAR(36) PRIMARY KEY,
  name VARCHAR(120) NOT NULL UNIQUE,
  color VARCHAR(40) NOT NULL DEFAULT '#2563eb',
  description TEXT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS patient_tag_assignments (
  patient_id CHAR(36) NOT NULL,
  tag_id CHAR(36) NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (patient_id, tag_id),
  INDEX idx_patient_tag_assignments_tag (tag_id),
  CONSTRAINT fk_patient_tag_assignments_tag FOREIGN KEY (tag_id) REFERENCES patient_tags(id)
);

CREATE TABLE IF NOT EXISTS patient_groups (
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

CREATE TABLE IF NOT EXISTS patient_group_members (
  group_id CHAR(36) NOT NULL,
  patient_id CHAR(36) NOT NULL,
  visit_id CHAR(36) NULL,
  added_by CHAR(36) NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (group_id, patient_id),
  INDEX idx_patient_group_members_patient (patient_id),
  CONSTRAINT fk_patient_group_members_group FOREIGN KEY (group_id) REFERENCES patient_groups(id)
);

-- +goose Down
DROP TABLE IF EXISTS patient_group_members;
DROP TABLE IF EXISTS patient_groups;
DROP TABLE IF EXISTS patient_tag_assignments;
DROP TABLE IF EXISTS patient_tags;
