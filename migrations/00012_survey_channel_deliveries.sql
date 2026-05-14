-- +goose Up
CREATE TABLE IF NOT EXISTS survey_channel_deliveries (
  id CHAR(36) PRIMARY KEY,
  project_id CHAR(36) NULL,
  share_id CHAR(36) NOT NULL,
  channel VARCHAR(40) NOT NULL,
  recipient VARCHAR(180) NOT NULL,
  recipient_name VARCHAR(120) NULL,
  status VARCHAR(40) NOT NULL DEFAULT 'queued',
  message TEXT NULL,
  error TEXT NULL,
  provider_ref VARCHAR(180) NULL,
  config_json JSON NULL,
  sent_at TIMESTAMP NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_survey_deliveries_project (project_id),
  INDEX idx_survey_deliveries_share (share_id),
  INDEX idx_survey_deliveries_status (status),
  INDEX idx_survey_deliveries_recipient (recipient)
);

-- +goose Down
DROP TABLE IF EXISTS survey_channel_deliveries;
