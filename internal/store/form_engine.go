package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"

	"reporter/internal/domain"
)

func (s *Store) EnsureFormEngineTables(ctx context.Context) error {
	db, err := s.openConfiguredDB()
	if err != nil {
		return err
	}
	defer db.Close()
	statements := []string{
		`CREATE TABLE IF NOT EXISTS form_schema_registry (
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
  INDEX idx_form_schema_hash (schema_hash)
)`,
		`CREATE TABLE IF NOT EXISTS component_templates (
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
)`,
		`CREATE TABLE IF NOT EXISTS question_bank_items (
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
)`,
		`CREATE TABLE IF NOT EXISTS form_attachments (
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
  INDEX idx_form_attachments_kind (file_kind)
)`,
	}
	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return seedFormEngine(ctx, db)
}

func seedFormEngine(ctx context.Context, db *sql.DB) error {
	templates := []domain.ComponentTemplate{
		{ID: "multi-sheet-table", Category: "多维数据", Name: "多维表格", Description: "用于明细、指标、规格、用药和费用等二维/多维采集", ComponentType: "table", Enabled: true, Tags: []string{"table", "sheet", "multimodal"}, Schema: map[string]interface{}{"type": "table", "label": "多维表格", "columns": []string{"项目", "数值", "单位"}, "rows": []string{"记录1"}, "config": map[string]interface{}{"addRows": true, "addColumns": false}}, Preview: map[string]interface{}{"columns": []string{"项目", "数值", "单位"}}},
		{ID: "computed-field", Category: "计算逻辑", Name: "计算字段", Description: "按表达式从其他字段实时计算得分、费用、风险值", ComponentType: "computed", Enabled: true, Tags: []string{"formula", "calculation"}, Schema: map[string]interface{}{"type": "computed", "label": "计算字段", "config": map[string]interface{}{"expression": "", "precision": 2, "readonly": true}}},
		{ID: "media-upload", Category: "多模态附件", Name: "文件/图片/视频/录音上传", Description: "统一上传附件到对象存储并写回附件索引", ComponentType: "attachment", Enabled: true, Tags: []string{"file", "image", "video", "audio", "object-storage"}, Schema: map[string]interface{}{"type": "attachment", "label": "附件上传", "config": map[string]interface{}{"accept": []string{"image/*", "video/*", "audio/*", "application/pdf"}, "maxSizeMb": 200, "multiple": true}}},
	}
	for _, item := range templates {
		if _, err := upsertComponentTemplate(ctx, db, item); err != nil {
			return err
		}
	}
	questions := []domain.QuestionBankItem{
		{Category: "满意度", QuestionID: "overall_satisfaction", Label: "总体满意度", QuestionType: "single_select", Enabled: true, Tags: []string{"satisfaction", "score"}, Options: []map[string]string{{"label": "很不满意", "value": "1"}, {"label": "不满意", "value": "2"}, {"label": "一般", "value": "3"}, {"label": "满意", "value": "4"}, {"label": "非常满意", "value": "5"}}, ValidationRules: map[string]interface{}{"required": true}},
		{Category: "满意度", QuestionID: "recommend_score", Label: "推荐意愿", QuestionType: "single_select", Enabled: true, Tags: []string{"nps", "score"}, Options: []map[string]string{{"label": "0", "value": "0"}, {"label": "1", "value": "1"}, {"label": "2", "value": "2"}, {"label": "3", "value": "3"}, {"label": "4", "value": "4"}, {"label": "5", "value": "5"}, {"label": "6", "value": "6"}, {"label": "7", "value": "7"}, {"label": "8", "value": "8"}, {"label": "9", "value": "9"}, {"label": "10", "value": "10"}}},
	}
	for _, item := range questions {
		if _, err := upsertQuestionBankItem(ctx, db, item); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) RegisterFormSchema(ctx context.Context, form domain.Form, version domain.FormVersion) error {
	db, err := s.openConfiguredDB()
	if err != nil {
		return err
	}
	defer db.Close()
	schema, hash, err := normalizedSchema(version.Schema)
	if err != nil {
		return err
	}
	jsonSchema := buildFormJSONSchema(version.Schema)
	rawJSONSchema, err := json.Marshal(jsonSchema)
	if err != nil {
		return err
	}
	status := "draft"
	if version.Published || form.CurrentVersionID == version.ID {
		status = "published"
	}
	_, err = db.ExecContext(ctx, `
INSERT INTO form_schema_registry (id, form_id, version_id, schema_name, schema_hash, status, description, json_schema, created_by)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, NULLIF(?, ''))
ON DUPLICATE KEY UPDATE schema_name=VALUES(schema_name), schema_hash=VALUES(schema_hash), status=VALUES(status), description=VALUES(description), json_schema=VALUES(json_schema)`,
		uuid.NewString(), form.ID, version.ID, form.Name, firstNonEmptyStore(hash, checksumBytes(schema)), status, form.Description, string(rawJSONSchema), version.CreatedBy)
	return err
}

func (s *Store) FormSchemaRegistry(ctx context.Context, formID string) ([]domain.FormSchemaRegistryItem, error) {
	db, err := s.openConfiguredDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()
	query := `SELECT id, form_id, version_id, schema_name, schema_hash, status, COALESCE(description, ''), COALESCE(CAST(json_schema AS CHAR), '{}'), COALESCE(created_by, ''), created_at FROM form_schema_registry`
	args := []interface{}{}
	if strings.TrimSpace(formID) != "" {
		query += ` WHERE form_id = ?`
		args = append(args, formID)
	}
	query += ` ORDER BY created_at DESC`
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.FormSchemaRegistryItem{}
	for rows.Next() {
		var item domain.FormSchemaRegistryItem
		var raw string
		if err := rows.Scan(&item.ID, &item.FormID, &item.VersionID, &item.SchemaName, &item.SchemaHash, &item.Status, &item.Description, &raw, &item.CreatedBy, &item.CreatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(raw), &item.JSONSchema)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) ComponentTemplates(ctx context.Context) ([]domain.ComponentTemplate, error) {
	db, err := s.openConfiguredDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.QueryContext(ctx, `SELECT id, category, name, COALESCE(description, ''), component_type, CAST(schema_json AS CHAR), COALESCE(CAST(preview_json AS CHAR), '{}'), COALESCE(CAST(tags_json AS CHAR), '[]'), enabled, created_at, updated_at FROM component_templates ORDER BY category, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.ComponentTemplate{}
	for rows.Next() {
		var item domain.ComponentTemplate
		var schemaRaw, previewRaw, tagsRaw string
		if err := rows.Scan(&item.ID, &item.Category, &item.Name, &item.Description, &item.ComponentType, &schemaRaw, &previewRaw, &tagsRaw, &item.Enabled, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(schemaRaw), &item.Schema)
		_ = json.Unmarshal([]byte(previewRaw), &item.Preview)
		_ = json.Unmarshal([]byte(tagsRaw), &item.Tags)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) UpsertComponentTemplate(ctx context.Context, item domain.ComponentTemplate) (domain.ComponentTemplate, error) {
	db, err := s.openConfiguredDB()
	if err != nil {
		return domain.ComponentTemplate{}, err
	}
	defer db.Close()
	return upsertComponentTemplate(ctx, db, item)
}

func upsertComponentTemplate(ctx context.Context, db *sql.DB, item domain.ComponentTemplate) (domain.ComponentTemplate, error) {
	if strings.TrimSpace(item.ID) == "" {
		item.ID = uuid.NewString()
	}
	if strings.TrimSpace(item.Category) == "" {
		item.Category = "通用"
	}
	if strings.TrimSpace(item.ComponentType) == "" {
		item.ComponentType = "text"
	}
	schemaRaw, _ := json.Marshal(nonNilMap(item.Schema))
	previewRaw, _ := json.Marshal(nonNilMap(item.Preview))
	tagsRaw, _ := json.Marshal(item.Tags)
	if string(tagsRaw) == "null" {
		tagsRaw = []byte("[]")
	}
	_, err := db.ExecContext(ctx, `
INSERT INTO component_templates (id, category, name, description, component_type, schema_json, preview_json, tags_json, enabled)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE category=VALUES(category), name=VALUES(name), description=VALUES(description), component_type=VALUES(component_type), schema_json=VALUES(schema_json), preview_json=VALUES(preview_json), tags_json=VALUES(tags_json), enabled=VALUES(enabled)`,
		item.ID, item.Category, item.Name, item.Description, item.ComponentType, string(schemaRaw), string(previewRaw), string(tagsRaw), item.Enabled)
	return item, err
}

func (s *Store) QuestionBankItems(ctx context.Context, category string) ([]domain.QuestionBankItem, error) {
	db, err := s.openConfiguredDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()
	query := `SELECT id, category, question_id, label, question_type, COALESCE(CAST(options_json AS CHAR), '[]'), COALESCE(CAST(validation_json AS CHAR), '{}'), COALESCE(CAST(tags_json AS CHAR), '[]'), enabled, created_at, updated_at FROM question_bank_items`
	args := []interface{}{}
	if strings.TrimSpace(category) != "" {
		query += ` WHERE category = ?`
		args = append(args, category)
	}
	query += ` ORDER BY category, label`
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.QuestionBankItem{}
	for rows.Next() {
		var item domain.QuestionBankItem
		var optionsRaw, validationRaw, tagsRaw string
		if err := rows.Scan(&item.ID, &item.Category, &item.QuestionID, &item.Label, &item.QuestionType, &optionsRaw, &validationRaw, &tagsRaw, &item.Enabled, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(optionsRaw), &item.Options)
		_ = json.Unmarshal([]byte(validationRaw), &item.ValidationRules)
		_ = json.Unmarshal([]byte(tagsRaw), &item.Tags)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) UpsertQuestionBankItem(ctx context.Context, item domain.QuestionBankItem) (domain.QuestionBankItem, error) {
	db, err := s.openConfiguredDB()
	if err != nil {
		return domain.QuestionBankItem{}, err
	}
	defer db.Close()
	return upsertQuestionBankItem(ctx, db, item)
}

func upsertQuestionBankItem(ctx context.Context, db *sql.DB, item domain.QuestionBankItem) (domain.QuestionBankItem, error) {
	if strings.TrimSpace(item.ID) == "" {
		item.ID = uuid.NewString()
	}
	if strings.TrimSpace(item.Category) == "" {
		item.Category = "通用"
	}
	if strings.TrimSpace(item.QuestionType) == "" {
		item.QuestionType = "text"
	}
	optionsRaw, _ := json.Marshal(item.Options)
	validationRaw, _ := json.Marshal(nonNilMap(item.ValidationRules))
	tagsRaw, _ := json.Marshal(item.Tags)
	if string(optionsRaw) == "null" {
		optionsRaw = []byte("[]")
	}
	if string(tagsRaw) == "null" {
		tagsRaw = []byte("[]")
	}
	_, err := db.ExecContext(ctx, `
INSERT INTO question_bank_items (id, category, question_id, label, question_type, options_json, validation_json, tags_json, enabled)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE label=VALUES(label), question_type=VALUES(question_type), options_json=VALUES(options_json), validation_json=VALUES(validation_json), tags_json=VALUES(tags_json), enabled=VALUES(enabled)`,
		item.ID, item.Category, item.QuestionID, item.Label, item.QuestionType, string(optionsRaw), string(validationRaw), string(tagsRaw), item.Enabled)
	return item, err
}

func (s *Store) CreateFormAttachment(ctx context.Context, item domain.FormAttachment) (domain.FormAttachment, error) {
	db, err := s.openConfiguredDB()
	if err != nil {
		return domain.FormAttachment{}, err
	}
	defer db.Close()
	if item.ID == "" {
		item.ID = uuid.NewString()
	}
	if item.FileKind == "" {
		item.FileKind = classifyFileKind(item.MimeType)
	}
	metadataRaw, _ := json.Marshal(nonNilMap(item.Metadata))
	item.CreatedAt = time.Now().UTC()
	_, err = db.ExecContext(ctx, `
INSERT INTO form_attachments (id, submission_id, form_id, form_version_id, component_id, file_name, mime_type, file_kind, size_bytes, storage_config_id, storage_uri, object_name, checksum, metadata_json, created_by, created_at)
VALUES (?, NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), ?, ?, ?, ?, ?, NULLIF(?, ''), ?, NULLIF(?, ''), NULLIF(?, ''), ?, NULLIF(?, ''), ?)`,
		item.ID, item.SubmissionID, item.FormID, item.FormVersionID, item.ComponentID, item.FileName, item.MimeType, item.FileKind, item.SizeBytes, item.StorageConfigID, item.StorageURI, item.ObjectName, item.Checksum, string(metadataRaw), item.CreatedBy, item.CreatedAt)
	return item, err
}

func (s *Store) FormAttachments(ctx context.Context, submissionID string) ([]domain.FormAttachment, error) {
	db, err := s.openConfiguredDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()
	query := `SELECT id, COALESCE(submission_id, ''), COALESCE(form_id, ''), COALESCE(form_version_id, ''), component_id, file_name, mime_type, file_kind, size_bytes, COALESCE(storage_config_id, ''), storage_uri, COALESCE(object_name, ''), COALESCE(checksum, ''), COALESCE(CAST(metadata_json AS CHAR), '{}'), COALESCE(created_by, ''), created_at FROM form_attachments`
	args := []interface{}{}
	if strings.TrimSpace(submissionID) != "" {
		query += ` WHERE submission_id = ?`
		args = append(args, submissionID)
	}
	query += ` ORDER BY created_at DESC`
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.FormAttachment{}
	for rows.Next() {
		var item domain.FormAttachment
		var metadataRaw string
		if err := rows.Scan(&item.ID, &item.SubmissionID, &item.FormID, &item.FormVersionID, &item.ComponentID, &item.FileName, &item.MimeType, &item.FileKind, &item.SizeBytes, &item.StorageConfigID, &item.StorageURI, &item.ObjectName, &item.Checksum, &metadataRaw, &item.CreatedBy, &item.CreatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(metadataRaw), &item.Metadata)
		items = append(items, item)
	}
	return items, rows.Err()
}

func buildFormJSONSchema(components []domain.FormComponent) map[string]interface{} {
	properties := map[string]interface{}{}
	required := []string{}
	for _, component := range components {
		if component.Type == "section" || component.Type == "static_text" {
			continue
		}
		field := map[string]interface{}{"title": component.Label, "x-component": component.Type}
		switch component.Type {
		case "number", "rating":
			field["type"] = "number"
		case "multi_select", "attachment", "table":
			field["type"] = "array"
		case "matrix":
			field["type"] = "object"
		default:
			field["type"] = "string"
		}
		properties[component.ID] = field
		if component.Required {
			required = append(required, component.ID)
		}
	}
	return map[string]interface{}{"type": "object", "properties": properties, "required": required}
}

func checksumBytes(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func classifyFileKind(mimeType string) string {
	switch {
	case strings.HasPrefix(mimeType, "image/"):
		return "image"
	case strings.HasPrefix(mimeType, "video/"):
		return "video"
	case strings.HasPrefix(mimeType, "audio/"):
		return "audio"
	default:
		return "file"
	}
}

func nonNilMap(value map[string]interface{}) map[string]interface{} {
	if value == nil {
		return map[string]interface{}{}
	}
	return value
}
