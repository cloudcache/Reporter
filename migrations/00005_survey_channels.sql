-- +goose Up
CREATE TABLE IF NOT EXISTS integration_channels (
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

CREATE TABLE IF NOT EXISTS survey_share_links (
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
);

CREATE TABLE IF NOT EXISTS survey_interviews (
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

-- +goose Down
DROP TABLE IF EXISTS survey_interviews;
DROP TABLE IF EXISTS survey_share_links;
DROP TABLE IF EXISTS integration_channels;
