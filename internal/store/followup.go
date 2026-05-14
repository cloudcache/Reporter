package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"

	"reporter/internal/domain"
)

func (s *MemoryStore) Departments() []domain.Department {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]domain.Department, 0, len(s.departments))
	for _, item := range s.departments {
		items = append(items, item)
	}
	return items
}

func (s *MemoryStore) Dictionaries() []domain.Dictionary {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]domain.Dictionary, 0, len(s.dictionaries))
	for _, item := range s.dictionaries {
		items = append(items, item)
	}
	return items
}

func (s *MemoryStore) CreateDictionary(item domain.Dictionary) domain.Dictionary {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	if item.ID == "" {
		item.ID = uuid.NewString()
	}
	item.CreatedAt = now
	item.UpdatedAt = now
	s.dictionaries[item.ID] = item
	return item
}

func (s *MemoryStore) UpdateDictionary(id string, patch domain.Dictionary) (domain.Dictionary, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.dictionaries[id]
	if !ok {
		return domain.Dictionary{}, ErrNotFound
	}
	item.Code = firstNonEmptyStore(patch.Code, item.Code)
	item.Name = firstNonEmptyStore(patch.Name, item.Name)
	item.Category = firstNonEmptyStore(patch.Category, item.Category)
	item.Description = patch.Description
	if patch.Items != nil {
		item.Items = patch.Items
	}
	item.UpdatedAt = time.Now().UTC()
	s.dictionaries[id] = item
	return item, nil
}

func (s *MemoryStore) LoadFollowupConfigFromSQL(ctx context.Context, driver, dsn string) error {
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
	departments, err := queryDepartments(ctx, db)
	if err != nil {
		return err
	}
	dictionaries, err := queryDictionaries(ctx, db)
	if err != nil {
		return err
	}
	plans, err := queryFollowupPlans(ctx, db)
	if err != nil {
		return err
	}
	tasks, err := queryFollowupTasks(ctx, db)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(departments) > 0 {
		s.departments = departments
	}
	if len(dictionaries) > 0 {
		s.dictionaries = dictionaries
	}
	if len(plans) > 0 {
		s.followPlans = plans
	}
	if len(tasks) > 0 {
		for id, task := range tasks {
			if patient, ok := s.patients[task.PatientID]; ok {
				task.PatientName = patient.Name
				task.PatientPhone = patient.Phone
			}
			if user, ok := s.users[task.AssigneeID]; ok {
				task.AssigneeName = user.DisplayName
			}
			tasks[id] = task
		}
		s.followTasks = tasks
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
VALUES (?, ?, ?, ?, ?, CAST(? AS JSON))
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
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CAST(? AS JSON))
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
SELECT id, COALESCE(plan_id, ''), patient_id, COALESCE(visit_id, ''), COALESCE(form_id, ''),
       COALESCE(form_template_id, ''), COALESCE(assignee_id, ''), COALESCE(role, ''), channel, status, priority,
       COALESCE(DATE_FORMAT(due_at, '%Y-%m-%d'), ''), COALESCE(CAST(result_json AS CHAR), '{}'), COALESCE(last_event, ''), created_at, updated_at
FROM followup_tasks
ORDER BY due_at IS NULL, due_at, updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := map[string]domain.FollowupTask{}
	for rows.Next() {
		var item domain.FollowupTask
		var raw string
		if err := rows.Scan(&item.ID, &item.PlanID, &item.PatientID, &item.VisitID, &item.FormID, &item.FormTemplateID, &item.AssigneeID, &item.Role, &item.Channel, &item.Status, &item.Priority, &item.DueAt, &raw, &item.LastEvent, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(raw), &item.Result); err != nil {
			return nil, err
		}
		items[item.ID] = item
	}
	return items, rows.Err()
}

func (s *MemoryStore) FormLibraryItem(id string) (domain.FormLibraryItem, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, item := range s.formLibrary {
		if item.ID == id {
			return item, true
		}
	}
	return domain.FormLibraryItem{}, false
}

func (s *MemoryStore) UpsertFormLibraryItem(item domain.FormLibraryItem) domain.FormLibraryItem {
	s.mu.Lock()
	defer s.mu.Unlock()
	if item.ID == "" {
		item.ID = uuid.NewString()
	}
	if item.Enabled == false {
		item.Enabled = true
	}
	for index, existing := range s.formLibrary {
		if existing.ID == item.ID {
			s.formLibrary[index] = item
			return item
		}
	}
	s.formLibrary = append(s.formLibrary, item)
	return item
}

func (s *MemoryStore) DeleteFormLibraryItem(id string) (domain.FormLibraryItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for index, item := range s.formLibrary {
		if item.ID == id {
			s.formLibrary = append(s.formLibrary[:index], s.formLibrary[index+1:]...)
			return item, nil
		}
	}
	return domain.FormLibraryItem{}, ErrNotFound
}

func (s *MemoryStore) FollowupPlans() []domain.FollowupPlan {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]domain.FollowupPlan, 0, len(s.followPlans))
	for _, item := range s.followPlans {
		items = append(items, item)
	}
	return items
}

func (s *MemoryStore) FollowupPlanByID(id string) (domain.FollowupPlan, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.followPlans[id]
	return item, ok
}

func (s *MemoryStore) CreateFollowupPlan(plan domain.FollowupPlan) domain.FollowupPlan {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	if plan.ID == "" {
		plan.ID = uuid.NewString()
	}
	if plan.Status == "" {
		plan.Status = "active"
	}
	plan.CreatedAt = now
	plan.UpdatedAt = now
	s.followPlans[plan.ID] = plan
	return plan
}

func (s *MemoryStore) UpdateFollowupPlan(id string, patch domain.FollowupPlan) (domain.FollowupPlan, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	plan, ok := s.followPlans[id]
	if !ok {
		return domain.FollowupPlan{}, ErrNotFound
	}
	plan.Name = firstNonEmptyStore(patch.Name, plan.Name)
	plan.Scenario = firstNonEmptyStore(patch.Scenario, plan.Scenario)
	plan.DiseaseCode = patch.DiseaseCode
	plan.DepartmentID = patch.DepartmentID
	plan.FormTemplateID = firstNonEmptyStore(patch.FormTemplateID, plan.FormTemplateID)
	plan.TriggerType = firstNonEmptyStore(patch.TriggerType, plan.TriggerType)
	if patch.TriggerOffset != 0 {
		plan.TriggerOffset = patch.TriggerOffset
	}
	plan.Channel = firstNonEmptyStore(patch.Channel, plan.Channel)
	plan.AssigneeRole = firstNonEmptyStore(patch.AssigneeRole, plan.AssigneeRole)
	plan.Status = firstNonEmptyStore(patch.Status, plan.Status)
	plan.Rules = patch.Rules
	plan.UpdatedAt = time.Now().UTC()
	s.followPlans[id] = plan
	return plan, nil
}

func (s *MemoryStore) FollowupTasks(status, assigneeID string) []domain.FollowupTask {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]domain.FollowupTask, 0, len(s.followTasks))
	for _, item := range s.followTasks {
		if status != "" && item.Status != status {
			continue
		}
		if assigneeID != "" && item.AssigneeID != assigneeID {
			continue
		}
		items = append(items, item)
	}
	return items
}

func (s *MemoryStore) CreateFollowupTask(task domain.FollowupTask) domain.FollowupTask {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	if task.ID == "" {
		task.ID = uuid.NewString()
	}
	if patient, ok := s.patients[task.PatientID]; ok {
		task.PatientName = patient.Name
		task.PatientPhone = patient.Phone
	}
	if user, ok := s.users[task.AssigneeID]; ok {
		task.AssigneeName = user.DisplayName
	}
	if task.Status == "" {
		task.Status = "pending"
	}
	if task.Priority == "" {
		task.Priority = "normal"
	}
	task.CreatedAt = now
	task.UpdatedAt = now
	s.followTasks[task.ID] = task
	return task
}

func (s *MemoryStore) UpdateFollowupTask(id string, patch domain.FollowupTask) (domain.FollowupTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	task, ok := s.followTasks[id]
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
	if patient, ok := s.patients[task.PatientID]; ok {
		task.PatientName = patient.Name
		task.PatientPhone = patient.Phone
	}
	if user, ok := s.users[task.AssigneeID]; ok {
		task.AssigneeName = user.DisplayName
	}
	task.UpdatedAt = time.Now().UTC()
	s.followTasks[id] = task
	return task, nil
}

func (s *MemoryStore) GenerateFollowupTasks(planID string) ([]domain.FollowupTask, error) {
	s.mu.RLock()
	plan, ok := s.followPlans[planID]
	if !ok {
		s.mu.RUnlock()
		return nil, ErrNotFound
	}
	patients := make([]domain.Patient, 0, len(s.patients))
	assignees := s.followupAssigneesLocked(plan.AssigneeRole)
	for _, patient := range s.patients {
		if plan.DiseaseCode != "" && !strings.Contains(patient.Diagnosis, "高血压") && plan.DiseaseCode == "I10" {
			continue
		}
		patients = append(patients, patient)
	}
	s.mu.RUnlock()

	tasks := make([]domain.FollowupTask, 0, len(patients))
	for index, patient := range patients {
		assigneeID := ""
		if len(assignees) > 0 {
			assigneeID = assignees[index%len(assignees)].ID
		}
		tasks = append(tasks, s.CreateFollowupTask(domain.FollowupTask{
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
		}))
	}
	return tasks, nil
}

func (s *MemoryStore) followupAssigneesLocked(role string) []domain.User {
	counts := map[string]int{}
	for _, task := range s.followTasks {
		if task.AssigneeID != "" && task.Status != "completed" && task.Status != "failed" {
			counts[task.AssigneeID]++
		}
	}
	users := []domain.User{}
	for _, user := range s.users {
		for _, userRole := range user.Roles {
			if userRole == role || (role == "agent" && userRole == "admin") {
				users = append(users, user)
				break
			}
		}
	}
	for i := 0; i < len(users); i++ {
		for j := i + 1; j < len(users); j++ {
			if counts[users[j].ID] < counts[users[i].ID] || (counts[users[j].ID] == counts[users[i].ID] && users[j].ID < users[i].ID) {
				users[i], users[j] = users[j], users[i]
			}
		}
	}
	return users
}

func firstNonEmptyStore(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
