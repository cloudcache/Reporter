package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"reporter/internal/domain"
)

func (s *Store) Departments() []domain.Department {
	items, _ := s.DepartmentsStrict(context.Background())
	return items
}

func (s *Store) DepartmentsStrict(ctx context.Context) ([]domain.Department, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	items, err := queryDepartments(ctx, db)
	if err != nil {
		return nil, err
	}
	result := make([]domain.Department, 0, len(items))
	for _, item := range items {
		result = append(result, item)
	}
	return result, nil
}

func (s *Store) CreateDepartment(ctx context.Context, item domain.Department) (domain.Department, error) {
	if s.dbDSN != "" {
		db, err := s.surveyDB(ctx)
		if err != nil {
			return domain.Department{}, err
		}
		defer db.Close()
		if item.ID == "" {
			item.ID = "DEPT-" + strings.ToUpper(strings.TrimSpace(item.Code))
		}
		if item.Kind == "" {
			item.Kind = "clinical"
		}
		if item.Status == "" {
			item.Status = "active"
		}
		if _, err := db.ExecContext(ctx, `INSERT INTO departments (id, code, name, kind, status) VALUES (?, ?, ?, ?, ?)`,
			item.ID, item.Code, item.Name, item.Kind, item.Status); err != nil {
			return domain.Department{}, err
		}
		return queryDepartmentByID(ctx, db, item.ID)
	}
	return domain.Department{}, errors.New("database dsn required")
}

func (s *Store) UpdateDepartment(ctx context.Context, id string, patch domain.Department) (domain.Department, error) {
	if s.dbDSN != "" {
		db, err := s.surveyDB(ctx)
		if err != nil {
			return domain.Department{}, err
		}
		defer db.Close()
		result, err := db.ExecContext(ctx, `UPDATE departments SET code = ?, name = ?, kind = ?, status = ? WHERE id = ?`,
			patch.Code, patch.Name, firstNonEmptyStore(patch.Kind, "clinical"), firstNonEmptyStore(patch.Status, "active"), id)
		if err != nil {
			return domain.Department{}, err
		}
		affected, _ := result.RowsAffected()
		if affected == 0 {
			return domain.Department{}, ErrNotFound
		}
		return queryDepartmentByID(ctx, db, id)
	}
	return domain.Department{}, errors.New("database dsn required")
}

func (s *Store) DeleteDepartment(ctx context.Context, id string) (domain.Department, error) {
	if s.dbDSN != "" {
		db, err := s.surveyDB(ctx)
		if err != nil {
			return domain.Department{}, err
		}
		defer db.Close()
		before, err := queryDepartmentByID(ctx, db, id)
		if err != nil {
			return domain.Department{}, err
		}
		if _, err := db.ExecContext(ctx, `DELETE FROM departments WHERE id = ?`, id); err != nil {
			return domain.Department{}, err
		}
		return before, nil
	}
	return domain.Department{}, errors.New("database dsn required")
}

func (s *Store) Dictionaries() []domain.Dictionary {
	db, err := s.surveyDB(context.Background())
	if err != nil {
		return nil
	}
	defer db.Close()
	items, err := queryDictionaries(context.Background(), db)
	if err != nil {
		return nil
	}
	result := make([]domain.Dictionary, 0, len(items))
	for _, item := range items {
		result = append(result, item)
	}
	return result
}

func (s *Store) CreateDictionary(item domain.Dictionary) domain.Dictionary {
	db, err := s.surveyDB(context.Background())
	if err != nil {
		return domain.Dictionary{}
	}
	defer db.Close()
	if item.ID == "" {
		item.ID = uuid.NewString()
	}
	raw, err := json.Marshal(item.Items)
	if err != nil {
		return domain.Dictionary{}
	}
	if _, err := db.ExecContext(context.Background(), `INSERT INTO dictionaries (id, code, name, category, description, items_json) VALUES (?, ?, ?, ?, ?, ?)`, item.ID, item.Code, item.Name, item.Category, nullableString(item.Description), string(raw)); err != nil {
		return domain.Dictionary{}
	}
	return item
}

func (s *Store) UpdateDictionary(id string, patch domain.Dictionary) (domain.Dictionary, error) {
	db, err := s.surveyDB(context.Background())
	if err != nil {
		return domain.Dictionary{}, err
	}
	defer db.Close()
	raw, err := json.Marshal(patch.Items)
	if err != nil {
		return domain.Dictionary{}, err
	}
	result, err := db.ExecContext(context.Background(), `UPDATE dictionaries SET code = ?, name = ?, category = ?, description = ?, items_json = ? WHERE id = ?`, patch.Code, patch.Name, patch.Category, nullableString(patch.Description), string(raw), id)
	if err != nil {
		return domain.Dictionary{}, err
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return domain.Dictionary{}, ErrNotFound
	}
	items, err := queryDictionaries(context.Background(), db)
	if err != nil {
		return domain.Dictionary{}, err
	}
	return items[id], nil
}

func (s *Store) LoadFollowupConfigFromSQL(ctx context.Context, driver, dsn string) error {
	if strings.TrimSpace(dsn) == "" {
		return nil
	}
	if strings.TrimSpace(driver) == "" {
		driver = "mysql"
	}
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := ensureFollowupTables(ctx, db); err != nil {
		return err
	}
	if err := seedFollowupConfig(ctx, db); err != nil {
		return err
	}
	return nil
}

func ensureFollowupTables(ctx context.Context, db *sql.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS departments (
  id VARCHAR(80) PRIMARY KEY,
  code VARCHAR(80) NOT NULL UNIQUE,
  name VARCHAR(180) NOT NULL,
  kind VARCHAR(60) NOT NULL DEFAULT 'clinical',
  status VARCHAR(40) NOT NULL DEFAULT 'active',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
)`,
		`CREATE TABLE IF NOT EXISTS dictionaries (
  id VARCHAR(80) PRIMARY KEY,
  code VARCHAR(120) NOT NULL UNIQUE,
  name VARCHAR(180) NOT NULL,
  category VARCHAR(120) NOT NULL,
  description TEXT NULL,
  items_json JSON NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
)`,
		`CREATE TABLE IF NOT EXISTS followup_plans (
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
)`,
		`CREATE TABLE IF NOT EXISTS followup_tasks (
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
)`,
	}
	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}

func seedFollowupConfig(ctx context.Context, db *sql.DB) error {
	now := time.Now().UTC()
	departments := []domain.Department{
		{ID: "DEPT-CARD", Code: "CARD", Name: "心内科", Kind: "clinical", Status: "active", CreatedAt: now, UpdatedAt: now},
		{ID: "DEPT-ENDO", Code: "ENDO", Name: "内分泌科", Kind: "clinical", Status: "active", CreatedAt: now, UpdatedAt: now},
	}
	for _, item := range departments {
		if _, err := db.ExecContext(ctx, `
INSERT INTO departments (id, code, name, kind, status)
VALUES (?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE name = VALUES(name), kind = VALUES(kind), status = VALUES(status)`,
			item.ID, item.Code, item.Name, item.Kind, item.Status,
		); err != nil {
			return err
		}
	}
	for _, item := range DefaultDictionaries() {
		raw, err := json.Marshal(item.Items)
		if err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, `
INSERT INTO dictionaries (id, code, name, category, description, items_json)
VALUES (?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE name = VALUES(name), category = VALUES(category), description = VALUES(description), items_json = VALUES(items_json)`,
			item.ID, item.Code, item.Name, item.Category, item.Description, string(raw),
		); err != nil {
			return err
		}
	}
	plans := []domain.FollowupPlan{
		{ID: "PLAN-HTN", Name: "高血压慢病随访", Scenario: "慢病", DiseaseCode: "I10", DepartmentID: "DEPT-CARD", FormTemplateID: "hypertension-follow-up", TriggerType: "定期", TriggerOffset: 30, Channel: "phone", AssigneeRole: "agent", Status: "active", Rules: map[string]interface{}{"ageMin": 45, "diagnosis": "高血压"}},
		{ID: "PLAN-DISCHARGE", Name: "出院后 7 日随访", Scenario: "随访", FormTemplateID: "discharge-follow-up", TriggerType: "出院后", TriggerOffset: 7, Channel: "phone", AssigneeRole: "nurse", Status: "active", Rules: map[string]interface{}{}},
	}
	for _, plan := range plans {
		raw, err := json.Marshal(plan.Rules)
		if err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, `
INSERT INTO followup_plans (id, name, scenario, disease_code, department_id, form_template_id, trigger_type, trigger_offset, channel, assignee_role, status, rules_json)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE name = VALUES(name), scenario = VALUES(scenario), disease_code = VALUES(disease_code), department_id = VALUES(department_id), form_template_id = VALUES(form_template_id), trigger_type = VALUES(trigger_type), trigger_offset = VALUES(trigger_offset), channel = VALUES(channel), assignee_role = VALUES(assignee_role), status = VALUES(status), rules_json = VALUES(rules_json)`,
			plan.ID, plan.Name, plan.Scenario, plan.DiseaseCode, plan.DepartmentID, plan.FormTemplateID, plan.TriggerType, plan.TriggerOffset, plan.Channel, plan.AssigneeRole, plan.Status, string(raw),
		); err != nil {
			return err
		}
	}
	return nil
}

func DefaultDictionaries() []domain.Dictionary {
	return []domain.Dictionary{
		{ID: "DICT-GENDER", Code: "gender", Name: "性别字典", Category: "患者基础", Items: []domain.DictionaryEntry{{Key: "M", Label: "男", Value: "男"}, {Key: "F", Label: "女", Value: "女"}, {Key: "O", Label: "其他", Value: "其他"}}},
		{ID: "DICT-FOLLOWUP-STATUS", Code: "followup_status", Name: "随访任务状态", Category: "随访中心", Items: []domain.DictionaryEntry{{Key: "pending", Label: "待随访", Value: "pending"}, {Key: "assigned", Label: "已分配", Value: "assigned"}, {Key: "in_progress", Label: "进行中", Value: "in_progress"}, {Key: "completed", Label: "已完成", Value: "completed"}, {Key: "failed", Label: "失败", Value: "failed"}}},
		{ID: "DICT-EMR-FIELDS", Code: "emr_common_fields", Name: "电子病历常用字段", Category: "电子病历", Description: "门诊、住院、专科病历同步和表单映射常用字段", Items: []domain.DictionaryEntry{
			{Key: "record_no", Label: "病历号", Value: "record_no"},
			{Key: "record_type", Label: "病历类型", Value: "record_type"},
			{Key: "record_title", Label: "病历标题", Value: "record_title"},
			{Key: "chief_complaint", Label: "主诉", Value: "chief_complaint"},
			{Key: "present_illness", Label: "现病史", Value: "present_illness"},
			{Key: "past_history", Label: "既往史", Value: "past_history"},
			{Key: "personal_history", Label: "个人史", Value: "personal_history"},
			{Key: "allergy_history", Label: "过敏史", Value: "allergy_history"},
			{Key: "physical_exam", Label: "体格检查", Value: "physical_exam"},
			{Key: "specialist_exam", Label: "专科检查", Value: "specialist_exam"},
			{Key: "auxiliary_exam", Label: "辅助检查", Value: "auxiliary_exam"},
			{Key: "diagnosis_code", Label: "诊断编码", Value: "diagnosis_code"},
			{Key: "diagnosis_name", Label: "诊断名称", Value: "diagnosis_name"},
			{Key: "treatment_plan", Label: "诊疗计划", Value: "treatment_plan"},
			{Key: "doctor_advice", Label: "医嘱", Value: "doctor_advice"},
			{Key: "recorded_at", Label: "记录时间", Value: "recorded_at"},
			{Key: "record_doctor", Label: "记录医生", Value: "record_doctor"},
			{Key: "department_code", Label: "科室编码", Value: "department_code"},
			{Key: "department_name", Label: "科室名称", Value: "department_name"},
			{Key: "source_system", Label: "来源系统", Value: "source_system"},
		}},
		{ID: "DICT-CASE-FIELDS", Code: "case_common_fields", Name: "病例常用字段", Category: "病例管理", Description: "病例建档、科研队列、病案首页和随访筛选常用字段", Items: []domain.DictionaryEntry{
			{Key: "case_no", Label: "病例号", Value: "case_no"},
			{Key: "patient_no", Label: "档案号", Value: "patient_no"},
			{Key: "patient_name", Label: "患者姓名", Value: "patient_name"},
			{Key: "gender", Label: "性别", Value: "gender"},
			{Key: "age", Label: "年龄", Value: "age"},
			{Key: "id_card_no", Label: "身份证号", Value: "id_card_no"},
			{Key: "phone", Label: "联系电话", Value: "phone"},
			{Key: "case_source", Label: "病例来源", Value: "case_source"},
			{Key: "disease_code", Label: "病种编码", Value: "disease_code"},
			{Key: "disease_name", Label: "病种名称", Value: "disease_name"},
			{Key: "primary_diagnosis_code", Label: "主要诊断编码", Value: "primary_diagnosis_code"},
			{Key: "primary_diagnosis_name", Label: "主要诊断名称", Value: "primary_diagnosis_name"},
			{Key: "tumor_stage", Label: "肿瘤分期", Value: "tumor_stage"},
			{Key: "pathology_no", Label: "病理号", Value: "pathology_no"},
			{Key: "pathology_diagnosis", Label: "病理诊断", Value: "pathology_diagnosis"},
			{Key: "operation_name", Label: "手术名称", Value: "operation_name"},
			{Key: "operation_date", Label: "手术日期", Value: "operation_date"},
			{Key: "discharge_status", Label: "出院情况", Value: "discharge_status"},
			{Key: "followup_flag", Label: "随访标识", Value: "followup_flag"},
			{Key: "case_created_at", Label: "建档时间", Value: "case_created_at"},
		}},
		{ID: "DICT-VISIT-FIELDS", Code: "visit_common_fields", Name: "就诊常用字段", Category: "就诊信息", Description: "门诊、急诊、住院、出院记录同步常用字段", Items: []domain.DictionaryEntry{
			{Key: "visit_no", Label: "就诊号", Value: "visit_no"},
			{Key: "visit_type", Label: "就诊类型", Value: "visit_type"},
			{Key: "outpatient_no", Label: "门诊号", Value: "outpatient_no"},
			{Key: "inpatient_no", Label: "住院号", Value: "inpatient_no"},
			{Key: "admission_no", Label: "入院登记号", Value: "admission_no"},
			{Key: "visit_at", Label: "就诊时间", Value: "visit_at"},
			{Key: "admission_at", Label: "入院时间", Value: "admission_at"},
			{Key: "discharge_at", Label: "出院时间", Value: "discharge_at"},
			{Key: "department_code", Label: "就诊科室编码", Value: "department_code"},
			{Key: "department_name", Label: "就诊科室", Value: "department_name"},
			{Key: "ward_name", Label: "病区", Value: "ward_name"},
			{Key: "bed_no", Label: "床号", Value: "bed_no"},
			{Key: "attending_doctor", Label: "主治医生", Value: "attending_doctor"},
			{Key: "responsible_nurse", Label: "责任护士", Value: "responsible_nurse"},
			{Key: "diagnosis_code", Label: "就诊诊断编码", Value: "diagnosis_code"},
			{Key: "diagnosis_name", Label: "就诊诊断", Value: "diagnosis_name"},
			{Key: "visit_status", Label: "就诊状态", Value: "visit_status"},
			{Key: "discharge_disposition", Label: "离院方式", Value: "discharge_disposition"},
			{Key: "total_fee", Label: "总费用", Value: "total_fee"},
			{Key: "insurance_type", Label: "医保类型", Value: "insurance_type"},
		}},
		{ID: "DICT-MEDICATION-FIELDS", Code: "medication_common_fields", Name: "用药常用字段", Category: "用药信息", Description: "处方、医嘱、用药随访和不良反应采集常用字段", Items: []domain.DictionaryEntry{
			{Key: "order_no", Label: "医嘱号", Value: "order_no"},
			{Key: "prescription_no", Label: "处方号", Value: "prescription_no"},
			{Key: "drug_code", Label: "药品编码", Value: "drug_code"},
			{Key: "drug_name", Label: "药品名称", Value: "drug_name"},
			{Key: "generic_name", Label: "通用名", Value: "generic_name"},
			{Key: "specification", Label: "规格", Value: "specification"},
			{Key: "dosage", Label: "单次剂量", Value: "dosage"},
			{Key: "dosage_unit", Label: "剂量单位", Value: "dosage_unit"},
			{Key: "frequency", Label: "用药频次", Value: "frequency"},
			{Key: "route", Label: "给药途径", Value: "route"},
			{Key: "start_at", Label: "开始时间", Value: "start_at"},
			{Key: "end_at", Label: "结束时间", Value: "end_at"},
			{Key: "days", Label: "用药天数", Value: "days"},
			{Key: "quantity", Label: "数量", Value: "quantity"},
			{Key: "manufacturer", Label: "生产厂家", Value: "manufacturer"},
			{Key: "doctor_name", Label: "开立医生", Value: "doctor_name"},
			{Key: "pharmacist_name", Label: "审核药师", Value: "pharmacist_name"},
			{Key: "medication_status", Label: "用药状态", Value: "medication_status"},
			{Key: "adverse_reaction", Label: "不良反应", Value: "adverse_reaction"},
			{Key: "compliance", Label: "用药依从性", Value: "compliance"},
		}},
	}
}

func queryDepartments(ctx context.Context, db *sql.DB) (map[string]domain.Department, error) {
	rows, err := db.QueryContext(ctx, `SELECT id, code, name, kind, status, created_at, updated_at FROM departments ORDER BY code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := map[string]domain.Department{}
	for rows.Next() {
		var item domain.Department
		if err := rows.Scan(&item.ID, &item.Code, &item.Name, &item.Kind, &item.Status, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items[item.ID] = item
	}
	return items, rows.Err()
}

func queryDepartmentByID(ctx context.Context, db *sql.DB, id string) (domain.Department, error) {
	var item domain.Department
	err := db.QueryRowContext(ctx, `SELECT id, code, name, kind, status, created_at, updated_at FROM departments WHERE id = ?`, id).
		Scan(&item.ID, &item.Code, &item.Name, &item.Kind, &item.Status, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Department{}, ErrNotFound
	}
	return item, err
}

func queryDictionaries(ctx context.Context, db *sql.DB) (map[string]domain.Dictionary, error) {
	rows, err := db.QueryContext(ctx, `SELECT id, code, name, category, COALESCE(description, ''), CAST(items_json AS CHAR), created_at, updated_at FROM dictionaries ORDER BY category, code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := map[string]domain.Dictionary{}
	for rows.Next() {
		var item domain.Dictionary
		var raw string
		if err := rows.Scan(&item.ID, &item.Code, &item.Name, &item.Category, &item.Description, &raw, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(raw), &item.Items); err != nil {
			return nil, err
		}
		items[item.ID] = item
	}
	return items, rows.Err()
}

func queryFollowupPlans(ctx context.Context, db *sql.DB) (map[string]domain.FollowupPlan, error) {
	rows, err := db.QueryContext(ctx, `
SELECT id, name, scenario, COALESCE(disease_code, ''), COALESCE(department_id, ''), form_template_id,
       trigger_type, trigger_offset, channel, assignee_role, status, COALESCE(CAST(rules_json AS CHAR), '{}'), created_at, updated_at
FROM followup_plans
ORDER BY updated_at DESC, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := map[string]domain.FollowupPlan{}
	for rows.Next() {
		var item domain.FollowupPlan
		var raw string
		if err := rows.Scan(&item.ID, &item.Name, &item.Scenario, &item.DiseaseCode, &item.DepartmentID, &item.FormTemplateID, &item.TriggerType, &item.TriggerOffset, &item.Channel, &item.AssigneeRole, &item.Status, &raw, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(raw), &item.Rules); err != nil {
			return nil, err
		}
		items[item.ID] = item
	}
	return items, rows.Err()
}

func queryFollowupTasks(ctx context.Context, db *sql.DB) (map[string]domain.FollowupTask, error) {
	rows, err := db.QueryContext(ctx, `
SELECT t.id, COALESCE(t.plan_id, ''), t.patient_id, COALESCE(t.visit_id, ''), COALESCE(t.form_id, ''),
       COALESCE(t.form_template_id, ''), COALESCE(t.assignee_id, ''), COALESCE(t.role, ''), t.channel, t.status, t.priority,
       COALESCE(DATE_FORMAT(t.due_at, '%Y-%m-%d'), ''), COALESCE(CAST(t.result_json AS CHAR), '{}'), COALESCE(t.last_event, ''),
       t.created_at, t.updated_at, COALESCE(p.name, ''), COALESCE(p.phone, ''), COALESCE(u.display_name, '')
FROM followup_tasks t
LEFT JOIN patients p ON p.id = t.patient_id
LEFT JOIN users u ON u.id = t.assignee_id
ORDER BY t.due_at IS NULL, t.due_at, t.updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := map[string]domain.FollowupTask{}
	for rows.Next() {
		var item domain.FollowupTask
		var raw string
		if err := rows.Scan(&item.ID, &item.PlanID, &item.PatientID, &item.VisitID, &item.FormID, &item.FormTemplateID, &item.AssigneeID, &item.Role, &item.Channel, &item.Status, &item.Priority, &item.DueAt, &raw, &item.LastEvent, &item.CreatedAt, &item.UpdatedAt, &item.PatientName, &item.PatientPhone, &item.AssigneeName); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(raw), &item.Result); err != nil {
			return nil, err
		}
		items[item.ID] = item
	}
	return items, rows.Err()
}

func (s *Store) dbFollowupPlan(ctx context.Context, id string) (domain.FollowupPlan, bool, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return domain.FollowupPlan{}, false, err
	}
	defer db.Close()
	plans, err := queryFollowupPlans(ctx, db)
	if err != nil {
		return domain.FollowupPlan{}, false, err
	}
	plan, ok := plans[id]
	return plan, ok, nil
}

func (s *Store) dbFollowupTasks(ctx context.Context, status, assigneeID string) ([]domain.FollowupTask, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	tasks, err := queryFollowupTasks(ctx, db)
	if err != nil {
		return nil, err
	}
	items := make([]domain.FollowupTask, 0, len(tasks))
	for _, item := range tasks {
		if status != "" && item.Status != status {
			continue
		}
		if assigneeID != "" && item.AssigneeID != assigneeID {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *Store) dbFollowupTask(ctx context.Context, id string) (domain.FollowupTask, bool, error) {
	tasks, err := s.dbFollowupTasks(ctx, "", "")
	if err != nil {
		return domain.FollowupTask{}, false, err
	}
	for _, task := range tasks {
		if task.ID == id {
			return task, true, nil
		}
	}
	return domain.FollowupTask{}, false, nil
}

func (s *Store) dbCreateFollowupTask(ctx context.Context, task domain.FollowupTask) (domain.FollowupTask, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return domain.FollowupTask{}, err
	}
	defer db.Close()
	now := time.Now().UTC()
	if task.ID == "" {
		task.ID = uuid.NewString()
	}
	if task.Status == "" {
		task.Status = "pending"
	}
	if task.Priority == "" {
		task.Priority = "normal"
	}
	task.CreatedAt = now
	task.UpdatedAt = now
	raw, err := json.Marshal(task.Result)
	if err != nil {
		return domain.FollowupTask{}, err
	}
	if string(raw) == "null" {
		raw = []byte("{}")
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO followup_tasks (id, plan_id, patient_id, visit_id, form_id, form_template_id, assignee_id, role, channel, status, priority, due_at, result_json, last_event, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE plan_id = VALUES(plan_id), patient_id = VALUES(patient_id), visit_id = VALUES(visit_id), form_id = VALUES(form_id), form_template_id = VALUES(form_template_id), assignee_id = VALUES(assignee_id), role = VALUES(role), channel = VALUES(channel), status = VALUES(status), priority = VALUES(priority), due_at = VALUES(due_at), result_json = VALUES(result_json), last_event = VALUES(last_event), updated_at = VALUES(updated_at)`,
		task.ID, nullStringStore(task.PlanID), task.PatientID, nullStringStore(task.VisitID), nullStringStore(task.FormID), nullStringStore(task.FormTemplateID), nullStringStore(task.AssigneeID), nullStringStore(task.Role), task.Channel, task.Status, task.Priority, nullStringStore(task.DueAt), string(raw), nullStringStore(task.LastEvent), task.CreatedAt, task.UpdatedAt,
	); err != nil {
		return domain.FollowupTask{}, err
	}
	created, ok, err := s.dbFollowupTask(ctx, task.ID)
	if err != nil {
		return domain.FollowupTask{}, err
	}
	if !ok {
		return domain.FollowupTask{}, ErrNotFound
	}
	return created, nil
}

func (s *Store) dbUpdateFollowupTask(ctx context.Context, id string, patch domain.FollowupTask) (domain.FollowupTask, error) {
	task, ok, err := s.dbFollowupTask(ctx, id)
	if err != nil {
		return domain.FollowupTask{}, err
	}
	if !ok {
		return domain.FollowupTask{}, ErrNotFound
	}
	task.PlanID = firstNonEmptyStore(patch.PlanID, task.PlanID)
	task.PatientID = firstNonEmptyStore(patch.PatientID, task.PatientID)
	task.VisitID = patch.VisitID
	task.FormID = patch.FormID
	task.FormTemplateID = firstNonEmptyStore(patch.FormTemplateID, task.FormTemplateID)
	task.AssigneeID = patch.AssigneeID
	task.Role = patch.Role
	task.Channel = firstNonEmptyStore(patch.Channel, task.Channel)
	task.Status = firstNonEmptyStore(patch.Status, task.Status)
	task.Priority = firstNonEmptyStore(patch.Priority, task.Priority)
	task.DueAt = patch.DueAt
	task.Result = patch.Result
	task.LastEvent = patch.LastEvent
	task.UpdatedAt = time.Now().UTC()
	updated, err := s.dbCreateFollowupTask(ctx, task)
	if err != nil {
		return domain.FollowupTask{}, err
	}
	return updated, nil
}

func (s *Store) dbFollowupAssignees(ctx context.Context, role string) ([]domain.User, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	countRows, err := db.QueryContext(ctx, `
SELECT assignee_id, COUNT(*)
FROM followup_tasks
WHERE assignee_id IS NOT NULL AND assignee_id <> '' AND status NOT IN ('completed', 'failed')
GROUP BY assignee_id`)
	if err != nil {
		return nil, err
	}
	counts := map[string]int{}
	for countRows.Next() {
		var id string
		var count int
		if err := countRows.Scan(&id, &count); err != nil {
			countRows.Close()
			return nil, err
		}
		counts[id] = count
	}
	if err := countRows.Close(); err != nil {
		return nil, err
	}

	rows, err := db.QueryContext(ctx, `
SELECT DISTINCT u.id, u.username, u.display_name, u.password_hash, u.created_at, u.updated_at
FROM users u
JOIN user_roles ur ON ur.user_id = u.id
WHERE ur.role_id = ? OR (? = 'agent' AND ur.role_id = 'admin')`, role, role)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	users := []domain.User{}
	for rows.Next() {
		var user domain.User
		if err := rows.Scan(&user.ID, &user.Username, &user.DisplayName, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := 0; i < len(users); i++ {
		for j := i + 1; j < len(users); j++ {
			if counts[users[j].ID] < counts[users[i].ID] || (counts[users[j].ID] == counts[users[i].ID] && users[j].ID < users[i].ID) {
				users[i], users[j] = users[j], users[i]
			}
		}
	}
	return users, nil
}

func (s *Store) FormLibraryItem(id string) (domain.FormLibraryItem, bool) {
	item, ok, _ := s.FormLibraryItemStrict(context.Background(), id)
	return item, ok
}

func (s *Store) UpsertFormLibraryItem(item domain.FormLibraryItem) domain.FormLibraryItem {
	saved, _ := s.UpsertFormLibraryItemStrict(context.Background(), item)
	return saved
}

func (s *Store) DeleteFormLibraryItem(id string) (domain.FormLibraryItem, error) {
	return s.DeleteFormLibraryItemStrict(context.Background(), id)
}

func (s *Store) FollowupPlans() []domain.FollowupPlan {
	db, err := s.surveyDB(context.Background())
	if err != nil {
		return nil
	}
	defer db.Close()
	items, err := queryFollowupPlans(context.Background(), db)
	if err != nil {
		return nil
	}
	result := make([]domain.FollowupPlan, 0, len(items))
	for _, item := range items {
		result = append(result, item)
	}
	return result
}

func (s *Store) FollowupPlanByID(id string) (domain.FollowupPlan, bool) {
	item, ok, _ := s.dbFollowupPlan(context.Background(), id)
	return item, ok
}

func (s *Store) CreateFollowupPlan(plan domain.FollowupPlan) domain.FollowupPlan {
	db, err := s.surveyDB(context.Background())
	if err != nil {
		return domain.FollowupPlan{}
	}
	defer db.Close()
	if plan.ID == "" {
		plan.ID = uuid.NewString()
	}
	if plan.Status == "" {
		plan.Status = "active"
	}
	raw, err := json.Marshal(nonNilMap(plan.Rules))
	if err != nil {
		return domain.FollowupPlan{}
	}
	_, err = db.ExecContext(context.Background(), `INSERT INTO followup_plans (id, name, scenario, disease_code, department_id, form_template_id, trigger_type, trigger_offset, channel, assignee_role, status, rules_json) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		plan.ID, plan.Name, plan.Scenario, nullableString(plan.DiseaseCode), nullableString(plan.DepartmentID), plan.FormTemplateID, plan.TriggerType, plan.TriggerOffset, plan.Channel, plan.AssigneeRole, plan.Status, string(raw))
	if err != nil {
		return domain.FollowupPlan{}
	}
	saved, _, _ := s.dbFollowupPlan(context.Background(), plan.ID)
	return saved
}

func (s *Store) UpdateFollowupPlan(id string, patch domain.FollowupPlan) (domain.FollowupPlan, error) {
	db, err := s.surveyDB(context.Background())
	if err != nil {
		return domain.FollowupPlan{}, err
	}
	defer db.Close()
	raw, err := json.Marshal(nonNilMap(patch.Rules))
	if err != nil {
		return domain.FollowupPlan{}, err
	}
	result, err := db.ExecContext(context.Background(), `UPDATE followup_plans SET name = ?, scenario = ?, disease_code = ?, department_id = ?, form_template_id = ?, trigger_type = ?, trigger_offset = ?, channel = ?, assignee_role = ?, status = ?, rules_json = ? WHERE id = ?`,
		patch.Name, patch.Scenario, nullableString(patch.DiseaseCode), nullableString(patch.DepartmentID), patch.FormTemplateID, patch.TriggerType, patch.TriggerOffset, patch.Channel, patch.AssigneeRole, firstNonEmptyStore(patch.Status, "active"), string(raw), id)
	if err != nil {
		return domain.FollowupPlan{}, err
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return domain.FollowupPlan{}, ErrNotFound
	}
	saved, _, err := s.dbFollowupPlan(context.Background(), id)
	return saved, err
}

func (s *Store) FollowupTasks(status, assigneeID string) []domain.FollowupTask {
	tasks, err := s.FollowupTasksStrict(context.Background(), status, assigneeID)
	if err != nil {
		return nil
	}
	return tasks
}

func (s *Store) FollowupTasksStrict(ctx context.Context, status, assigneeID string) ([]domain.FollowupTask, error) {
	return s.dbFollowupTasks(ctx, status, assigneeID)
}

func (s *Store) CreateFollowupTask(task domain.FollowupTask) domain.FollowupTask {
	created, err := s.CreateFollowupTaskStrict(context.Background(), task)
	if err != nil {
		return domain.FollowupTask{}
	}
	return created
}

func (s *Store) CreateFollowupTaskStrict(ctx context.Context, task domain.FollowupTask) (domain.FollowupTask, error) {
	return s.dbCreateFollowupTask(ctx, task)
}

func (s *Store) UpdateFollowupTask(id string, patch domain.FollowupTask) (domain.FollowupTask, error) {
	return s.UpdateFollowupTaskStrict(context.Background(), id, patch)
}

func (s *Store) UpdateFollowupTaskStrict(ctx context.Context, id string, patch domain.FollowupTask) (domain.FollowupTask, error) {
	return s.dbUpdateFollowupTask(ctx, id, patch)
}

func (s *Store) GenerateFollowupTasks(planID string) ([]domain.FollowupTask, error) {
	return s.GenerateFollowupTasksStrict(context.Background(), planID)
}

func (s *Store) GenerateFollowupTasksStrict(ctx context.Context, planID string) ([]domain.FollowupTask, error) {
	plan, ok, err := s.dbFollowupPlan(ctx, planID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotFound
	}
	patients, err := s.PatientsStrict(ctx, "")
	if err != nil {
		return nil, err
	}
	assignees, err := s.dbFollowupAssignees(ctx, plan.AssigneeRole)
	if err != nil {
		return nil, err
	}
	targets := make([]domain.Patient, 0, len(patients))
	for _, patient := range patients {
		if plan.DiseaseCode != "" && !strings.Contains(patient.Diagnosis, "高血压") && plan.DiseaseCode == "I10" {
			continue
		}
		targets = append(targets, patient)
	}

	tasks := make([]domain.FollowupTask, 0, len(targets))
	for index, patient := range targets {
		assigneeID := ""
		if len(assignees) > 0 {
			assigneeID = assignees[index%len(assignees)].ID
		}
		task, err := s.CreateFollowupTaskStrict(ctx, domain.FollowupTask{
			PlanID:         plan.ID,
			PatientID:      patient.ID,
			FormTemplateID: plan.FormTemplateID,
			AssigneeID:     assigneeID,
			Role:           plan.AssigneeRole,
			Channel:        plan.Channel,
			Status:         firstNonEmptyStore(map[bool]string{true: "assigned", false: "pending"}[assigneeID != ""]),
			Priority:       "normal",
			DueAt:          time.Now().AddDate(0, 0, plan.TriggerOffset).Format("2006-01-02"),
			LastEvent:      "按随访方案批量生成，采用最少任务优先的轮询分配",
		})
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}

func firstNonEmptyStore(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func nullStringStore(value string) sql.NullString {
	if strings.TrimSpace(value) == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value, Valid: true}
}
